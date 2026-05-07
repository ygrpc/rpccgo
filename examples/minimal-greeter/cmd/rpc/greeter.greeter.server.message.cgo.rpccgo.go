package main

import (
	v1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

/*
#include <stdint.h>

typedef int32_t (*GreeterSayHelloCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);

typedef struct GreeterCGOMessageServerCallbacks {
GreeterSayHelloCGOMessageUnaryCallback SayHello;
} GreeterCGOMessageServerCallbacks;

static inline int32_t callGreeterSayHelloCGOMessageUnary(GreeterSayHelloCGOMessageUnaryCallback callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {
	return callback(request_ptr, request_len, response_ptr, response_len);
}

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

func RegisterGreeterCGOMessageServer(callbacks *C.GreeterCGOMessageServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterMessageAdapter], error) {
	if callbacks == nil {
		return rpcruntime.AdapterSnapshot[v1.GreeterMessageAdapter]{}, greeterCGOMessageServerCallbacksNil
	}
	if callbacks.SayHello == nil {
		return rpcruntime.AdapterSnapshot[v1.GreeterMessageAdapter]{}, greeterCGOMessageServerUnaryCallbackMissing
	}
	callbacksCopy := *callbacks
	return v1.RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindCGOMessage, &greeterCGOMessageAdapter{callbacks: callbacksCopy})
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
