//go:build cgo

package main

import (
	"log"
	"os"
	"strings"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"example.com/rpccgo-grpc/internal/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var remoteConn *grpc.ClientConn

func init() {
	if target := argValue("--grpc-target"); target != "" {
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
	if argValue("--server") == "grpc_server" {
		if err := greeterv1.RegisterGreeterGRPCServer(backend.GRPCGreeter{}); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}
}

func argValue(name string) string {
	prefix := name + "="
	for index, arg := range os.Args[1:] {
		if arg == name && index+2 <= len(os.Args[1:]) {
			return os.Args[index+2]
		}
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}
