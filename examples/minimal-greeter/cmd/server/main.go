package main

import (
	"log"
	"net/http"
	"os"

	greeterv1 "example.com/rpccgo-minimal/gen/greeter/v1"
	"example.com/rpccgo-minimal/internal/backend"
)

func main() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}

	path, handler := greeterv1.NewGreeterConnectHandler()
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	log.Fatal(http.ListenAndServe(envOrDefault("RPCCGO_MINIMAL_CONNECT_ADDR", "127.0.0.1:8080"), mux))
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
