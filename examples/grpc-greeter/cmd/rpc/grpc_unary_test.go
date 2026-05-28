package main

import (
	"context"
	"net"
	"testing"
	"unsafe"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"example.com/rpccgo-grpc/internal/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

func TestGRPCNativeAndMessageClients(t *testing.T) {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	name := []byte("native")
	var nativeMessagePtr uintptr
	var nativeMessageLen int32
	if errID := CallGreeterSayHelloNativeUnary(
		context.Background(),
		uintptr(unsafe.Pointer(unsafe.SliceData(name))),
		int32(len(name)),
		0,
		&nativeMessagePtr,
		&nativeMessageLen,
	); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() error id = %d", errID)
	}
	nativeMessage := unsafe.Slice((*byte)(unsafe.Pointer(nativeMessagePtr)), nativeMessageLen)
	if got := string(nativeMessage); got != "hello, native" {
		t.Fatalf("native response = %q, want hello, native", got)
	}
	rpcruntime.Release(nativeMessagePtr)

	messageInput, err := proto.Marshal(&greeterv1.SayHelloRequest{Name: "message"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	messageOutput := &GreeterMessageOutput{}
	if errID := CallGreeterSayHelloMessageUnary(context.Background(), uintptr(unsafe.Pointer(&messageInput[0])), int32(len(messageInput)), messageOutput); errID != 0 {
		t.Fatalf("CallGreeterSayHelloMessageUnary() error id = %d", errID)
	}
	messageBytes := unsafe.Slice((*byte)(unsafe.Pointer(messageOutput.DataPtr)), messageOutput.DataLen)
	var messageResponse greeterv1.SayHelloResponse
	if err := proto.Unmarshal(messageBytes, &messageResponse); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := messageResponse.GetMessage(); got != "hello, message" {
		t.Fatalf("message response = %q, want hello, message", got)
	}
	rpcruntime.Release(messageOutput.DataPtr)
}

func TestGRPCRemoteServerAdapter(t *testing.T) {
	ctx := context.Background()
	conn := startGRPCServer(t)
	client := greeterv1.NewGreeterClient(conn)
	if _, err := greeterv1.RegisterGreeterGRPCRemoteServer(client); err != nil {
		t.Fatalf("RegisterGreeterGRPCRemoteServer() error = %v", err)
	}

	messageInput, err := proto.Marshal(&greeterv1.SayHelloRequest{Name: "remote-message"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	messageOutput := &GreeterMessageOutput{}
	if errID := CallGreeterSayHelloMessageUnary(ctx, uintptr(unsafe.Pointer(&messageInput[0])), int32(len(messageInput)), messageOutput); errID != 0 {
		t.Fatalf("CallGreeterSayHelloMessageUnary() error id = %d", errID)
	}
	messageBytes := unsafe.Slice((*byte)(unsafe.Pointer(messageOutput.DataPtr)), messageOutput.DataLen)
	var messageResponse greeterv1.SayHelloResponse
	if err := proto.Unmarshal(messageBytes, &messageResponse); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := messageResponse.GetMessage(); got != "hello, remote-message" {
		t.Fatalf("message response = %q, want hello, remote-message", got)
	}
	rpcruntime.Release(messageOutput.DataPtr)

	name := []byte("remote-native")
	var nativeMessagePtr uintptr
	var nativeMessageLen int32
	if errID := CallGreeterSayHelloNativeUnary(
		ctx,
		uintptr(unsafe.Pointer(unsafe.SliceData(name))),
		int32(len(name)),
		0,
		&nativeMessagePtr,
		&nativeMessageLen,
	); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() error id = %d", errID)
	}
	nativeMessage := unsafe.Slice((*byte)(unsafe.Pointer(nativeMessagePtr)), nativeMessageLen)
	if got := string(nativeMessage); got != "hello, remote-native" {
		t.Fatalf("native response = %q, want hello, remote-native", got)
	}
	rpcruntime.Release(nativeMessagePtr)
}

func startGRPCServer(t *testing.T) *grpc.ClientConn {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	server := grpc.NewServer()
	greeterv1.RegisterGreeterServer(server, backend.GRPCGreeter{})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}
