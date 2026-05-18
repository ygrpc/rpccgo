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
	sync "sync"
	unsafe "unsafe"
)

// rpccgo native generated file for Greeter cgo native server

var (
	greeterCGONativeServerCallbacksNil         = errors.New("rpccgo: Greeter cgo native server callbacks are nil")
	greeterCGONativeServerUnaryCallbackMissing = errors.New("rpccgo: Greeter cgo native server unary callback is missing")
	greeterCGONativeServerUnsupportedField     = errors.New("rpccgo: cgo native server field bridge is not implemented")
	greeterCGONativeServerStreamNotImplemented = errors.New("rpccgo: cgo native server streaming is not implemented")
	greeterCGONativeServerAdapterMu            sync.Mutex
	greeterCGONativeServerAdapter              = &greeterCGONativeAdapter{}
)

type greeterCGONativeAdapter struct {
	SayHelloCallback C.GreeterSayHelloCGONativeUnaryCallback
	CollectStart     C.GreeterCollectCGONativeClientStreamStartCallback
	CollectSend      C.GreeterCollectCGONativeClientStreamSendCallback
	CollectFinish    C.GreeterCollectCGONativeClientStreamFinishCallback
	CollectCancel    C.GreeterCollectCGONativeClientStreamCancelCallback
	BroadcastStart   C.GreeterBroadcastCGONativeServerStreamStartCallback
	BroadcastRecv    C.GreeterBroadcastCGONativeServerStreamRecvCallback
	BroadcastDone    C.GreeterBroadcastCGONativeServerStreamDoneCallback
	BroadcastCancel  C.GreeterBroadcastCGONativeServerStreamCancelCallback
	ChatStart        C.GreeterChatCGONativeBidiStreamStartCallback
	ChatSend         C.GreeterChatCGONativeBidiStreamSendCallback
	ChatRecv         C.GreeterChatCGONativeBidiStreamRecvCallback
	ChatCloseSend    C.GreeterChatCGONativeBidiStreamCloseSendCallback
	ChatDone         C.GreeterChatCGONativeBidiStreamDoneCallback
	ChatCancel       C.GreeterChatCGONativeBidiStreamCancelCallback
}

