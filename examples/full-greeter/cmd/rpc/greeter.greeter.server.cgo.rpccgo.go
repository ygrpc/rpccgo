package main

import (
	proto "example.com/rpccgo-full/proto"
)

/*
#include <stdint.h>

typedef struct GreeterSayHelloCGONativeUnaryRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t CityPtr;
int32_t CityLen;
} GreeterSayHelloCGONativeUnaryRequest;

typedef struct GreeterSayHelloCGONativeUnaryResponse {
uintptr_t MessagePtr;
int32_t MessageLen;
int32_t MessageOwnership;
} GreeterSayHelloCGONativeUnaryResponse;

typedef int32_t (*GreeterSayHelloCGONativeUnaryCallback)(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output);

typedef struct GreeterCollectCGONativeClientStreamRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t CityPtr;
int32_t CityLen;
} GreeterCollectCGONativeClientStreamRequest;

typedef struct GreeterCollectCGONativeClientStreamResponse {
uintptr_t MessagePtr;
int32_t MessageLen;
int32_t MessageOwnership;
} GreeterCollectCGONativeClientStreamResponse;

typedef int32_t (*GreeterCollectCGONativeClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterCollectCGONativeClientStreamSendCallback)(int32_t stream, GreeterCollectCGONativeClientStreamRequest* input);
typedef int32_t (*GreeterCollectCGONativeClientStreamFinishCallback)(int32_t stream, GreeterCollectCGONativeClientStreamResponse* output);
typedef int32_t (*GreeterCollectCGONativeClientStreamCancelCallback)(int32_t stream);

typedef struct GreeterBroadcastCGONativeServerStreamRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t CityPtr;
int32_t CityLen;
} GreeterBroadcastCGONativeServerStreamRequest;

typedef struct GreeterBroadcastCGONativeServerStreamResponse {
uintptr_t MessagePtr;
int32_t MessageLen;
int32_t MessageOwnership;
} GreeterBroadcastCGONativeServerStreamResponse;

typedef int32_t (*GreeterBroadcastCGONativeServerStreamStartCallback)(GreeterBroadcastCGONativeServerStreamRequest* input, int32_t* stream);
typedef int32_t (*GreeterBroadcastCGONativeServerStreamRecvCallback)(int32_t stream, GreeterBroadcastCGONativeServerStreamResponse* output);
typedef int32_t (*GreeterBroadcastCGONativeServerStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterBroadcastCGONativeServerStreamCancelCallback)(int32_t stream);

typedef struct GreeterChatCGONativeBidiStreamRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t CityPtr;
int32_t CityLen;
} GreeterChatCGONativeBidiStreamRequest;

typedef struct GreeterChatCGONativeBidiStreamResponse {
uintptr_t MessagePtr;
int32_t MessageLen;
int32_t MessageOwnership;
} GreeterChatCGONativeBidiStreamResponse;

typedef int32_t (*GreeterChatCGONativeBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamSendCallback)(int32_t stream, GreeterChatCGONativeBidiStreamRequest* input);
typedef int32_t (*GreeterChatCGONativeBidiStreamRecvCallback)(int32_t stream, GreeterChatCGONativeBidiStreamResponse* output);
typedef int32_t (*GreeterChatCGONativeBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGONativeServerCallbacks {
GreeterSayHelloCGONativeUnaryCallback SayHello;
GreeterCollectCGONativeClientStreamStartCallback CollectStart;
GreeterCollectCGONativeClientStreamSendCallback CollectSend;
GreeterCollectCGONativeClientStreamFinishCallback CollectFinish;
GreeterCollectCGONativeClientStreamCancelCallback CollectCancel;
GreeterBroadcastCGONativeServerStreamStartCallback BroadcastStart;
GreeterBroadcastCGONativeServerStreamRecvCallback BroadcastRecv;
GreeterBroadcastCGONativeServerStreamDoneCallback BroadcastDone;
GreeterBroadcastCGONativeServerStreamCancelCallback BroadcastCancel;
GreeterChatCGONativeBidiStreamStartCallback ChatStart;
GreeterChatCGONativeBidiStreamSendCallback ChatSend;
GreeterChatCGONativeBidiStreamRecvCallback ChatRecv;
GreeterChatCGONativeBidiStreamCloseSendCallback ChatCloseSend;
GreeterChatCGONativeBidiStreamDoneCallback ChatDone;
GreeterChatCGONativeBidiStreamCancelCallback ChatCancel;
} GreeterCGONativeServerCallbacks;

static inline int32_t callGreeterSayHelloCGONativeUnaryCallback(GreeterSayHelloCGONativeUnaryCallback callback, GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	return callback(input, output);
}

static inline int32_t callGreeterCollectCGONativeClientStreamStartCallback(GreeterCollectCGONativeClientStreamStartCallback callback, int32_t* stream) {
	return callback(stream);
}

static inline int32_t callGreeterCollectCGONativeClientStreamSendCallback(GreeterCollectCGONativeClientStreamSendCallback callback, int32_t stream, GreeterCollectCGONativeClientStreamRequest* input) {
	return callback(stream, input);
}

static inline int32_t callGreeterCollectCGONativeClientStreamFinishCallback(GreeterCollectCGONativeClientStreamFinishCallback callback, int32_t stream, GreeterCollectCGONativeClientStreamResponse* output) {
	return callback(stream, output);
}

static inline int32_t callGreeterCollectCGONativeClientStreamCancelCallback(GreeterCollectCGONativeClientStreamCancelCallback callback, int32_t stream) {
	return callback(stream);
}

static inline int32_t callGreeterBroadcastCGONativeServerStreamStartCallback(GreeterBroadcastCGONativeServerStreamStartCallback callback, GreeterBroadcastCGONativeServerStreamRequest* input, int32_t* stream) {
	return callback(input, stream);
}

static inline int32_t callGreeterBroadcastCGONativeServerStreamRecvCallback(GreeterBroadcastCGONativeServerStreamRecvCallback callback, int32_t stream, GreeterBroadcastCGONativeServerStreamResponse* output) {
	return callback(stream, output);
}

static inline int32_t callGreeterBroadcastCGONativeServerStreamDoneCallback(GreeterBroadcastCGONativeServerStreamDoneCallback callback, int32_t stream) {
	return callback(stream);
}

static inline int32_t callGreeterBroadcastCGONativeServerStreamCancelCallback(GreeterBroadcastCGONativeServerStreamCancelCallback callback, int32_t stream) {
	return callback(stream);
}

static inline int32_t callGreeterChatCGONativeBidiStreamStartCallback(GreeterChatCGONativeBidiStreamStartCallback callback, int32_t* stream) {
	return callback(stream);
}

static inline int32_t callGreeterChatCGONativeBidiStreamSendCallback(GreeterChatCGONativeBidiStreamSendCallback callback, int32_t stream, GreeterChatCGONativeBidiStreamRequest* input) {
	return callback(stream, input);
}

static inline int32_t callGreeterChatCGONativeBidiStreamRecvCallback(GreeterChatCGONativeBidiStreamRecvCallback callback, int32_t stream, GreeterChatCGONativeBidiStreamResponse* output) {
	return callback(stream, output);
}

static inline int32_t callGreeterChatCGONativeBidiStreamCloseSendCallback(GreeterChatCGONativeBidiStreamCloseSendCallback callback, int32_t stream) {
	return callback(stream);
}

static inline int32_t callGreeterChatCGONativeBidiStreamDoneCallback(GreeterChatCGONativeBidiStreamDoneCallback callback, int32_t stream) {
	return callback(stream);
}

static inline int32_t callGreeterChatCGONativeBidiStreamCancelCallback(GreeterChatCGONativeBidiStreamCancelCallback callback, int32_t stream) {
	return callback(stream);
}

*/
import "C"

