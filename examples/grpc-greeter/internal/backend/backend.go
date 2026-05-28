package backend

import (
	"context"
	"fmt"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, name *rpcruntime.RpcString) (string, error) {
	value := name.SafeString()
	return format(value), nil
}

type GRPCGreeter struct {
	greeterv1.UnimplementedGreeterServer
}

func (GRPCGreeter) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: format(req.GetName())}, nil
}

func format(value string) string {
	if value == "" {
		value = "world"
	}
	return fmt.Sprintf("hello, %s", value)
}
