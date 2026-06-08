package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"example.com/rpccgo-grpc/internal/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

func TestGRPCGreeterTransportAndStreamingMatrix(t *testing.T) {
	ctx := context.Background()
	registerNativeServer(t)

	t.Run("native_cgo", func(t *testing.T) {
		assertNativeUnary(t, ctx, "native", "local", "hello native from local")
		assertNativeCollect(t, ctx, []string{"ada", "grace"}, "collect:ada,grace")
		assertNativeBroadcast(t, ctx, "stream", []string{"broadcast[0]:stream", "broadcast[1]:stream"})
		assertNativeChat(t, ctx, "bidi", "chat:bidi")
		assertNativeChatConversation(t, ctx, []string{"ada", "grace"}, []string{"chat:ada", "chat:grace"})
	})

	t.Run("message_cgo", func(t *testing.T) {
		registerMessageServer(t)
		assertMessageUnary(t, ctx, "message", "local", "hello message from local")
		assertMessageCollect(t, ctx, []string{"client", "stream"}, "collect:client,stream")
		assertMessageBroadcast(t, ctx, "server", []string{"broadcast[0]:server", "broadcast[1]:server"})
		assertMessageChat(t, ctx, "bidi-message", "chat:bidi-message")
		assertMessageChatConversation(t, ctx, []string{"ada-message", "grace-message"}, []string{"chat:ada-message", "chat:grace-message"})
	})

	t.Run("grpc_remote", func(t *testing.T) {
		remote := startExampleServer(t)
		conn, err := grpc.NewClient(remote.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })

		client := greeterv1.NewGreeterClient(conn)
		if err := greeterv1.RegisterGreeterGRPCRemoteServer(client); err != nil {
			t.Fatalf("RegisterGreeterGRPCRemoteServer() error = %v", err)
		}
		assertMessageUnary(t, ctx, "grpc", "remote", "hello grpc from remote")
		assertMessageCollect(t, ctx, []string{"grpc", "collect"}, "collect:grpc,collect")
		assertMessageBroadcast(t, ctx, "grpc-broadcast", []string{"broadcast[0]:grpc-broadcast", "broadcast[1]:grpc-broadcast"})
		assertMessageChat(t, ctx, "grpc-chat", "chat:grpc-chat")
		assertMessageChatConversation(t, ctx, []string{"grpc-chat-1", "grpc-chat-2"}, []string{"chat:grpc-chat-1", "chat:grpc-chat-2"})
	})
}

func registerNativeServer(t *testing.T) {
	t.Helper()
	if err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
}

func registerMessageServer(t *testing.T) {
	t.Helper()
	if err := greeterv1.RegisterGreeterCGOMessageServer(localMessageServer{}); err != nil {
		t.Fatalf("RegisterGreeterCGOMessageServer() error = %v", err)
	}
}

type localMessageServer struct{}

func (localMessageServer) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: fmt.Sprintf("hello %s from %s", req.GetName(), req.GetCity())}, nil
}

func (localMessageServer) Collect(ctx context.Context, stream rpcruntime.CGOMessageClientStream[*greeterv1.SayHelloRequest]) (*greeterv1.SayHelloResponse, error) {
	var names []string
	for {
		req, err := stream.Recv(ctx)
		if err == io.EOF {
			return &greeterv1.SayHelloResponse{Message: "collect:" + strings.Join(names, ",")}, nil
		}
		if err != nil {
			return nil, err
		}
		names = append(names, req.GetName())
	}
}

func (localMessageServer) Broadcast(ctx context.Context, req *greeterv1.SayHelloRequest, stream rpcruntime.CGOMessageServerStream[*greeterv1.SayHelloResponse]) error {
	for index := 0; index < 2; index++ {
		resp := &greeterv1.SayHelloResponse{Message: fmt.Sprintf("broadcast[%d]:%s", index, req.GetName())}
		if err := stream.Send(ctx, resp); err != nil {
			return err
		}
	}
	return nil
}

