package backend

import (
	"context"
	"fmt"

	greeterv1 "example.com/rpccgo-minimal/gen/greeter/v1"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	name := req.GetName()
	if name == "" {
		name = "world"
	}
	return &greeterv1.SayHelloResponse{Message: fmt.Sprintf("hello, %s", name)}, nil
}
