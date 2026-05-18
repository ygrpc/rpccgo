//go:build cgo

package main

import (
	backend "example.com/rpccgo-full/internal/backend"
	greeterv1 "example.com/rpccgo-full/proto"
)

func init() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		panic(err)
	}
}
