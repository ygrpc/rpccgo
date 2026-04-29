package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeUnaryClientRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/nativeunary\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n")
	writeFile(t, filepath.Join(tmp, "nativeunary/runtime.rpccgo.go"), nativeUnaryRuntimeSource)
	writeFile(t, filepath.Join(tmp, "nativeunary/server.native.rpccgo.go"), nativeUnaryServerSource)
	writeFile(t, filepath.Join(tmp, "nativeunary/client.cgo.rpccgo.go"), nativeUnaryClientSource)
	writeFile(t, filepath.Join(tmp, "nativeunary/native_unary_test.go"), nativeUnaryFixtureTestSource)

	cmd := exec.Command("go", "test", "./nativeunary", "-run", "TestNativeUnary", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native unary fixture failed: %v\n%s", err, out)
	}
}

func writeFile(t *testing.T, target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

const nativeUnaryRuntimeSource = `package nativeunary

import (
	context "context"
	rpcruntime "rpccgo/rpcruntime"
)

type GreeterNativeAdapter interface {
	SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error)
}

var greeterDispatcher rpcruntime.Dispatcher[GreeterNativeAdapter]

func registerGreeterActiveServer(kind rpcruntime.ServerKind, adapter GreeterNativeAdapter) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	return greeterDispatcher.Register(kind, rpcruntime.ServerContractNative, adapter)
}
`

const nativeUnaryServerSource = `package nativeunary

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

var greeterNativeRequestBridgeNotImplemented = errors.New("rpccgo: native request bridge is not implemented")

type HelloRequest struct {
	Name string
	Enabled bool
}

type HelloReply struct {
	Accepted bool
	Payload []byte
}

type GreeterNativeServer interface {
	SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error)
}

type greeterGoNativeAdapter struct {
	server GreeterNativeServer
}

func (a *greeterGoNativeAdapter) SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	if req == nil {
		return nil, greeterNativeRequestBridgeNotImplemented
	}
	return a.server.SayHello(ctx, req)
}

func RegisterGreeterGoNativeServer(server GreeterNativeServer) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	if server == nil {
		return rpcruntime.AdapterSnapshot[GreeterNativeAdapter]{}, errors.New("rpccgo: Greeter go native server is nil")
	}
	return registerGreeterActiveServer(rpcruntime.ServerKindGoNative, &greeterGoNativeAdapter{server: server})
}
`

const nativeUnaryClientSource = `package nativeunary

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
)

type GreeterSayHelloNativeUnaryInput struct {
	NamePtr uintptr
	NameLen int32
	NameOwnership int32
	Enabled int8
}

type GreeterSayHelloNativeUnaryOutput struct {
	Accepted int8
	PayloadPtr uintptr
	PayloadLen int32
}

func CallGreeterSayHelloNativeUnary(ctx context.Context, input *GreeterSayHelloNativeUnaryInput, output *GreeterSayHelloNativeUnaryOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary client input is nil")))
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary client output is nil")))
	}
	req := &HelloRequest{}
	req.Name = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0).SafeString()
	req.Enabled = input.Enabled != 0
	var resp *HelloReply
	err := greeterDispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterNativeAdapter]) error {
		var callErr error
		resp, callErr = snapshot.Adapter.SayHello(ctx, req)
		return callErr
	})
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if resp == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary server returned nil response")))
	}
	if resp.Accepted {
		output.Accepted = 1
	} else {
		output.Accepted = 0
	}
	ptr, err := rpcruntime.PinBytes(resp.Payload)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	length, err := rpcruntime.LengthToInt32(len(resp.Payload))
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	output.PayloadPtr = ptr
	output.PayloadLen = length
	return 0
}
`

const nativeUnaryFixtureTestSource = `package nativeunary

import (
	context "context"
	errors "errors"
	strings "strings"
	"testing"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

type recordingServer struct {
	called bool
	err error
}

func (s *recordingServer) SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	s.called = true
	if s.err != nil {
		return nil, s.err
	}
	if req.Name != "stage3" || !req.Enabled {
		return nil, errors.New("request did not cross native bridge")
	}
	return &HelloReply{Accepted: true, Payload: []byte("dispatcher:"+req.Name)}, nil
}

func TestNativeUnaryClientRoutesToGoNativeServer(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	server := &recordingServer{}
	if _, err := RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	name := []byte("stage3")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		Enabled: 1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if !server.called {
		t.Fatal("server was not called through dispatcher")
	}
	if output.Accepted != 1 {
		t.Fatalf("Accepted = %d, want 1", output.Accepted)
	}
	got := unsafe.Slice((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen)
	if string(got) != "dispatcher:stage3" {
		t.Fatalf("Payload = %q", got)
	}
	rpcruntime.Release(output.PayloadPtr)
}

func TestNativeUnaryMissingActiveServerStoresError(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	input := &GreeterSayHelloNativeUnaryInput{}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("missing active server returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "active server") {
		t.Fatalf("missing active server error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryServerErrorStoresError(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	if _, err := RegisterGreeterGoNativeServer(&recordingServer{err: errors.New("server exploded")}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &GreeterSayHelloNativeUnaryInput{}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("server error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "server exploded") {
		t.Fatalf("server error text = %q, ok=%v", text, ok)
	}
}
`
