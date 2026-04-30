package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rpccgo/internal/generator"
)

func TestNativeCGOServerUnaryRoutesThroughDispatcher(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeUnaryTestPluginForPackage(t, "example.com/nativecgoserver/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/nativecgoserver\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n")
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".server.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
		if !include {
			continue
		}
		writeFile(t, filepath.Join(tmp, name), generated.GetContent())
	}
	writeFile(t, filepath.Join(tmp, "test/v1/native_unary_stubs.go"), nativeUnaryStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_cgo_server_callbacks.go"), nativeCGOServerUnaryFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_cgo_server_unary_test.go"), nativeCGOServerUnaryFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "TestNativeCGOServerUnary", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native cgo server unary fixture failed: %v\n%s", err, out)
	}
}

const nativeCGOServerUnaryFixtureTestSource = `package main

import (
	context "context"
	runtime "runtime"
	strings "strings"
	"testing"
	"unsafe"

	v1 "example.com/nativecgoserver/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type cgoOverrideGoServer struct{}

func (cgoOverrideGoServer) SayHello(context.Context, *v1.HelloRequest) (*v1.HelloReply, error) {
	return &v1.HelloReply{Accepted: true, Payload: []byte("go-server"), Note: "go"}, nil
}

func (cgoOverrideGoServer) SayUnsupported(context.Context, *v1.HelloRequest) (*v1.UnsupportedReply, error) {
	return &v1.UnsupportedReply{}, nil
}

func TestNativeCGOServerUnaryRoutesThroughDispatcher(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerCFreeCallback()
	if err := registerGreeterCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerCallbacks() error = %v", err)
	}

	name := []byte("stage3")
	payload := []byte("bytes")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	got := unsafe.Slice((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen)
	if string(got) != "cgo-server:stage3" {
		t.Fatalf("Payload = %q", got)
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
		runtime.Gosched()
	}
	if got := frees(); got != 1 {
		t.Fatalf("free callback calls after GC = %d, want 1", got)
	}
}

func TestNativeCGOServerUnaryRegistrationValidation(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := RegisterGreeterCGONativeServer(nil); err == nil || !strings.Contains(err.Error(), "callbacks are nil") {
		t.Fatalf("RegisterGreeterCGONativeServer(nil) error = %v", err)
	}
	err := registerGreeterCGONativeServerEmptyCallbacks()
	if err == nil || !strings.Contains(err.Error(), "unary callback is missing") {
		t.Fatalf("registerGreeterCGONativeServerEmptyCallbacks() error = %v", err)
	}
}

func TestNativeCGOServerUnaryCallbackErrorPropagates(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if err := registerGreeterCGONativeServerErrorCallback(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerErrorCallback() error = %v", err)
	}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), &GreeterSayHelloNativeUnaryInput{}, &GreeterSayHelloNativeUnaryOutput{})
	if errID == 0 {
		t.Fatal("callback error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "callback exploded") {
		t.Fatalf("callback error text = %q, ok=%v", text, ok)
	}
	_, _, ok = rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if ok {
		t.Fatal("callback error id should be consumed after first read")
	}
}

func TestNativeCGOServerUnaryUnknownCallbackErrorIDIsExplicit(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if err := registerGreeterCGONativeServerUnknownErrorCallback(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerUnknownErrorCallback() error = %v", err)
	}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), &GreeterSayHelloNativeUnaryInput{}, &GreeterSayHelloNativeUnaryOutput{})
	if errID == 0 {
		t.Fatal("unknown callback error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "unknown error id 99999") {
		t.Fatalf("unknown callback error text = %q, ok=%v", text, ok)
	}
}

func TestNativeCGOServerUnaryOwnedOutputCleanupOnDecodeError(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerCFreeCallback()
	if err := registerGreeterCGONativeServerNegativeLengthCallback(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerNegativeLengthCallback() error = %v", err)
	}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), &GreeterSayHelloNativeUnaryInput{}, &GreeterSayHelloNativeUnaryOutput{})
	if errID == 0 {
		t.Fatal("negative length returned errID 0")
	}
	if got := frees(); got != 1 {
		t.Fatalf("free callback calls = %d, want 1", got)
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
		runtime.Gosched()
	}
	if got := frees(); got != 1 {
		t.Fatalf("free callback calls after GC = %d, want 1", got)
	}
}

func TestNativeCGOServerUnaryOwnedOutputCleanupOnCallbackError(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerCFreeCallback()
	if err := registerGreeterCGONativeServerPartialErrorCallback(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerPartialErrorCallback() error = %v", err)
	}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), &GreeterSayHelloNativeUnaryInput{}, &GreeterSayHelloNativeUnaryOutput{})
	if errID == 0 {
		t.Fatal("partial callback error returned errID 0")
	}
	if got := frees(); got != 1 {
		t.Fatalf("free callback calls = %d, want 1", got)
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
		runtime.Gosched()
	}
	if got := frees(); got != 1 {
		t.Fatalf("free callback calls after GC = %d, want 1", got)
	}
}

func TestNativeCGOServerUnaryOwnedOutputCleanupErrorPropagates(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	if err := registerGreeterCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerCallbacks() error = %v", err)
	}
	name := []byte("stage3")
	payload := []byte("bytes")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr:    uintptr(unsafe.Pointer(&name[0])),
		NameLen:    int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled:    1,
	}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, &GreeterSayHelloNativeUnaryOutput{})
	if errID == 0 {
		t.Fatal("missing free callback returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "ownership requires registered free func") {
		t.Fatalf("cleanup error text = %q, ok=%v", text, ok)
	}
}

func TestNativeCGOServerUnaryCallbackTableIsCopiedAtRegistration(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	registerCFreeCallback()
	if err := registerGreeterCGONativeServerCallbacksThenClearLocalTable(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerCallbacksThenClearLocalTable() error = %v", err)
	}
	assertUnaryPayload(t, "cgo-server:stage3")
}

func TestNativeCGOServerUnaryRegistrationOverridesGoServer(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	registerCFreeCallback()
	if _, err := v1.RegisterGreeterGoNativeServer(cgoOverrideGoServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	if err := registerGreeterCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerCallbacks() error = %v", err)
	}
	assertUnaryPayload(t, "cgo-server:stage3")

	if _, err := v1.RegisterGreeterGoNativeServer(cgoOverrideGoServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() second error = %v", err)
	}
	assertUnaryPayload(t, "go-server")
}

func assertUnaryPayload(t *testing.T, want string) {
	t.Helper()
	name := []byte("stage3")
	payload := []byte("bytes")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr:    uintptr(unsafe.Pointer(&name[0])),
		NameLen:    int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled:    1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	got := unsafe.Slice((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen)
	if string(got) != want {
		t.Fatalf("Payload = %q, want %q", got, want)
	}
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.ExtraPayloadPtr)
}
`

