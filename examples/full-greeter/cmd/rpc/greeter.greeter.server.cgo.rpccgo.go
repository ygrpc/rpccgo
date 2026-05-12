package main

import (
	proto "example.com/rpccgo-full/proto"
)

/*
#include <stdint.h>

typedef int32_t (*GreeterSayHelloCGONativeUnaryCallback)(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership);

typedef int32_t (*GreeterCollectCGONativeClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterCollectCGONativeClientStreamSendCallback)(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership);
typedef int32_t (*GreeterCollectCGONativeClientStreamFinishCallback)(int32_t stream, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership);
typedef int32_t (*GreeterCollectCGONativeClientStreamCancelCallback)(int32_t stream);

typedef int32_t (*GreeterBroadcastCGONativeServerStreamStartCallback)(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership, int32_t *outStream);
typedef int32_t (*GreeterBroadcastCGONativeServerStreamRecvCallback)(int32_t stream, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership);
typedef int32_t (*GreeterBroadcastCGONativeServerStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterBroadcastCGONativeServerStreamCancelCallback)(int32_t stream);

typedef int32_t (*GreeterChatCGONativeBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamSendCallback)(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership);
typedef int32_t (*GreeterChatCGONativeBidiStreamRecvCallback)(int32_t stream, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership);
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

static inline int32_t callGreeterSayHelloCGONativeUnaryCallback(GreeterSayHelloCGONativeUnaryCallback callback, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership) {
	return callback(NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership, outMessagePtr, outMessageLen, outMessageOwnership);
}

static inline int32_t callGreeterCollectCGONativeClientStreamStartCallback(GreeterCollectCGONativeClientStreamStartCallback callback, int32_t* stream) {
	return callback(stream);
}

static inline int32_t callGreeterCollectCGONativeClientStreamSendCallback(GreeterCollectCGONativeClientStreamSendCallback callback, int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership) {
	return callback(stream, NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership);
}

static inline int32_t callGreeterCollectCGONativeClientStreamFinishCallback(GreeterCollectCGONativeClientStreamFinishCallback callback, int32_t stream, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership) {
	return callback(stream, outMessagePtr, outMessageLen, outMessageOwnership);
}

static inline int32_t callGreeterCollectCGONativeClientStreamCancelCallback(GreeterCollectCGONativeClientStreamCancelCallback callback, int32_t stream) {
	return callback(stream);
}

static inline int32_t callGreeterBroadcastCGONativeServerStreamStartCallback(GreeterBroadcastCGONativeServerStreamStartCallback callback, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership, int32_t* stream) {
	return callback(NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership, stream);
}

static inline int32_t callGreeterBroadcastCGONativeServerStreamRecvCallback(GreeterBroadcastCGONativeServerStreamRecvCallback callback, int32_t stream, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership) {
	return callback(stream, outMessagePtr, outMessageLen, outMessageOwnership);
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

static inline int32_t callGreeterChatCGONativeBidiStreamSendCallback(GreeterChatCGONativeBidiStreamSendCallback callback, int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t CityPtr, int32_t CityLen, int32_t CityOwnership) {
	return callback(stream, NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership);
}

static inline int32_t callGreeterChatCGONativeBidiStreamRecvCallback(GreeterChatCGONativeBidiStreamRecvCallback callback, int32_t stream, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership) {
	return callback(stream, outMessagePtr, outMessageLen, outMessageOwnership);
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

type GreeterSayHelloCGONativeUnaryRequest struct {
	NamePtr       C.uintptr_t
	NameLen       C.int32_t
	NameOwnership C.int32_t
	CityPtr       C.uintptr_t
	CityLen       C.int32_t
	CityOwnership C.int32_t
}

type GreeterSayHelloCGONativeUnaryResponse struct {
	MessagePtr       C.uintptr_t
	MessageLen       C.int32_t
	MessageOwnership C.int32_t
}

type GreeterCollectCGONativeClientStreamRequest struct {
	NamePtr       C.uintptr_t
	NameLen       C.int32_t
	NameOwnership C.int32_t
	CityPtr       C.uintptr_t
	CityLen       C.int32_t
	CityOwnership C.int32_t
}

type GreeterCollectCGONativeClientStreamResponse struct {
	MessagePtr       C.uintptr_t
	MessageLen       C.int32_t
	MessageOwnership C.int32_t
}

type GreeterBroadcastCGONativeServerStreamRequest struct {
	NamePtr       C.uintptr_t
	NameLen       C.int32_t
	NameOwnership C.int32_t
	CityPtr       C.uintptr_t
	CityLen       C.int32_t
	CityOwnership C.int32_t
}

type GreeterBroadcastCGONativeServerStreamResponse struct {
	MessagePtr       C.uintptr_t
	MessageLen       C.int32_t
	MessageOwnership C.int32_t
}

type GreeterChatCGONativeBidiStreamRequest struct {
	NamePtr       C.uintptr_t
	NameLen       C.int32_t
	NameOwnership C.int32_t
	CityPtr       C.uintptr_t
	CityLen       C.int32_t
	CityOwnership C.int32_t
}

type GreeterChatCGONativeBidiStreamResponse struct {
	MessagePtr       C.uintptr_t
	MessageLen       C.int32_t
	MessageOwnership C.int32_t
}

var (
	greeterCGONativeServerCallbacksNil         = errors.New("rpccgo: Greeter cgo native server callbacks are nil")
	greeterCGONativeServerUnaryCallbackMissing = errors.New("rpccgo: Greeter cgo native server unary callback is missing")
	greeterCGONativeServerUnsupportedField     = errors.New("rpccgo: cgo native server field bridge is not implemented")
	greeterCGONativeServerStreamNotImplemented = errors.New("rpccgo: cgo native server streaming is not implemented")
)

type greeterCGONativeAdapter struct {
	callbacks C.GreeterCGONativeServerCallbacks
}

func (a *greeterCGONativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	if a == nil {
		return "", greeterCGONativeServerCallbacksNil
	}
	callback := a.callbacks.SayHello
	if callback == nil {
		return "", greeterCGONativeServerUnaryCallbackMissing
	}
	input, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(name, city)
	_ = input
	if err != nil {
		return "", err
	}
	defer cleanup()
	output := &GreeterSayHelloCGONativeUnaryResponse{}
	errID := int32(C.callGreeterSayHelloCGONativeUnaryCallback(callback, input.NamePtr, input.NameLen, input.NameOwnership, input.CityPtr, input.CityLen, input.CityOwnership, &output.MessagePtr, &output.MessageLen, &output.MessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterSayHelloCGONativeUnaryResponse(output)
	cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
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

func (s *greeterCollectCGONativeClientStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	input, cleanup, err := encodeGreeterCollectCGONativeClientStreamRequest(name, city)
	_ = input
	if err != nil {
		return err
	}
	defer cleanup()
	errID := int32(C.callGreeterCollectCGONativeClientStreamSendCallback(s.callbacks.CollectSend, s.stream, input.NamePtr, input.NameLen, input.NameOwnership, input.CityPtr, input.CityLen, input.CityOwnership))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterCollectCGONativeClientStreamSession) Finish(ctx context.Context) (string, error) {
	output := &GreeterCollectCGONativeClientStreamResponse{}
	errID := int32(C.callGreeterCollectCGONativeClientStreamFinishCallback(s.callbacks.CollectFinish, s.stream, &output.MessagePtr, &output.MessageLen, &output.MessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterCollectCGONativeClientStreamResponse(output)
	cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (s *greeterCollectCGONativeClientStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterCollectCGONativeClientStreamCancelCallback(s.callbacks.CollectCancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterCGONativeAdapter) StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (proto.GreeterBroadcastNativeStreamSession, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	if a.callbacks.BroadcastStart == nil || a.callbacks.BroadcastRecv == nil || a.callbacks.BroadcastDone == nil || a.callbacks.BroadcastCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	input, cleanup, err := encodeGreeterBroadcastCGONativeServerStreamRequest(name, city)
	_ = input
	if err != nil {
		return nil, err
	}
	defer cleanup()
	var stream C.int32_t
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamStartCallback(a.callbacks.BroadcastStart, input.NamePtr, input.NameLen, input.NameOwnership, input.CityPtr, input.CityLen, input.CityOwnership, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterBroadcastCGONativeServerStreamSession{callbacks: a.callbacks, stream: stream}, nil
}

type greeterBroadcastCGONativeServerStreamSession struct {
	callbacks C.GreeterCGONativeServerCallbacks
	stream    C.int32_t
}

func (s *greeterBroadcastCGONativeServerStreamSession) Recv(ctx context.Context) (string, error) {
	output := &GreeterBroadcastCGONativeServerStreamResponse{}
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamRecvCallback(s.callbacks.BroadcastRecv, s.stream, &output.MessagePtr, &output.MessageLen, &output.MessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterBroadcastCGONativeServerStreamResponse(output)
	cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
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

func (s *greeterChatCGONativeBidiStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	input, cleanup, err := encodeGreeterChatCGONativeBidiStreamRequest(name, city)
	_ = input
	if err != nil {
		return err
	}
	defer cleanup()
	errID := int32(C.callGreeterChatCGONativeBidiStreamSendCallback(s.callbacks.ChatSend, s.stream, input.NamePtr, input.NameLen, input.NameOwnership, input.CityPtr, input.CityLen, input.CityOwnership))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Recv(ctx context.Context) (string, error) {
	output := &GreeterChatCGONativeBidiStreamResponse{}
	errID := int32(C.callGreeterChatCGONativeBidiStreamRecvCallback(s.callbacks.ChatRecv, s.stream, &output.MessagePtr, &output.MessageLen, &output.MessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterChatCGONativeBidiStreamResponse(output)
	cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
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

func encodeGreeterSayHelloCGONativeUnaryRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (*GreeterSayHelloCGONativeUnaryRequest, func(), error) {
	input := &GreeterSayHelloCGONativeUnaryRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLen, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, namePtr, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if namePtr != 0 {
		pinned = append(pinned, namePtr)
	}
	input.NamePtr = C.uintptr_t(namePtr)
	input.NameLen = C.int32_t(nameLen)
	cityLen, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, cityPtr, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if cityPtr != 0 {
		pinned = append(pinned, cityPtr)
	}
	input.CityPtr = C.uintptr_t(cityPtr)
	input.CityLen = C.int32_t(cityLen)
	return input, cleanup, nil
}

func decodeGreeterSayHelloCGONativeUnaryResponse(output *GreeterSayHelloCGONativeUnaryResponse) (string, error) {
	if output == nil {
		return "", errors.New("rpccgo: cgo native server response output is nil")
	}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	MessageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	messageResult := MessageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterSayHelloCGONativeUnaryResponse(output *GreeterSayHelloCGONativeUnaryResponse) error {
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

func encodeGreeterCollectCGONativeClientStreamRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (*GreeterCollectCGONativeClientStreamRequest, func(), error) {
	input := &GreeterCollectCGONativeClientStreamRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLen, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, namePtr, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if namePtr != 0 {
		pinned = append(pinned, namePtr)
	}
	input.NamePtr = C.uintptr_t(namePtr)
	input.NameLen = C.int32_t(nameLen)
	cityLen, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, cityPtr, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if cityPtr != 0 {
		pinned = append(pinned, cityPtr)
	}
	input.CityPtr = C.uintptr_t(cityPtr)
	input.CityLen = C.int32_t(cityLen)
	return input, cleanup, nil
}

func decodeGreeterCollectCGONativeClientStreamResponse(output *GreeterCollectCGONativeClientStreamResponse) (string, error) {
	if output == nil {
		return "", errors.New("rpccgo: cgo native server response output is nil")
	}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	MessageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	messageResult := MessageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterCollectCGONativeClientStreamResponse(output *GreeterCollectCGONativeClientStreamResponse) error {
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

func encodeGreeterBroadcastCGONativeServerStreamRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (*GreeterBroadcastCGONativeServerStreamRequest, func(), error) {
	input := &GreeterBroadcastCGONativeServerStreamRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLen, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, namePtr, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if namePtr != 0 {
		pinned = append(pinned, namePtr)
	}
	input.NamePtr = C.uintptr_t(namePtr)
	input.NameLen = C.int32_t(nameLen)
	cityLen, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, cityPtr, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if cityPtr != 0 {
		pinned = append(pinned, cityPtr)
	}
	input.CityPtr = C.uintptr_t(cityPtr)
	input.CityLen = C.int32_t(cityLen)
	return input, cleanup, nil
}

func decodeGreeterBroadcastCGONativeServerStreamResponse(output *GreeterBroadcastCGONativeServerStreamResponse) (string, error) {
	if output == nil {
		return "", errors.New("rpccgo: cgo native server response output is nil")
	}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	MessageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	messageResult := MessageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterBroadcastCGONativeServerStreamResponse(output *GreeterBroadcastCGONativeServerStreamResponse) error {
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

func encodeGreeterChatCGONativeBidiStreamRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (*GreeterChatCGONativeBidiStreamRequest, func(), error) {
	input := &GreeterChatCGONativeBidiStreamRequest{}
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLen, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, namePtr, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if namePtr != 0 {
		pinned = append(pinned, namePtr)
	}
	input.NamePtr = C.uintptr_t(namePtr)
	input.NameLen = C.int32_t(nameLen)
	cityLen, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	_, cityPtr, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if cityPtr != 0 {
		pinned = append(pinned, cityPtr)
	}
	input.CityPtr = C.uintptr_t(cityPtr)
	input.CityLen = C.int32_t(cityLen)
	return input, cleanup, nil
}

func decodeGreeterChatCGONativeBidiStreamResponse(output *GreeterChatCGONativeBidiStreamResponse) (string, error) {
	if output == nil {
		return "", errors.New("rpccgo: cgo native server response output is nil")
	}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	MessageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(output.MessagePtr))), int32(output.MessageLen), false)
	messageResult := MessageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterChatCGONativeBidiStreamResponse(output *GreeterChatCGONativeBidiStreamResponse) error {
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
	SayHello        func(ctx context.Context, input *GreeterSayHelloCGONativeUnaryRequest, output *GreeterSayHelloCGONativeUnaryResponse) int32
	CollectStart    func(ctx context.Context, stream *C.int32_t) int32
	CollectSend     func(ctx context.Context, stream C.int32_t, input *GreeterCollectCGONativeClientStreamRequest) int32
	CollectFinish   func(ctx context.Context, stream C.int32_t, output *GreeterCollectCGONativeClientStreamResponse) int32
	CollectCancel   func(ctx context.Context, stream C.int32_t) int32
	BroadcastStart  func(ctx context.Context, input *GreeterBroadcastCGONativeServerStreamRequest, stream *C.int32_t) int32
	BroadcastRecv   func(ctx context.Context, stream C.int32_t, output *GreeterBroadcastCGONativeServerStreamResponse) int32
	BroadcastDone   func(ctx context.Context, stream C.int32_t) int32
	BroadcastCancel func(ctx context.Context, stream C.int32_t) int32
	ChatStart       func(ctx context.Context, stream *C.int32_t) int32
	ChatSend        func(ctx context.Context, stream C.int32_t, input *GreeterChatCGONativeBidiStreamRequest) int32
	ChatRecv        func(ctx context.Context, stream C.int32_t, output *GreeterChatCGONativeBidiStreamResponse) int32
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

func (a *greeterGoCGONativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	input, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(name, city)
	if err != nil {
		return "", err
	}
	defer cleanup()
	output := &GreeterSayHelloCGONativeUnaryResponse{}
	errID := a.callbacks.SayHello(ctx, input, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterSayHelloCGONativeUnaryResponse(output)
	cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
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

func (s *greeterCollectGoCGONativeClientStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	input, cleanup, err := encodeGreeterCollectCGONativeClientStreamRequest(name, city)
	_ = input
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

func (s *greeterCollectGoCGONativeClientStreamSession) Finish(ctx context.Context) (string, error) {
	output := &GreeterCollectCGONativeClientStreamResponse{}
	errID := s.callbacks.CollectFinish(ctx, s.stream, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterCollectCGONativeClientStreamResponse(output)
	cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (s *greeterCollectGoCGONativeClientStreamSession) Cancel(ctx context.Context) error {
	errID := s.callbacks.CollectCancel(ctx, s.stream)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterGoCGONativeAdapter) StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (proto.GreeterBroadcastNativeStreamSession, error) {
	input, cleanup, err := encodeGreeterBroadcastCGONativeServerStreamRequest(name, city)
	_ = input
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

func (s *greeterBroadcastGoCGONativeServerStreamSession) Recv(ctx context.Context) (string, error) {
	output := &GreeterBroadcastCGONativeServerStreamResponse{}
	errID := s.callbacks.BroadcastRecv(ctx, s.stream, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterBroadcastCGONativeServerStreamResponse(output)
	cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
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

func (s *greeterChatGoCGONativeBidiStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	input, cleanup, err := encodeGreeterChatCGONativeBidiStreamRequest(name, city)
	_ = input
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

func (s *greeterChatGoCGONativeBidiStreamSession) Recv(ctx context.Context) (string, error) {
	output := &GreeterChatCGONativeBidiStreamResponse{}
	errID := s.callbacks.ChatRecv(ctx, s.stream, output)
	if errID != 0 {
		cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterChatCGONativeBidiStreamResponse(output)
	cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(output)
	if cleanupErr != nil {
		if err != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", cleanupErr
	}
	if err != nil {
		return "", err
	}
	return messageResult, nil
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
