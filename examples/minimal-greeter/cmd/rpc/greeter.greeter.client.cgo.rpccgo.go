package main

import (
	v1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

import (
	context "context"
	errors "errors"
	fmt "fmt"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
)

// rpccgo native stage file for Greeter cgo native client

var greeterNativeClientUnsupportedField = errors.New("rpccgo: native unary client field bridge is not implemented")
var greeterNativeClientStreamHandleInvalid = errors.New("rpccgo: native client stream handle is invalid")

type GreeterSayHelloNativeUnaryInput struct {
	NamePtr       uintptr
	NameLen       int32
	NameOwnership int32
}

type GreeterSayHelloNativeUnaryOutput struct {
	MessagePtr uintptr
	MessageLen int32
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
	req, err := decodeGreeterSayHelloNativeUnaryRequest(input)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	resp, err := v1.NewGreeterCGONativeClientBridge().SayHello(ctx, req)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if resp == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary server returned nil response")))
	}
	if err := encodeGreeterSayHelloNativeUnaryResponse(resp, output); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterSayHelloNativeUnaryRequest(input *GreeterSayHelloNativeUnaryInput) (*v1.SayHelloRequest, error) {
	req := &v1.SayHelloRequest{}
	if _, err := rpcruntime.LengthFromInt32(input.NameLen); err != nil {
		return nil, fmt.Errorf("examples.minimal.greeter.v1.SayHelloRequest.name: %w", err)
	}
	Name := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0)
	req.Name = Name.SafeString()
	if err := Name.Release(); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeGreeterSayHelloNativeUnaryResponse(resp *v1.SayHelloResponse, output *GreeterSayHelloNativeUnaryOutput) error {
	MessageLen, err := rpcruntime.LengthToInt32(len(resp.Message))
	if err != nil {
		return err
	}
	data, MessagePtr, err := rpcruntime.PinString(resp.Message)
	_ = data
	if err != nil {
		return err
	}
	_ = MessagePtr
	output.MessagePtr = MessagePtr
	output.MessageLen = MessageLen
	return nil
}