const nativeCGOServerUnaryFixtureCallbackSource = `package main

/*
#include <stdint.h>
#include <stdlib.h>

extern int32_t StoreGreeterCGONativeServerErrorTextForExport(char* text, int32_t textLen);

typedef struct GreeterSayHelloCGONativeUnaryRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t PayloadPtr;
int32_t PayloadLen;
int8_t Enabled;
} GreeterSayHelloCGONativeUnaryRequest;

typedef struct GreeterSayHelloCGONativeUnaryResponse {
int8_t Accepted;
uintptr_t PayloadPtr;
int32_t PayloadLen;
int32_t PayloadOwnership;
uintptr_t NotePtr;
int32_t NoteLen;
int32_t NoteOwnership;
uintptr_t ExtraPayloadPtr;
int32_t ExtraPayloadLen;
int32_t ExtraPayloadOwnership;
} GreeterSayHelloCGONativeUnaryResponse;

typedef struct GreeterSayUnsupportedCGONativeUnaryRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t PayloadPtr;
int32_t PayloadLen;
int8_t Enabled;
} GreeterSayUnsupportedCGONativeUnaryRequest;

typedef struct GreeterSayUnsupportedCGONativeUnaryResponse {
uintptr_t PayloadPtr;
int32_t PayloadLen;
int32_t PayloadOwnership;
uintptr_t NotePtr;
int32_t NoteLen;
int32_t NoteOwnership;
uintptr_t Unsupported;
} GreeterSayUnsupportedCGONativeUnaryResponse;

typedef int32_t (*GreeterSayHelloCGONativeUnaryCallback)(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output);
typedef int32_t (*GreeterSayUnsupportedCGONativeUnaryCallback)(GreeterSayUnsupportedCGONativeUnaryRequest* input, GreeterSayUnsupportedCGONativeUnaryResponse* output);

typedef struct GreeterCGONativeServerCallbacks {
GreeterSayHelloCGONativeUnaryCallback SayHello;
GreeterSayUnsupportedCGONativeUnaryCallback SayUnsupported;
} GreeterCGONativeServerCallbacks;

static int32_t greeterSayHelloCallback(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	if (input == NULL || output == NULL) {
		char msg[] = "callback input/output missing";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	if (input->NameLen != 6 || input->PayloadLen != 5 || input->Enabled != 1) {
		char msg[] = "request did not reach cgo callback";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	char* resp = (char*)malloc(17);
	if (resp == NULL) {
		char msg[] = "callback malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	resp[0] = 'c'; resp[1] = 'g'; resp[2] = 'o'; resp[3] = '-'; resp[4] = 's'; resp[5] = 'e'; resp[6] = 'r'; resp[7] = 'v'; resp[8] = 'e'; resp[9] = 'r'; resp[10] = ':'; resp[11] = 's'; resp[12] = 't'; resp[13] = 'a'; resp[14] = 'g'; resp[15] = 'e'; resp[16] = '3';
	output->Accepted = 1;
	output->PayloadPtr = (uintptr_t)resp;
	output->PayloadLen = 17;
	output->PayloadOwnership = 1;
	return 0;
}

static int32_t greeterSayUnsupportedCallback(GreeterSayUnsupportedCGONativeUnaryRequest* input, GreeterSayUnsupportedCGONativeUnaryResponse* output) {
	return 0;
}

static int32_t greeterErrorCallback(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	char msg[] = "callback exploded";
	return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
}

static int32_t greeterUnknownErrorCallback(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	return 99999;
}

static int32_t greeterNegativeLengthCallback(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	char* resp = (char*)malloc(1);
	if (resp == NULL) {
		char msg[] = "callback malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	output->PayloadPtr = (uintptr_t)resp;
	output->PayloadLen = -1;
	output->PayloadOwnership = 1;
	return 0;
}

static int32_t greeterPartialErrorCallback(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	char* resp = (char*)malloc(1);
	if (resp == NULL) {
		char msg[] = "callback malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	output->PayloadPtr = (uintptr_t)resp;
	output->PayloadLen = 1;
	output->PayloadOwnership = 1;
	char msg[] = "partial failure";
	return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
}

static GreeterCGONativeServerCallbacks greeterCallbacks(void) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.SayHello = greeterSayHelloCallback;
	callbacks.SayUnsupported = greeterSayUnsupportedCallback;
	return callbacks;
}

static GreeterCGONativeServerCallbacks greeterCallbacksWithSayHello(GreeterSayHelloCGONativeUnaryCallback sayHello) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.SayHello = sayHello;
	callbacks.SayUnsupported = greeterSayUnsupportedCallback;
	return callbacks;
}

static GreeterCGONativeServerCallbacks greeterErrorCallbacks(void) {
	return greeterCallbacksWithSayHello(greeterErrorCallback);
}

static GreeterCGONativeServerCallbacks greeterUnknownErrorCallbacks(void) {
	return greeterCallbacksWithSayHello(greeterUnknownErrorCallback);
}

static GreeterCGONativeServerCallbacks greeterNegativeLengthCallbacks(void) {
	return greeterCallbacksWithSayHello(greeterNegativeLengthCallback);
}

static GreeterCGONativeServerCallbacks greeterPartialErrorCallbacks(void) {
	return greeterCallbacksWithSayHello(greeterPartialErrorCallback);
}
*/
import "C"

import (
	"sync/atomic"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func registerGreeterCGONativeServerCallbacks() error {
	callbacks := C.greeterCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func registerGreeterCGONativeServerCallbacksThenClearLocalTable() error {
	callbacks := C.greeterCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	callbacks.SayHello = nil
	callbacks.SayUnsupported = nil
	return err
}

func registerGreeterCGONativeServerEmptyCallbacks() error {
	_, err := RegisterGreeterCGONativeServer(&C.GreeterCGONativeServerCallbacks{})
	return err
}

func registerGreeterCGONativeServerErrorCallback() error {
	callbacks := C.greeterErrorCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func registerGreeterCGONativeServerUnknownErrorCallback() error {
	callbacks := C.greeterUnknownErrorCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func registerGreeterCGONativeServerNegativeLengthCallback() error {
	callbacks := C.greeterNegativeLengthCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func registerGreeterCGONativeServerPartialErrorCallback() error {
	callbacks := C.greeterPartialErrorCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func registerCFreeCallback() func() int32 {
	var frees atomic.Int32
	rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {
		frees.Add(1)
		C.free(ptr)
	})
	return frees.Load
}
`
