package main

import (
	"log"
	"net/http"
	"os"

	"example.com/rpccgo-connect/internal/backend"
	greeterv1 "example.com/rpccgo-connect/proto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}

	go func() {
		path, handler := greeterv1.NewGreeterConnectHandler()
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		log.Fatal(http.ListenAndServe(envOrDefault("RPCCGO_CONNECT_CONNECT_ADDR", "127.0.0.1:8081"), h2c.NewHandler(mux, &http2.Server{})))
	}()

	select {}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
