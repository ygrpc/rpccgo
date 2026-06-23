//go:build cgo

package main

import (
	"log"

	"example.com/rpccgo-android-foreground-service-so/internal/backend"
	foregroundservicev1 "example.com/rpccgo-android-foreground-service-so/proto"
)

func init() {
	if err := foregroundservicev1.RegisterForegroundServiceDemoConnectHandler(backend.NewForegroundServiceDemoServer()); err != nil {
		log.Fatal(err)
	}
}
