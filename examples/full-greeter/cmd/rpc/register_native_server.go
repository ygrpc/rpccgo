//go:build cgo

package main

/*
#include <stdint.h>
*/
import "C"

import (
	backend "example.com/rpccgo-full/internal/backend"
	greeterv1 "example.com/rpccgo-full/proto"
	rpcruntime "rpccgo/rpcruntime"
)

//export rpccgo_full_greeter_register_native_server
func rpccgo_full_greeter_register_native_server() C.int32_t {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}
