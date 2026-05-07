package main

import (
	proto "example.com/rpccgo-full/proto"
)

/*
#include <stdint.h>

typedef int32_t (*GreeterSayHelloCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);

typedef int32_t (*GreeterCollectCGOMessageClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterCollectCGOMessageClientStreamSendCallback)(int32_t stream, uintptr_t request_ptr, int32_t request_len);
typedef int32_t (*GreeterCollectCGOMessageClientStreamFinishCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterCollectCGOMessageClientStreamCancelCallback)(int32_t stream);

typedef int32_t (*GreeterBroadcastCGOMessageServerStreamStartCallback)(uintptr_t request_ptr, int32_t request_len, int32_t* stream);
typedef int32_t (*GreeterBroadcastCGOMessageServerStreamRecvCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterBroadcastCGOMessageServerStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterBroadcastCGOMessageServerStreamCancelCallback)(int32_t stream);

typedef int32_t (*GreeterChatCGOMessageBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamSendCallback)(int32_t stream, uintptr_t request_ptr, int32_t request_len);
typedef int32_t (*GreeterChatCGOMessageBidiStreamRecvCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterChatCGOMessageBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGOMessageServerCallbacks {
GreeterSayHelloCGOMessageUnaryCallback SayHello;
GreeterCollectCGOMessageClientStreamStartCallback CollectStart;
GreeterCollectCGOMessageClientStreamSendCallback CollectSend;
GreeterCollectCGOMessageClientStreamFinishCallback CollectFinish;
GreeterCollectCGOMessageClientStreamCancelCallback CollectCancel;
GreeterBroadcastCGOMessageServerStreamStartCallback BroadcastStart;
GreeterBroadcastCGOMessageServerStreamRecvCallback BroadcastRecv;
GreeterBroadcastCGOMessageServerStreamDoneCallback BroadcastDone;
GreeterBroadcastCGOMessageServerStreamCancelCallback BroadcastCancel;
GreeterChatCGOMessageBidiStreamStartCallback ChatStart;
GreeterChatCGOMessageBidiStreamSendCallback ChatSend;
GreeterChatCGOMessageBidiStreamRecvCallback ChatRecv;
GreeterChatCGOMessageBidiStreamCloseSendCallback ChatCloseSend;
GreeterChatCGOMessageBidiStreamDoneCallback ChatDone;
GreeterChatCGOMessageBidiStreamCancelCallback ChatCancel;
} GreeterCGOMessageServerCallbacks;

static inline int32_t callGreeterSayHelloCGOMessageUnary(GreeterSayHelloCGOMessageUnaryCallback callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {
	return callback(request_ptr, request_len, response_ptr, response_len);
}

static inline int32_t callGreeterCollectCGOMessageClientStreamStart(GreeterCollectCGOMessageClientStreamStartCallback callback, int32_t* stream) { return callback(stream); }
static inline int32_t callGreeterCollectCGOMessageClientStreamSend(GreeterCollectCGOMessageClientStreamSendCallback callback, int32_t stream, uintptr_t request_ptr, int32_t request_len) { return callback(stream, request_ptr, request_len); }
static inline int32_t callGreeterCollectCGOMessageClientStreamFinish(GreeterCollectCGOMessageClientStreamFinishCallback callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }
static inline int32_t callGreeterCollectCGOMessageClientStreamCancel(GreeterCollectCGOMessageClientStreamCancelCallback callback, int32_t stream) { return callback(stream); }

static inline int32_t callGreeterBroadcastCGOMessageServerStreamStart(GreeterBroadcastCGOMessageServerStreamStartCallback callback, uintptr_t request_ptr, int32_t request_len, int32_t* stream) { return callback(request_ptr, request_len, stream); }
static inline int32_t callGreeterBroadcastCGOMessageServerStreamRecv(GreeterBroadcastCGOMessageServerStreamRecvCallback callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }
static inline int32_t callGreeterBroadcastCGOMessageServerStreamDone(GreeterBroadcastCGOMessageServerStreamDoneCallback callback, int32_t stream) { return callback(stream); }
static inline int32_t callGreeterBroadcastCGOMessageServerStreamCancel(GreeterBroadcastCGOMessageServerStreamCancelCallback callback, int32_t stream) { return callback(stream); }

static inline int32_t callGreeterChatCGOMessageBidiStreamStart(GreeterChatCGOMessageBidiStreamStartCallback callback, int32_t* stream) { return callback(stream); }
static inline int32_t callGreeterChatCGOMessageBidiStreamSend(GreeterChatCGOMessageBidiStreamSendCallback callback, int32_t stream, uintptr_t request_ptr, int32_t request_len) { return callback(stream, request_ptr, request_len); }
static inline int32_t callGreeterChatCGOMessageBidiStreamRecv(GreeterChatCGOMessageBidiStreamRecvCallback callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }
static inline int32_t callGreeterChatCGOMessageBidiStreamCloseSend(GreeterChatCGOMessageBidiStreamCloseSendCallback callback, int32_t stream) { return callback(stream); }
static inline int32_t callGreeterChatCGOMessageBidiStreamDone(GreeterChatCGOMessageBidiStreamDoneCallback callback, int32_t stream) { return callback(stream); }
static inline int32_t callGreeterChatCGOMessageBidiStreamCancel(GreeterChatCGOMessageBidiStreamCancelCallback callback, int32_t stream) { return callback(stream); }

*/
import "C"

