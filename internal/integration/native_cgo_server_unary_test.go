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
	plugin := newNativeUnaryTestPlugin(t)
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
	writeFile(t, filepath.Join(tmp, "test/v1/native_cgo_server_unary_test.go"), nativeCGOServerUnaryFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1", "-run", "TestNativeCGOServerUnary", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native cgo server unary fixture failed: %v\n%s", err, out)
	}
}

const nativeCGOServerUnaryFixtureTestSource = `package testv1

import (
	context "context"
	errors "errors"
	strings "strings"
	"testing"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

type cgoOverrideGoServer struct{}

func (cgoOverrideGoServer) SayHello(context.Context, *HelloRequest) (*HelloReply, error) {
	return &HelloReply{Accepted: true, Payload: []byte("go-server"), Note: "go"}, nil
}

func (cgoOverrideGoServer) SayUnsupported(context.Context, *HelloRequest) (*UnsupportedReply, error) {
	return &UnsupportedReply{}, nil
}

func TestNativeCGOServerUnaryRoutesThroughDispatcher(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	rpcruntime.RegisterFreeCallback(func(unsafe.Pointer) {})
	callbacks := &GreeterCGONativeServerCallbacks{
		SayHello: func(ctx context.Context, input *GreeterSayHelloCGONativeUnaryRequest, output *GreeterSayHelloCGONativeUnaryResponse) int32 {
			if input == nil || output == nil {
				return int32(rpcruntime.StoreError(errors.New("callback input/output missing")))
			}
			name := unsafe.String((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen)
			payload := unsafe.Slice((*byte)(unsafe.Pointer(input.PayloadPtr)), input.PayloadLen)
			if name != "stage3" || string(payload) != "bytes" || input.Enabled != 1 {
				return int32(rpcruntime.StoreError(errors.New("request did not reach cgo callback")))
			}
			resp := []byte("cgo-server:" + name)
			output.Accepted = 1
			output.PayloadPtr = uintptr(unsafe.Pointer(&resp[0]))
			output.PayloadLen = int32(len(resp))
			output.PayloadOwnership = 1
			return 0
		},
		SayUnsupported: func(context.Context, *GreeterSayUnsupportedCGONativeUnaryRequest, *GreeterSayUnsupportedCGONativeUnaryResponse) int32 {
			return 0
		},
	}
	if _, err := RegisterGreeterCGONativeServer(callbacks); err != nil {
		t.Fatalf("RegisterGreeterCGONativeServer() error = %v", err)
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
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.ExtraPayloadPtr)
}

func TestNativeCGOServerUnaryRegistrationValidation(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	if _, err := RegisterGreeterCGONativeServer(nil); err == nil || !strings.Contains(err.Error(), "callbacks are nil") {
		t.Fatalf("RegisterGreeterCGONativeServer(nil) error = %v", err)
	}
	_, err := RegisterGreeterCGONativeServer(&GreeterCGONativeServerCallbacks{})
	if err == nil || !strings.Contains(err.Error(), "unary callback is missing") {
		t.Fatalf("RegisterGreeterCGONativeServer(empty) error = %v", err)
	}
}

func TestNativeCGOServerUnaryCallbackErrorPropagates(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	if _, err := RegisterGreeterCGONativeServer(&GreeterCGONativeServerCallbacks{
		SayHello: func(ctx context.Context, input *GreeterSayHelloCGONativeUnaryRequest, output *GreeterSayHelloCGONativeUnaryResponse) int32 {
			return int32(rpcruntime.StoreError(errors.New("callback exploded")))
		},
		SayUnsupported: func(context.Context, *GreeterSayUnsupportedCGONativeUnaryRequest, *GreeterSayUnsupportedCGONativeUnaryResponse) int32 {
			return 0
		},
	}); err != nil {
		t.Fatalf("RegisterGreeterCGONativeServer() error = %v", err)
	}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), &GreeterSayHelloNativeUnaryInput{}, &GreeterSayHelloNativeUnaryOutput{})
	if errID == 0 {
		t.Fatal("callback error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "callback exploded") {
		t.Fatalf("callback error text = %q, ok=%v", text, ok)
	}
}

func TestNativeCGOServerUnaryRegistrationOverridesGoServer(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	rpcruntime.RegisterFreeCallback(func(unsafe.Pointer) {})
	if _, err := RegisterGreeterGoNativeServer(cgoOverrideGoServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	callbacks := &GreeterCGONativeServerCallbacks{
		SayHello: func(ctx context.Context, input *GreeterSayHelloCGONativeUnaryRequest, output *GreeterSayHelloCGONativeUnaryResponse) int32 {
			resp := []byte("cgo-after-go")
			output.PayloadPtr = uintptr(unsafe.Pointer(&resp[0]))
			output.PayloadLen = int32(len(resp))
			output.PayloadOwnership = 1
			return 0
		},
		SayUnsupported: func(context.Context, *GreeterSayUnsupportedCGONativeUnaryRequest, *GreeterSayUnsupportedCGONativeUnaryResponse) int32 {
			return 0
		},
	}
	if _, err := RegisterGreeterCGONativeServer(callbacks); err != nil {
		t.Fatalf("RegisterGreeterCGONativeServer() error = %v", err)
	}
	assertUnaryPayload(t, "cgo-after-go")

	if _, err := RegisterGreeterGoNativeServer(cgoOverrideGoServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() second error = %v", err)
	}
	assertUnaryPayload(t, "go-server")
}

func assertUnaryPayload(t *testing.T, want string) {
	t.Helper()
	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(context.Background(), &GreeterSayHelloNativeUnaryInput{}, output); errID != 0 {
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
