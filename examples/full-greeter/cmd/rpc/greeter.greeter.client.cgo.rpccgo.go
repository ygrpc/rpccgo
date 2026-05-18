package main

import (
	proto "example.com/rpccgo-full/proto"
)

/*
#include <stdint.h>
*/
import "C"

import (
	context "context"
	errors "errors"
	fmt "fmt"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
)

// rpccgo native generated file for Greeter cgo native client

var greeterNativeClientUnsupportedField = errors.New("rpccgo: native unary client field bridge is not implemented")
var greeterNativeClientStreamHandleInvalid = errors.New("rpccgo: native client stream handle is invalid")

func CallGreeterSayHelloNativeUnary(ctx context.Context, NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateGreeterSayHelloNativeUnaryResponse(outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	nameValue, cityValue, err := decodeGreeterSayHelloNativeUnaryRequest(NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	messageResult, err := proto.NewGreeterCGONativeClientBridge().SayHello(ctx, nameValue, cityValue)
	if cleanupErr := errors.Join(nameValue.Release(), cityValue.Release()); cleanupErr != nil {
		err = errors.Join(err, cleanupErr)
	}
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := encodeGreeterSayHelloNativeUnaryResponse(messageResult, outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterSayHelloNativeUnaryRequest(NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var decoded []interface{ Release() error }
	cleanupDecoded := func() error {
		var errs []error
		for i := len(decoded) - 1; i >= 0; i-- {
			errs = append(errs, decoded[i].Release())
		}
		return errors.Join(errs...)
	}
	if _, err := rpcruntime.LengthFromInt32(NameLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err), cleanupDecoded())
	}
	var nameValue *rpcruntime.RpcString
	if NamePtr == 0 || NameLen == 0 {
		nameValue = rpcruntime.EmptyRpcString()
	} else {
		nameValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(NamePtr)), NameLen, NameOwnership > 0)
	}
	decoded = append(decoded, nameValue)
	if _, err := rpcruntime.LengthFromInt32(CityLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err), cleanupDecoded())
	}
	var cityValue *rpcruntime.RpcString
	if CityPtr == 0 || CityLen == 0 {
		cityValue = rpcruntime.EmptyRpcString()
	} else {
		cityValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(CityPtr)), CityLen, CityOwnership > 0)
	}
	decoded = append(decoded, cityValue)
	return nameValue, cityValue, nil
}

func validateGreeterSayHelloNativeUnaryResponse(outMessagePtr *uintptr, outMessageLen *int32) error {
	if outMessagePtr == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	if outMessageLen == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	return nil
}

func encodeGreeterSayHelloNativeUnaryResponse(messageResult string, outMessagePtr *uintptr, outMessageLen *int32) error {
	if err := validateGreeterSayHelloNativeUnaryResponse(outMessagePtr, outMessageLen); err != nil {
		return err
	}
	messageLenValue, err := rpcruntime.LengthToInt32(len(messageResult))
	if err != nil {
		return err
	}
	data, messagePtrValue, err := rpcruntime.PinString(messageResult)
	_ = data
	if err != nil {
		return err
	}
	_ = messagePtrValue
	*outMessagePtr = messagePtrValue
	*outMessageLen = messageLenValue
	return nil
}