import (
	context "context"
	errors "errors"
	fmt "fmt"
	io "io"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
)

// rpccgo message direct stage file for Greeter cgo message server callbacks

var (
	greeterCGOMessageServerCallbacksNil         = errors.New("rpccgo: Greeter cgo message server callbacks are nil")
	greeterCGOMessageServerUnaryCallbackMissing = errors.New("rpccgo: Greeter cgo message server unary callback is missing")
)

type greeterCGOMessageAdapter struct {
	callbacks C.GreeterCGOMessageServerCallbacks
}

func (a *greeterCGOMessageAdapter) SayHelloMessage(ctx context.Context, req []byte) ([]byte, error) {
	if a == nil {
		return nil, greeterCGOMessageServerCallbacksNil
	}
	callback := a.callbacks.SayHello
	if callback == nil {
		return nil, greeterCGOMessageServerUnaryCallbackMissing
	}
	var requestPtr uintptr
	if len(req) != 0 {
		requestPtr = uintptr(unsafe.Pointer(&req[0]))
	}
	requestLen, err := rpcruntime.LengthToInt32(len(req))
	if err != nil {
		return nil, err
	}
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := int32(C.callGreeterSayHelloCGOMessageUnary(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	if responseLen < 0 {
		return nil, errors.New("rpccgo: message server response length is negative")
	}
	if responseLen == 0 {
		return nil, nil
	}
	if responsePtr == 0 {
		return nil, errors.New("rpccgo: message server response pointer is nil")
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(uintptr(responsePtr))), int(responseLen))...), nil
}

func (a *greeterCGOMessageAdapter) StartCollectMessage(ctx context.Context) (proto.GreeterCollectMessageStreamSession, error) {
	if a == nil {
		return nil, greeterCGOMessageServerCallbacksNil
	}
	if a.callbacks.CollectStart == nil {
		return nil, greeterCGOMessageServerUnaryCallbackMissing
	}
	var stream C.int32_t
	errID := int32(C.callGreeterCollectCGOMessageClientStreamStart(a.callbacks.CollectStart, &stream))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	return &greeterCollectCGOMessageClientStreamSession{callbacks: a.callbacks, stream: int32(stream)}, nil
}

type greeterCollectCGOMessageClientStreamSession struct {
	callbacks C.GreeterCGOMessageServerCallbacks
	stream    int32
}

