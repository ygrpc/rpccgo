package main

/*
#include <stdint.h>
*/
import "C"

import (
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo cgo support generated file for shared exports

//export rpccgo_take_error_text
func rpccgo_take_error_text(errID C.int32_t, textPtr *C.uintptr_t, textLen *C.int32_t) C.int32_t {
	if textPtr != nil {
		*textPtr = 0
	}
	if textLen != nil {
		*textLen = 0
	}
	if textPtr == nil || textLen == nil {
		return 1
	}
	var goPtr uintptr
	var goLen int32
	status := rpcruntime.TakeErrorTextForExport(int32(errID), &goPtr, &goLen)
	if status != 0 {
		return C.int32_t(status)
	}
	*textPtr = C.uintptr_t(goPtr)
	*textLen = C.int32_t(goLen)
	return 0
}

//export rpccgo_release
func rpccgo_release(ptr C.uintptr_t) C.int32_t {
	if !rpcruntime.Release(uintptr(ptr)) {
		return 1
	}
	return 0
}
