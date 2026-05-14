package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo native generated file for Greeter go native server

var (
	greeterNativeRequestBridgeNotImplemented = errors.New("rpccgo: native request bridge is not implemented")
	greeterNativeStreamBridgeNotImplemented  = errors.New("rpccgo: native stream bridge is not implemented")
	greeterNativeStreamIsNil                 = errors.New("rpccgo: native stream is nil")
	greeterNativeStreamClosed                = errors.New("rpccgo: native stream is closed")
)

type GreeterNativeServer interface {
	SayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error)
}

type greeterGoNativeAdapter struct {
	server GreeterNativeServer
}

func (a *greeterGoNativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error) {
	return a.server.SayHello(ctx, name)
}

func RegisterGreeterGoNativeServer(server GreeterNativeServer) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	if server == nil {
		return rpcruntime.AdapterSnapshot[GreeterNativeAdapter]{}, errors.New("rpccgo: Greeter go native server is nil")
	}
	return registerGreeterActiveServer(rpcruntime.ServerKindGoNative, &greeterGoNativeAdapter{server: server})
}
