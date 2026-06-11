package rpcruntime

import (
	"context"
	"io"
	"reflect"
	"sync"
)

// LocalStreamOptions configures an in-process streaming call.
type LocalStreamOptions struct {
	RequestBuffer  int
	ResponseBuffer int
	StreamClosed   error
	NilRequest     error
	NilResponse    error
}

type streamItem[T any] struct {
	value    T
	received chan struct{}
}

func isNilStreamValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

type clientStreamingState[Req, Resp any] struct {
	ctx           context.Context
	cancel        context.CancelFunc
	requests      chan streamItem[Req]
	sendDone      chan struct{}
	closeSendOnce sync.Once
	done          chan struct{}
	completeOnce  sync.Once
	resp          Resp
	err           error
	streamClosed  error
	nilRequest    error
}

// ClientStreamForClient is the client-side endpoint of an in-process client-streaming RPC.
type ClientStreamForClient[Req, Resp any] struct {
	state *clientStreamingState[Req, Resp]
}

// ClientStreamForServer is the server-side endpoint of an in-process client-streaming RPC.
type ClientStreamForServer[Req, Resp any] struct {
	state *clientStreamingState[Req, Resp]
}

// NewClientStreaming constructs the client and server endpoints of an in-process client-streaming RPC.
func NewClientStreaming[Req, Resp any](ctx context.Context, options LocalStreamOptions) (*ClientStreamForClient[Req, Resp], *ClientStreamForServer[Req, Resp], context.Context) {
	streamCtx, cancel := context.WithCancel(ctx)
	state := &clientStreamingState[Req, Resp]{
		ctx:          streamCtx,
		cancel:       cancel,
		requests:     make(chan streamItem[Req], options.RequestBuffer),
		sendDone:     make(chan struct{}),
		done:         make(chan struct{}),
		streamClosed: options.StreamClosed,
		nilRequest:   options.NilRequest,
	}
	return &ClientStreamForClient[Req, Resp]{state: state}, &ClientStreamForServer[Req, Resp]{state: state}, streamCtx
}

// Send sends a request to the server endpoint.
func (c *ClientStreamForClient[Req, Resp]) Send(ctx context.Context, req Req) error {
	s := c.state
	if s.nilRequest != nil && isNilStreamValue(req) {
		return s.nilRequest
	}
	select {
	case <-s.sendDone:
		return s.streamClosed
	default:
	}
	item := streamItem[Req]{value: req, received: make(chan struct{})}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.done:
		if s.err != nil {
			return s.err
		}
		return s.streamClosed
	case <-s.sendDone:
		return s.streamClosed
	case s.requests <- item:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-s.done:
			select {
			case <-item.received:
				return nil
			default:
			}
			if s.err != nil {
				return s.err
			}
			return s.streamClosed
		case <-s.sendDone:
			return s.streamClosed
		case <-item.received:
			return nil
		}
	}
}

// Finish closes the request side and waits for the unary response.
func (c *ClientStreamForClient[Req, Resp]) Finish(ctx context.Context) (Resp, error) {
	s := c.state
	s.closeSendOnce.Do(func() { close(s.sendDone) })
	select {
	case <-ctx.Done():
		s.cancel()
		var zero Resp
		return zero, ctx.Err()
	case <-s.ctx.Done():
		var zero Resp
		return zero, s.ctx.Err()
	case <-s.done:
		s.cancel()
		return s.resp, s.err
	}
}

// Cancel aborts the RPC and waits for the server endpoint to complete.
func (c *ClientStreamForClient[Req, Resp]) Cancel(ctx context.Context) error {
	s := c.state
	s.cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return nil
	}
}

