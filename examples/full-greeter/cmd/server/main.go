package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"example.com/rpccgo-full/internal/backend"
	greeterv1 "example.com/rpccgo-full/proto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func main() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}

	go func() {
		path, handler := greeterv1.NewGreeterConnectHandler()
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		log.Fatal(http.ListenAndServe(envOrDefault("RPCCGO_FULL_CONNECT_ADDR", "127.0.0.1:8081"), h2c.NewHandler(mux, &http2.Server{})))
	}()

	listener, err := net.Listen("tcp", envOrDefault("RPCCGO_FULL_GRPC_ADDR", "127.0.0.1:8082"))
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()
	if err := greeterv1.RegisterGreeterGRPCServer(server); err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.Serve(listener))
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
