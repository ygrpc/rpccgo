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
	errID := rpccgoNativeGreeterv1GreeterSayHello(C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterNativeCollectStart() (int32, int32) {
	var stream C.int32_t
	errID := rpccgoNativeGreeterv1GreeterCollectStart(&stream)
	return int32(stream), int32(errID)
}

func greeterNativeCollectSend(stream int32, namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32) int32 {
	return int32(rpccgoNativeGreeterv1GreeterCollectSend(C.int32_t(stream), C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership)))
}

func greeterNativeCollectFinish(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgoNativeGreeterv1GreeterCollectFinish(C.int32_t(stream), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterNativeBroadcastStart(namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32) (int32, int32) {
	var stream C.int32_t
	errID := rpccgoNativeGreeterv1GreeterBroadcastStart(C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership), &stream)
	return int32(stream), int32(errID)
}

func greeterNativeBroadcastRecv(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgoNativeGreeterv1GreeterBroadcastRecv(C.int32_t(stream), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterNativeBroadcastFinish(stream int32) int32 {
	return int32(rpccgoNativeGreeterv1GreeterBroadcastFinish(C.int32_t(stream)))
}

func greeterNativeChatStart() (int32, int32) {
	var stream C.int32_t
	errID := rpccgoNativeGreeterv1GreeterChatStart(&stream)
	return int32(stream), int32(errID)
}

func greeterNativeChatSend(stream int32, namePtr uintptr, nameLen int32, nameOwnership int32, cityPtr uintptr, cityLen int32, cityOwnership int32) int32 {
	return int32(rpccgoNativeGreeterv1GreeterChatSend(C.int32_t(stream), C.uintptr_t(namePtr), C.int32_t(nameLen), C.int32_t(nameOwnership), C.uintptr_t(cityPtr), C.int32_t(cityLen), C.int32_t(cityOwnership)))
}

func greeterNativeChatRecv(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	var messageOwnership C.int32_t
	errID := rpccgoNativeGreeterv1GreeterChatRecv(C.int32_t(stream), &messagePtr, &messageLen, &messageOwnership)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterNativeChatCloseSend(stream int32) int32 {
	return int32(rpccgoNativeGreeterv1GreeterChatCloseSend(C.int32_t(stream)))
}

func greeterNativeChatFinish(stream int32) int32 {
	return int32(rpccgoNativeGreeterv1GreeterChatFinish(C.int32_t(stream)))
}

func callGreeterSayHelloMessageUnary(requestPtr uintptr, requestLen int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgoMsgGreeterv1GreeterSayHello(C.uintptr_t(requestPtr), C.int32_t(requestLen), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterMessageCollectStart() (int32, int32) {
	var stream C.int32_t
	errID := rpccgoMsgGreeterv1GreeterCollectStart(&stream)
	return int32(stream), int32(errID)
}

func greeterMessageCollectSend(stream int32, requestPtr uintptr, requestLen int32) int32 {
	return int32(rpccgoMsgGreeterv1GreeterCollectSend(C.int32_t(stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
}

func greeterMessageCollectFinish(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgoMsgGreeterv1GreeterCollectFinish(C.int32_t(stream), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterMessageBroadcastStart(requestPtr uintptr, requestLen int32) (int32, int32) {
	var stream C.int32_t
	errID := rpccgoMsgGreeterv1GreeterBroadcastStart(C.uintptr_t(requestPtr), C.int32_t(requestLen), &stream)
	return int32(stream), int32(errID)
}

func greeterMessageBroadcastRecv(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgoMsgGreeterv1GreeterBroadcastRecv(C.int32_t(stream), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterMessageBroadcastFinish(stream int32) int32 {
	return int32(rpccgoMsgGreeterv1GreeterBroadcastFinish(C.int32_t(stream)))
}

func greeterMessageChatStart() (int32, int32) {
	var stream C.int32_t
	errID := rpccgoMsgGreeterv1GreeterChatStart(&stream)
	return int32(stream), int32(errID)
}

func greeterMessageChatSend(stream int32, requestPtr uintptr, requestLen int32) int32 {
	return int32(rpccgoMsgGreeterv1GreeterChatSend(C.int32_t(stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
}

func greeterMessageChatRecv(stream int32, outMessagePtr *uintptr, outMessageLen *int32) int32 {
	var messagePtr C.uintptr_t
	var messageLen C.int32_t
	errID := rpccgoMsgGreeterv1GreeterChatRecv(C.int32_t(stream), &messagePtr, &messageLen)
	*outMessagePtr = uintptr(messagePtr)
	*outMessageLen = int32(messageLen)
	return int32(errID)
}

func greeterMessageChatCloseSend(stream int32) int32 {
	return int32(rpccgoMsgGreeterv1GreeterChatCloseSend(C.int32_t(stream)))
}

func greeterMessageChatFinish(stream int32) int32 {
	return int32(rpccgoMsgGreeterv1GreeterChatFinish(C.int32_t(stream)))
}
