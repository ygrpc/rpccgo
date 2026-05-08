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
	greeterv1 "example.com/rpccgo-full/proto"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	connectBaseURL := strings.TrimRight(envOrDefault("RPCCGO_FULL_CONNECT_URL", "http://127.0.0.1:8081"), "/")
	grpcAddr := envOrDefault("RPCCGO_FULL_GRPC_ADDR", "127.0.0.1:8082")

	if err := runConnectDemo(ctx, connectBaseURL); err != nil {
		return err
	}
	if err := runGRPCDemo(ctx, grpcAddr); err != nil {
		return err
	}
	return nil
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

func runGRPCDemo(ctx context.Context, addr string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	unaryResp := new(greeterv1.SayHelloResponse)
	if err := conn.Invoke(ctx, greeterv1.GreeterSayHelloGRPCFullMethodName, &greeterv1.SayHelloRequest{Name: "grpc-demo", City: "local"}, unaryResp); err != nil {
		return err
	}
	fmt.Println("grpc unary:", unaryResp.GetMessage())

	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, greeterv1.GreeterBroadcastGRPCFullMethodName)
	if err != nil {
		return err
	}
	client := &grpc.GenericClientStream[greeterv1.SayHelloRequest, greeterv1.SayHelloResponse]{ClientStream: stream}
	if err := client.Send(&greeterv1.SayHelloRequest{Name: "grpc-broadcast"}); err != nil {
		return err
	}
	if err := client.CloseSend(); err != nil {
		return err
	}
	for {
		resp, err := client.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		fmt.Println("grpc server-stream:", resp.GetMessage())
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
