//go:build cgo

package main

import (
	greeterv1 "example.com/rpccgo-minimal/gen/greeter/v1"
	"example.com/rpccgo-minimal/internal/backend"
)

func init() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		panic(err)
	}
}
