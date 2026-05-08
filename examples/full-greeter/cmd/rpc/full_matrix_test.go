package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"example.com/rpccgo-full/internal/backend"
	greeterv1 "example.com/rpccgo-full/proto"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

func TestFullGreeterTransportAndStreamingMatrix(t *testing.T) {
	ctx := context.Background()
	registerNativeServer(t)

	t.Run("native_cgo", func(t *testing.T) {
		assertNativeUnary(t, ctx, "native", "local", "hello native from local")
		assertNativeCollect(t, ctx, []string{"ada", "grace"}, "collect:ada,grace")
		assertNativeBroadcast(t, ctx, "stream", []string{"broadcast[0]:stream", "broadcast[1]:stream"})
		assertNativeChat(t, ctx, "bidi", "chat:bidi")
	})

	t.Run("message_cgo", func(t *testing.T) {
		assertMessageUnary(t, ctx, "message", "local", "hello message from local")
		assertMessageCollect(t, ctx, []string{"client", "stream"}, "collect:client,stream")
		assertMessageBroadcast(t, ctx, "server", []string{"broadcast[0]:server", "broadcast[1]:server"})
		assertMessageChat(t, ctx, "bidi-message", "chat:bidi-message")
	})

	t.Run("connect_remote", func(t *testing.T) {
		remote := startExampleServer(t)

		if _, err := greeterv1.RegisterGreeterConnectRemoteServer(httpClient(), "http://"+remote.connectAddr); err != nil {
			t.Fatalf("RegisterGreeterConnectRemoteServer() error = %v", err)
		}
		assertMessageUnary(t, ctx, "connect", "remote", "hello connect from remote")
		assertMessageCollect(t, ctx, []string{"connect", "collect"}, "collect:connect,collect")
		assertMessageBroadcast(t, ctx, "connect-broadcast", []string{"broadcast[0]:connect-broadcast", "broadcast[1]:connect-broadcast"})
		assertMessageChat(t, ctx, "connect-chat", "chat:connect-chat")
	})

	t.Run("grpc_remote", func(t *testing.T) {
		remote := startExampleServer(t)
		conn := newGRPCConn(t, remote.grpcAddr)
		defer conn.Close()

		if _, err := greeterv1.RegisterGreeterGRPCRemoteServer(conn); err != nil {
			t.Fatalf("RegisterGreeterGRPCRemoteServer() error = %v", err)
		}
		assertMessageUnary(t, ctx, "grpc", "remote", "hello grpc from remote")
		assertMessageCollect(t, ctx, []string{"grpc", "collect"}, "collect:grpc,collect")
		assertMessageBroadcast(t, ctx, "grpc-broadcast", []string{"broadcast[0]:grpc-broadcast", "broadcast[1]:grpc-broadcast"})
		assertMessageChat(t, ctx, "grpc-chat", "chat:grpc-chat")
	})
}

func registerNativeServer(t *testing.T) {
	t.Helper()
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
}

func newGRPCConn(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	return conn
}

type exampleServer struct {
	connectAddr string
	grpcAddr    string
}

func startExampleServer(t *testing.T) exampleServer {
	t.Helper()

	connectAddr := reserveTCPAddr(t)
	grpcAddr := reserveTCPAddr(t)
	serverBin := filepath.Join(t.TempDir(), "full-example-server-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	build := exec.Command("go", "build", "-o", serverBin, "./cmd/server")
	build.Dir = "../.."
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build full example server error = %v\n%s", err, out)
	}

	cmd := exec.Command(serverBin)
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"RPCCGO_FULL_CONNECT_ADDR="+connectAddr,
		"RPCCGO_FULL_GRPC_ADDR="+grpcAddr,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start full example server error = %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	waitForTCP(t, connectAddr)
	waitForTCP(t, grpcAddr)
	return exampleServer{connectAddr: connectAddr, grpcAddr: grpcAddr}
}

func reserveTCPAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp addr error = %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close reserved listener error = %v", err)
	}
	return addr
}

func waitForTCP(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not start listening on %s", addr)
}

func httpClient() interface {
	Do(*http.Request) (*http.Response, error)
} {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
}

func assertNativeUnary(t *testing.T, ctx context.Context, name, city, want string) {
	t.Helper()

	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(ctx, nativeUnaryInput(name, city), output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() error id = %d", errID)
	}
	assertNativeOutput(t, output.MessagePtr, output.MessageLen, want)
}

