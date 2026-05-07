package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	greeterv1 "example.com/rpccgo-full/proto"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: format(req.GetName(), req.GetCity())}, nil
}

func (Greeter) Collect(_ context.Context) (greeterv1.GreeterCollectNativeClientStream, error) {
	return &collectStream{}, nil
}

func (Greeter) Broadcast(_ context.Context, req *greeterv1.SayHelloRequest) (greeterv1.GreeterBroadcastNativeServerStream, error) {
	return &broadcastStream{
		name:      req.GetName(),
		remaining: 2,
	}, nil
}

func (Greeter) Chat(_ context.Context) (greeterv1.GreeterChatNativeBidiStream, error) {
	return &chatStream{}, nil
}

type collectStream struct {
	names []string
}

func (s *collectStream) Send(_ context.Context, req *greeterv1.SayHelloRequest) error {
	s.names = append(s.names, req.GetName())
	return nil
}

func (s *collectStream) Finish(context.Context) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: "collect:" + strings.Join(s.names, ",")}, nil
}

func (*collectStream) Cancel(context.Context) error {
	return nil
}

type broadcastStream struct {
	name      string
	remaining int
}

func (s *broadcastStream) Recv(context.Context) (*greeterv1.SayHelloResponse, error) {
	if s.remaining == 0 {
		return nil, io.EOF
	}
	index := 2 - s.remaining
	s.remaining--
	return &greeterv1.SayHelloResponse{Message: fmt.Sprintf("broadcast[%d]:%s", index, s.name)}, nil
}

func (*broadcastStream) Cancel(context.Context) error {
	return nil
}

type chatStream struct {
	closed bool
	queue  []*greeterv1.SayHelloRequest
}

func (s *chatStream) Send(_ context.Context, req *greeterv1.SayHelloRequest) error {
	if s.closed {
		return errors.New("chat send closed")
	}
	s.queue = append(s.queue, req)
	return nil
}

func (s *chatStream) Recv(context.Context) (*greeterv1.SayHelloResponse, error) {
	if len(s.queue) == 0 {
		if s.closed {
			return nil, io.EOF
		}
		return nil, errors.New("chat receive before send")
	}
	req := s.queue[0]
	s.queue = s.queue[1:]
	return &greeterv1.SayHelloResponse{Message: "chat:" + req.GetName()}, nil
}

func (s *chatStream) CloseSend(context.Context) error {
	s.closed = true
	return nil
}

func (*chatStream) Cancel(context.Context) error {
	return nil
}

func format(name, city string) string {
	if name == "" {
		name = "world"
	}
	if city == "" {
		city = "somewhere"
	}
	return fmt.Sprintf("hello %s from %s", name, city)
}