func (s *greeterCollectCGOMessageClientStreamSession) Send(ctx context.Context, req []byte) error {
	var requestPtr uintptr
	if len(req) != 0 {
		requestPtr = uintptr(unsafe.Pointer(&req[0]))
	}
	requestLen, err := rpcruntime.LengthToInt32(len(req))
	if err != nil {
		return err
	}
	errID := int32(C.callGreeterCollectCGOMessageClientStreamSend(s.callbacks.CollectSend, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (s *greeterCollectCGOMessageClientStreamSession) Finish(ctx context.Context) ([]byte, error) {
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := int32(C.callGreeterCollectCGOMessageClientStreamFinish(s.callbacks.CollectFinish, C.int32_t(s.stream), &responsePtr, &responseLen))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	if responseLen < 0 {
		return nil, errors.New("rpccgo: message server response length is negative")
	}
	if responseLen == 0 {
		return nil, nil
	}
	if responsePtr == 0 {
		return nil, errors.New("rpccgo: message server response pointer is nil")
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(uintptr(responsePtr))), int(responseLen))...), nil
}

func (s *greeterCollectCGOMessageClientStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterCollectCGOMessageClientStreamCancel(s.callbacks.CollectCancel, C.int32_t(s.stream)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (a *greeterCGOMessageAdapter) StartBroadcastMessage(ctx context.Context, req []byte) (proto.GreeterBroadcastMessageStreamSession, error) {
	if a == nil {
		return nil, greeterCGOMessageServerCallbacksNil
	}
	if a.callbacks.BroadcastStart == nil {
		return nil, greeterCGOMessageServerUnaryCallbackMissing
	}
	var requestPtr uintptr
	if len(req) != 0 {
		requestPtr = uintptr(unsafe.Pointer(&req[0]))
	}
	requestLen, err := rpcruntime.LengthToInt32(len(req))
	if err != nil {
		return nil, err
	}
	var stream C.int32_t
	errID := int32(C.callGreeterBroadcastCGOMessageServerStreamStart(a.callbacks.BroadcastStart, C.uintptr_t(requestPtr), C.int32_t(requestLen), &stream))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	return &greeterBroadcastCGOMessageServerStreamSession{callbacks: a.callbacks, stream: int32(stream)}, nil
}

type greeterBroadcastCGOMessageServerStreamSession struct {
	callbacks C.GreeterCGOMessageServerCallbacks
	stream    int32
}

func (s *greeterBroadcastCGOMessageServerStreamSession) Recv(ctx context.Context) ([]byte, error) {
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := int32(C.callGreeterBroadcastCGOMessageServerStreamRecv(s.callbacks.BroadcastRecv, C.int32_t(s.stream), &responsePtr, &responseLen))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	if responseLen < 0 {
		return nil, errors.New("rpccgo: message server response length is negative")
	}
	if responseLen == 0 {
		return nil, nil
	}
	if responsePtr == 0 {
		return nil, errors.New("rpccgo: message server response pointer is nil")
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(uintptr(responsePtr))), int(responseLen))...), nil
}

func (s *greeterBroadcastCGOMessageServerStreamSession) Done(ctx context.Context) error {
	errID := int32(C.callGreeterBroadcastCGOMessageServerStreamDone(s.callbacks.BroadcastDone, C.int32_t(s.stream)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (s *greeterBroadcastCGOMessageServerStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterBroadcastCGOMessageServerStreamCancel(s.callbacks.BroadcastCancel, C.int32_t(s.stream)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (a *greeterCGOMessageAdapter) StartChatMessage(ctx context.Context) (proto.GreeterChatMessageStreamSession, error) {
	if a == nil {
		return nil, greeterCGOMessageServerCallbacksNil
	}
	if a.callbacks.ChatStart == nil {
		return nil, greeterCGOMessageServerUnaryCallbackMissing
	}
	var stream C.int32_t
	errID := int32(C.callGreeterChatCGOMessageBidiStreamStart(a.callbacks.ChatStart, &stream))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	return &greeterChatCGOMessageBidiStreamSession{callbacks: a.callbacks, stream: int32(stream)}, nil
}

type greeterChatCGOMessageBidiStreamSession struct {
	callbacks C.GreeterCGOMessageServerCallbacks
	stream    int32
}

func (s *greeterChatCGOMessageBidiStreamSession) Send(ctx context.Context, req []byte) error {
	var requestPtr uintptr
	if len(req) != 0 {
		requestPtr = uintptr(unsafe.Pointer(&req[0]))
	}
	requestLen, err := rpcruntime.LengthToInt32(len(req))
	if err != nil {
		return err
	}
	errID := int32(C.callGreeterChatCGOMessageBidiStreamSend(s.callbacks.ChatSend, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (s *greeterChatCGOMessageBidiStreamSession) Recv(ctx context.Context) ([]byte, error) {
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := int32(C.callGreeterChatCGOMessageBidiStreamRecv(s.callbacks.ChatRecv, C.int32_t(s.stream), &responsePtr, &responseLen))
	if errID != 0 {
		return nil, greeterCGOMessageServerError(errID)
	}
	if responseLen < 0 {
		return nil, errors.New("rpccgo: message server response length is negative")
	}
	if responseLen == 0 {
		return nil, nil
	}
	if responsePtr == 0 {
		return nil, errors.New("rpccgo: message server response pointer is nil")
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(uintptr(responsePtr))), int(responseLen))...), nil
}

func (s *greeterChatCGOMessageBidiStreamSession) CloseSend(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGOMessageBidiStreamCloseSend(s.callbacks.ChatCloseSend, C.int32_t(s.stream)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (s *greeterChatCGOMessageBidiStreamSession) Done(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGOMessageBidiStreamDone(s.callbacks.ChatDone, C.int32_t(s.stream)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func (s *greeterChatCGOMessageBidiStreamSession) Cancel(ctx context.Context) error {
	errID := int32(C.callGreeterChatCGOMessageBidiStreamCancel(s.callbacks.ChatCancel, C.int32_t(s.stream)))
	if errID != 0 {
		return greeterCGOMessageServerError(errID)
	}
	return nil
}

func RegisterGreeterCGOMessageServer(callbacks *C.GreeterCGOMessageServerCallbacks) (rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter], error) {
	if callbacks == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerCallbacksNil
	}
	if callbacks.SayHello == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.CollectStart == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.CollectSend == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.CollectFinish == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.CollectCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.BroadcastStart == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.BroadcastRecv == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.BroadcastDone == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.BroadcastCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.ChatStart == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.ChatSend == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.ChatRecv == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.ChatCloseSend == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.ChatDone == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	if callbacks.ChatCancel == nil {
		return rpcruntime.AdapterSnapshot[proto.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	callbacksCopy := *callbacks
	return proto.RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindCGOMessage, &greeterCGOMessageAdapter{callbacks: callbacksCopy})
}

func greeterCGOMessageServerError(errID int32) error {
	if errID == 0 {
		return nil
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if ok {
		if ptr != 0 {
			defer rpcruntime.Release(ptr)
		}
		if string(text) == io.EOF.Error() {
			return io.EOF
		}
		return errors.New(string(text))
	}
	return fmt.Errorf("rpccgo: cgo message server callback returned unknown error id %d", errID)
}

func GreeterCGOMessageStreamEOFErrorID() int32 {
	return int32(rpcruntime.StoreError(io.EOF))
}
