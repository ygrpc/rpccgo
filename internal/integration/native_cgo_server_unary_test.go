package integration

import (
	"os"
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
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/nativecgoserver\n\ngo 1.24.4\n\nrequire (\n\tgoogle.golang.org/protobuf v1.36.11\n\trpccgo v0.0.0\n)\n\nreplace rpccgo => "+repoRoot+"\n")
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.sum"), string(goSum))
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".server.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go")
		if !include {
			continue
		}
		writeFile(t, filepath.Join(tmp, name), generated.GetContent())
	}
	writeFile(t, filepath.Join(tmp, "test/v1/native_unary_stubs.go"), nativeUnaryStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_cgo_server_callbacks.go"), nativeCGOServerUnaryFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_cgo_server_unary_test.go"), nativeCGOServerUnaryFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeCGOServerUnary", "-count=1")
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

type sayHelloInput struct {
	NamePtr uintptr
	NameLen int32
	NameOwnership int32
	PayloadPtr uintptr
	PayloadLen int32
	PayloadOwnership int32
	Enabled int8
}

type sayHelloOutput struct {
	Accepted int8
	PayloadPtr uintptr
	PayloadLen int32
	NotePtr uintptr
	NoteLen int32
	ExtraPayloadPtr uintptr
	ExtraPayloadLen int32
}

func callSayHello(ctx context.Context, input *sayHelloInput, output *sayHelloOutput) int32 {
	if input == nil {
		input = &sayHelloInput{}
	}
	if output == nil {
		output = &sayHelloOutput{}
	}
	return CallGreeterSayHelloNativeUnary(ctx,
		input.NamePtr, input.NameLen, input.NameOwnership,
		input.PayloadPtr, input.PayloadLen, input.PayloadOwnership,
		input.Enabled,
		&output.Accepted,
		&output.PayloadPtr, &output.PayloadLen,
		&output.NotePtr, &output.NoteLen,
		&output.ExtraPayloadPtr, &output.ExtraPayloadLen,
	)
}

func (cgoOverrideGoServer) SayHello(context.Context, *rpcruntime.RpcString, *rpcruntime.RpcBytes, bool) (bool, []byte, string, []byte, error) {
	return true, []byte("go-server"), "go", nil, nil
}

func (cgoOverrideGoServer) SayUnsupported(context.Context, *rpcruntime.RpcString, *rpcruntime.RpcBytes, bool) ([]byte, string, []byte, error) {
	return nil, "", nil, nil
}

