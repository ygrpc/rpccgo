package main

import (
	proto "example.com/rpccgo-full/proto"
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
	CityPtr       uintptr
	CityLen       int32
	CityOwnership int32
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
	resp, err := proto.NewGreeterCGONativeClientBridge().SayHello(ctx, req)
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

func decodeGreeterSayHelloNativeUnaryRequest(input *GreeterSayHelloNativeUnaryInput) (*proto.SayHelloRequest, error) {
	req := &proto.SayHelloRequest{}
	if _, err := rpcruntime.LengthFromInt32(input.NameLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err)
	}
	Name := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0)
	req.Name = Name.SafeString()
	if err := Name.Release(); err != nil {
		return nil, err
	}
	if _, err := rpcruntime.LengthFromInt32(input.CityLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err)
	}
	City := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.CityPtr)), input.CityLen, input.CityOwnership > 0)
	req.City = City.SafeString()
	if err := City.Release(); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeGreeterSayHelloNativeUnaryResponse(resp *proto.SayHelloResponse, output *GreeterSayHelloNativeUnaryOutput) error {
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

type GreeterCollectNativeClientStreamInput struct {
	NamePtr       uintptr
	NameLen       int32
	NameOwnership int32
	CityPtr       uintptr
	CityLen       int32
	CityOwnership int32
}

type GreeterCollectNativeClientStreamOutput struct {
	MessagePtr uintptr
	MessageLen int32
}

func StartGreeterCollectNativeClientStream(ctx context.Context) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	handle, err := proto.NewGreeterCGONativeClientBridge().StartCollect(ctx)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func SendGreeterCollectNativeClientStream(ctx context.Context, handle int32, input *GreeterCollectNativeClientStreamInput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native client stream input is nil")))
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().LoadCollectNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	req, err := decodeGreeterCollectNativeClientStreamRequest(input)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := session.Send(ctx, req); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func FinishGreeterCollectNativeClientStream(ctx context.Context, handle int32, output *GreeterCollectNativeClientStreamOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native client stream output is nil")))
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().TakeCollectNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	resp, err := session.Finish(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if resp == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native client stream server returned nil response")))
	}
	if err := encodeGreeterCollectNativeClientStreamResponse(resp, output); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CancelGreeterCollectNativeClientStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().TakeCollectNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	if err := session.Cancel(ctx); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterCollectNativeClientStreamRequest(input *GreeterCollectNativeClientStreamInput) (*proto.SayHelloRequest, error) {
	req := &proto.SayHelloRequest{}
	if _, err := rpcruntime.LengthFromInt32(input.NameLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err)
	}
	Name := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0)
	req.Name = Name.SafeString()
	if err := Name.Release(); err != nil {
		return nil, err
	}
	if _, err := rpcruntime.LengthFromInt32(input.CityLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err)
	}
	City := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.CityPtr)), input.CityLen, input.CityOwnership > 0)
	req.City = City.SafeString()
	if err := City.Release(); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeGreeterCollectNativeClientStreamResponse(resp *proto.SayHelloResponse, output *GreeterCollectNativeClientStreamOutput) error {
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

type GreeterBroadcastNativeServerStreamInput struct {
	NamePtr       uintptr
	NameLen       int32
	NameOwnership int32
	CityPtr       uintptr
	CityLen       int32
	CityOwnership int32
}

type GreeterBroadcastNativeServerStreamOutput struct {
	MessagePtr uintptr
	MessageLen int32
}

func StartGreeterBroadcastNativeServerStream(ctx context.Context, input *GreeterBroadcastNativeServerStreamInput) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		return 0, int32(rpcruntime.StoreError(errors.New("rpccgo: native server stream input is nil")))
	}
	req, err := decodeGreeterBroadcastNativeServerStreamRequest(input)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	handle, err := proto.NewGreeterCGONativeClientBridge().StartBroadcast(ctx, req)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func ReadGreeterBroadcastNativeServerStream(ctx context.Context, handle int32, output *GreeterBroadcastNativeServerStreamOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native server stream output is nil")))
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().LoadBroadcastNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	resp, err := session.Recv(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if resp == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native server stream server returned nil response")))
	}
	if err := encodeGreeterBroadcastNativeServerStreamResponse(resp, output); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func DoneGreeterBroadcastNativeServerStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().TakeBroadcastNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	if done, ok := session.(interface{ Done(context.Context) error }); ok {
		if err := done.Done(ctx); err != nil {
			return int32(rpcruntime.StoreError(err))
		}
	}
	return 0
}

func CancelGreeterBroadcastNativeServerStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().TakeBroadcastNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	if err := session.Cancel(ctx); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterBroadcastNativeServerStreamRequest(input *GreeterBroadcastNativeServerStreamInput) (*proto.SayHelloRequest, error) {
	req := &proto.SayHelloRequest{}
	if _, err := rpcruntime.LengthFromInt32(input.NameLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err)
	}
	Name := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0)
	req.Name = Name.SafeString()
	if err := Name.Release(); err != nil {
		return nil, err
	}
	if _, err := rpcruntime.LengthFromInt32(input.CityLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err)
	}
	City := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.CityPtr)), input.CityLen, input.CityOwnership > 0)
	req.City = City.SafeString()
	if err := City.Release(); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeGreeterBroadcastNativeServerStreamResponse(resp *proto.SayHelloResponse, output *GreeterBroadcastNativeServerStreamOutput) error {
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

type GreeterChatNativeBidiStreamInput struct {
	NamePtr       uintptr
	NameLen       int32
	NameOwnership int32
	CityPtr       uintptr
	CityLen       int32
	CityOwnership int32
}

type GreeterChatNativeBidiStreamOutput struct {
	MessagePtr uintptr
	MessageLen int32
}

func StartGreeterChatNativeBidiStream(ctx context.Context) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	handle, err := proto.NewGreeterCGONativeClientBridge().StartChat(ctx)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func SendGreeterChatNativeBidiStream(ctx context.Context, handle int32, input *GreeterChatNativeBidiStreamInput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native bidi stream input is nil")))
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().LoadChatNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	req, err := decodeGreeterChatNativeBidiStreamRequest(input)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := session.Send(ctx, req); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func ReadGreeterChatNativeBidiStream(ctx context.Context, handle int32, output *GreeterChatNativeBidiStreamOutput) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native bidi stream output is nil")))
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().LoadChatNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	resp, err := session.Recv(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if resp == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: native bidi stream server returned nil response")))
	}
	if err := encodeGreeterChatNativeBidiStreamResponse(resp, output); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CloseSendGreeterChatNativeBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().LoadChatNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	if err := session.CloseSend(ctx); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func DoneGreeterChatNativeBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().TakeChatNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	if done, ok := session.(interface{ Done(context.Context) error }); ok {
		if err := done.Done(ctx); err != nil {
			return int32(rpcruntime.StoreError(err))
		}
	}
	return 0
}

func CancelGreeterChatNativeBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := proto.NewGreeterCGONativeClientBridge().TakeChatNativeStream(rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	if err := session.Cancel(ctx); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterChatNativeBidiStreamRequest(input *GreeterChatNativeBidiStreamInput) (*proto.SayHelloRequest, error) {
	req := &proto.SayHelloRequest{}
	if _, err := rpcruntime.LengthFromInt32(input.NameLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err)
	}
	Name := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0)
	req.Name = Name.SafeString()
	if err := Name.Release(); err != nil {
		return nil, err
	}
	if _, err := rpcruntime.LengthFromInt32(input.CityLen); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err)
	}
	City := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.CityPtr)), input.CityLen, input.CityOwnership > 0)
	req.City = City.SafeString()
	if err := City.Release(); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeGreeterChatNativeBidiStreamResponse(resp *proto.SayHelloResponse, output *GreeterChatNativeBidiStreamOutput) error {
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