func (localMessageServer) Chat(ctx context.Context, stream rpcruntime.CGOMessageBidiStream[*greeterv1.SayHelloRequest, *greeterv1.SayHelloResponse]) error {
	for {
		req, err := stream.Recv(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(ctx, &greeterv1.SayHelloResponse{Message: "chat:" + req.GetName()}); err != nil {
			return err
		}
	}
}

type exampleServer struct {
	grpcAddr string
}

func startExampleServer(t *testing.T) exampleServer {
	t.Helper()

	grpcAddr := reserveTCPAddr(t)
	serverBin := filepath.Join(t.TempDir(), "grpc-example-server-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	build := exec.Command("go", "build", "-o", serverBin, "./cmd/server")
	build.Dir = "../.."
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build grpc example server error = %v\n%s", err, out)
	}

	cmd := exec.Command(serverBin, "--addr", grpcAddr)
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start grpc example server error = %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	waitForTCP(t, grpcAddr)
	return exampleServer{grpcAddr: grpcAddr}
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

func assertNativeUnary(t *testing.T, ctx context.Context, name, city, want string) {
	t.Helper()

	input := nativeInput(name, city)
	var messagePtr uintptr
	var messageLen int32
	if errID := callGreeterSayHelloNativeUnary(
		input.namePtr(), input.nameLen(), 0,
		input.cityPtr(), input.cityLen(), 0,
		&messagePtr, &messageLen,
	); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() error id = %d", errID)
	}
	assertNativeOutput(t, messagePtr, messageLen, want)
}

func assertNativeCollect(t *testing.T, ctx context.Context, names []string, want string) {
	t.Helper()

	handle, errID := startGreeterCollectNativeClientStream()
	if errID != 0 {
		t.Fatalf("StartGreeterCollectNativeClientStream() error id = %d", errID)
	}
	for _, name := range names {
		input := nativeInput(name, "local")
		if errID := sendGreeterCollectNativeClientStream(
			handle,
			input.namePtr(), input.nameLen(), 0,
			input.cityPtr(), input.cityLen(), 0,
		); errID != 0 {
			t.Fatalf("SendGreeterCollectNativeClientStream() error id = %d", errID)
		}
	}
	var messagePtr uintptr
	var messageLen int32
	if errID := finishGreeterCollectNativeClientStream(handle, &messagePtr, &messageLen); errID != 0 {
		t.Fatalf("FinishGreeterCollectNativeClientStream() error id = %d", errID)
	}
	assertNativeOutput(t, messagePtr, messageLen, want)
}

func assertNativeBroadcast(t *testing.T, ctx context.Context, name string, wants []string) {
	t.Helper()

	input := nativeInput(name, "local")
	handle, errID := startGreeterBroadcastNativeServerStream(
		input.namePtr(), input.nameLen(), 0,
		input.cityPtr(), input.cityLen(), 0,
	)
	if errID != 0 {
		t.Fatalf("StartGreeterBroadcastNativeServerStream() error id = %d", errID)
	}
	for _, want := range wants {
		var messagePtr uintptr
		var messageLen int32
		if errID := readGreeterBroadcastNativeServerStream(handle, &messagePtr, &messageLen); errID != 0 {
			t.Fatalf("ReadGreeterBroadcastNativeServerStream() error id = %d", errID)
		}
		assertNativeOutput(t, messagePtr, messageLen, want)
	}
	if errID := finishGreeterBroadcastNativeServerStream(handle); errID != 0 {
		t.Fatalf("FinishGreeterBroadcastNativeServerStream() error id = %d", errID)
	}
}

func assertNativeChat(t *testing.T, ctx context.Context, name, want string) {
	t.Helper()

	handle, errID := startGreeterChatNativeBidiStream()
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() error id = %d", errID)
	}
	input := nativeInput(name, "local")
	if errID := sendGreeterChatNativeBidiStream(
		handle,
		input.namePtr(), input.nameLen(), 0,
		input.cityPtr(), input.cityLen(), 0,
	); errID != 0 {
		t.Fatalf("SendGreeterChatNativeBidiStream() error id = %d", errID)
	}
	var messagePtr uintptr
	var messageLen int32
	if errID := readGreeterChatNativeBidiStream(handle, &messagePtr, &messageLen); errID != 0 {
		t.Fatalf("ReadGreeterChatNativeBidiStream() error id = %d", errID)
	}
	assertNativeOutput(t, messagePtr, messageLen, want)
	if errID := closeSendGreeterChatNativeBidiStream(handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatNativeBidiStream() error id = %d", errID)
	}
	if errID := finishGreeterChatNativeBidiStream(handle); errID != 0 {
		t.Fatalf("FinishGreeterChatNativeBidiStream() error id = %d", errID)
	}
}

