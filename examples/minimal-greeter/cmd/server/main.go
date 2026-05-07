package main

import (
	"log"
	"net/http"

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
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}
