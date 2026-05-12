package backend

import (
	"context"
	"fmt"

	rpcruntime "rpccgo/rpcruntime"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, name *rpcruntime.RpcString) (string, error) {
	value := name.SafeString()
	if value == "" {
		value = "world"
	}
	return fmt.Sprintf("hello, %s", value), nil
}