//export rpccgo_native_greeterv1_Greeter_SayHello
func rpccgo_native_greeterv1_Greeter_SayHello(NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32, outMessagePtr *uintptr, outMessageLen *int32) C.int32_t {
	if outMessagePtr != nil {
		*outMessagePtr = 0
	}
	if outMessageLen != nil {
		*outMessageLen = 0
	}
	if outMessagePtr == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	if outMessageLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	return C.int32_t(CallGreeterSayHelloNativeUnary(context.Background(), uintptr(NamePtr), int32(NameLen), int32(NameOwnership), uintptr(CityPtr), int32(CityLen), int32(CityOwnership), (*uintptr)(unsafe.Pointer(outMessagePtr)), (*int32)(unsafe.Pointer(outMessageLen))))
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

func SendGreeterCollectNativeClientStream(ctx context.Context, handle int32, NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.LoadDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterCollectNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	var err error
	nameValue, cityValue, err := decodeGreeterCollectNativeClientStreamRequest(NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	err = session.Send(ctx, nameValue, cityValue)
	if cleanupErr := errors.Join(nameValue.Release(), cityValue.Release()); cleanupErr != nil {
		err = errors.Join(err, cleanupErr)
	}
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func FinishGreeterCollectNativeClientStream(ctx context.Context, handle int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateGreeterCollectNativeClientStreamResponse(outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	session, ok := rpcruntime.TakeDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterCollectNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	messageResult, err := session.Finish(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := encodeGreeterCollectNativeClientStreamResponse(messageResult, outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CancelGreeterCollectNativeClientStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.TakeDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterCollectNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	err := session.Cancel(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterCollectNativeClientStreamRequest(NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var decoded []interface{ Release() error }
	cleanupDecoded := func() error {
		var errs []error
		for i := len(decoded) - 1; i >= 0; i-- {
			errs = append(errs, decoded[i].Release())
		}
		return errors.Join(errs...)
	}
	if _, err := rpcruntime.LengthFromInt32(NameLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err), cleanupDecoded())
	}
	var nameValue *rpcruntime.RpcString
	if NamePtr == 0 || NameLen == 0 {
		nameValue = rpcruntime.EmptyRpcString()
	} else {
		nameValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(NamePtr)), NameLen, NameOwnership > 0)
	}
	decoded = append(decoded, nameValue)
	if _, err := rpcruntime.LengthFromInt32(CityLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err), cleanupDecoded())
	}
	var cityValue *rpcruntime.RpcString
	if CityPtr == 0 || CityLen == 0 {
		cityValue = rpcruntime.EmptyRpcString()
	} else {
		cityValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(CityPtr)), CityLen, CityOwnership > 0)
	}
	decoded = append(decoded, cityValue)
	return nameValue, cityValue, nil
}

func validateGreeterCollectNativeClientStreamResponse(outMessagePtr *uintptr, outMessageLen *int32) error {
	if outMessagePtr == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	if outMessageLen == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	return nil
}

func encodeGreeterCollectNativeClientStreamResponse(messageResult string, outMessagePtr *uintptr, outMessageLen *int32) error {
	if err := validateGreeterCollectNativeClientStreamResponse(outMessagePtr, outMessageLen); err != nil {
		return err
	}
	messageLenValue, err := rpcruntime.LengthToInt32(len(messageResult))
	if err != nil {
		return err
	}
	data, messagePtrValue, err := rpcruntime.PinString(messageResult)
	_ = data
	if err != nil {
		return err
	}
	_ = messagePtrValue
	*outMessagePtr = messagePtrValue
	*outMessageLen = messageLenValue
	return nil
}

//export rpccgo_native_greeterv1_Greeter_Collect_start
func rpccgo_native_greeterv1_Greeter_Collect_start(handle *C.int32_t) C.int32_t {
	if handle != nil {
		*handle = 0
	}
	if handle == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client handle pointer is nil")))
	}
	handleValue, errID := StartGreeterCollectNativeClientStream(context.Background())
	if errID != 0 {
		return C.int32_t(errID)
	}
	*handle = C.int32_t(handleValue)
	return 0
}

//export rpccgo_native_greeterv1_Greeter_Collect_send
func rpccgo_native_greeterv1_Greeter_Collect_send(handle C.int32_t, NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) C.int32_t {
	return C.int32_t(SendGreeterCollectNativeClientStream(context.Background(), int32(handle), uintptr(NamePtr), int32(NameLen), int32(NameOwnership), uintptr(CityPtr), int32(CityLen), int32(CityOwnership)))
}

