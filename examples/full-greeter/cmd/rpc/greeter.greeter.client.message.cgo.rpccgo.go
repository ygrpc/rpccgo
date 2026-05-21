package main

import (
	proto "example.com/rpccgo-full/proto"
)

/*
#include <stdint.h>
*/
import "C"

import (
	context "context"
	errors "errors"
	fmt "fmt"
	protobuf "google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
)

// rpccgo message direct generated file for Greeter cgo message client

type GreeterMessageOutput struct {
	DataPtr uintptr
	DataLen int32
}

func CallGreeterSayHelloMessageUnary(ctx context.Context, requestPtr uintptr, requestLen int32, output *GreeterMessageOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))
	}
	req, err := decodeGreeterSayHelloMessageRequestBytes(requestPtr, requestLen)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(req, &proto.SayHelloRequest{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)))
	}
	resp, err := proto.NewGreeterCGOMessageClientBridge().SayHello(ctx, req)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(resp, &proto.SayHelloResponse{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)))
	}
	ptr, length, err := encodeGreeterSayHelloMessageResponseBytes(resp)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	output.DataPtr = ptr
	output.DataLen = length
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_SayHello
func rpccgo_msg_greeterv1_Greeter_SayHello(requestPtr C.uintptr_t, requestLen C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {
	if responsePtr != nil {
		*responsePtr = 0
	}
	if responseLen != nil {
		*responseLen = 0
	}
	if responsePtr == nil || responseLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))
	}
	var output GreeterMessageOutput
	errID := CallGreeterSayHelloMessageUnary(context.Background(), uintptr(requestPtr), int32(requestLen), &output)
	if errID != 0 {
		return C.int32_t(errID)
	}
	*responsePtr = C.uintptr_t(output.DataPtr)
	*responseLen = C.int32_t(output.DataLen)
	return 0
}

func decodeGreeterSayHelloMessageRequestBytes(ptr uintptr, length int32) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("rpccgo: message request length is negative")
	}
	if ptr == 0 || length == 0 {
		return nil, nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))...), nil
}

func encodeGreeterSayHelloMessageResponseBytes(data []byte) (uintptr, int32, error) {
	length, err := rpcruntime.LengthToInt32(len(data))
	if err != nil {
		return 0, 0, err
	}
	ptr, err := rpcruntime.PinBytes(data)
	if err != nil {
		return 0, 0, err
	}
	return ptr, length, nil
}

func StartGreeterCollectMessageClientStream(ctx context.Context) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	handle, err := proto.NewGreeterCGOMessageClientBridge().StartCollect(ctx)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func SendGreeterCollectMessageClientStream(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := decodeGreeterCollectMessageRequestBytes(requestPtr, requestLen)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(req, &proto.SayHelloRequest{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)))
	}
	err = proto.NewGreeterCollectMessageStream(rpcruntime.StreamHandle(handle)).Send(ctx, req)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func FinishGreeterCollectMessageClientStream(ctx context.Context, handle int32, output *GreeterMessageOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))
	}
	var resp []byte
	var err error
	resp, err = proto.NewGreeterCollectMessageStream(rpcruntime.StreamHandle(handle)).Finish(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(resp, &proto.SayHelloResponse{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)))
	}
	ptr, length, err := encodeGreeterCollectMessageResponseBytes(resp)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	output.DataPtr = ptr
	output.DataLen = length
	return 0
}

func CancelGreeterCollectMessageClientStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	err = proto.NewGreeterCollectMessageStream(rpcruntime.StreamHandle(handle)).Cancel(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterCollectMessageRequestBytes(ptr uintptr, length int32) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("rpccgo: message request length is negative")
	}
	if ptr == 0 || length == 0 {
		return nil, nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))...), nil
}

func encodeGreeterCollectMessageResponseBytes(data []byte) (uintptr, int32, error) {
	length, err := rpcruntime.LengthToInt32(len(data))
	if err != nil {
		return 0, 0, err
	}
	ptr, err := rpcruntime.PinBytes(data)
	if err != nil {
		return 0, 0, err
	}
	return ptr, length, nil
}