func TestNativeCGOServerUnaryRoutesThroughDispatcher(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerCFreeCallback()
	if err := registerGreeterCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterCGONativeServerCallbacks() error = %v", err)
	}

	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	got := unsafe.Slice((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen)
	if string(got) != "cgo-server:native" {
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
	if err := registerGreeterCGONativeServerNilCallback(); err == nil || !strings.Contains(err.Error(), "unary callback is missing") {
		t.Fatalf("registerGreeterCGONativeServerNilCallback() error = %v", err)
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
	errID := callSayHello(context.Background(), &sayHelloInput{}, &sayHelloOutput{})
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
	errID := callSayHello(context.Background(), &sayHelloInput{}, &sayHelloOutput{})
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
	errID := callSayHello(context.Background(), &sayHelloInput{}, &sayHelloOutput{})
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
	errID := callSayHello(context.Background(), &sayHelloInput{}, &sayHelloOutput{})
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
	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr:    uintptr(unsafe.Pointer(&name[0])),
		NameLen:    int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled:    1,
	}
	errID := callSayHello(context.Background(), input, &sayHelloOutput{})
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
	assertUnaryPayload(t, "cgo-server:native")
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
	assertUnaryPayload(t, "cgo-server:native")

	if _, err := v1.RegisterGreeterGoNativeServer(cgoOverrideGoServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() second error = %v", err)
	}
	assertUnaryPayload(t, "go-server")
}

func assertUnaryPayload(t *testing.T, want string) {
	t.Helper()
	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr:    uintptr(unsafe.Pointer(&name[0])),
		NameLen:    int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled:    1,
	}
	output := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), input, output); errID != 0 {
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

typedef int32_t (*GreeterSayHelloCGONativeUnaryCallback)(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, int8_t* outAccepted, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outExtraPayloadPtr, int32_t* outExtraPayloadLen, int32_t* outExtraPayloadOwnership);
typedef int32_t (*GreeterSayUnsupportedCGONativeUnaryCallback)(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outUnsupportedPtr, int32_t* outUnsupportedLen, int32_t* outUnsupportedOwnership);

typedef struct GreeterCGONativeServerCallbacks {
GreeterSayHelloCGONativeUnaryCallback SayHello;
GreeterSayUnsupportedCGONativeUnaryCallback SayUnsupported;
} GreeterCGONativeServerCallbacks;

static int32_t greeterSayHelloCallback(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, int8_t* outAccepted, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outExtraPayloadPtr, int32_t* outExtraPayloadLen, int32_t* outExtraPayloadOwnership) {
	if (outAccepted == NULL || outPayloadPtr == NULL || outPayloadLen == NULL || outPayloadOwnership == NULL) {
		char msg[] = "callback output missing";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	if (NameLen != 6 || PayloadLen != 5 || Enabled != 1) {
		char msg[] = "request did not reach cgo callback";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	char* resp = (char*)malloc(17);
	if (resp == NULL) {
		char msg[] = "callback malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	resp[0] = 'c'; resp[1] = 'g'; resp[2] = 'o'; resp[3] = '-'; resp[4] = 's'; resp[5] = 'e'; resp[6] = 'r'; resp[7] = 'v'; resp[8] = 'e'; resp[9] = 'r'; resp[10] = ':'; resp[11] = 'n'; resp[12] = 'a'; resp[13] = 't'; resp[14] = 'i'; resp[15] = 'v'; resp[16] = 'e';
	*outAccepted = 1;
	*outPayloadPtr = (uintptr_t)resp;
	*outPayloadLen = 17;
	*outPayloadOwnership = 1;
	return 0;
}

static int32_t greeterSayUnsupportedCallback(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outUnsupportedPtr, int32_t* outUnsupportedLen, int32_t* outUnsupportedOwnership) {
	return 0;
}

static int32_t greeterErrorCallback(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, int8_t* outAccepted, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outExtraPayloadPtr, int32_t* outExtraPayloadLen, int32_t* outExtraPayloadOwnership) {
	char msg[] = "callback exploded";
	return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
}

static int32_t greeterUnknownErrorCallback(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, int8_t* outAccepted, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outExtraPayloadPtr, int32_t* outExtraPayloadLen, int32_t* outExtraPayloadOwnership) {
	return 99999;
}

static int32_t greeterNegativeLengthCallback(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, int8_t* outAccepted, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outExtraPayloadPtr, int32_t* outExtraPayloadLen, int32_t* outExtraPayloadOwnership) {
	char* resp = (char*)malloc(1);
	if (resp == NULL) {
		char msg[] = "callback malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	*outPayloadPtr = (uintptr_t)resp;
	*outPayloadLen = -1;
	*outPayloadOwnership = 1;
	return 0;
}

static int32_t greeterPartialErrorCallback(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership, int8_t Enabled, int8_t* outAccepted, uintptr_t* outPayloadPtr, int32_t* outPayloadLen, int32_t* outPayloadOwnership, uintptr_t* outNotePtr, int32_t* outNoteLen, int32_t* outNoteOwnership, uintptr_t* outExtraPayloadPtr, int32_t* outExtraPayloadLen, int32_t* outExtraPayloadOwnership) {
	char* resp = (char*)malloc(1);
	if (resp == NULL) {
		char msg[] = "callback malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	*outPayloadPtr = (uintptr_t)resp;
	*outPayloadLen = 1;
	*outPayloadOwnership = 1;
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
	"errors"
	"sync/atomic"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func registerGreeterCGONativeServerCallbacks() error {
	callbacks := C.greeterCallbacks()
	return registerGreeterCGONativeServerCallbacksTable(callbacks)
}

func registerGreeterCGONativeServerCallbacksThenClearLocalTable() error {
	callbacks := C.greeterCallbacks()
	err := registerGreeterCGONativeServerCallbacksTable(callbacks)
	callbacks.SayHello = nil
	callbacks.SayUnsupported = nil
	return err
}

func registerGreeterCGONativeServerNilCallback() error {
	return nativeCGORegistrationError(rpccgo_native_testv1_Greeter_SayHello_register(nil))
}

func registerGreeterCGONativeServerEmptyCallbacks() error {
	return registerGreeterCGONativeServerCallbacksTable(C.GreeterCGONativeServerCallbacks{})
}

func registerGreeterCGONativeServerErrorCallback() error {
	callbacks := C.greeterErrorCallbacks()
	return registerGreeterCGONativeServerCallbacksTable(callbacks)
}

func registerGreeterCGONativeServerUnknownErrorCallback() error {
	callbacks := C.greeterUnknownErrorCallbacks()
	return registerGreeterCGONativeServerCallbacksTable(callbacks)
}

func registerGreeterCGONativeServerNegativeLengthCallback() error {
	callbacks := C.greeterNegativeLengthCallbacks()
	return registerGreeterCGONativeServerCallbacksTable(callbacks)
}

func registerGreeterCGONativeServerPartialErrorCallback() error {
	callbacks := C.greeterPartialErrorCallbacks()
	return registerGreeterCGONativeServerCallbacksTable(callbacks)
}

func registerGreeterCGONativeServerCallbacksTable(callbacks C.GreeterCGONativeServerCallbacks) error {
	for _, errID := range []C.int32_t{
		rpccgo_native_testv1_Greeter_SayHello_register(callbacks.SayHello),
		rpccgo_native_testv1_Greeter_SayUnsupported_register(callbacks.SayUnsupported),
	} {
		if err := nativeCGORegistrationError(errID); err != nil {
			return err
		}
	}
	return nil
}

func nativeCGORegistrationError(errID C.int32_t) error {
	if errID == 0 {
		return nil
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok {
		return nil
	}
	if ptr != 0 {
		rpcruntime.Release(ptr)
	}
	return errors.New(string(text))
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