import (
	context "context"
	errors "errors"
	fmt "fmt"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
)

// rpccgo native stage file for Greeter cgo native server

var (
	greeterCGONativeServerCallbacksNil         = errors.New("rpccgo: Greeter cgo native server callbacks are nil")
	greeterCGONativeServerUnaryCallbackMissing = errors.New("rpccgo: Greeter cgo native server unary callback is missing")
	greeterCGONativeServerUnsupportedField     = errors.New("rpccgo: cgo native server field bridge is not implemented")
	greeterCGONativeServerStreamNotImplemented = errors.New("rpccgo: cgo native server streaming is not implemented")
)

type greeterCGONativeAdapter struct {
	callbacks C.GreeterCGONativeServerCallbacks
}

func (a *greeterCGONativeAdapter) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	callback := a.callbacks.SayHello
	if callback == nil {
		return nil, greeterCGONativeServerUnaryCallbackMissing
	}
	input, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(req)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	output := &C.GreeterSayHelloCGONativeUnaryResponse{}
	errID := int32(C.callGreeterSayHelloCGONativeUnaryCallback(callback, input, output))
	if errID != 0 {
		cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterSayHelloCGONativeUnaryResponse(output)
	cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *greeterCGONativeAdapter) StartCollect(ctx context.Context) (proto.GreeterCollectNativeStreamSession, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	if a.callbacks.CollectStart == nil || a.callbacks.CollectSend == nil || a.callbacks.CollectFinish == nil || a.callbacks.CollectCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	var stream C.int32_t
	errID := int32(C.callGreeterCollectCGONativeClientStreamStartCallback(a.callbacks.CollectStart, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterCollectCGONativeClientStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterCollectCGONativeClientStreamSession struct {
	callbacks C.GreeterCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterCollectCGONativeClientStreamSession) Send(ctx context.Context, req *proto.SayHelloRequest) error {
	input, cleanup, err := encodeGreeterCollectCGONativeClientStreamRequest(req)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := int32(C.callGreeterCollectCGONativeClientStreamSendCallback(s.callbacks.CollectSend, s.stream, input))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterCollectCGONativeClientStreamSession) Finish(ctx context.Context) (*proto.SayHelloResponse, error) {
	output := &C.GreeterCollectCGONativeClientStreamResponse{}
	errID := int32(C.callGreeterCollectCGONativeClientStreamFinishCallback(s.callbacks.CollectFinish, s.stream, output))
	if errID != 0 {
		cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterCollectCGONativeClientStreamResponse(output)
	cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *greeterCollectCGONativeClientStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterCollectCGONativeClientStreamCancelCallback(s.callbacks.CollectCancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterCGONativeAdapter) StartBroadcast(ctx context.Context, req *proto.SayHelloRequest) (proto.GreeterBroadcastNativeStreamSession, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	if a.callbacks.BroadcastStart == nil || a.callbacks.BroadcastRecv == nil || a.callbacks.BroadcastDone == nil || a.callbacks.BroadcastCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	input, cleanup, err := encodeGreeterBroadcastCGONativeServerStreamRequest(req)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	var stream C.int32_t
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamStartCallback(a.callbacks.BroadcastStart, input, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterBroadcastCGONativeServerStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterBroadcastCGONativeServerStreamSession struct {
	callbacks C.GreeterCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterBroadcastCGONativeServerStreamSession) Recv(ctx context.Context) (*proto.SayHelloResponse, error) {
	output := &C.GreeterBroadcastCGONativeServerStreamResponse{}
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamRecvCallback(s.callbacks.BroadcastRecv, s.stream, output))
	if errID != 0 {
		cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterBroadcastCGONativeServerStreamResponse(output)
	cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *greeterBroadcastCGONativeServerStreamSession) Done(ctx context.Context) error {
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamDoneCallback(s.callbacks.BroadcastDone, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterBroadcastCGONativeServerStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamCancelCallback(s.callbacks.BroadcastCancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterCGONativeAdapter) StartChat(ctx context.Context) (proto.GreeterChatNativeStreamSession, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	if a.callbacks.ChatStart == nil || a.callbacks.ChatSend == nil || a.callbacks.ChatRecv == nil || a.callbacks.ChatCloseSend == nil || a.callbacks.ChatDone == nil || a.callbacks.ChatCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	var stream C.int32_t
	errID := int32(C.callGreeterChatCGONativeBidiStreamStartCallback(a.callbacks.ChatStart, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterChatCGONativeBidiStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterChatCGONativeBidiStreamSession struct {
	callbacks C.GreeterCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterChatCGONativeBidiStreamSession) Send(ctx context.Context, req *proto.SayHelloRequest) error {
	input, cleanup, err := encodeGreeterChatCGONativeBidiStreamRequest(req)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := int32(C.callGreeterChatCGONativeBidiStreamSendCallback(s.callbacks.ChatSend, s.stream, input))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Recv(ctx context.Context) (*proto.SayHelloResponse, error) {
	output := &C.GreeterChatCGONativeBidiStreamResponse{}
	errID := int32(C.callGreeterChatCGONativeBidiStreamRecvCallback(s.callbacks.ChatRecv, s.stream, output))
	if errID != 0 {
		cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterChatCGONativeBidiStreamResponse(output)
	cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *greeterChatCGONativeBidiStreamSession) CloseSend(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGONativeBidiStreamCloseSendCallback(s.callbacks.ChatCloseSend, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Done(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGONativeBidiStreamDoneCallback(s.callbacks.ChatDone, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGONativeBidiStreamCancelCallback(s.callbacks.ChatCancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func encodeGreeterSayHelloCGONativeUnaryRequest(req *proto.SayHelloRequest) (*C.GreeterSayHelloCGONativeUnaryRequest, func(), error) {
	if req == nil {
		return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")
	}
	input := &C.GreeterSayHelloCGONativeUnaryRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	NameLen, err := rpcruntime.LengthToInt32(len(req.Name))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, NamePtr, err := rpcruntime.PinString(req.Name)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if NamePtr != 0 {
		pinned = append(pinned, NamePtr)
	}
	input.NamePtr = C.uintptr_t(NamePtr)
	input.NameLen = C.int32_t(NameLen)
	CityLen, err := rpcruntime.LengthToInt32(len(req.City))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, CityPtr, err := rpcruntime.PinString(req.City)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if CityPtr != 0 {
		pinned = append(pinned, CityPtr)
	}
	input.CityPtr = C.uintptr_t(CityPtr)
	input.CityLen = C.int32_t(CityLen)
	return input, cleanup, nil
}

func decodeGreeterSayHelloCGONativeUnaryResponse(output *C.GreeterSayHelloCGONativeUnaryResponse) (*proto.SayHelloResponse, error) {
	if output == nil {
		return nil, errors.New("rpccgo: cgo native server response output is nil")
	}
	resp := &proto.SayHelloResponse{}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	Message := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	resp.Message = Message.SafeString()
	return resp, nil
}

func cleanupGreeterSayHelloCGONativeUnaryResponse(output *C.GreeterSayHelloCGONativeUnaryResponse) error {
	if output == nil {
		return nil
	}
	var cleanupErr error
	if output.MessageOwnership > 0 && output.MessagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.MessagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		output.MessagePtr = 0
		output.MessageLen = 0
		output.MessageOwnership = 0
	}
	return cleanupErr
}

func encodeGreeterCollectCGONativeClientStreamRequest(req *proto.SayHelloRequest) (*C.GreeterCollectCGONativeClientStreamRequest, func(), error) {
	if req == nil {
		return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")
	}
	input := &C.GreeterCollectCGONativeClientStreamRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	NameLen, err := rpcruntime.LengthToInt32(len(req.Name))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, NamePtr, err := rpcruntime.PinString(req.Name)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if NamePtr != 0 {
		pinned = append(pinned, NamePtr)
	}
	input.NamePtr = C.uintptr_t(NamePtr)
	input.NameLen = C.int32_t(NameLen)
	CityLen, err := rpcruntime.LengthToInt32(len(req.City))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, CityPtr, err := rpcruntime.PinString(req.City)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if CityPtr != 0 {
		pinned = append(pinned, CityPtr)
	}
	input.CityPtr = C.uintptr_t(CityPtr)
	input.CityLen = C.int32_t(CityLen)
	return input, cleanup, nil
}

func decodeGreeterCollectCGONativeClientStreamResponse(output *C.GreeterCollectCGONativeClientStreamResponse) (*proto.SayHelloResponse, error) {
	if output == nil {
		return nil, errors.New("rpccgo: cgo native server response output is nil")
	}
	resp := &proto.SayHelloResponse{}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	Message := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	resp.Message = Message.SafeString()
	return resp, nil
}

func cleanupGreeterCollectCGONativeClientStreamResponse(output *C.GreeterCollectCGONativeClientStreamResponse) error {
	if output == nil {
		return nil
	}
	var cleanupErr error
	if output.MessageOwnership > 0 && output.MessagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.MessagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		output.MessagePtr = 0
		output.MessageLen = 0
		output.MessageOwnership = 0
	}
	return cleanupErr
}

func encodeGreeterBroadcastCGONativeServerStreamRequest(req *proto.SayHelloRequest) (*C.GreeterBroadcastCGONativeServerStreamRequest, func(), error) {
	if req == nil {
		return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")
	}
	input := &C.GreeterBroadcastCGONativeServerStreamRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	NameLen, err := rpcruntime.LengthToInt32(len(req.Name))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, NamePtr, err := rpcruntime.PinString(req.Name)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if NamePtr != 0 {
		pinned = append(pinned, NamePtr)
	}
	input.NamePtr = C.uintptr_t(NamePtr)
	input.NameLen = C.int32_t(NameLen)
	CityLen, err := rpcruntime.LengthToInt32(len(req.City))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, CityPtr, err := rpcruntime.PinString(req.City)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if CityPtr != 0 {
		pinned = append(pinned, CityPtr)
	}
	input.CityPtr = C.uintptr_t(CityPtr)
	input.CityLen = C.int32_t(CityLen)
	return input, cleanup, nil
}

func decodeGreeterBroadcastCGONativeServerStreamResponse(output *C.GreeterBroadcastCGONativeServerStreamResponse) (*proto.SayHelloResponse, error) {
	if output == nil {
		return nil, errors.New("rpccgo: cgo native server response output is nil")
	}
	resp := &proto.SayHelloResponse{}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	Message := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	resp.Message = Message.SafeString()
	return resp, nil
}

func cleanupGreeterBroadcastCGONativeServerStreamResponse(output *C.GreeterBroadcastCGONativeServerStreamResponse) error {
	if output == nil {
		return nil
	}
	var cleanupErr error
	if output.MessageOwnership > 0 && output.MessagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.MessagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		output.MessagePtr = 0
		output.MessageLen = 0
		output.MessageOwnership = 0
	}
	return cleanupErr
}

func encodeGreeterChatCGONativeBidiStreamRequest(req *proto.SayHelloRequest) (*C.GreeterChatCGONativeBidiStreamRequest, func(), error) {
	if req == nil {
		return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")
	}
	input := &C.GreeterChatCGONativeBidiStreamRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	NameLen, err := rpcruntime.LengthToInt32(len(req.Name))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, NamePtr, err := rpcruntime.PinString(req.Name)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if NamePtr != 0 {
		pinned = append(pinned, NamePtr)
	}
	input.NamePtr = C.uintptr_t(NamePtr)
	input.NameLen = C.int32_t(NameLen)
	CityLen, err := rpcruntime.LengthToInt32(len(req.City))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, CityPtr, err := rpcruntime.PinString(req.City)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if CityPtr != 0 {
		pinned = append(pinned, CityPtr)
	}
	input.CityPtr = C.uintptr_t(CityPtr)
	input.CityLen = C.int32_t(CityLen)
	return input, cleanup, nil
}

func decodeGreeterChatCGONativeBidiStreamResponse(output *C.GreeterChatCGONativeBidiStreamResponse) (*proto.SayHelloResponse, error) {
	if output == nil {
		return nil, errors.New("rpccgo: cgo native server response output is nil")
	}
	resp := &proto.SayHelloResponse{}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return nil, fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	Message := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	resp.Message = Message.SafeString()
	return resp, nil
}

func cleanupGreeterChatCGONativeBidiStreamResponse(output *C.GreeterChatCGONativeBidiStreamResponse) error {
	if output == nil {
		return nil
	}
	var cleanupErr error
	if output.MessageOwnership > 0 && output.MessagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.MessagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		output.MessagePtr = 0
		output.MessageLen = 0
		output.MessageOwnership = 0
	}
	return cleanupErr
}

func greeterCGONativeServerErrorFromID(errID int32) error {
	if errID == 0 {
		return nil
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if ok {
		if ptr != 0 {
			defer rpcruntime.Release(ptr)
		}
		return errors.New(string(text))
	}
	return fmt.Errorf("rpccgo: cgo native server callback returned unknown error id %d", errID)
}

func RegisterGreeterCGONativeServer(callbacks *C.GreeterCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter], error) {
	if callbacks == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerCallbacksNil
	}
	if callbacks.SayHello == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerUnaryCallbackMissing
	}
	if callbacks.CollectStart == nil || callbacks.CollectSend == nil || callbacks.CollectFinish == nil || callbacks.CollectCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerStreamNotImplemented
	}
	if callbacks.BroadcastStart == nil || callbacks.BroadcastRecv == nil || callbacks.BroadcastDone == nil || callbacks.BroadcastCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerStreamNotImplemented
	}
	if callbacks.ChatStart == nil || callbacks.ChatSend == nil || callbacks.ChatRecv == nil || callbacks.ChatCloseSend == nil || callbacks.ChatDone == nil || callbacks.ChatCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerStreamNotImplemented
	}
	callbacksCopy := *callbacks
	return proto.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, &greeterCGONativeAdapter{callbacks: callbacksCopy})
}

type GreeterGoCGONativeServerCallbacks struct {
	SayHello        func(ctx context.Context, input *C.GreeterSayHelloCGONativeUnaryRequest, output *C.GreeterSayHelloCGONativeUnaryResponse) int32
	CollectStart    func(ctx context.Context, stream *C.int32_t) int32
	CollectSend     func(ctx context.Context, stream C.int32_t, input *C.GreeterCollectCGONativeClientStreamRequest) int32
	CollectFinish   func(ctx context.Context, stream C.int32_t, output *C.GreeterCollectCGONativeClientStreamResponse) int32
	CollectCancel   func(ctx context.Context, stream C.int32_t) int32
	BroadcastStart  func(ctx context.Context, input *C.GreeterBroadcastCGONativeServerStreamRequest, stream *C.int32_t) int32
	BroadcastRecv   func(ctx context.Context, stream C.int32_t, output *C.GreeterBroadcastCGONativeServerStreamResponse) int32
	BroadcastDone   func(ctx context.Context, stream C.int32_t) int32
	BroadcastCancel func(ctx context.Context, stream C.int32_t) int32
	ChatStart       func(ctx context.Context, stream *C.int32_t) int32
	ChatSend        func(ctx context.Context, stream C.int32_t, input *C.GreeterChatCGONativeBidiStreamRequest) int32
	ChatRecv        func(ctx context.Context, stream C.int32_t, output *C.GreeterChatCGONativeBidiStreamResponse) int32
	ChatCloseSend   func(ctx context.Context, stream C.int32_t) int32
	ChatDone        func(ctx context.Context, stream C.int32_t) int32
	ChatCancel      func(ctx context.Context, stream C.int32_t) int32
}

func RegisterGreeterGoCGONativeServerForTesting(callbacks *GreeterGoCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter], error) {
	if callbacks == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerCallbacksNil
	}
	if callbacks.SayHello == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerUnaryCallbackMissing
	}
	if callbacks.CollectStart == nil || callbacks.CollectSend == nil || callbacks.CollectFinish == nil || callbacks.CollectCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerStreamNotImplemented
	}
	if callbacks.BroadcastStart == nil || callbacks.BroadcastRecv == nil || callbacks.BroadcastDone == nil || callbacks.BroadcastCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerStreamNotImplemented
	}
	if callbacks.ChatStart == nil || callbacks.ChatSend == nil || callbacks.ChatRecv == nil || callbacks.ChatCloseSend == nil || callbacks.ChatDone == nil || callbacks.ChatCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterNativeAdapter]{}, greeterCGONativeServerStreamNotImplemented
	}
	return proto.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, &greeterGoCGONativeAdapter{callbacks: callbacks})
}

type greeterGoCGONativeAdapter struct {
	callbacks *GreeterGoCGONativeServerCallbacks
}

func (a *greeterGoCGONativeAdapter) SayHello(ctx context.Context, req *proto.SayHelloRequest) (*proto.SayHelloResponse, error) {
	input, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(req)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	output := &C.GreeterSayHelloCGONativeUnaryResponse{}
	errID := a.callbacks.SayHello(ctx, input, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterSayHelloCGONativeUnaryResponse(output)
	cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (a *greeterGoCGONativeAdapter) StartCollect(ctx context.Context) (proto.GreeterCollectNativeStreamSession, error) {
	var stream C.int32_t
	errID := a.callbacks.CollectStart(ctx, &stream)
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterCollectGoCGONativeClientStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterCollectGoCGONativeClientStreamSession struct {
	callbacks *GreeterGoCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterCollectGoCGONativeClientStreamSession) Send(ctx context.Context, req *proto.SayHelloRequest) error {
	input, cleanup, err := encodeGreeterCollectCGONativeClientStreamRequest(req)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := s.callbacks.CollectSend(ctx, s.stream, input)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterCollectGoCGONativeClientStreamSession) Finish(ctx context.Context) (*proto.SayHelloResponse, error) {
	output := &C.GreeterCollectCGONativeClientStreamResponse{}
	errID := s.callbacks.CollectFinish(ctx, s.stream, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterCollectCGONativeClientStreamResponse(output)
	cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *greeterCollectGoCGONativeClientStreamSession) Cancel(ctx context.Context) error {
	errID := s.callbacks.CollectCancel(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterGoCGONativeAdapter) StartBroadcast(ctx context.Context, req *proto.SayHelloRequest) (proto.GreeterBroadcastNativeStreamSession, error) {
	input, cleanup, err := encodeGreeterBroadcastCGONativeServerStreamRequest(req)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	var stream C.int32_t
	errID := a.callbacks.BroadcastStart(ctx, input, &stream)
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterBroadcastGoCGONativeServerStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterBroadcastGoCGONativeServerStreamSession struct {
	callbacks *GreeterGoCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterBroadcastGoCGONativeServerStreamSession) Recv(ctx context.Context) (*proto.SayHelloResponse, error) {
	output := &C.GreeterBroadcastCGONativeServerStreamResponse{}
	errID := s.callbacks.BroadcastRecv(ctx, s.stream, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterBroadcastCGONativeServerStreamResponse(output)
	cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *greeterBroadcastGoCGONativeServerStreamSession) Done(ctx context.Context) error {
	errID := s.callbacks.BroadcastDone(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterBroadcastGoCGONativeServerStreamSession) Cancel(ctx context.Context) error {
	errID := s.callbacks.BroadcastCancel(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterGoCGONativeAdapter) StartChat(ctx context.Context) (proto.GreeterChatNativeStreamSession, error) {
	var stream C.int32_t
	errID := a.callbacks.ChatStart(ctx, &stream)
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterChatGoCGONativeBidiStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterChatGoCGONativeBidiStreamSession struct {
	callbacks *GreeterGoCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterChatGoCGONativeBidiStreamSession) Send(ctx context.Context, req *proto.SayHelloRequest) error {
	input, cleanup, err := encodeGreeterChatCGONativeBidiStreamRequest(req)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := s.callbacks.ChatSend(ctx, s.stream, input)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatGoCGONativeBidiStreamSession) Recv(ctx context.Context) (*proto.SayHelloResponse, error) {
	output := &C.GreeterChatCGONativeBidiStreamResponse{}
	errID := s.callbacks.ChatRecv(ctx, s.stream, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return nil, errors.Join(callbackErr, cleanupErr)
		}
		return nil, callbackErr
	}
	resp, err := decodeGreeterChatCGONativeBidiStreamResponse(output)
	cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return nil, errors.Join(err, cleanupErr)
		}
		return nil, cleanupErr
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *greeterChatGoCGONativeBidiStreamSession) CloseSend(ctx context.Context) error {
	errID := s.callbacks.ChatCloseSend(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatGoCGONativeBidiStreamSession) Done(ctx context.Context) error {
	errID := s.callbacks.ChatDone(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatGoCGONativeBidiStreamSession) Cancel(ctx context.Context) error {
	errID := s.callbacks.ChatCancel(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

//export StoreGreeterCGONativeServerErrorTextForExport
func StoreGreeterCGONativeServerErrorTextForExport(text *C.char, textLen C.int32_t) C.int32_t {
	length, err := rpcruntime.LengthFromInt32(int32(textLen))
	if err != nil {
		return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: cgo native server error text: %w", err)))
	}
	if text == nil && length != 0 {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: cgo native server error text pointer is nil")))
	}
	var data []byte
	if length != 0 {
		data = unsafe.Slice((*byte)(unsafe.Pointer(text)), length)
	}
	return C.int32_t(rpcruntime.StoreError(errors.New(string(data))))
}
