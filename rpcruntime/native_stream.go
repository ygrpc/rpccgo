package rpcruntime

import "context"

// ClientStreamingClient is the client-side operation surface for a client-streaming RPC.
type ClientStreamingClient[Req, Resp any] interface {
	Send(context.Context, Req) error
	Finish(context.Context) (Resp, error)
	Cancel(context.Context) error
}

// ClientStreamingServer is the server-side operation surface for a client-streaming RPC.
type ClientStreamingServer[Req any] interface {
	Recv(context.Context) (Req, error)
}

// ServerStreamingClient is the client-side operation surface for a server-streaming RPC.
type ServerStreamingClient[Resp any] interface {
	Recv(context.Context) (Resp, error)
	Finish(context.Context) error
	Cancel(context.Context) error
}

// ServerStreamingServer is the server-side operation surface for a server-streaming RPC.
type ServerStreamingServer[Resp any] interface {
	FinishRequested() <-chan struct{}
	Send(context.Context, Resp) error
}

// BidiStreamingClient is the client-side operation surface for a bidirectional-streaming RPC.
type BidiStreamingClient[Req, Resp any] interface {
	Send(context.Context, Req) error
	Recv(context.Context) (Resp, error)
	CloseSend(context.Context) error
	Finish(context.Context) error
	Cancel(context.Context) error
}

// BidiStreamingServer is the server-side operation surface for a bidirectional-streaming RPC.
type BidiStreamingServer[Req, Resp any] interface {
	FinishRequested() <-chan struct{}
	Recv(context.Context) (Req, error)
	Send(context.Context, Resp) error
}