//export rpccgo_msg_greeterv1_Greeter_Collect_start
func rpccgo_msg_greeterv1_Greeter_Collect_start(handle *C.int32_t) C.int32_t {
	if handle != nil {
		*handle = 0
	}
	if handle == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client handle pointer is nil")))
	}
	handleValue, errID := StartGreeterCollectMessageClientStream(context.Background())
	if errID != 0 {
		return C.int32_t(errID)
	}
	*handle = C.int32_t(handleValue)
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_Collect_send
func rpccgo_msg_greeterv1_Greeter_Collect_send(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {
	return C.int32_t(SendGreeterCollectMessageClientStream(context.Background(), int32(handle), uintptr(requestPtr), int32(requestLen)))
}

//export rpccgo_msg_greeterv1_Greeter_Collect_finish
func rpccgo_msg_greeterv1_Greeter_Collect_finish(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {
	if responsePtr != nil {
		*responsePtr = 0
	}
	if responseLen != nil {
		*responseLen = 0
	}
	if responsePtr == nil || responseLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))
	}
	var output GreeterMessageOutput
	errID := FinishGreeterCollectMessageClientStream(context.Background(), int32(handle), &output)
	if errID != 0 {
		return C.int32_t(errID)
	}
	*responsePtr = C.uintptr_t(output.DataPtr)
	*responseLen = C.int32_t(output.DataLen)
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_Collect_cancel
func rpccgo_msg_greeterv1_Greeter_Collect_cancel(handle C.int32_t) C.int32_t {
	return C.int32_t(CancelGreeterCollectMessageClientStream(context.Background(), int32(handle)))
}

func StartGreeterBroadcastMessageServerStream(ctx context.Context, requestPtr uintptr, requestLen int32) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := decodeGreeterBroadcastMessageRequestBytes(requestPtr, requestLen)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(req, &proto.SayHelloRequest{}); err != nil {
		return 0, int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)))
	}
	handle, err := proto.NewGreeterCGOMessageClientBridge().StartBroadcast(ctx, req)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func ReadGreeterBroadcastMessageServerStream(ctx context.Context, handle int32, output *GreeterMessageOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))
	}
	var resp []byte
	var err error
	resp, err = proto.NewGreeterBroadcastMessageStream(rpcruntime.StreamHandle(handle)).Recv(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(resp, &proto.SayHelloResponse{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)))
	}
	ptr, length, err := encodeGreeterBroadcastMessageResponseBytes(resp)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	output.DataPtr = ptr
	output.DataLen = length
	return 0
}

func DoneGreeterBroadcastMessageServerStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	err = proto.NewGreeterBroadcastMessageStream(rpcruntime.StreamHandle(handle)).Done(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CancelGreeterBroadcastMessageServerStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	err = proto.NewGreeterBroadcastMessageStream(rpcruntime.StreamHandle(handle)).Cancel(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterBroadcastMessageRequestBytes(ptr uintptr, length int32) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("rpccgo: message request length is negative")
	}
	if ptr == 0 || length == 0 {
		return nil, nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))...), nil
}

func encodeGreeterBroadcastMessageResponseBytes(data []byte) (uintptr, int32, error) {
	length, err := rpcruntime.LengthToInt32(len(data))
	if err != nil {
		return 0, 0, err
	}
	ptr, err := rpcruntime.PinBytes(data)
	if err != nil {
		return 0, 0, err
	}
	return ptr, length, nil
}

//export rpccgo_msg_greeterv1_Greeter_Broadcast_start
func rpccgo_msg_greeterv1_Greeter_Broadcast_start(requestPtr C.uintptr_t, requestLen C.int32_t, handle *C.int32_t) C.int32_t {
	if handle != nil {
		*handle = 0
	}
	if handle == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client handle pointer is nil")))
	}
	handleValue, errID := StartGreeterBroadcastMessageServerStream(context.Background(), uintptr(requestPtr), int32(requestLen))
	if errID != 0 {
		return C.int32_t(errID)
	}
	*handle = C.int32_t(handleValue)
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_Broadcast_read
func rpccgo_msg_greeterv1_Greeter_Broadcast_read(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {
	if responsePtr != nil {
		*responsePtr = 0
	}
	if responseLen != nil {
		*responseLen = 0
	}
	if responsePtr == nil || responseLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))
	}
	var output GreeterMessageOutput
	errID := ReadGreeterBroadcastMessageServerStream(context.Background(), int32(handle), &output)
	if errID != 0 {
		return C.int32_t(errID)
	}
	*responsePtr = C.uintptr_t(output.DataPtr)
	*responseLen = C.int32_t(output.DataLen)
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_Broadcast_done
func rpccgo_msg_greeterv1_Greeter_Broadcast_done(handle C.int32_t) C.int32_t {
	return C.int32_t(DoneGreeterBroadcastMessageServerStream(context.Background(), int32(handle)))
}

