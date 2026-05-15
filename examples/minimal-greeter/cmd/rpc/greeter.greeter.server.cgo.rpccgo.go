package main

import (
	v1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

/*
#include <stdint.h>

typedef int32_t (*GreeterSayHelloCGONativeUnaryCallback)(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership);

static inline int32_t callGreeterSayHelloCGONativeUnaryCallback(GreeterSayHelloCGONativeUnaryCallback callback, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t *outMessagePtr, int32_t *outMessageLen, int32_t *outMessageOwnership) {
	return callback(NamePtr, NameLen, NameOwnership, outMessagePtr, outMessageLen, outMessageOwnership);
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

type GreeterSayHelloCGONativeUnaryRequest struct {
	NamePtr       C.uintptr_t
	NameLen       C.int32_t
	NameOwnership C.int32_t
}

type GreeterSayHelloCGONativeUnaryResponse struct {
	MessagePtr       C.uintptr_t
	MessageLen       C.int32_t
	MessageOwnership C.int32_t
}

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
}

func (a *greeterCGONativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error) {
	if a == nil {
		return "", greeterCGONativeServerCallbacksNil
	}
	callback := a.SayHelloCallback
	if callback == nil {
		return "", greeterCGONativeServerUnaryCallbackMissing
	}
	input, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(name)
	_ = input
	if err != nil {
		return "", err
	}
	defer cleanup()
	output := &GreeterSayHelloCGONativeUnaryResponse{}
	errID := int32(C.callGreeterSayHelloCGONativeUnaryCallback(callback, input.NamePtr, input.NameLen, input.NameOwnership, &output.MessagePtr, &output.MessageLen, &output.MessageOwnership))
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

func encodeGreeterSayHelloCGONativeUnaryRequest(name *rpcruntime.RpcString) (*GreeterSayHelloCGONativeUnaryRequest, func(), error) {
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
	return input, cleanup, nil
}

func decodeGreeterSayHelloCGONativeUnaryResponse(output *GreeterSayHelloCGONativeUnaryResponse) (string, error) {
	if output == nil {
		return "", errors.New("rpccgo: cgo native server response output is nil")
	}
	if _, err := rpcruntime.LengthFromInt32(int32(output.MessageLen)); err != nil {
		return "", fmt.Errorf("examples.minimal.greeter.v1.SayHelloResponse.message: %w", err)
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

//export rpccgo_native_greeterv1_Greeter_SayHello_register
func rpccgo_native_greeterv1_Greeter_SayHello_register(callback C.GreeterSayHelloCGONativeUnaryCallback) C.int32_t {
	if callback == nil {
		return C.int32_t(rpcruntime.StoreError(greeterCGONativeServerUnaryCallbackMissing))
	}
	greeterCGONativeServerAdapterMu.Lock()
	defer greeterCGONativeServerAdapterMu.Unlock()
	greeterCGONativeServerAdapter.SayHelloCallback = callback
	_, err := v1.RegisterGreeterCGONativeActiveServer(rpcruntime.ServerKindCGONative, greeterCGONativeServerAdapter)
	if err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}

type GreeterGoCGONativeServerCallbacks struct {
	SayHello func(ctx context.Context, input *GreeterSayHelloCGONativeUnaryRequest, output *GreeterSayHelloCGONativeUnaryResponse) int32
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

func (a *greeterGoCGONativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error) {
	input, cleanup, err := encodeGreeterSayHelloCGONativeUnaryRequest(name)
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
