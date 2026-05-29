package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	connect "connectrpc.com/connect"
	greeterv1 "example.com/rpccgo-connect/proto"
	"golang.org/x/net/http2"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	connectBaseURL := strings.TrimRight(envOrDefault("RPCCGO_CONNECT_CONNECT_URL", "http://127.0.0.1:8081"), "/")
	return runConnectDemo(ctx, connectBaseURL)
}

func runConnectDemo(ctx context.Context, baseURL string) error {
	client := connect.NewClient[greeterv1.SayHelloRequest, greeterv1.SayHelloResponse](
		h2cClient(),
		baseURL+greeterv1.GreeterSayHelloConnectProcedure,
	)
	response, err := client.CallUnary(ctx, connect.NewRequest(&greeterv1.SayHelloRequest{Name: "connect-demo", City: "local"}))
	if err != nil {
		return err
	}
	fmt.Println("connect unary:", response.Msg.GetMessage())

	collect := connect.NewClient[greeterv1.SayHelloRequest, greeterv1.SayHelloResponse](
		h2cClient(),
		baseURL+greeterv1.GreeterCollectConnectProcedure,
	).CallClientStream(ctx)
	for _, name := range []string{"connect", "stream"} {
		if err := collect.Send(&greeterv1.SayHelloRequest{Name: name}); err != nil {
			return err
		}
	}
	collectResp, err := collect.CloseAndReceive()
	if err != nil {
		return err
	}
	fmt.Println("connect client-stream:", collectResp.Msg.GetMessage())
	return nil
}

func h2cClient() connect.HTTPClient {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
