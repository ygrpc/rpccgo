package main

import (
	"context"
	"testing"
	"unsafe"

	greeterv1 "example.com/rpccgo-minimal/gen/greeter/v1"
	"example.com/rpccgo-minimal/internal/backend"
	"google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

func TestMinimalNativeAndMessageClients(t *testing.T) {
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
