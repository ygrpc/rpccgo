package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	connectBaseURL := strings.TrimRight(envOrDefault("RPCCGO_CONNECT_URL", "http://127.0.0.1:8081"), "/")
	return runConnectDemo(ctx, connectBaseURL)
}

func runConnectDemo(ctx context.Context, baseURL string) error {
	client := greeterv1.NewGreeterClient(h2cClient(), baseURL)

	response, err := client.SayHello(ctx, &greeterv1.SayHelloRequest{Name: "connect-demo", City: "local"})
	if err != nil {
		return err
	}
	fmt.Println("connect unary:", response.GetMessage())

	collect, err := client.Collect(ctx)
	if err != nil {
		return err
	}
	for _, name := range []string{"connect", "stream"} {
		if err := collect.Send(&greeterv1.SayHelloRequest{Name: name}); err != nil {
			return err
		}
	}
	collectResp, err := collect.CloseAndReceive()
	if err != nil {
		return err
	}
	fmt.Println("connect client-stream:", collectResp.GetMessage())

	broadcast, err := client.Broadcast(ctx, &greeterv1.SayHelloRequest{Name: "connect-broadcast", City: "local"})
	if err != nil {
		return err
	}
	for broadcast.Receive() {
		fmt.Println("connect server-stream:", broadcast.Msg().GetMessage())
	}
	if err := broadcast.Err(); err != nil {
		return err
	}

	chat, err := client.Chat(ctx)
	if err != nil {
		return err
	}
	for _, name := range []string{"connect-chat-1", "connect-chat-2"} {
		if err := chat.Send(&greeterv1.SayHelloRequest{Name: name, City: "local"}); err != nil {
			return err
		}
		chatResp, err := chat.Receive()
		if err != nil {
			return err
		}
		fmt.Println("connect bidi:", chatResp.GetMessage())
	}
	if err := chat.CloseRequest(); err != nil {
		return err
	}
	if _, err := chat.Receive(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("connect bidi final receive succeeded, want EOF")
		}
		return err
	}
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