//export rpccgo_native_greeterv1_Greeter_Collect_finish
func rpccgo_native_greeterv1_Greeter_Collect_finish(handle C.int32_t, outMessagePtr *uintptr, outMessageLen *int32) C.int32_t {
	if outMessagePtr != nil {
		*outMessagePtr = 0
	}
	if outMessageLen != nil {
		*outMessageLen = 0
	}
	if outMessagePtr == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	if outMessageLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	return C.int32_t(FinishGreeterCollectNativeClientStream(context.Background(), int32(handle), (*uintptr)(unsafe.Pointer(outMessagePtr)), (*int32)(unsafe.Pointer(outMessageLen))))
}

//export rpccgo_native_greeterv1_Greeter_Collect_cancel
func rpccgo_native_greeterv1_Greeter_Collect_cancel(handle C.int32_t) C.int32_t {
	return C.int32_t(CancelGreeterCollectNativeClientStream(context.Background(), int32(handle)))
}

func StartGreeterBroadcastNativeServerStream(ctx context.Context, NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) (int32, int32) {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	nameValue, cityValue, err := decodeGreeterBroadcastNativeServerStreamRequest(NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership)
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	handle, err := proto.NewGreeterCGONativeClientBridge().StartBroadcast(ctx, nameValue, cityValue)
	if cleanupErr := errors.Join(nameValue.Release(), cityValue.Release()); cleanupErr != nil {
		err = errors.Join(err, cleanupErr)
	}
	if err != nil {
		return 0, int32(rpcruntime.StoreError(err))
	}
	return int32(handle), 0
}

func ReadGreeterBroadcastNativeServerStream(ctx context.Context, handle int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateGreeterBroadcastNativeServerStreamResponse(outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	session, ok := rpcruntime.LoadDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterBroadcastNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	messageResult, err := session.Recv(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := encodeGreeterBroadcastNativeServerStreamResponse(messageResult, outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func DoneGreeterBroadcastNativeServerStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.TakeDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterBroadcastNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	var err error
	if done, ok := session.(interface{ Done(context.Context) error }); ok {
		err = done.Done(ctx)
	}
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CancelGreeterBroadcastNativeServerStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.TakeDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterBroadcastNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	err := session.Cancel(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterBroadcastNativeServerStreamRequest(NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var decoded []interface{ Release() error }
	cleanupDecoded := func() error {
		var errs []error
		for i := len(decoded) - 1; i >= 0; i-- {
			errs = append(errs, decoded[i].Release())
		}
		return errors.Join(errs...)
	}
	if _, err := rpcruntime.LengthFromInt32(NameLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err), cleanupDecoded())
	}
	var nameValue *rpcruntime.RpcString
	if NamePtr == 0 || NameLen == 0 {
		nameValue = rpcruntime.EmptyRpcString()
	} else {
		nameValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(NamePtr)), NameLen, NameOwnership > 0)
	}
	decoded = append(decoded, nameValue)
	if _, err := rpcruntime.LengthFromInt32(CityLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err), cleanupDecoded())
	}
	var cityValue *rpcruntime.RpcString
	if CityPtr == 0 || CityLen == 0 {
		cityValue = rpcruntime.EmptyRpcString()
	} else {
		cityValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(CityPtr)), CityLen, CityOwnership > 0)
	}
	decoded = append(decoded, cityValue)
	return nameValue, cityValue, nil
}

func validateGreeterBroadcastNativeServerStreamResponse(outMessagePtr *uintptr, outMessageLen *int32) error {
	if outMessagePtr == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	if outMessageLen == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	return nil
}

func encodeGreeterBroadcastNativeServerStreamResponse(messageResult string, outMessagePtr *uintptr, outMessageLen *int32) error {
	if err := validateGreeterBroadcastNativeServerStreamResponse(outMessagePtr, outMessageLen); err != nil {
		return err
	}
	messageLenValue, err := rpcruntime.LengthToInt32(len(messageResult))
	if err != nil {
		return err
	}
	data, messagePtrValue, err := rpcruntime.PinString(messageResult)
	_ = data
	if err != nil {
		return err
	}
	_ = messagePtrValue
	*outMessagePtr = messagePtrValue
	*outMessageLen = messageLenValue
	return nil
}

