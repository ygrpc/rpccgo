package rpcruntime

import (
	"net/http"
	"unsafe"

	"connectrpc.com/connect"
)

type ConnectStreamingHandlerConn struct {
	SpecValue            connect.Spec
	PeerValue            connect.Peer
	RequestHeaderValue   http.Header
	ResponseHeaderValue  http.Header
	ResponseTrailerValue http.Header
	ReceiveFunc          func(any) error
	SendFunc             func(any) error
}

func (c *ConnectStreamingHandlerConn) Spec() connect.Spec { return c.SpecValue }

func (c *ConnectStreamingHandlerConn) Peer() connect.Peer { return c.PeerValue }

func (c *ConnectStreamingHandlerConn) Receive(msg any) error {
	if c.ReceiveFunc == nil {
		return ErrStreamInvalidHandle
	}
	return c.ReceiveFunc(msg)
}

func (c *ConnectStreamingHandlerConn) RequestHeader() http.Header {
	if c.RequestHeaderValue == nil {
		c.RequestHeaderValue = http.Header{}
	}
	return c.RequestHeaderValue
}

func (c *ConnectStreamingHandlerConn) Send(msg any) error {
	if c.SendFunc == nil {
		return ErrStreamInvalidHandle
	}
	return c.SendFunc(msg)
}

func (c *ConnectStreamingHandlerConn) ResponseHeader() http.Header {
	if c.ResponseHeaderValue == nil {
		c.ResponseHeaderValue = http.Header{}
	}
	return c.ResponseHeaderValue
}

func (c *ConnectStreamingHandlerConn) ResponseTrailer() http.Header {
	if c.ResponseTrailerValue == nil {
		c.ResponseTrailerValue = http.Header{}
	}
	return c.ResponseTrailerValue
}

type connectMaybeInitializerLayout struct {
	initializer func(connect.Spec, any) error
}

type connectClientStreamLayout[Req any] struct {
	conn        connect.StreamingHandlerConn
	initializer connectMaybeInitializerLayout
	msg         *Req
	err         error
}

type connectServerStreamLayout[Res any] struct {
	conn connect.StreamingHandlerConn
}

type connectBidiStreamLayout[Req, Res any] struct {
	conn        connect.StreamingHandlerConn
	initializer connectMaybeInitializerLayout
}

func NewConnectClientStream[Req any](conn connect.StreamingHandlerConn) *connect.ClientStream[Req] {
	stream := &connectClientStreamLayout[Req]{conn: conn}
	return (*connect.ClientStream[Req])(unsafe.Pointer(stream))
}

func NewConnectServerStream[Res any](conn connect.StreamingHandlerConn) *connect.ServerStream[Res] {
	stream := &connectServerStreamLayout[Res]{conn: conn}
	return (*connect.ServerStream[Res])(unsafe.Pointer(stream))
}

func NewConnectBidiStream[Req, Res any](conn connect.StreamingHandlerConn) *connect.BidiStream[Req, Res] {
	stream := &connectBidiStreamLayout[Req, Res]{conn: conn}
	return (*connect.BidiStream[Req, Res])(unsafe.Pointer(stream))
}
