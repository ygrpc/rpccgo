package main

import (
	"flag"
	"log"
	"net/http"

	"example.com/rpccgo-connect/internal/backend"
	greeterv1 "example.com/rpccgo-connect/proto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8081", "connect server listen address")
	flag.Parse()

	server := backend.Greeter{}
	if err := greeterv1.RegisterGreeterGoNativeServer(server); err != nil {
		log.Fatal(err)
	}

	go func() {
		path, handler := greeterv1.NewGreeterHandler(backend.ConnectGreeter{})
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		log.Fatal(http.ListenAndServe(*addr, h2c.NewHandler(mux, &http2.Server{})))
	}()

	select {}
}