func assertNativeChatConversation(t *testing.T, ctx context.Context, names []string, wants []string) {
	t.Helper()
	if len(names) != len(wants) {
		t.Fatalf("native chat conversation names=%d wants=%d", len(names), len(wants))
	}

	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = streamCtx

	handle, errID := startGreeterChatNativeBidiStream()
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() error id = %d", errID)
	}
	for index, name := range names {
		input := nativeInput(name, "local")
		if errID := sendGreeterChatNativeBidiStream(
			handle,
			input.namePtr(), input.nameLen(), 0,
			input.cityPtr(), input.cityLen(), 0,
		); errID != 0 {
			t.Fatalf("SendGreeterChatNativeBidiStream(%q) error id = %d: %s", name, errID, cgoErrorText(errID))
		}
		var messagePtr uintptr
		var messageLen int32
		if errID := readGreeterChatNativeBidiStream(handle, &messagePtr, &messageLen); errID != 0 {
			t.Fatalf("ReadGreeterChatNativeBidiStream(%q) error id = %d: %s", name, errID, cgoErrorText(errID))
		}
		assertNativeOutput(t, messagePtr, messageLen, wants[index])
	}
	if errID := closeSendGreeterChatNativeBidiStream(handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatNativeBidiStream() error id = %d", errID)
	}
	if errID := finishGreeterChatNativeBidiStream(handle); errID != 0 {
		t.Fatalf("FinishGreeterChatNativeBidiStream() error id = %d", errID)
	}
}

func assertMessageUnary(t *testing.T, ctx context.Context, name, city, want string) {
	t.Helper()

	request := messageRequestBytes(t, name, city)
	var messagePtr uintptr
	var messageLen int32
	if errID := callGreeterSayHelloMessageUnary(bytesPtr(request), int32(len(request)), &messagePtr, &messageLen); errID != 0 {
		t.Fatalf("CallGreeterSayHelloMessageUnary() error id = %d", errID)
	}
	assertMessageOutput(t, messagePtr, messageLen, want)
}

func assertMessageCollect(t *testing.T, ctx context.Context, names []string, want string) {
	t.Helper()

	handle, errID := startGreeterCollectMessageClientStream()
	if errID != 0 {
		t.Fatalf("StartGreeterCollectMessageClientStream() error id = %d", errID)
	}
	for _, name := range names {
		request := messageRequestBytes(t, name, "remote")
		if errID := sendGreeterCollectMessageClientStream(handle, bytesPtr(request), int32(len(request))); errID != 0 {
			t.Fatalf("SendGreeterCollectMessageClientStream() error id = %d", errID)
		}
	}
	var messagePtr uintptr
	var messageLen int32
	if errID := finishGreeterCollectMessageClientStream(handle, &messagePtr, &messageLen); errID != 0 {
		t.Fatalf("FinishGreeterCollectMessageClientStream() error id = %d", errID)
	}
	assertMessageOutput(t, messagePtr, messageLen, want)
}

func assertMessageBroadcast(t *testing.T, ctx context.Context, name string, wants []string) {
	t.Helper()

	request := messageRequestBytes(t, name, "remote")
	handle, errID := startGreeterBroadcastMessageServerStream(bytesPtr(request), int32(len(request)))
	if errID != 0 {
		t.Fatalf("StartGreeterBroadcastMessageServerStream() error id = %d", errID)
	}
	for _, want := range wants {
		var messagePtr uintptr
		var messageLen int32
		if errID := readGreeterBroadcastMessageServerStream(handle, &messagePtr, &messageLen); errID != 0 {
			t.Fatalf("ReadGreeterBroadcastMessageServerStream() error id = %d", errID)
		}
		assertMessageOutput(t, messagePtr, messageLen, want)
	}
	if errID := finishGreeterBroadcastMessageServerStream(handle); errID != 0 {
		t.Fatalf("FinishGreeterBroadcastMessageServerStream() error id = %d", errID)
	}
}

