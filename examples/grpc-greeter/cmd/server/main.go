package main

import (
	"flag"
	"log"
	"net"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"example.com/rpccgo-grpc/internal/backend"
	"google.golang.org/grpc"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "grpc server listen address")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()
	greeterv1.RegisterGreeterServer(server, backend.GRPCGreeter{})
	log.Fatal(server.Serve(listener))
}
