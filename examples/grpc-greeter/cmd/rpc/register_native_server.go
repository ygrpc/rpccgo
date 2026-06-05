//go:build cgo

package main

import (
	"log"
	"os"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"example.com/rpccgo-grpc/internal/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var remoteConn *grpc.ClientConn

func init() {
	if target := os.Getenv("RPCCGO_GRPC_TARGET"); target != "" {
		conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatal(err)
		}
		remoteConn = conn
		if err := greeterv1.RegisterGreeterGRPCRemoteServer(greeterv1.NewGreeterClient(conn)); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}
}