//export rpccgo_msg_greeterv1_Greeter_Broadcast_cancel
func rpccgo_msg_greeterv1_Greeter_Broadcast_cancel(handle C.int32_t) C.int32_t {
	return C.int32_t(CancelGreeterBroadcastMessageServerStream(context.Background(), int32(handle)))
}

func StartGreeterChatMessageBidiStream(ctx context.Context) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	handle, err := proto.NewGreeterCGOMessageClientBridge().StartChat(ctx)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func SendGreeterChatMessageBidiStream(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := decodeGreeterChatMessageRequestBytes(requestPtr, requestLen)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(req, &proto.SayHelloRequest{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)))
	}
	err = proto.NewGreeterChatMessageStream(rpcruntime.StreamHandle(handle)).Send(ctx, req)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func ReadGreeterChatMessageBidiStream(ctx context.Context, handle int32, output *GreeterMessageOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))
	}
	var resp []byte
	var err error
	resp, err = proto.NewGreeterChatMessageStream(rpcruntime.StreamHandle(handle)).Recv(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(resp, &proto.SayHelloResponse{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)))
	}
	ptr, length, err := encodeGreeterChatMessageResponseBytes(resp)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	output.DataPtr = ptr
	output.DataLen = length
	return 0
}

func CloseSendGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	err = proto.NewGreeterChatMessageStream(rpcruntime.StreamHandle(handle)).CloseSend(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func DoneGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	err = proto.NewGreeterChatMessageStream(rpcruntime.StreamHandle(handle)).Done(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CancelGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	err = proto.NewGreeterChatMessageStream(rpcruntime.StreamHandle(handle)).Cancel(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterChatMessageRequestBytes(ptr uintptr, length int32) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("rpccgo: message request length is negative")
	}
	if ptr == 0 || length == 0 {
		return nil, nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))...), nil
}

func encodeGreeterChatMessageResponseBytes(data []byte) (uintptr, int32, error) {
	length, err := rpcruntime.LengthToInt32(len(data))
	if err != nil {
		return 0, 0, err
	}
	ptr, err := rpcruntime.PinBytes(data)
	if err != nil {
		return 0, 0, err
	}
	return ptr, length, nil
}

//export rpccgo_msg_greeterv1_Greeter_Chat_start
func rpccgo_msg_greeterv1_Greeter_Chat_start(handle *C.int32_t) C.int32_t {
	if handle != nil {
		*handle = 0
	}
	if handle == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client handle pointer is nil")))
	}
	handleValue, errID := StartGreeterChatMessageBidiStream(context.Background())
	if errID != 0 {
		return C.int32_t(errID)
	}
	*handle = C.int32_t(handleValue)
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_Chat_send
func rpccgo_msg_greeterv1_Greeter_Chat_send(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {
	return C.int32_t(SendGreeterChatMessageBidiStream(context.Background(), int32(handle), uintptr(requestPtr), int32(requestLen)))
}

//export rpccgo_msg_greeterv1_Greeter_Chat_read
func rpccgo_msg_greeterv1_Greeter_Chat_read(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {
	if responsePtr != nil {
		*responsePtr = 0
	}
	if responseLen != nil {
		*responseLen = 0
	}
	if responsePtr == nil || responseLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))
	}
	var output GreeterMessageOutput
	errID := ReadGreeterChatMessageBidiStream(context.Background(), int32(handle), &output)
	if errID != 0 {
		return C.int32_t(errID)
	}
	*responsePtr = C.uintptr_t(output.DataPtr)
	*responseLen = C.int32_t(output.DataLen)
	return 0
}

//export rpccgo_msg_greeterv1_Greeter_Chat_close_send
func rpccgo_msg_greeterv1_Greeter_Chat_close_send(handle C.int32_t) C.int32_t {
	return C.int32_t(CloseSendGreeterChatMessageBidiStream(context.Background(), int32(handle)))
}

//export rpccgo_msg_greeterv1_Greeter_Chat_cancel
func rpccgo_msg_greeterv1_Greeter_Chat_cancel(handle C.int32_t) C.int32_t {
	return C.int32_t(CancelGreeterChatMessageBidiStream(context.Background(), int32(handle)))
}
