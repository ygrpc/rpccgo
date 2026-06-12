package rpcruntime

import (
	"context"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc/metadata"
)

var (
	errGRPCStreamMessageType = errors.New("grpc stream message type mismatch")
	errGRPCStreamNilResponse = errors.New("grpc stream response is nil")
	errGRPCStreamNoResponse  = errors.New("grpc client stream completed without SendAndClose")
)

type grpcServerStream struct {
	ctx     context.Context
	mu      sync.Mutex
	header  metadata.MD
	trailer metadata.MD
}

func (s *grpcServerStream) SetHeader(md metadata.MD) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.header = joinMetadata(s.header, md)
	return nil
}

func (s *grpcServerStream) SendHeader(md metadata.MD) error {
	return s.SetHeader(md)
}

func (s *grpcServerStream) SetTrailer(md metadata.MD) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trailer = joinMetadata(s.trailer, md)
}

func (s *grpcServerStream) Context() context.Context { return s.ctx }

func joinMetadata(current, next metadata.MD) metadata.MD {
	if next == nil {
		return current
	}
	if current == nil {
		return next.Copy()
	}
	return metadata.Join(current, next)
}

// GRPCClientStreamingServer adapts a local client-streaming server endpoint to grpc-go's typed server stream.
type GRPCClientStreamingServer[Req, Resp any] struct {
	grpcServerStream
	stream *ClientStreamForServer[*Req, *Resp]
}

// NewGRPCClientStreamingServer constructs a grpc-go client-streaming server adapter.
func NewGRPCClientStreamingServer[Req, Resp any](ctx context.Context, stream *ClientStreamForServer[*Req, *Resp]) *GRPCClientStreamingServer[Req, Resp] {
	return &GRPCClientStreamingServer[Req, Resp]{grpcServerStream: grpcServerStream{ctx: ctx}, stream: stream}
}

// Recv receives the next request from the local client endpoint.
func (s *GRPCClientStreamingServer[Req, Resp]) Recv() (*Req, error) {
	return s.stream.Recv(s.ctx)
}

// SendAndClose completes the local stream with its unary response.
func (s *GRPCClientStreamingServer[Req, Resp]) SendAndClose(resp *Resp) error {
	if resp == nil {
		return errGRPCStreamNilResponse
	}
	s.stream.Complete(resp, nil)
	return nil
}

// Complete records the grpc handler result when it did not already complete through SendAndClose.
func (s *GRPCClientStreamingServer[Req, Resp]) Complete(err error) {
	if err == nil {
		err = errGRPCStreamNoResponse
	}
	s.stream.Complete(nil, err)
}

// SendMsg implements the untyped grpc.ServerStream response operation.
func (s *GRPCClientStreamingServer[Req, Resp]) SendMsg(message any) error {
	resp, ok := message.(*Resp)
	if !ok {
		return errGRPCStreamMessageType
	}
	return s.SendAndClose(resp)
}

// RecvMsg implements the untyped grpc.ServerStream request operation.
func (s *GRPCClientStreamingServer[Req, Resp]) RecvMsg(message any) error {
	req, ok := message.(*Req)
	if !ok || req == nil {
		return errGRPCStreamMessageType
	}
	next, err := s.Recv()
	if err != nil {
		return err
	}
	*req = *next
	return nil
}

// GRPCServerStreamingServer adapts a local server-streaming server endpoint to grpc-go's typed server stream.
type GRPCServerStreamingServer[Resp any] struct {
	grpcServerStream
	stream *ServerStreamForServer[*Resp]
}

// NewGRPCServerStreamingServer constructs a grpc-go server-streaming server adapter.
func NewGRPCServerStreamingServer[Resp any](ctx context.Context, stream *ServerStreamForServer[*Resp]) *GRPCServerStreamingServer[Resp] {
	return &GRPCServerStreamingServer[Resp]{grpcServerStream: grpcServerStream{ctx: ctx}, stream: stream}
}

// Send sends a response to the local client endpoint.
func (s *GRPCServerStreamingServer[Resp]) Send(resp *Resp) error {
	if resp == nil {
		return errGRPCStreamNilResponse
	}
	return s.stream.Send(s.ctx, resp)
}

// SendMsg implements the untyped grpc.ServerStream response operation.
func (s *GRPCServerStreamingServer[Resp]) SendMsg(message any) error {
	resp, ok := message.(*Resp)
	if !ok {
		return errGRPCStreamMessageType
	}
	return s.Send(resp)
}

// RecvMsg reports EOF because server-streaming RPCs have no streaming request side.
func (s *GRPCServerStreamingServer[Resp]) RecvMsg(any) error { return io.EOF }

// GRPCBidiStreamingServer adapts a local bidi server endpoint to grpc-go's typed server stream.
type GRPCBidiStreamingServer[Req, Resp any] struct {
	grpcServerStream
	stream *BidiStreamForServer[*Req, *Resp]
}

// NewGRPCBidiStreamingServer constructs a grpc-go bidirectional-streaming server adapter.
func NewGRPCBidiStreamingServer[Req, Resp any](ctx context.Context, stream *BidiStreamForServer[*Req, *Resp]) *GRPCBidiStreamingServer[Req, Resp] {
	return &GRPCBidiStreamingServer[Req, Resp]{grpcServerStream: grpcServerStream{ctx: ctx}, stream: stream}
}

// Recv receives the next request from the local client endpoint.
func (s *GRPCBidiStreamingServer[Req, Resp]) Recv() (*Req, error) {
	return s.stream.Recv(s.ctx)
}

// Send sends a response to the local client endpoint.
func (s *GRPCBidiStreamingServer[Req, Resp]) Send(resp *Resp) error {
	if resp == nil {
		return errGRPCStreamNilResponse
	}
	return s.stream.Send(s.ctx, resp)
}

// SendMsg implements the untyped grpc.ServerStream response operation.
func (s *GRPCBidiStreamingServer[Req, Resp]) SendMsg(message any) error {
	resp, ok := message.(*Resp)
	if !ok {
		return errGRPCStreamMessageType
	}
	return s.Send(resp)
}

// RecvMsg implements the untyped grpc.ServerStream request operation.
func (s *GRPCBidiStreamingServer[Req, Resp]) RecvMsg(message any) error {
	req, ok := message.(*Req)
	if !ok || req == nil {
		return errGRPCStreamMessageType
	}
	next, err := s.Recv()
	if err != nil {
		return err
	}
	*req = *next
	return nil
}