// Recv receives the next request from the client endpoint.
func (s *ClientStreamForServer[Req, Resp]) Recv(ctx context.Context) (Req, error) {
	select {
	case req := <-s.state.requests:
		close(req.received)
		return req.value, nil
	default:
	}
	var zero Req
	select {
	case <-ctx.Done():
		select {
		case <-s.state.sendDone:
			return zero, io.EOF
		default:
		}
		return zero, ctx.Err()
	case <-s.state.ctx.Done():
		select {
		case <-s.state.sendDone:
			return zero, io.EOF
		default:
		}
		return zero, s.state.ctx.Err()
	case req := <-s.state.requests:
		close(req.received)
		return req.value, nil
	case <-s.state.sendDone:
		return zero, io.EOF
	}
}

// Complete records the handler result and completes the RPC.
func (s *ClientStreamForServer[Req, Resp]) Complete(resp Resp, err error) {
	s.state.completeOnce.Do(func() {
		s.state.resp = resp
		s.state.err = err
		close(s.state.done)
	})
}

type serverStreamingState[Resp any] struct {
	ctx          context.Context
	cancel       context.CancelFunc
	finishCtx    context.Context
	finishCancel context.CancelFunc
	responses    chan streamItem[Resp]
	done         chan struct{}
	completeOnce sync.Once
	err          error
	streamClosed error
	nilResponse  error
}

// ServerStreamForClient is the client-side endpoint of an in-process server-streaming RPC.
type ServerStreamForClient[Resp any] struct {
	state *serverStreamingState[Resp]
}

// ServerStreamForServer is the server-side endpoint of an in-process server-streaming RPC.
type ServerStreamForServer[Resp any] struct {
	state *serverStreamingState[Resp]
}

// NewServerStreaming constructs the client and server endpoints of an in-process server-streaming RPC.
func NewServerStreaming[Resp any](ctx context.Context, options LocalStreamOptions) (*ServerStreamForClient[Resp], *ServerStreamForServer[Resp], context.Context) {
	streamCtx, cancel := context.WithCancel(ctx)
	finishCtx, finishCancel := context.WithCancel(context.Background())
	state := &serverStreamingState[Resp]{
		ctx:          streamCtx,
		cancel:       cancel,
		finishCtx:    finishCtx,
		finishCancel: finishCancel,
		responses:    make(chan streamItem[Resp], options.ResponseBuffer),
		done:         make(chan struct{}),
		streamClosed: options.StreamClosed,
		nilResponse:  options.NilResponse,
	}
	return &ServerStreamForClient[Resp]{state: state}, &ServerStreamForServer[Resp]{state: state}, streamCtx
}

// Recv receives the next response from the server endpoint.
func (c *ServerStreamForClient[Resp]) Recv(ctx context.Context) (Resp, error) {
	s := c.state
	var zero Resp
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case <-s.ctx.Done():
		return zero, s.ctx.Err()
	case resp := <-s.responses:
		close(resp.received)
		return resp.value, nil
	case <-s.done:
		if s.err != nil {
			return zero, s.err
		}
		return zero, io.EOF
	}
}

// Finish asks the server endpoint to stop producing responses and waits for completion.
func (c *ServerStreamForClient[Resp]) Finish(ctx context.Context) error {
	s := c.state
	s.finishCancel()
	defer s.cancel()
	select {
	case <-ctx.Done():
		s.cancel()
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.done:
		return nil
	}
}

// Cancel aborts the RPC and waits for the server endpoint to complete.
func (c *ServerStreamForClient[Resp]) Cancel(ctx context.Context) error {
	s := c.state
	s.cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return nil
	}
}

// Send sends a response to the client endpoint.
func (s *ServerStreamForServer[Resp]) Send(ctx context.Context, resp Resp) error {
	if s.state.nilResponse != nil && isNilStreamValue(resp) {
		return s.state.nilResponse
	}
	return sendStreamResponse(ctx, s.state.ctx, s.state.finishCtx, s.state.done, s.state.responses, s.state.streamClosed, func() error { return s.state.err }, resp)
}

// FinishRequested returns a channel closed when the client asks the server to finish gracefully.
func (s *ServerStreamForServer[Resp]) FinishRequested() <-chan struct{} {
	return s.state.finishCtx.Done()
}

