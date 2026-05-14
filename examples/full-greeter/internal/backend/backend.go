package backend

import (
	"context"
	"fmt"
	"io"
	"strings"

	greeterv1 "example.com/rpccgo-full/proto"
	rpcruntime "rpccgo/rpcruntime"
)

type Greeter struct{}

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

func format(name, city string) string {
	if name == "" {
		name = "world"
	}
	if city == "" {
		city = "somewhere"
	}
	return fmt.Sprintf("hello %s from %s", name, city)
}
