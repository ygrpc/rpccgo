package cgotest_connect_suffix

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockConnectSuffixHandler struct {
	pingCalled bool
	lastMsg    string
}

func (m *mockConnectSuffixHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	m.pingCalled = true
	m.lastMsg = req.GetMsg()
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func TestConnectSuffixAdaptor(t *testing.T) {
	t.Run("NoProtocol_DefaultConnect", func(t *testing.T) {
		mock := &mockConnectSuffixHandler{}
		testutil.RunUnaryTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, msg string) (string, error) {
				resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
				if err != nil {
					return "", err
				}
				return resp.GetMsg(), nil
			},
			"hello",
			"pong: hello",
		)

		if !mock.pingCalled {
			t.Error("expected handler to be called")
		}
		if mock.lastMsg != "hello" {
			t.Errorf("expected lastMsg to be 'hello', got %q", mock.lastMsg)
		}
	})

	t.Run("ExplicitConnectRPC", func(t *testing.T) {
		testutil.RunUnaryTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, &mockConnectSuffixHandler{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, msg string) (string, error) {
				ctx = rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolConnectRPC)
				resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
				if err != nil {
					return "", err
				}
				return resp.GetMsg(), nil
			},
			"hello",
			"pong: hello",
		)
	})
}

type mockConnectSuffixStreamServiceHandler struct{}

func (m *mockConnectSuffixStreamServiceHandler) ClientStreamCall(
	ctx context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	_ = ctx
	var msgs []string
	for stream.Receive() {
		msgs = append(msgs, stream.Msg().GetData())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	joined := strings.Join(msgs, "")
	return &StreamResponse{Result: "received:" + joined}, nil
}

func (m *mockConnectSuffixStreamServiceHandler) ServerStreamCall(
	ctx context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	_ = ctx
	if err := stream.Send(&StreamResponse{Result: req.GetData() + "-a"}); err != nil {
		return err
	}
	if err := stream.Send(&StreamResponse{Result: req.GetData() + "-b"}); err != nil {
		return err
	}
	if err := stream.Send(&StreamResponse{Result: req.GetData() + "-c"}); err != nil {
		return err
	}
	return nil
}

func (m *mockConnectSuffixStreamServiceHandler) BidiStreamCall(
	ctx context.Context,
	stream *connect.BidiStream[StreamRequest, StreamResponse],
) error {
	_ = ctx
	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

func TestConnectSuffixAdaptor_StreamServiceStreaming(t *testing.T) {
	t.Run("ClientStreaming", func(t *testing.T) {
		testutil.RunClientStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, &mockConnectSuffixStreamServiceHandler{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context) (uint64, error) {
				return StreamService_ClientStreamCallStart(ctx)
			},
			func(handle uint64, data string) error {
				return StreamService_ClientStreamCallSend(handle, &StreamRequest{Data: data})
			},
			func(handle uint64) (string, error) {
				resp, err := StreamService_ClientStreamCallFinish(handle)
				if err != nil {
					return "", err
				}
				return resp.GetResult(), nil
			},
			[]string{"A", "B", "C"},
			"received:ABC",
		)
	})

	t.Run("ServerStreaming", func(t *testing.T) {
		testutil.RunServerStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, &mockConnectSuffixStreamServiceHandler{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, msg string, onRead func(string) bool) error {
				done := make(chan error, 1)
				onDone := func(err error) {
					done <- err
				}
				if err := StreamService_ServerStreamCall(
					ctx,
					&StreamRequest{Data: msg},
					func(resp *StreamResponse) bool {
						return onRead(resp.GetResult())
					},
					onDone,
				); err != nil {
					return err
				}
				select {
				case err := <-done:
					return err
				case <-time.After(5 * time.Second):
					return context.DeadlineExceeded
				}
			},
			"test",
			[]string{"test-a", "test-b", "test-c"},
		)
	})

	t.Run("BidiStreaming", func(t *testing.T) {
		testutil.RunBidiStreamTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, &mockConnectSuffixStreamServiceHandler{})
				testutil.RequireNoError(t, err)
				return func() {}
			},
			func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
				wrappedDone, startTimer := wrapOnDoneWithTimeout(onDone)
				handle, err := StreamService_BidiStreamCallStart(
					ctx,
					func(resp *StreamResponse) bool {
						return onRead(resp.GetResult())
					},
					wrappedDone,
				)
				if err != nil {
					return 0, err
				}
				startTimer()
				return handle, nil
			},
			func(handle uint64, data string) error {
				return StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: data})
			},
			func(handle uint64) {
				if err := StreamService_BidiStreamCallCloseSend(handle); err != nil {
					t.Fatalf("StreamService_BidiStreamCallCloseSend failed: %v", err)
				}
			},
			[]string{"X", "Y", "Z"},
			[]string{"echo:X", "echo:Y", "echo:Z"},
		)
	})
}

func wrapOnDoneWithTimeout(onDone func(error)) (func(error), func()) {
	done := make(chan struct{})
	var once sync.Once
	wrapped := func(err error) {
		once.Do(func() {
			close(done)
			onDone(err)
		})
	}
	startTimer := func() {
		go func() {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				wrapped(context.DeadlineExceeded)
			}
		}()
	}
	return wrapped, startTimer
}