func assertNativeCollect(t *testing.T, ctx context.Context, names []string, want string) {
	t.Helper()

	handle, errID := StartGreeterCollectNativeClientStream(ctx)
	if errID != 0 {
		t.Fatalf("StartGreeterCollectNativeClientStream() error id = %d", errID)
	}
	for _, name := range names {
		if errID := SendGreeterCollectNativeClientStream(ctx, handle, nativeCollectInput(name, "local")); errID != 0 {
			t.Fatalf("SendGreeterCollectNativeClientStream() error id = %d", errID)
		}
	}
	output := &GreeterCollectNativeClientStreamOutput{}
	if errID := FinishGreeterCollectNativeClientStream(ctx, handle, output); errID != 0 {
		t.Fatalf("FinishGreeterCollectNativeClientStream() error id = %d", errID)
	}
	assertNativeOutput(t, output.MessagePtr, output.MessageLen, want)
}

func assertNativeBroadcast(t *testing.T, ctx context.Context, name string, wants []string) {
	t.Helper()

	handle, errID := StartGreeterBroadcastNativeServerStream(ctx, nativeBroadcastInput(name, "local"))
	if errID != 0 {
		t.Fatalf("StartGreeterBroadcastNativeServerStream() error id = %d", errID)
	}
	for _, want := range wants {
		output := &GreeterBroadcastNativeServerStreamOutput{}
		if errID := ReadGreeterBroadcastNativeServerStream(ctx, handle, output); errID != 0 {
			t.Fatalf("ReadGreeterBroadcastNativeServerStream() error id = %d", errID)
		}
		assertNativeOutput(t, output.MessagePtr, output.MessageLen, want)
	}
	if errID := DoneGreeterBroadcastNativeServerStream(ctx, handle); errID != 0 {
		t.Fatalf("DoneGreeterBroadcastNativeServerStream() error id = %d", errID)
	}
}

func assertNativeChat(t *testing.T, ctx context.Context, name, want string) {
	t.Helper()

	handle, errID := StartGreeterChatNativeBidiStream(ctx)
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() error id = %d", errID)
	}
	if errID := SendGreeterChatNativeBidiStream(ctx, handle, nativeChatInput(name, "local")); errID != 0 {
		t.Fatalf("SendGreeterChatNativeBidiStream() error id = %d", errID)
	}
	output := &GreeterChatNativeBidiStreamOutput{}
	if errID := ReadGreeterChatNativeBidiStream(ctx, handle, output); errID != 0 {
		t.Fatalf("ReadGreeterChatNativeBidiStream() error id = %d", errID)
	}
	assertNativeOutput(t, output.MessagePtr, output.MessageLen, want)
	if errID := CloseSendGreeterChatNativeBidiStream(ctx, handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatNativeBidiStream() error id = %d", errID)
	}
	if errID := DoneGreeterChatNativeBidiStream(ctx, handle); errID != 0 {
		t.Fatalf("DoneGreeterChatNativeBidiStream() error id = %d", errID)
	}
}

func assertMessageUnary(t *testing.T, ctx context.Context, name, city, want string) {
	t.Helper()

	output := &GreeterMessageOutput{}
	request := messageRequestBytes(t, name, city)
	if errID := CallGreeterSayHelloMessageUnary(ctx, bytesPtr(request), int32(len(request)), output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloMessageUnary() error id = %d", errID)
	}
	assertMessageOutput(t, output, want)
}

func assertMessageCollect(t *testing.T, ctx context.Context, names []string, want string) {
	t.Helper()

	handle, errID := StartGreeterCollectMessageClientStream(ctx)
	if errID != 0 {
		t.Fatalf("StartGreeterCollectMessageClientStream() error id = %d", errID)
	}
	for _, name := range names {
		request := messageRequestBytes(t, name, "remote")
		if errID := SendGreeterCollectMessageClientStream(ctx, handle, bytesPtr(request), int32(len(request))); errID != 0 {
			t.Fatalf("SendGreeterCollectMessageClientStream() error id = %d", errID)
		}
	}
	output := &GreeterMessageOutput{}
	if errID := FinishGreeterCollectMessageClientStream(ctx, handle, output); errID != 0 {
		t.Fatalf("FinishGreeterCollectMessageClientStream() error id = %d", errID)
	}
	assertMessageOutput(t, output, want)
}

func assertMessageBroadcast(t *testing.T, ctx context.Context, name string, wants []string) {
	t.Helper()

	request := messageRequestBytes(t, name, "remote")
	handle, errID := StartGreeterBroadcastMessageServerStream(ctx, bytesPtr(request), int32(len(request)))
	if errID != 0 {
		t.Fatalf("StartGreeterBroadcastMessageServerStream() error id = %d", errID)
	}
	for _, want := range wants {
		output := &GreeterMessageOutput{}
		if errID := ReadGreeterBroadcastMessageServerStream(ctx, handle, output); errID != 0 {
			t.Fatalf("ReadGreeterBroadcastMessageServerStream() error id = %d", errID)
		}
		assertMessageOutput(t, output, want)
	}
	if errID := DoneGreeterBroadcastMessageServerStream(ctx, handle); errID != 0 {
		t.Fatalf("DoneGreeterBroadcastMessageServerStream() error id = %d", errID)
	}
}