// Complete records the handler result and completes the RPC.
func (s *ServerStreamForServer[Resp]) Complete(err error) {
	s.state.completeOnce.Do(func() {
		s.state.err = err
		close(s.state.done)
	})
}

type bidiStreamingState[Req, Resp any] struct {
	ctx           context.Context
	cancel        context.CancelFunc
	finishCtx     context.Context
	finishCancel  context.CancelFunc
	requests      chan streamItem[Req]
	sendDone      chan struct{}
	closeSendOnce sync.Once
	responses     chan streamItem[Resp]
	done          chan struct{}
	completeOnce  sync.Once
	err           error
	streamClosed  error
	nilRequest    error
	nilResponse   error
}

// BidiStreamForClient is the client-side endpoint of an in-process bidirectional-streaming RPC.
type BidiStreamForClient[Req, Resp any] struct {
	state *bidiStreamingState[Req, Resp]
}

// BidiStreamForServer is the server-side endpoint of an in-process bidirectional-streaming RPC.
type BidiStreamForServer[Req, Resp any] struct {
	state *bidiStreamingState[Req, Resp]
}

// NewBidiStreaming constructs the client and server endpoints of an in-process bidirectional-streaming RPC.
func NewBidiStreaming[Req, Resp any](ctx context.Context, options LocalStreamOptions) (*BidiStreamForClient[Req, Resp], *BidiStreamForServer[Req, Resp], context.Context) {
	streamCtx, cancel := context.WithCancel(ctx)
	finishCtx, finishCancel := context.WithCancel(context.Background())
	state := &bidiStreamingState[Req, Resp]{
		ctx:          streamCtx,
		cancel:       cancel,
		finishCtx:    finishCtx,
		finishCancel: finishCancel,
		requests:     make(chan streamItem[Req], options.RequestBuffer),
		sendDone:     make(chan struct{}),
		responses:    make(chan streamItem[Resp], options.ResponseBuffer),
		done:         make(chan struct{}),
		streamClosed: options.StreamClosed,
		nilRequest:   options.NilRequest,
		nilResponse:  options.NilResponse,
	}
	return &BidiStreamForClient[Req, Resp]{state: state}, &BidiStreamForServer[Req, Resp]{state: state}, streamCtx
}

// Send sends a request to the server endpoint.
func (c *BidiStreamForClient[Req, Resp]) Send(ctx context.Context, req Req) error {
	s := c.state
	if s.nilRequest != nil && isNilStreamValue(req) {
		return s.nilRequest
	}
	select {
	case <-s.sendDone:
		return s.streamClosed
	default:
	}
	item := streamItem[Req]{value: req, received: make(chan struct{})}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.done:
		if s.err != nil {
			return s.err
		}
		return s.streamClosed
	case <-s.sendDone:
		return s.streamClosed
	case s.requests <- item:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-s.done:
			select {
			case <-item.received:
				return nil
			default:
			}
			if s.err != nil {
				return s.err
			}
			return nil
		case <-s.sendDone:
			return s.streamClosed
		case <-item.received:
			return nil
		}
	}
}

// Recv receives the next response from the server endpoint.
func (c *BidiStreamForClient[Req, Resp]) Recv(ctx context.Context) (Resp, error) {
	s := c.state
	var zero Resp
	select {
	case resp := <-s.responses:
		close(resp.received)
		return resp.value, nil
	default:
	}
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case <-s.ctx.Done():
		return zero, s.ctx.Err()
	case resp := <-s.responses:
		close(resp.received)
		return resp.value, nil
	case <-s.done:
		select {
		case resp := <-s.responses:
			close(resp.received)
			return resp.value, nil
		default:
		}
		if s.err != nil {
			return zero, s.err
		}
		return zero, io.EOF
	}
}

