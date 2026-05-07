package main

import (
	v1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

/*
#include <stdint.h>

typedef struct GreeterSayHelloCGONativeUnaryRequest {
uintptr_t NamePtr;
int32_t NameLen;
} GreeterSayHelloCGONativeUnaryRequest;

typedef struct GreeterSayHelloCGONativeUnaryResponse {
uintptr_t MessagePtr;
int32_t MessageLen;
int32_t MessageOwnership;
} GreeterSayHelloCGONativeUnaryResponse;

typedef int32_t (*GreeterSayHelloCGONativeUnaryCallback)(GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output);

typedef struct GreeterCGONativeServerCallbacks {
GreeterSayHelloCGONativeUnaryCallback SayHello;
} GreeterCGONativeServerCallbacks;

static inline int32_t callGreeterSayHelloCGONativeUnaryCallback(GreeterSayHelloCGONativeUnaryCallback callback, GreeterSayHelloCGONativeUnaryRequest* input, GreeterSayHelloCGONativeUnaryResponse* output) {
	return callback(input, output);
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

func (a *greeterCGONativeAdapter) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
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

func encodeGreeterSayHelloCGONativeUnaryRequest(req *v1.SayHelloRequest) (*C.GreeterSayHelloCGONativeUnaryRequest, func(), error) {
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
	return input, cleanup, nil
}

func decodeGreeterSayHelloCGONativeUnaryResponse(output *C.GreeterSayHelloCGONativeUnaryResponse) (*v1.SayHelloResponse, error) {
	if output == nil {
		return nil, errors.New("rpccgo: cgo native server response output is nil")
	}
	resp := &v1.SayHelloResponse{}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return nil, fmt.Errorf("examples.minimal.greeter.v1.SayHelloResponse.message: %w", err)
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
		if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.MessagePtr)), true, "examples.minimal.greeter.v1.SayHelloResponse.message"); err != nil {
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

func RegisterGreeterCGONativeServer(callbacks *C.GreeterCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter], error) {
	if callbacks == nil {
		return rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter]{}, greeterCGONativeServerCallbacksNil
	}
	if callbacks.SayHello == nil {
		return rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter]{}, greeterCGONativeServerUnaryCallbackMissing
	}
	callbacksCopy := *callbacks
	return v1.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, &greeterCGONativeAdapter{callbacks: callbacksCopy})
}

type GreeterGoCGONativeServerCallbacks struct {
	SayHello func(ctx context.Context, input *C.GreeterSayHelloCGONativeUnaryRequest, output *C.GreeterSayHelloCGONativeUnaryResponse) int32
}

func RegisterGreeterGoCGONativeServerForTesting(callbacks *GreeterGoCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter], error) {
	if callbacks == nil {
		return rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter]{}, greeterCGONativeServerCallbacksNil
	}
	if callbacks.SayHello == nil {
		return rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter]{}, greeterCGONativeServerUnaryCallbackMissing
	}
	return v1.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, &greeterGoCGONativeAdapter{callbacks: callbacks})
}

type greeterGoCGONativeAdapter struct {
	callbacks *GreeterGoCGONativeServerCallbacks
}

func (a *greeterGoCGONativeAdapter) SayHello(ctx context.Context, req *v1.SayHelloRequest) (*v1.SayHelloResponse, error) {
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
