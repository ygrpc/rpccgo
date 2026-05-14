//go:build cgo

package main

/*
#include <stdint.h>
*/
import "C"

import (
	greeterv1 "example.com/rpccgo-minimal/gen/greeter/v1"
	"example.com/rpccgo-minimal/internal/backend"
	rpcruntime "rpccgo/rpcruntime"
)

//export rpccgo_minimal_greeter_register_native_server
func rpccgo_minimal_greeter_register_native_server() C.int32_t {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		return C.int32_t(rpcruntime.StoreError(err))
	}
	return 0
}