// CloseSend closes the request side without waiting for the server endpoint to observe EOF.
func (c *BidiStreamForClient[Req, Resp]) CloseSend(ctx context.Context) error {
	s := c.state
	s.closeSendOnce.Do(func() { close(s.sendDone) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.done:
		return s.err
	default:
		return nil
	}
}

// Finish closes both directions gracefully and waits for the server endpoint to complete.
func (c *BidiStreamForClient[Req, Resp]) Finish(ctx context.Context) error {
	s := c.state
	s.closeSendOnce.Do(func() { close(s.sendDone) })
	s.finishCancel()
	defer s.cancel()
	select {
	case <-ctx.Done():
		s.cancel()
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.done:
		return nil
	}
}

// Cancel aborts the RPC and waits for the server endpoint to complete.
func (c *BidiStreamForClient[Req, Resp]) Cancel(ctx context.Context) error {
	s := c.state
	s.cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return nil
	}
}

// Recv receives the next request from the client endpoint.
func (s *BidiStreamForServer[Req, Resp]) Recv(ctx context.Context) (Req, error) {
	select {
	case req := <-s.state.requests:
		close(req.received)
		return req.value, nil
	default:
	}
	var zero Req
	select {
	case <-s.state.sendDone:
		return zero, io.EOF
	default:
	}
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case <-s.state.ctx.Done():
		return zero, s.state.ctx.Err()
	case req := <-s.state.requests:
		close(req.received)
		return req.value, nil
	case <-s.state.sendDone:
		return zero, io.EOF
	}
}

// Send sends a response to the client endpoint.
func (s *BidiStreamForServer[Req, Resp]) Send(ctx context.Context, resp Resp) error {
	if s.state.nilResponse != nil && isNilStreamValue(resp) {
		return s.state.nilResponse
	}
	return sendBidiStreamResponse(ctx, s.state.ctx, s.state.finishCtx, s.state.done, s.state.responses, s.state.streamClosed, func() error { return s.state.err }, resp)
}

// FinishRequested returns a channel closed when the client asks the server to finish gracefully.
func (s *BidiStreamForServer[Req, Resp]) FinishRequested() <-chan struct{} {
	return s.state.finishCtx.Done()
}

// Complete records the handler result and completes the RPC.
func (s *BidiStreamForServer[Req, Resp]) Complete(err error) {
	s.state.completeOnce.Do(func() {
		s.state.err = err
		close(s.state.done)
	})
}

func sendStreamResponse[T any](ctx, streamCtx, finishCtx context.Context, done <-chan struct{}, responses chan<- streamItem[T], streamClosed error, streamErr func() error, value T) error {
	item := streamItem[T]{value: value, received: make(chan struct{})}
	select {
	case <-done:
		if err := streamErr(); err != nil {
			return err
		}
		return streamClosed
	case <-finishCtx.Done():
		return io.EOF
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-streamCtx.Done():
		select {
		case <-finishCtx.Done():
			return io.EOF
		default:
			return streamCtx.Err()
		}
	case <-finishCtx.Done():
		return io.EOF
	case <-done:
		if err := streamErr(); err != nil {
			return err
		}
		return streamClosed
	case responses <- item:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-streamCtx.Done():
			select {
			case <-finishCtx.Done():
				return io.EOF
			default:
				return streamCtx.Err()
			}
		case <-finishCtx.Done():
			return io.EOF
		case <-done:
			if err := streamErr(); err != nil {
				return err
			}
			return streamClosed
		case <-item.received:
			return nil
		}
	}
}

func sendBidiStreamResponse[T any](ctx, streamCtx, finishCtx context.Context, done <-chan struct{}, responses chan<- streamItem[T], streamClosed error, streamErr func() error, value T) error {
	item := streamItem[T]{value: value, received: make(chan struct{})}
	select {
	case <-done:
		if err := streamErr(); err != nil {
			return err
		}
		return streamClosed
	case <-finishCtx.Done():
		return io.EOF
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-streamCtx.Done():
		select {
		case <-finishCtx.Done():
			return io.EOF
		default:
			return streamCtx.Err()
		}
	case <-finishCtx.Done():
		return io.EOF
	case <-done:
		if err := streamErr(); err != nil {
			return err
		}
		return streamClosed
	case responses <- item:
		return nil
	}
}
