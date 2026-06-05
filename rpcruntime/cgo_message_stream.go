package rpcruntime

import "context"

type CGOMessageClientStream[Req any] interface {
	Recv(context.Context) (Req, error)
}

type CGOMessageServerStream[Resp any] interface {
	Send(context.Context, Resp) error
}

type CGOMessageBidiStream[Req, Resp any] interface {
	Recv(context.Context) (Req, error)
	Send(context.Context, Resp) error
}

type CGOMessageClientStreamSession[Req, Resp any] interface {
	Send(context.Context, Req) error
	Finish(context.Context) (Resp, error)
	Cancel(context.Context) error
}

type CGOMessageServerStreamSession[Resp any] interface {
	Recv(context.Context) (Resp, error)
	Finish(context.Context) error
	Cancel(context.Context) error
}

type CGOMessageBidiStreamSession[Req, Resp any] interface {
	Send(context.Context, Req) error
	Recv(context.Context) (Resp, error)
	CloseSend(context.Context) error
	Finish(context.Context) error
	Cancel(context.Context) error
}
