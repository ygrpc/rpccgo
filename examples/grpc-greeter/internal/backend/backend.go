package backend

import (
	"context"
	"fmt"
	"io"
	"strings"

	greeterv1 "example.com/rpccgo-grpc/gen/greeter/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type Greeter struct {
	greeterv1.UnimplementedGreeterNativeServer
}

func (Greeter) SayHello(_ context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	return format(name.SafeString(), city.SafeString()), nil
}

func (Greeter) Collect(ctx context.Context, stream greeterv1.GreeterCollectNativeClientStream) (string, error) {
	var names []string
	for {
		name, _, err := stream.Recv(ctx)
		if err == io.EOF {
			return "collect:" + strings.Join(names, ","), nil
		}
		if err != nil {
			return "", err
		}
		names = append(names, name.SafeString())
	}
}

func (Greeter) Broadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString, stream greeterv1.GreeterBroadcastNativeServerStream) error {
	for index := 0; index < 2; index++ {
		if err := stream.Send(ctx, fmt.Sprintf("broadcast[%d]:%s", index, name.SafeString())); err != nil {
			return err
		}
	}
	return nil
}

func (Greeter) Chat(ctx context.Context, stream greeterv1.GreeterChatNativeBidiStream) error {
	for {
		name, _, err := stream.Recv(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(ctx, "chat:"+name.SafeString()); err != nil {
			return err
		}
	}
}

type GRPCGreeter struct {
	greeterv1.UnimplementedGreeterServer
}

func (GRPCGreeter) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: format(req.GetName(), req.GetCity())}, nil
}

func (GRPCGreeter) Collect(stream greeterv1.Greeter_CollectServer) error {
	var names []string
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&greeterv1.SayHelloResponse{Message: "collect:" + strings.Join(names, ",")})
		}
		if err != nil {
			return err
		}
		names = append(names, req.GetName())
	}
}

func (GRPCGreeter) Broadcast(req *greeterv1.SayHelloRequest, stream greeterv1.Greeter_BroadcastServer) error {
	for index := 0; index < 2; index++ {
		if err := stream.Send(&greeterv1.SayHelloResponse{Message: fmt.Sprintf("broadcast[%d]:%s", index, req.GetName())}); err != nil {
			return err
		}
	}
	return nil
}

func (GRPCGreeter) Chat(stream greeterv1.Greeter_ChatServer) error {
	for {
		req, err := stream.Recv()
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
