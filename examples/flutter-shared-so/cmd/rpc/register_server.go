//go:build cgo

package main

import (
	"log"

	backend "example.com/rpccgo-flutter-shared-so/internal/backend"
	fluttersharedv1 "example.com/rpccgo-flutter-shared-so/proto"
)

func init() {
	if err := fluttersharedv1.RegisterSharedSoDemoConnectHandler(backend.NewSharedSoDemoServer()); err != nil {
		log.Fatal(err)
	}
}
