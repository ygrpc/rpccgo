//go:build android && cgo

package main

/*
#include <jni.h>
#include <stdlib.h>

static const char* rpccgoGetStringUTFChars(JNIEnv* env, jstring value) {
	if (value == NULL) {
		return NULL;
	}
	return (*env)->GetStringUTFChars(env, value, NULL);
}

static void rpccgoReleaseStringUTFChars(JNIEnv* env, jstring value, const char* chars) {
	if (value == NULL || chars == NULL) {
		return;
	}
	(*env)->ReleaseStringUTFChars(env, value, chars);
}

static jstring rpccgoNewStringUTF(JNIEnv* env, const char* chars) {
	return (*env)->NewStringUTF(env, chars);
}
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"

	fluttersharedv1 "example.com/rpccgo-flutter-shared-so/proto"
)

// Java_com_ygrpc_examples_rpccgofluttersharedso_MainActivity_nativeComposeGreeting
// forwards the Kotlin/JNI request into the rpccgo-generated message invoke path
// inside the same shared library that Flutter FFI resolves from the process.
//
//export Java_com_ygrpc_examples_rpccgofluttersharedso_MainActivity_nativeComposeGreeting
func Java_com_ygrpc_examples_rpccgofluttersharedso_MainActivity_nativeComposeGreeting(env *C.JNIEnv, _ C.jobject, value C.jstring) C.jstring {
	name := goJNIString(env, value)
	resp, err := fluttersharedv1.InvokeSharedSoDemoMessageComposeGreeting(context.Background(), &fluttersharedv1.ComposeGreetingRequest{
		Name:   name,
		Caller: "kotlin-jni",
	})
	if err != nil {
		return newJNIString(env, fmt.Sprintf("jni error: %v", err))
	}
	return newJNIString(env, fmt.Sprintf("%s | served_by=%s | library=%s", resp.GetMessage(), resp.GetServedBy(), resp.GetLibrary()))
}

// Java_com_ygrpc_examples_rpccgofluttersharedso_MainActivity_nativeReadRuntimeState
// reads the Go state previously modified through Flutter FFI.
//
//export Java_com_ygrpc_examples_rpccgofluttersharedso_MainActivity_nativeReadRuntimeState
func Java_com_ygrpc_examples_rpccgofluttersharedso_MainActivity_nativeReadRuntimeState(env *C.JNIEnv, _ C.jobject) C.jstring {
	resp, err := fluttersharedv1.InvokeSharedSoDemoMessageReadRuntimeState(context.Background(), &fluttersharedv1.ReadRuntimeStateRequest{
		Caller: "kotlin-jni",
	})
	if err != nil {
		return newJNIString(env, fmt.Sprintf("jni runtime read error: %v", err))
	}
	return newJNIString(env, fmt.Sprintf("runtime_id=%s | value=%d | revision=%d | caller=%s", resp.GetRuntimeId(), resp.GetValue(), resp.GetRevision(), resp.GetCaller()))
}

func goJNIString(env *C.JNIEnv, value C.jstring) string {
	chars := C.rpccgoGetStringUTFChars(env, value)
	if chars == nil {
		return ""
	}
	defer C.rpccgoReleaseStringUTFChars(env, value, chars)
	return C.GoString(chars)
}

func newJNIString(env *C.JNIEnv, value string) C.jstring {
	cstr := C.CString(value)
	defer C.free(unsafe.Pointer(cstr))
	return C.rpccgoNewStringUTF(env, cstr)
}
