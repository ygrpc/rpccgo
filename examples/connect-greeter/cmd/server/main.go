package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	connect "connectrpc.com/connect"
	"example.com/rpccgo-connect/internal/backend"
	greeterv1 "example.com/rpccgo-connect/proto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	server := backend.Greeter{}
	if err := greeterv1.RegisterGreeterGoNativeServer(server); err != nil {
		log.Fatal(err)
	}

	go func() {
		path, handler := greeterv1.NewGreeterHandler(connectGreeter{server: server})
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		log.Fatal(http.ListenAndServe(envOrDefault("RPCCGO_CONNECT_CONNECT_ADDR", "127.0.0.1:8081"), h2c.NewHandler(mux, &http2.Server{})))
	}()

	select {}
}

type connectGreeter struct {
	server backend.Greeter
}

func (g connectGreeter) SayHello(ctx context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: format(req.GetName(), req.GetCity())}, nil
}

func (g connectGreeter) Collect(ctx context.Context, stream *connect.ClientStream[greeterv1.SayHelloRequest]) (*greeterv1.SayHelloResponse, error) {
	var names []string
	for stream.Receive() {
		names = append(names, stream.Msg().GetName())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &greeterv1.SayHelloResponse{Message: "collect:" + strings.Join(names, ",")}, nil
}

func (g connectGreeter) Broadcast(ctx context.Context, req *greeterv1.SayHelloRequest, stream *connect.ServerStream[greeterv1.SayHelloResponse]) error {
	for index := 0; index < 2; index++ {
		if err := stream.Send(&greeterv1.SayHelloResponse{Message: fmt.Sprintf("broadcast[%d]:%s", index, req.GetName())}); err != nil {
			return err
		}
	}
	return nil
}

func (g connectGreeter) Chat(ctx context.Context, stream *connect.BidiStream[greeterv1.SayHelloRequest, greeterv1.SayHelloResponse]) error {
	for {
		req, err := stream.Receive()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&greeterv1.SayHelloResponse{Message: "chat:" + req.GetName()}); err != nil {
			return err
		}
	}
}

func format(name, city string) string {
	if name == "" {
		name = "world"
	}
	if city == "" {
		city = "somewhere"
	}
	return fmt.Sprintf("hello %s from %s", name, city)
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
