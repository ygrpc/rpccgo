package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

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
	mu     sync.Mutex
	notify chan struct{}
	closed bool
	queue  []string
}

func (s *chatStream) Send(_ context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("chat send closed")
	}
	s.queue = append(s.queue, name.SafeString())
	s.signalLocked()
	return nil
}

func (s *chatStream) Recv(ctx context.Context) (string, error) {
	s.mu.Lock()
	for len(s.queue) == 0 && !s.closed {
		notify := s.notifyLocked()
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-notify:
		}
		s.mu.Lock()
	}
	if len(s.queue) == 0 {
		s.mu.Unlock()
		if s.closed {
			return "", io.EOF
		}
		return "", ctx.Err()
	}
	name := s.queue[0]
	s.queue = s.queue[1:]
	s.mu.Unlock()
	return "chat:" + name, nil
}

func (s *chatStream) CloseSend(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.signalLocked()
	return nil
}

func (*chatStream) Cancel(context.Context) error {
	return nil
}

func (s *chatStream) notifyLocked() <-chan struct{} {
	if s.notify == nil {
		s.notify = make(chan struct{})
	}
	return s.notify
}

func (s *chatStream) signalLocked() {
	if s.notify == nil {
		return
	}
	close(s.notify)
	s.notify = nil
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
