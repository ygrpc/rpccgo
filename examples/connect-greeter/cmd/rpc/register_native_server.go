//go:build cgo

package main

import (
	backend "example.com/rpccgo-connect/internal/backend"
	greeterv1 "example.com/rpccgo-connect/proto"
)

func init() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		panic(err)
	}
}
