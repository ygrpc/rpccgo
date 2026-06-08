package main

/*
#include <stdint.h>
*/
import "C"

// Go test files cannot import C, so these helpers keep C export coverage in Go tests.
func callGreeterSayHelloNativeUnary(namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_SayHello(C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func startGreeterCollectNativeClientStream() (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_Collect_start(&stream)
	return int32(stream), int32(errID)
}

func sendGreeterCollectNativeClientStream(stream int32, namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32) int32 {
	return int32(rpccgo_native_greeterv1_Greeter_Collect_send(C.int32_t(stream), C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership)))
}

func finishGreeterCollectNativeClientStream(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_Collect_finish(C.int32_t(stream), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func startGreeterBroadcastNativeServerStream(namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32) (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_Broadcast_start(C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership), &stream)
	return int32(stream), int32(errID)
}

func readGreeterBroadcastNativeServerStream(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_Broadcast_read(C.int32_t(stream), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func finishGreeterBroadcastNativeServerStream(stream int32) int32 {
	return int32(rpccgo_native_greeterv1_Greeter_Broadcast_finish(C.int32_t(stream)))
}

func startGreeterChatNativeBidiStream() (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_Chat_start(&stream)
	return int32(stream), int32(errID)
}

func sendGreeterChatNativeBidiStream(stream int32, namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32) int32 {
	return int32(rpccgo_native_greeterv1_Greeter_Chat_send(C.int32_t(stream), C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership)))
}

func readGreeterChatNativeBidiStream(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgo_native_greeterv1_Greeter_Chat_read(C.int32_t(stream), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func closeSendGreeterChatNativeBidiStream(stream int32) int32 {
	return int32(rpccgo_native_greeterv1_Greeter_Chat_close_send(C.int32_t(stream)))
}

func finishGreeterChatNativeBidiStream(stream int32) int32 {
	return int32(rpccgo_native_greeterv1_Greeter_Chat_finish(C.int32_t(stream)))
}

func callGreeterSayHelloMessageUnary(requestPtr uintptr, requestLen int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_SayHello(C.uintptr_t(requestPtr), C.int32_t(requestLen), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func startGreeterCollectMessageClientStream() (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_Collect_start(&stream)
	return int32(stream), int32(errID)
}

func sendGreeterCollectMessageClientStream(stream int32, requestPtr uintptr, requestLen int32) int32 {
	return int32(rpccgo_msg_greeterv1_Greeter_Collect_send(C.int32_t(stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
}

func finishGreeterCollectMessageClientStream(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_Collect_finish(C.int32_t(stream), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func startGreeterBroadcastMessageServerStream(requestPtr uintptr, requestLen int32) (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_Broadcast_start(C.uintptr_t(requestPtr), C.int32_t(requestLen), &stream)
	return int32(stream), int32(errID)
}

func readGreeterBroadcastMessageServerStream(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_Broadcast_read(C.int32_t(stream), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func finishGreeterBroadcastMessageServerStream(stream int32) int32 {
	return int32(rpccgo_msg_greeterv1_Greeter_Broadcast_finish(C.int32_t(stream)))
}

func startGreeterChatMessageBidiStream() (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_Chat_start(&stream)
	return int32(stream), int32(errID)
}

func sendGreeterChatMessageBidiStream(stream int32, requestPtr uintptr, requestLen int32) int32 {
	return int32(rpccgo_msg_greeterv1_Greeter_Chat_send(C.int32_t(stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
}

func readGreeterChatMessageBidiStream(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgo_msg_greeterv1_Greeter_Chat_read(C.int32_t(stream), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func closeSendGreeterChatMessageBidiStream(stream int32) int32 {
	return int32(rpccgo_msg_greeterv1_Greeter_Chat_close_send(C.int32_t(stream)))
}

func finishGreeterChatMessageBidiStream(stream int32) int32 {
	return int32(rpccgo_msg_greeterv1_Greeter_Chat_finish(C.int32_t(stream)))
}