//export rpccgo_native_greeterv1_Greeter_Broadcast_start
func rpccgo_native_greeterv1_Greeter_Broadcast_start(NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32, handle *C.int32_t) C.int32_t {
	if handle != nil {
		*handle = 0
	}
	if handle == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client handle pointer is nil")))
	}
	handleValue, errID := StartGreeterBroadcastNativeServerStream(context.Background(), uintptr(NamePtr), int32(NameLen), int32(NameOwnership), uintptr(CityPtr), int32(CityLen), int32(CityOwnership))
	if errID != 0 {
		return C.int32_t(errID)
	}
	*handle = C.int32_t(handleValue)
	return 0
}

//export rpccgo_native_greeterv1_Greeter_Broadcast_read
func rpccgo_native_greeterv1_Greeter_Broadcast_read(handle C.int32_t, outMessagePtr *uintptr, outMessageLen *int32) C.int32_t {
	if outMessagePtr != nil {
		*outMessagePtr = 0
	}
	if outMessageLen != nil {
		*outMessageLen = 0
	}
	if outMessagePtr == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	if outMessageLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	return C.int32_t(ReadGreeterBroadcastNativeServerStream(context.Background(), int32(handle), (*uintptr)(unsafe.Pointer(outMessagePtr)), (*int32)(unsafe.Pointer(outMessageLen))))
}

//export rpccgo_native_greeterv1_Greeter_Broadcast_done
func rpccgo_native_greeterv1_Greeter_Broadcast_done(handle C.int32_t) C.int32_t {
	return C.int32_t(DoneGreeterBroadcastNativeServerStream(context.Background(), int32(handle)))
}

//export rpccgo_native_greeterv1_Greeter_Broadcast_cancel
func rpccgo_native_greeterv1_Greeter_Broadcast_cancel(handle C.int32_t) C.int32_t {
	return C.int32_t(CancelGreeterBroadcastNativeServerStream(context.Background(), int32(handle)))
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

func SendGreeterChatNativeBidiStream(ctx context.Context, handle int32, NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.LoadDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterChatNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	var err error
	nameValue, cityValue, err := decodeGreeterChatNativeBidiStreamRequest(NamePtr, NameLen, NameOwnership, CityPtr, CityLen, CityOwnership)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	err = session.Send(ctx, nameValue, cityValue)
	if cleanupErr := errors.Join(nameValue.Release(), cityValue.Release()); cleanupErr != nil {
		err = errors.Join(err, cleanupErr)
	}
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func ReadGreeterChatNativeBidiStream(ctx context.Context, handle int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateGreeterChatNativeBidiStreamResponse(outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	session, ok := rpcruntime.LoadDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterChatNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	messageResult, err := session.Recv(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	if err := encodeGreeterChatNativeBidiStreamResponse(messageResult, outMessagePtr, outMessageLen); err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CloseSendGreeterChatNativeBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.LoadDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterChatNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
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
	session, ok := rpcruntime.TakeDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterChatNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	var err error
	if done, ok := session.(interface{ Done(context.Context) error }); ok {
		err = done.Done(ctx)
	}
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func CancelGreeterChatNativeBidiStream(ctx context.Context, handle int32) int32 {
	if ctx == nil {
		ctx = context.Background()
	}
	session, ok := rpcruntime.TakeDispatcherStream[proto.GreeterActiveAdapter, proto.GreeterChatNativeStreamSession](proto.GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return int32(rpcruntime.StoreError(greeterNativeClientStreamHandleInvalid))
	}
	err := session.Cancel(ctx)
	if err != nil {
		return int32(rpcruntime.StoreError(err))
	}
	return 0
}

func decodeGreeterChatNativeBidiStreamRequest(NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var decoded []interface{ Release() error }
	cleanupDecoded := func() error {
		var errs []error
		for i := len(decoded) - 1; i >= 0; i-- {
			errs = append(errs, decoded[i].Release())
		}
		return errors.Join(errs...)
	}
	if _, err := rpcruntime.LengthFromInt32(NameLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.name: %w", err), cleanupDecoded())
	}
	var nameValue *rpcruntime.RpcString
	if NamePtr == 0 || NameLen == 0 {
		nameValue = rpcruntime.EmptyRpcString()
	} else {
		nameValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(NamePtr)), NameLen, NameOwnership > 0)
	}
	decoded = append(decoded, nameValue)
	if _, err := rpcruntime.LengthFromInt32(CityLen); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("examples.full.greeter.v1.SayHelloRequest.city: %w", err), cleanupDecoded())
	}
	var cityValue *rpcruntime.RpcString
	if CityPtr == 0 || CityLen == 0 {
		cityValue = rpcruntime.EmptyRpcString()
	} else {
		cityValue = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(CityPtr)), CityLen, CityOwnership > 0)
	}
	decoded = append(decoded, cityValue)
	return nameValue, cityValue, nil
}

