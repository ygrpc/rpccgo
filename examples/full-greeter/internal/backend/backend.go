package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	greeterv1 "example.com/rpccgo-full/proto"
	rpcruntime "rpccgo/rpcruntime"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	return format(name.SafeString(), city.SafeString()), nil
}

func (Greeter) Collect(_ context.Context) (greeterv1.GreeterCollectNativeClientStream, error) {
	return &collectStream{}, nil
}

func (Greeter) Broadcast(_ context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (greeterv1.GreeterBroadcastNativeServerStream, error) {
	return &broadcastStream{
		name:      name.SafeString(),
		remaining: 2,
	}, nil
}

func (Greeter) Chat(_ context.Context) (greeterv1.GreeterChatNativeBidiStream, error) {
	return &chatStream{}, nil
}

type collectStream struct {
	names []string
}

func (s *collectStream) Send(_ context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	s.names = append(s.names, name.SafeString())
	return nil
}

func (s *collectStream) Finish(context.Context) (string, error) {
	return "collect:" + strings.Join(s.names, ","), nil
}

func (*collectStream) Cancel(context.Context) error {
	return nil
}

type broadcastStream struct {
	name      string
	remaining int
}

func (s *broadcastStream) Recv(context.Context) (string, error) {
	if s.remaining == 0 {
		return "", io.EOF
	}
	index := 2 - s.remaining
	s.remaining--
	return fmt.Sprintf("broadcast[%d]:%s", index, s.name), nil
}

func (*broadcastStream) Cancel(context.Context) error {
	return nil
}

type chatStream struct {
	closed bool
	queue  []string
}

func (s *chatStream) Send(_ context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	if s.closed {
		return errors.New("chat send closed")
	}
	s.queue = append(s.queue, name.SafeString())
	return nil
}

func (s *chatStream) Recv(context.Context) (string, error) {
	if len(s.queue) == 0 {
		if s.closed {
			return "", io.EOF
		}
		return "", errors.New("chat receive before send")
	}
	name := s.queue[0]
	s.queue = s.queue[1:]
	return "chat:" + name, nil
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
