package main

import (
	v1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

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
	if err := protobuf.Unmarshal(req, &v1.SayHelloRequest{}); err != nil {
		return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)))
	}
	resp, err := v1.NewGreeterCGOMessageClientBridge().SayHello(ctx, req)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := protobuf.Unmarshal(resp, &v1.SayHelloResponse{}); err != nil {
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