func validateGreeterChatNativeBidiStreamResponse(outMessagePtr *uintptr, outMessageLen *int32) error {
	if outMessagePtr == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	if outMessageLen == nil {
		return errors.New("rpccgo: native client output pointer is nil")
	}
	return nil
}

func encodeGreeterChatNativeBidiStreamResponse(messageResult string, outMessagePtr *uintptr, outMessageLen *int32) error {
	if err := validateGreeterChatNativeBidiStreamResponse(outMessagePtr, outMessageLen); err != nil {
		return err
	}
	messageLenValue, err := rpcruntime.LengthToInt32(len(messageResult))
	if err != nil {
		return err
	}
	data, messagePtrValue, err := rpcruntime.PinString(messageResult)
	_ = data
	if err != nil {
		return err
	}
	_ = messagePtrValue
	*outMessagePtr = messagePtrValue
	*outMessageLen = messageLenValue
	return nil
}

//export rpccgo_native_greeterv1_Greeter_Chat_start
func rpccgo_native_greeterv1_Greeter_Chat_start(handle *C.int32_t) C.int32_t {
	if handle != nil {
		*handle = 0
	}
	if handle == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client handle pointer is nil")))
	}
	handleValue, errID := StartGreeterChatNativeBidiStream(context.Background())
	if errID != 0 {
		return C.int32_t(errID)
	}
	*handle = C.int32_t(handleValue)
	return 0
}

//export rpccgo_native_greeterv1_Greeter_Chat_send
func rpccgo_native_greeterv1_Greeter_Chat_send(handle C.int32_t, NamePtr uintptr, NameLen int32, NameOwnership int32, CityPtr uintptr, CityLen int32, CityOwnership int32) C.int32_t {
	return C.int32_t(SendGreeterChatNativeBidiStream(context.Background(), int32(handle), uintptr(NamePtr), int32(NameLen), int32(NameOwnership), uintptr(CityPtr), int32(CityLen), int32(CityOwnership)))
}

//export rpccgo_native_greeterv1_Greeter_Chat_read
func rpccgo_native_greeterv1_Greeter_Chat_read(handle C.int32_t, outMessagePtr *uintptr, outMessageLen *int32) C.int32_t {
	if outMessagePtr != nil {
		*outMessagePtr = 0
	}
	if outMessageLen != nil {
		*outMessageLen = 0
	}
	if outMessagePtr == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	if outMessageLen == nil {
		return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))
	}
	return C.int32_t(ReadGreeterChatNativeBidiStream(context.Background(), int32(handle), (*uintptr)(unsafe.Pointer(outMessagePtr)), (*int32)(unsafe.Pointer(outMessageLen))))
}

//export rpccgo_native_greeterv1_Greeter_Chat_close_send
func rpccgo_native_greeterv1_Greeter_Chat_close_send(handle C.int32_t) C.int32_t {
	return C.int32_t(CloseSendGreeterChatNativeBidiStream(context.Background(), int32(handle)))
}

//export rpccgo_native_greeterv1_Greeter_Chat_cancel
func rpccgo_native_greeterv1_Greeter_Chat_cancel(handle C.int32_t) C.int32_t {
	return C.int32_t(CancelGreeterChatNativeBidiStream(context.Background(), int32(handle)))
}
