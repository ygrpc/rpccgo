package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	connect "connectrpc.com/connect"
	greeterv1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	baseURL := strings.TrimRight(envOrDefault("RPCCGO_MINIMAL_CONNECT_URL", "http://127.0.0.1:8080"), "/")
	client := connect.NewClient[greeterv1.SayHelloRequest, greeterv1.SayHelloResponse](
		httpClient(),
		baseURL+greeterv1.GreeterSayHelloConnectProcedure,
	)
	response, err := client.CallUnary(ctx, connect.NewRequest(&greeterv1.SayHelloRequest{Name: "minimal-demo"}))
	if err != nil {
		return err
	}
	fmt.Println("connect:", response.Msg.GetMessage())
	return nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func httpClient() connect.HTTPClient {
	return http.DefaultClient
}
