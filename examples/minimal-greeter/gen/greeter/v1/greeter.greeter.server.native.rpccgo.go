package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo native stage file for Greeter go native server

var (
	greeterNativeRequestBridgeNotImplemented = errors.New("rpccgo: native request bridge is not implemented")
	greeterNativeStreamBridgeNotImplemented  = errors.New("rpccgo: native stream bridge is not implemented")
	greeterNativeStreamIsNil                 = errors.New("rpccgo: native stream is nil")
)

type GreeterNativeServer interface {
	SayHello(ctx context.Context, req *SayHelloRequest) (*SayHelloResponse, error)
}

type greeterGoNativeAdapter struct {
	server GreeterNativeServer
}

func (a *greeterGoNativeAdapter) SayHello(ctx context.Context, req *SayHelloRequest) (*SayHelloResponse, error) {
	if req == nil {
		return nil, greeterNativeRequestBridgeNotImplemented
	}
	return a.server.SayHello(ctx, req)
}

func RegisterGreeterGoNativeServer(server GreeterNativeServer) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	if server == nil {
		return rpcruntime.AdapterSnapshot[GreeterNativeAdapter]{}, errors.New("rpccgo: Greeter go native server is nil")
	}
	return registerGreeterActiveServer(rpcruntime.ServerKindGoNative, &greeterGoNativeAdapter{server: server})
}
