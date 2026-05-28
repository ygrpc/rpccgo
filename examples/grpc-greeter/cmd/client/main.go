package main

import (
	"context"
	"fmt"
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
	response, err := client.SayHello(ctx, &greeterv1.SayHelloRequest{Name: "grpc-demo"})
	if err != nil {
		return err
	}
	fmt.Println("grpc:", response.GetMessage())
	return nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