func assertMessageChat(t *testing.T, ctx context.Context, name, want string) {
	t.Helper()

	handle, errID := startGreeterChatMessageBidiStream()
	if errID != 0 {
		t.Fatalf("StartGreeterChatMessageBidiStream() error id = %d", errID)
	}
	request := messageRequestBytes(t, name, "remote")
	if errID := sendGreeterChatMessageBidiStream(handle, bytesPtr(request), int32(len(request))); errID != 0 {
		t.Fatalf("SendGreeterChatMessageBidiStream() error id = %d", errID)
	}
	var messagePtr uintptr
	var messageLen int32
	if errID := readGreeterChatMessageBidiStream(handle, &messagePtr, &messageLen); errID != 0 {
		t.Fatalf("ReadGreeterChatMessageBidiStream() error id = %d: %s", errID, cgoErrorText(errID))
	}
	assertMessageOutput(t, messagePtr, messageLen, want)
	if errID := finishGreeterChatMessageBidiStream(handle); errID != 0 {
		t.Fatalf("FinishGreeterChatMessageBidiStream() error id = %d", errID)
	}
}

func assertMessageChatConversation(t *testing.T, ctx context.Context, names []string, wants []string) {
	t.Helper()
	if len(names) != len(wants) {
		t.Fatalf("message chat conversation names=%d wants=%d", len(names), len(wants))
	}

	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = streamCtx

	handle, errID := startGreeterChatMessageBidiStream()
	if errID != 0 {
		t.Fatalf("StartGreeterChatMessageBidiStream() error id = %d", errID)
	}
	for index, name := range names {
		request := messageRequestBytes(t, name, "remote")
		if errID := sendGreeterChatMessageBidiStream(handle, bytesPtr(request), int32(len(request))); errID != 0 {
			t.Fatalf("SendGreeterChatMessageBidiStream(%q) error id = %d: %s", name, errID, cgoErrorText(errID))
		}
		var messagePtr uintptr
		var messageLen int32
		if errID := readGreeterChatMessageBidiStream(handle, &messagePtr, &messageLen); errID != 0 {
			t.Fatalf("ReadGreeterChatMessageBidiStream(%q) error id = %d: %s", name, errID, cgoErrorText(errID))
		}
		assertMessageOutput(t, messagePtr, messageLen, wants[index])
	}
	if errID := closeSendGreeterChatMessageBidiStream(handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatMessageBidiStream() error id = %d", errID)
	}
	if errID := finishGreeterChatMessageBidiStream(handle); errID != 0 {
		t.Fatalf("FinishGreeterChatMessageBidiStream() error id = %d", errID)
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

type nativeInputArgs struct {
	name []byte
	city []byte
}

func nativeInput(name, city string) nativeInputArgs {
	return nativeInputArgs{
		name: []byte(name),
		city: []byte(city),
	}
}

func (a nativeInputArgs) namePtr() uintptr {
	return bytesPtr(a.name)
}

func (a nativeInputArgs) nameLen() int32 {
	return int32(len(a.name))
}

func (a nativeInputArgs) cityPtr() uintptr {
	return bytesPtr(a.city)
}

func (a nativeInputArgs) cityLen() int32 {
	return int32(len(a.city))
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

func assertMessageOutput(t *testing.T, ptr uintptr, length int32, want string) {
	t.Helper()
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length)
	var response greeterv1.SayHelloResponse
	if err := proto.Unmarshal(data, &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := response.GetMessage(); got != want {
		t.Fatalf("message response = %q, want %q", got, want)
	}
	rpcruntime.Release(ptr)
}
