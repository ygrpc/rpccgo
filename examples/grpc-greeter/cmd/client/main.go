package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	conn, err := grpc.NewClient(envOrDefault("RPCCGO_GRPC_TARGET", "127.0.0.1:8080"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := greeterv1.NewGreeterClient(conn)
	response, err := client.SayHello(ctx, &greeterv1.SayHelloRequest{Name: "grpc-demo", City: "local"})
	if err != nil {
		return err
	}
	fmt.Println("grpc unary:", response.GetMessage())

	collect, err := client.Collect(ctx)
	if err != nil {
		return err
	}
	for _, name := range []string{"grpc", "stream"} {
		if err := collect.Send(&greeterv1.SayHelloRequest{Name: name, City: "local"}); err != nil {
			return err
		}
	}
	collectResp, err := collect.CloseAndRecv()
	if err != nil {
		return err
	}
	fmt.Println("grpc client-stream:", collectResp.GetMessage())

	broadcast, err := client.Broadcast(ctx, &greeterv1.SayHelloRequest{Name: "grpc-broadcast", City: "local"})
	if err != nil {
		return err
	}
	for {
		resp, err := broadcast.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fmt.Println("grpc server-stream:", resp.GetMessage())
	}

	chat, err := client.Chat(ctx)
	if err != nil {
		return err
	}
	for _, name := range []string{"grpc-chat-1", "grpc-chat-2"} {
		if err := chat.Send(&greeterv1.SayHelloRequest{Name: name, City: "local"}); err != nil {
			return err
		}
		chatResp, err := chat.Recv()
		if err != nil {
			return err
		}
		fmt.Println("grpc bidi:", chatResp.GetMessage())
	}
	if err := chat.CloseSend(); err != nil {
		return err
	}
	if _, err := chat.Recv(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("grpc bidi final recv succeeded, want EOF")
		}
		return err
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