func (a *greeterCGONativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	if a == nil {
		return "", greeterCGONativeServerCallbacksNil
	}
	callback := a.SayHelloCallback
	if callback == nil {
		return "", greeterCGONativeServerUnaryCallbackMissing
	}
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(name, city)
	if err != nil {
		return "", err
	}
	defer cleanup()
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := int32(C.callGreeterSayHelloCGONativeUnaryCallback(callback, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, &outMessagePtr, &outMessageLen, &outMessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterSayHelloCGONativeUnaryResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	if a.CollectStart == nil || a.CollectSend == nil || a.CollectFinish == nil || a.CollectCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	var stream C.int32_t
	errID := int32(C.callGreeterCollectCGONativeClientStreamStartCallback(a.CollectStart, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterCollectCGONativeClientStreamSession{send: a.CollectSend, finish: a.CollectFinish, cancel: a.CollectCancel, stream: stream}, nil
}

type greeterCollectCGONativeClientStreamSession struct {
	send   C.GreeterCollectCGONativeClientStreamSendCallback
	finish C.GreeterCollectCGONativeClientStreamFinishCallback
	cancel C.GreeterCollectCGONativeClientStreamCancelCallback
	stream C.int32_t
}

func (s *greeterCollectCGONativeClientStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterCollectCGONativeClientStreamRequest(name, city)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := int32(C.callGreeterCollectCGONativeClientStreamSendCallback(s.send, s.stream, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterCollectCGONativeClientStreamSession) Finish(ctx context.Context) (string, error) {
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := int32(C.callGreeterCollectCGONativeClientStreamFinishCallback(s.finish, s.stream, &outMessagePtr, &outMessageLen, &outMessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterCollectCGONativeClientStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	errID := int32(C.callGreeterCollectCGONativeClientStreamCancelCallback(s.cancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterCGONativeAdapter) StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (proto.GreeterBroadcastNativeStreamSession, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	if a.BroadcastStart == nil || a.BroadcastRecv == nil || a.BroadcastDone == nil || a.BroadcastCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterBroadcastCGONativeServerStreamRequest(name, city)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	var stream C.int32_t
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamStartCallback(a.BroadcastStart, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterBroadcastCGONativeServerStreamSession{recv: a.BroadcastRecv, done: a.BroadcastDone, cancel: a.BroadcastCancel, stream: stream}, nil
}

type greeterBroadcastCGONativeServerStreamSession struct {
	recv   C.GreeterBroadcastCGONativeServerStreamRecvCallback
	done   C.GreeterBroadcastCGONativeServerStreamDoneCallback
	cancel C.GreeterBroadcastCGONativeServerStreamCancelCallback
	stream C.int32_t
}

func (s *greeterBroadcastCGONativeServerStreamSession) Recv(ctx context.Context) (string, error) {
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamRecvCallback(s.recv, s.stream, &outMessagePtr, &outMessageLen, &outMessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterBroadcastCGONativeServerStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamDoneCallback(s.done, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterBroadcastCGONativeServerStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterBroadcastCGONativeServerStreamCancelCallback(s.cancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (a *greeterCGONativeAdapter) StartChat(ctx context.Context) (proto.GreeterChatNativeStreamSession, error) {
	if a == nil {
		return nil, greeterCGONativeServerCallbacksNil
	}
	if a.ChatStart == nil || a.ChatSend == nil || a.ChatRecv == nil || a.ChatCloseSend == nil || a.ChatDone == nil || a.ChatCancel == nil {
		return nil, greeterCGONativeServerStreamNotImplemented
	}
	var stream C.int32_t
	errID := int32(C.callGreeterChatCGONativeBidiStreamStartCallback(a.ChatStart, &stream))
	if errID != 0 {
		return nil, greeterCGONativeServerErrorFromID(errID)
	}
	return &greeterChatCGONativeBidiStreamSession{send: a.ChatSend, recv: a.ChatRecv, closeSend: a.ChatCloseSend, done: a.ChatDone, cancel: a.ChatCancel, stream: stream}, nil
}

type greeterChatCGONativeBidiStreamSession struct {
	send      C.GreeterChatCGONativeBidiStreamSendCallback
	recv      C.GreeterChatCGONativeBidiStreamRecvCallback
	closeSend C.GreeterChatCGONativeBidiStreamCloseSendCallback
	done      C.GreeterChatCGONativeBidiStreamDoneCallback
	cancel    C.GreeterChatCGONativeBidiStreamCancelCallback
	stream    C.int32_t
}

func (s *greeterChatCGONativeBidiStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterChatCGONativeBidiStreamRequest(name, city)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := int32(C.callGreeterChatCGONativeBidiStreamSendCallback(s.send, s.stream, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Recv(ctx context.Context) (string, error) {
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := int32(C.callGreeterChatCGONativeBidiStreamRecvCallback(s.recv, s.stream, &outMessagePtr, &outMessageLen, &outMessageOwnership))
	if errID != 0 {
		cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterChatCGONativeBidiStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	errID := int32(C.callGreeterChatCGONativeBidiStreamCloseSendCallback(s.closeSend, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Done(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGONativeBidiStreamDoneCallback(s.done, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatCGONativeBidiStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGONativeBidiStreamCancelCallback(s.cancel, s.stream))
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func encodeGreeterSayHelloCGONativeUnaryRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (C.uintptr_t, C.int32_t, C.int32_t, C.uintptr_t, C.int32_t, C.int32_t, func(), error) {
	var namePtr C.uintptr_t
	var nameLen C.int32_t
	var nameOwnership C.int32_t
	var cityPtr C.uintptr_t
	var cityLen C.int32_t
	var cityOwnership C.int32_t
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLenValue, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, namePtrValue, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if namePtrValue != 0 {
		pinned = append(pinned, namePtrValue)
	}
	namePtr = C.uintptr_t(namePtrValue)
	nameLen = C.int32_t(nameLenValue)
	cityLenValue, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, cityPtrValue, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if cityPtrValue != 0 {
		pinned = append(pinned, cityPtrValue)
	}
	cityPtr = C.uintptr_t(cityPtrValue)
	cityLen = C.int32_t(cityLenValue)
	return namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, nil
}

func decodeGreeterSayHelloCGONativeUnaryResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) (string, error) {
	if _, err := rpcruntime.LengthFromInt32(int32(messageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	messageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(messagePtr))), int32(messageLen), false)
	messageResult := messageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterSayHelloCGONativeUnaryResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) error {
	var cleanupErr error
	if messageOwnership > 0 && messagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(messagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func encodeGreeterCollectCGONativeClientStreamRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (C.uintptr_t, C.int32_t, C.int32_t, C.uintptr_t, C.int32_t, C.int32_t, func(), error) {
	var namePtr C.uintptr_t
	var nameLen C.int32_t
	var nameOwnership C.int32_t
	var cityPtr C.uintptr_t
	var cityLen C.int32_t
	var cityOwnership C.int32_t
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLenValue, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, namePtrValue, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if namePtrValue != 0 {
		pinned = append(pinned, namePtrValue)
	}
	namePtr = C.uintptr_t(namePtrValue)
	nameLen = C.int32_t(nameLenValue)
	cityLenValue, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, cityPtrValue, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if cityPtrValue != 0 {
		pinned = append(pinned, cityPtrValue)
	}
	cityPtr = C.uintptr_t(cityPtrValue)
	cityLen = C.int32_t(cityLenValue)
	return namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, nil
}

func decodeGreeterCollectCGONativeClientStreamResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) (string, error) {
	if _, err := rpcruntime.LengthFromInt32(int32(messageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	messageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(messagePtr))), int32(messageLen), false)
	messageResult := messageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterCollectCGONativeClientStreamResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) error {
	var cleanupErr error
	if messageOwnership > 0 && messagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(messagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func encodeGreeterBroadcastCGONativeServerStreamRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (C.uintptr_t, C.int32_t, C.int32_t, C.uintptr_t, C.int32_t, C.int32_t, func(), error) {
	var namePtr C.uintptr_t
	var nameLen C.int32_t
	var nameOwnership C.int32_t
	var cityPtr C.uintptr_t
	var cityLen C.int32_t
	var cityOwnership C.int32_t
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLenValue, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, namePtrValue, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if namePtrValue != 0 {
		pinned = append(pinned, namePtrValue)
	}
	namePtr = C.uintptr_t(namePtrValue)
	nameLen = C.int32_t(nameLenValue)
	cityLenValue, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, cityPtrValue, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if cityPtrValue != 0 {
		pinned = append(pinned, cityPtrValue)
	}
	cityPtr = C.uintptr_t(cityPtrValue)
	cityLen = C.int32_t(cityLenValue)
	return namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, nil
}

func decodeGreeterBroadcastCGONativeServerStreamResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) (string, error) {
	if _, err := rpcruntime.LengthFromInt32(int32(messageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	messageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(messagePtr))), int32(messageLen), false)
	messageResult := messageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterBroadcastCGONativeServerStreamResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) error {
	var cleanupErr error
	if messageOwnership > 0 && messagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(messagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func encodeGreeterChatCGONativeBidiStreamRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) (C.uintptr_t, C.int32_t, C.int32_t, C.uintptr_t, C.int32_t, C.int32_t, func(), error) {
	var namePtr C.uintptr_t
	var nameLen C.int32_t
	var nameOwnership C.int32_t
	var cityPtr C.uintptr_t
	var cityLen C.int32_t
	var cityOwnership C.int32_t
	var pinned []uintptr
	cleanup := func() {
		for i := len(pinned) - 1; i >= 0; i-- {
			rpcruntime.Release(pinned[i])
		}
	}
	nameLenValue, err := rpcruntime.LengthToInt32(len(name.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, namePtrValue, err := rpcruntime.PinString(name.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if namePtrValue != 0 {
		pinned = append(pinned, namePtrValue)
	}
	namePtr = C.uintptr_t(namePtrValue)
	nameLen = C.int32_t(nameLenValue)
	cityLenValue, err := rpcruntime.LengthToInt32(len(city.SafeString()))
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	_, cityPtrValue, err := rpcruntime.PinString(city.SafeString())
	if err != nil {
		cleanup()
		return 0, 0, 0, 0, 0, 0, func() {}, err
	}
	if cityPtrValue != 0 {
		pinned = append(pinned, cityPtrValue)
	}
	cityPtr = C.uintptr_t(cityPtrValue)
	cityLen = C.int32_t(cityLenValue)
	return namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, nil
}

func decodeGreeterChatCGONativeBidiStreamResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) (string, error) {
	if _, err := rpcruntime.LengthFromInt32(int32(messageLen)); err != nil {
		return "", fmt.Errorf("examples.full.greeter.v1.SayHelloResponse.message: %w", err)
	}
	messageWrapper := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(uintptr(messagePtr))), int32(messageLen), false)
	messageResult := messageWrapper.SafeString()
	return messageResult, nil
}

func cleanupGreeterChatCGONativeBidiStreamResponse(messagePtr C.uintptr_t, messageLen C.int32_t, messageOwnership C.int32_t) error {
	var cleanupErr error
	if messageOwnership > 0 && messagePtr != 0 {
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(messagePtr)), true, "examples.full.greeter.v1.SayHelloResponse.message"); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
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

//export rpccgo_native_greeterv1_Greeter_SayHello_register
func rpccgo_native_greeterv1_Greeter_SayHello_register(callback C.GreeterSayHelloCGONativeUnaryCallback) C.int32_t {
	if callback == nil {
		return C.int32_t(rpcruntime.StoreError(greeterCGONativeServerUnaryCallbackMissing))
	}
	greeterCGONativeServerAdapterMu.Lock()
	defer greeterCGONativeServerAdapterMu.Unlock()
	greeterCGONativeServerAdapter.SayHelloCallback = callback
	_, err := proto.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, greeterCGONativeServerAdapter)
	if err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}

//export rpccgo_native_greeterv1_Greeter_Collect_register
func rpccgo_native_greeterv1_Greeter_Collect_register(start C.GreeterCollectCGONativeClientStreamStartCallback, send C.GreeterCollectCGONativeClientStreamSendCallback, finish C.GreeterCollectCGONativeClientStreamFinishCallback, cancel C.GreeterCollectCGONativeClientStreamCancelCallback) C.int32_t {
	if start == nil || send == nil || finish == nil || cancel == nil {
		return C.int32_t(rpcruntime.StoreError(greeterCGONativeServerStreamNotImplemented))
	}
	greeterCGONativeServerAdapterMu.Lock()
	defer greeterCGONativeServerAdapterMu.Unlock()
	greeterCGONativeServerAdapter.CollectStart = start
	greeterCGONativeServerAdapter.CollectSend = send
	greeterCGONativeServerAdapter.CollectFinish = finish
	greeterCGONativeServerAdapter.CollectCancel = cancel
	_, err := proto.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, greeterCGONativeServerAdapter)
	if err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}

//export rpccgo_native_greeterv1_Greeter_Broadcast_register
func rpccgo_native_greeterv1_Greeter_Broadcast_register(start C.GreeterBroadcastCGONativeServerStreamStartCallback, recv C.GreeterBroadcastCGONativeServerStreamRecvCallback, done C.GreeterBroadcastCGONativeServerStreamDoneCallback, cancel C.GreeterBroadcastCGONativeServerStreamCancelCallback) C.int32_t {
	if start == nil || recv == nil || done == nil || cancel == nil {
		return C.int32_t(rpcruntime.StoreError(greeterCGONativeServerStreamNotImplemented))
	}
	greeterCGONativeServerAdapterMu.Lock()
	defer greeterCGONativeServerAdapterMu.Unlock()
	greeterCGONativeServerAdapter.BroadcastStart = start
	greeterCGONativeServerAdapter.BroadcastRecv = recv
	greeterCGONativeServerAdapter.BroadcastDone = done
	greeterCGONativeServerAdapter.BroadcastCancel = cancel
	_, err := proto.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, greeterCGONativeServerAdapter)
	if err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}

//export rpccgo_native_greeterv1_Greeter_Chat_register
func rpccgo_native_greeterv1_Greeter_Chat_register(start C.GreeterChatCGONativeBidiStreamStartCallback, send C.GreeterChatCGONativeBidiStreamSendCallback, recv C.GreeterChatCGONativeBidiStreamRecvCallback, closeSend C.GreeterChatCGONativeBidiStreamCloseSendCallback, done C.GreeterChatCGONativeBidiStreamDoneCallback, cancel C.GreeterChatCGONativeBidiStreamCancelCallback) C.int32_t {
	if start == nil || send == nil || recv == nil || closeSend == nil || done == nil || cancel == nil {
		return C.int32_t(rpcruntime.StoreError(greeterCGONativeServerStreamNotImplemented))
	}
	greeterCGONativeServerAdapterMu.Lock()
	defer greeterCGONativeServerAdapterMu.Unlock()
	greeterCGONativeServerAdapter.ChatStart = start
	greeterCGONativeServerAdapter.ChatSend = send
	greeterCGONativeServerAdapter.ChatRecv = recv
	greeterCGONativeServerAdapter.ChatCloseSend = closeSend
	greeterCGONativeServerAdapter.ChatDone = done
	greeterCGONativeServerAdapter.ChatCancel = cancel
	_, err := proto.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, greeterCGONativeServerAdapter)
	if err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}

type GreeterGoCGONativeServerCallbacks struct {
	SayHello        func(ctx context.Context, namePtr C.uintptr_t, nameLen C.int32_t, nameOwnership C.int32_t, cityPtr C.uintptr_t, cityLen C.int32_t, cityOwnership C.int32_t, outMessagePtr *C.uintptr_t, outMessageLen *C.int32_t, outMessageOwnership *C.int32_t) int32
	CollectStart    func(ctx context.Context, stream *C.int32_t) int32
	CollectSend     func(ctx context.Context, stream C.int32_t, namePtr C.uintptr_t, nameLen C.int32_t, nameOwnership C.int32_t, cityPtr C.uintptr_t, cityLen C.int32_t, cityOwnership C.int32_t) int32
	CollectFinish   func(ctx context.Context, stream C.int32_t, outMessagePtr *C.uintptr_t, outMessageLen *C.int32_t, outMessageOwnership *C.int32_t) int32
	CollectCancel   func(ctx context.Context, stream C.int32_t) int32
	BroadcastStart  func(ctx context.Context, namePtr C.uintptr_t, nameLen C.int32_t, nameOwnership C.int32_t, cityPtr C.uintptr_t, cityLen C.int32_t, cityOwnership C.int32_t, stream *C.int32_t) int32
	BroadcastRecv   func(ctx context.Context, stream C.int32_t, outMessagePtr *C.uintptr_t, outMessageLen *C.int32_t, outMessageOwnership *C.int32_t) int32
	BroadcastDone   func(ctx context.Context, stream C.int32_t) int32
	BroadcastCancel func(ctx context.Context, stream C.int32_t) int32
	ChatStart       func(ctx context.Context, stream *C.int32_t) int32
	ChatSend        func(ctx context.Context, stream C.int32_t, namePtr C.uintptr_t, nameLen C.int32_t, nameOwnership C.int32_t, cityPtr C.uintptr_t, cityLen C.int32_t, cityOwnership C.int32_t) int32
	ChatRecv        func(ctx context.Context, stream C.int32_t, outMessagePtr *C.uintptr_t, outMessageLen *C.int32_t, outMessageOwnership *C.int32_t) int32
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
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(name, city)
	if err != nil {
		return "", err
	}
	defer cleanup()
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := a.callbacks.SayHello(ctx, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, &outMessagePtr, &outMessageLen, &outMessageOwnership)
	if errID != 0 {
		cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterSayHelloCGONativeUnaryResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterSayHelloCGONativeUnaryResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterCollectCGONativeClientStreamRequest(name, city)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := s.callbacks.CollectSend(ctx, s.stream, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterCollectGoCGONativeClientStreamSession) Finish(ctx context.Context) (string, error) {
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := s.callbacks.CollectFinish(ctx, s.stream, &outMessagePtr, &outMessageLen, &outMessageOwnership)
	if errID != 0 {
		cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterCollectCGONativeClientStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterCollectCGONativeClientStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterBroadcastCGONativeServerStreamRequest(name, city)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	var stream C.int32_t
	errID := a.callbacks.BroadcastStart(ctx, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, &stream)
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
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := s.callbacks.BroadcastRecv(ctx, s.stream, &outMessagePtr, &outMessageLen, &outMessageOwnership)
	if errID != 0 {
		cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterBroadcastCGONativeServerStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterBroadcastCGONativeServerStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
	namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership, cleanup, err := encodeGreeterChatCGONativeBidiStreamRequest(name, city)
	if err != nil {
		return err
	}
	defer cleanup()
	errID := s.callbacks.ChatSend(ctx, s.stream, namePtr, nameLen, nameOwnership, cityPtr, cityLen, cityOwnership)
	if errID != 0 {
		return greeterCGONativeServerErrorFromID(errID)
	}
	return nil
}

func (s *greeterChatGoCGONativeBidiStreamSession) Recv(ctx context.Context) (string, error) {
	var outMessagePtr C.uintptr_t
	var outMessageLen C.int32_t
	var outMessageOwnership C.int32_t
	errID := s.callbacks.ChatRecv(ctx, s.stream, &outMessagePtr, &outMessageLen, &outMessageOwnership)
	if errID != 0 {
		cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
		callbackErr := greeterCGONativeServerErrorFromID(errID)
		if cleanupErr != nil {
			return "", errors.Join(callbackErr, cleanupErr)
		}
		return "", callbackErr
	}
	messageResult, err := decodeGreeterChatCGONativeBidiStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
	cleanupErr := cleanupGreeterChatCGONativeBidiStreamResponse(outMessagePtr, outMessageLen, outMessageOwnership)
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
