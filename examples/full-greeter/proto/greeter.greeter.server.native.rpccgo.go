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
)

type GreeterNativeServer interface {
	SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error)
	Collect(ctx context.Context) (GreeterCollectNativeClientStream, error)
	Broadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (GreeterBroadcastNativeServerStream, error)
	Chat(ctx context.Context) (GreeterChatNativeBidiStream, error)
}

type GreeterCollectNativeClientStream interface {
	Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error
	Finish(ctx context.Context) (string, error)
	Cancel(ctx context.Context) error
}

type GreeterBroadcastNativeServerStream interface {
	Recv(ctx context.Context) (string, error)
	Cancel(ctx context.Context) error
}

type GreeterChatNativeBidiStream interface {
	Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error
	Recv(ctx context.Context) (string, error)
	CloseSend(ctx context.Context) error
	Cancel(ctx context.Context) error
}

type greeterGoNativeAdapter struct {
	server GreeterNativeServer
}

func (a *greeterGoNativeAdapter) SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	return a.server.SayHello(ctx, name, city)
}

func (a *greeterGoNativeAdapter) StartCollect(ctx context.Context) (GreeterCollectNativeStreamSession, error) {
	stream, err := a.server.Collect(ctx)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, greeterNativeStreamIsNil
	}
	return &greeterCollectGoNativeClientStreamSession{stream: stream}, nil
}

type greeterCollectGoNativeClientStreamSession struct {
	stream GreeterCollectNativeClientStream
}

func (s *greeterCollectGoNativeClientStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	if s.stream == nil {
		return greeterNativeStreamIsNil
	}
	return s.stream.Send(ctx, name, city)
}

func (s *greeterCollectGoNativeClientStreamSession) Finish(ctx context.Context) (string, error) {
	if s.stream == nil {
		return "", greeterNativeStreamIsNil
	}
	return s.stream.Finish(ctx)
}

func (s *greeterCollectGoNativeClientStreamSession) Cancel(ctx context.Context) error {
	if s.stream == nil {
		return greeterNativeStreamIsNil
	}
	return s.stream.Cancel(ctx)
}

func (a *greeterGoNativeAdapter) StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (GreeterBroadcastNativeStreamSession, error) {
	stream, err := a.server.Broadcast(ctx, name, city)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, greeterNativeStreamIsNil
	}
	return &greeterBroadcastGoNativeServerStreamSession{stream: stream}, nil
}

type greeterBroadcastGoNativeServerStreamSession struct {
	stream GreeterBroadcastNativeServerStream
}

func (s *greeterBroadcastGoNativeServerStreamSession) Recv(ctx context.Context) (string, error) {
	if s.stream == nil {
		return "", greeterNativeStreamIsNil
	}
	return s.stream.Recv(ctx)
}

func (s *greeterBroadcastGoNativeServerStreamSession) Cancel(ctx context.Context) error {
	if s.stream == nil {
		return greeterNativeStreamIsNil
	}
	return s.stream.Cancel(ctx)
}

func (a *greeterGoNativeAdapter) StartChat(ctx context.Context) (GreeterChatNativeStreamSession, error) {
	stream, err := a.server.Chat(ctx)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, greeterNativeStreamIsNil
	}
	return &greeterChatGoNativeBidiStreamSession{stream: stream}, nil
}

type greeterChatGoNativeBidiStreamSession struct {
	stream GreeterChatNativeBidiStream
}

func (s *greeterChatGoNativeBidiStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	if s.stream == nil {
		return greeterNativeStreamIsNil
	}
	return s.stream.Send(ctx, name, city)
}

func (s *greeterChatGoNativeBidiStreamSession) Recv(ctx context.Context) (string, error) {
	if s.stream == nil {
		return "", greeterNativeStreamIsNil
	}
	return s.stream.Recv(ctx)
}

func (s *greeterChatGoNativeBidiStreamSession) CloseSend(ctx context.Context) error {
	if s.stream == nil {
		return greeterNativeStreamIsNil
	}
	return s.stream.CloseSend(ctx)
}

func (s *greeterChatGoNativeBidiStreamSession) Cancel(ctx context.Context) error {
	if s.stream == nil {
		return greeterNativeStreamIsNil
	}
	return s.stream.Cancel(ctx)
}

func RegisterGreeterGoNativeServer(server GreeterNativeServer) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	if server == nil {
		return rpcruntime.AdapterSnapshot[GreeterNativeAdapter]{}, errors.New("rpccgo: Greeter go native server is nil")
	}
	return registerGreeterActiveServer(rpcruntime.ServerKindGoNative, &greeterGoNativeAdapter{server: server})
}