func assertMessageChat(t *testing.T, ctx context.Context, name, want string) {
	t.Helper()

	handle, errID := StartGreeterChatMessageBidiStream(ctx)
	if errID != 0 {
		t.Fatalf("StartGreeterChatMessageBidiStream() error id = %d", errID)
	}
	request := messageRequestBytes(t, name, "remote")
	if errID := SendGreeterChatMessageBidiStream(ctx, handle, bytesPtr(request), int32(len(request))); errID != 0 {
		t.Fatalf("SendGreeterChatMessageBidiStream() error id = %d", errID)
	}
	output := &GreeterMessageOutput{}
	if errID := ReadGreeterChatMessageBidiStream(ctx, handle, output); errID != 0 {
		t.Fatalf("ReadGreeterChatMessageBidiStream() error id = %d: %s", errID, cgoErrorText(errID))
	}
	assertMessageOutput(t, output, want)
	if errID := CloseSendGreeterChatMessageBidiStream(ctx, handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatMessageBidiStream() error id = %d", errID)
	}
	if errID := DoneGreeterChatMessageBidiStream(ctx, handle); errID != 0 {
		t.Fatalf("DoneGreeterChatMessageBidiStream() error id = %d", errID)
	}
}

func cgoErrorText(errorID int32) string {
	data, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errorID))
	if !ok {
		return "<missing>"
	}
	if ptr != 0 {
		defer rpcruntime.Release(ptr)
	}
	return string(data)
}

func nativeUnaryInput(name, city string) *GreeterSayHelloNativeUnaryInput {
	nameBytes := []byte(name)
	cityBytes := []byte(city)
	return &GreeterSayHelloNativeUnaryInput{
		NamePtr: uintptr(unsafe.Pointer(unsafe.SliceData(nameBytes))),
		NameLen: int32(len(nameBytes)),
		CityPtr: uintptr(unsafe.Pointer(unsafe.SliceData(cityBytes))),
		CityLen: int32(len(cityBytes)),
	}
}

func nativeCollectInput(name, city string) *GreeterCollectNativeClientStreamInput {
	nameBytes := []byte(name)
	cityBytes := []byte(city)
	return &GreeterCollectNativeClientStreamInput{
		NamePtr: uintptr(unsafe.Pointer(unsafe.SliceData(nameBytes))),
		NameLen: int32(len(nameBytes)),
		CityPtr: uintptr(unsafe.Pointer(unsafe.SliceData(cityBytes))),
		CityLen: int32(len(cityBytes)),
	}
}

func nativeBroadcastInput(name, city string) *GreeterBroadcastNativeServerStreamInput {
	nameBytes := []byte(name)
	cityBytes := []byte(city)
	return &GreeterBroadcastNativeServerStreamInput{
		NamePtr: uintptr(unsafe.Pointer(unsafe.SliceData(nameBytes))),
		NameLen: int32(len(nameBytes)),
		CityPtr: uintptr(unsafe.Pointer(unsafe.SliceData(cityBytes))),
		CityLen: int32(len(cityBytes)),
	}
}

func nativeChatInput(name, city string) *GreeterChatNativeBidiStreamInput {
	nameBytes := []byte(name)
	cityBytes := []byte(city)
	return &GreeterChatNativeBidiStreamInput{
		NamePtr: uintptr(unsafe.Pointer(unsafe.SliceData(nameBytes))),
		NameLen: int32(len(nameBytes)),
		CityPtr: uintptr(unsafe.Pointer(unsafe.SliceData(cityBytes))),
		CityLen: int32(len(cityBytes)),
	}
}

func messageRequestBytes(t *testing.T, name, city string) []byte {
	t.Helper()
	data, err := proto.Marshal(&greeterv1.SayHelloRequest{Name: name, City: city})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return data
}

func bytesPtr(data []byte) uintptr {
	return uintptr(unsafe.Pointer(unsafe.SliceData(data)))
}

func assertNativeOutput(t *testing.T, ptr uintptr, length int32, want string) {
	t.Helper()
	got := string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length))
	if got != want {
		t.Fatalf("native response = %q, want %q", got, want)
	}
	rpcruntime.Release(ptr)
}

func assertMessageOutput(t *testing.T, output *GreeterMessageOutput, want string) {
	t.Helper()
	data := unsafe.Slice((*byte)(unsafe.Pointer(output.DataPtr)), output.DataLen)
	var response greeterv1.SayHelloResponse
	if err := proto.Unmarshal(data, &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := response.GetMessage(); got != want {
		t.Fatalf("message response = %q, want %q", got, want)
	}
	rpcruntime.Release(output.DataPtr)
}
