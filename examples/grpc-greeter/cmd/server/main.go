package main

import (
	"log"
	"net"
	"os"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"example.com/rpccgo-grpc/internal/backend"
	"google.golang.org/grpc"
)

func main() {
	listener, err := net.Listen("tcp", envOrDefault("RPCCGO_GRPC_ADDR", "127.0.0.1:8080"))
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()
	greeterv1.RegisterGreeterServer(server, backend.GRPCGreeter{})
	log.Fatal(server.Serve(listener))
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
