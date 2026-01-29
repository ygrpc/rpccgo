// Package cgotest_mix tests the multi-protocol (grpc|connectrpc) adaptor with fallback.
//
// This package contains TWO types of tests:
// 1. Common tests (using testutil.RunXxxTest): Basic RPC functionality shared across protocols
// 2. Protocol-specific tests: Fallback behavior, explicit protocol selection
//
// Protocol-specific subtests that MUST remain in this file:
//   - ExplicitGrpc_NoFallback: Ensures explicit gRPC selection does not fallback to Connect
//   - NoProtocol_FallbackToConnect: Verifies fallback from gRPC to ConnectRPC when gRPC handler absent
//   - ExplicitGrpc_UsesGrpc: Validates explicit gRPC selection actually uses gRPC handler
//
// These test behaviors unique to the multi-protocol adaptor and cannot be unified.
package cgotest_mix

import (
	"context"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockConnectTestServiceHandler struct {
	pingCalled int32
}

func (m *mockConnectTestServiceHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	atomic.AddInt32(&m.pingCalled, 1)
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

func TestAllAdaptor_ContextSelection(t *testing.T) {
	t.Run("ExplicitGrpc_NoFallback", func(t *testing.T) {
		mock := &mockConnectTestServiceHandler{}
		_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
		if err != nil {
			t.Fatalf("RegisterConnectHandler failed: %v", err)
		}

		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolGrpc)
		_, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		if callErr != rpcruntime.ErrServiceNotRegistered {
			t.Fatalf("expected ErrServiceNotRegistered, got %v", callErr)
		}
		if atomic.LoadInt32(&mock.pingCalled) > 0 {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("NoProtocol_FallbackToConnect", func(t *testing.T) {
		testutil.RunUnaryTest(
			t,
			func() func() {
				_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, &mockConnectTestServiceHandler{})
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
	})
}

type mockMixConnectStreamServiceHandler struct {
	clientStreamCalled int32
	serverStreamCalled int32
	bidiStreamCalled   int32
}

func (m *mockMixConnectStreamServiceHandler) ClientStreamCall(
	ctx context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	_ = ctx
	atomic.AddInt32(&m.clientStreamCalled, 1)

	var msgs []string
	for stream.Receive() {
		msgs = append(msgs, stream.Msg().GetData())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	joined := ""
	for _, s := range msgs {
		joined += s
	}
	return &StreamResponse{Result: "connect:received:" + joined}, nil
}

func (m *mockMixConnectStreamServiceHandler) ServerStreamCall(
	ctx context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	_ = ctx
	atomic.AddInt32(&m.serverStreamCalled, 1)

	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: "connect:" + req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockMixConnectStreamServiceHandler) BidiStreamCall(
	ctx context.Context,
	stream *connect.BidiStream[StreamRequest, StreamResponse],
) error {
	_ = ctx
	atomic.AddInt32(&m.bidiStreamCalled, 1)

	for {
		req, err := stream.Receive()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "connect:echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

type mockMixGrpcStreamServiceServer struct {
	UnimplementedStreamServiceServer
	clientStreamCalled int32
	serverStreamCalled int32
	bidiStreamCalled   int32
}

func (m *mockMixGrpcStreamServiceServer) ClientStreamCall(stream StreamService_ClientStreamCallServer) error {
	atomic.AddInt32(&m.clientStreamCalled, 1)

	var msgs []string
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		msgs = append(msgs, req.GetData())
	}
	joined := ""
	for _, s := range msgs {
		joined += s
	}
	return stream.SendAndClose(&StreamResponse{Result: "grpc:received:" + joined})
}

func (m *mockMixGrpcStreamServiceServer) ServerStreamCall(
	req *StreamRequest,
	stream StreamService_ServerStreamCallServer,
) error {
	atomic.AddInt32(&m.serverStreamCalled, 1)

	for i := 0; i < 3; i++ {
		resp := &StreamResponse{Result: "grpc:" + req.GetData() + "-" + string(rune('a'+i))}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockMixGrpcStreamServiceServer) BidiStreamCall(stream StreamService_BidiStreamCallServer) error {
	atomic.AddInt32(&m.bidiStreamCalled, 1)

	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		if err := stream.Send(&StreamResponse{Result: "grpc:echo:" + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

func TestMixAdaptor_StreamServiceStreaming(t *testing.T) {
	connectMock := &mockMixConnectStreamServiceHandler{}
	_, err := rpcruntime.RegisterConnectHandler(StreamService_ServiceName, connectMock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	t.Run("NoProtocol_FallbackToConnect", func(t *testing.T) {
		t.Run("ClientStream", func(t *testing.T) {
			testutil.RunClientStreamTest(
				t,
				func() func() {
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
				"connect:received:ABC",
			)
		})

		t.Run("ServerStream", func(t *testing.T) {
			testutil.RunServerStreamTest(
				t,
				func() func() {
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
					return <-done
				},
				"test",
				[]string{"connect:test-a", "connect:test-b", "connect:test-c"},
			)
		})

		t.Run("BidiStream", func(t *testing.T) {
			testutil.RunBidiStreamTest(
				t,
				func() func() {
					return func() {}
				},
				func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
					return StreamService_BidiStreamCallStart(
						ctx,
						func(resp *StreamResponse) bool {
							return onRead(resp.GetResult())
						},
						onDone,
					)
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
				[]string{"connect:echo:X", "connect:echo:Y", "connect:echo:Z"},
			)
		})
	})

	t.Run("ExplicitGrpc_NoFallback", func(t *testing.T) {
		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolGrpc)
		before := atomic.LoadInt32(&connectMock.serverStreamCalled)

		done := make(chan error, 1)
		onRead := func(*StreamResponse) bool { return true }
		onDone := func(err error) { done <- err }
		err := StreamService_ServerStreamCall(ctx, &StreamRequest{Data: "test"}, onRead, onDone)
		if err != rpcruntime.ErrServiceNotRegistered {
			t.Fatalf("expected ErrServiceNotRegistered, got %v", err)
		}
		if after := atomic.LoadInt32(&connectMock.serverStreamCalled); after != before {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("ExplicitGrpc_UsesGrpc", func(t *testing.T) {
		grpcMock := &mockMixGrpcStreamServiceServer{}
		_, err := rpcruntime.RegisterGrpcHandler(StreamService_ServiceName, grpcMock)
		if err != nil {
			t.Fatalf("RegisterGrpcHandler failed: %v", err)
		}

		t.Run("ClientStream", func(t *testing.T) {
			testutil.RunClientStreamTest(
				t,
				func() func() {
					return func() {}
				},
				func(ctx context.Context) (uint64, error) {
					ctx = rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)
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
				"grpc:received:ABC",
			)
		})

		t.Run("ServerStream", func(t *testing.T) {
			testutil.RunServerStreamTest(
				t,
				func() func() {
					return func() {}
				},
				func(ctx context.Context, msg string, onRead func(string) bool) error {
					ctx = rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)
					done := make(chan error, 1)
					onDone := func(err error) { done <- err }
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
					return <-done
				},
				"test",
				[]string{"grpc:test-a", "grpc:test-b", "grpc:test-c"},
			)
		})

		t.Run("BidiStream", func(t *testing.T) {
			testutil.RunBidiStreamTest(
				t,
				func() func() {
					return func() {}
				},
				func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
					ctx = rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)
					return StreamService_BidiStreamCallStart(
						ctx,
						func(resp *StreamResponse) bool {
							return onRead(resp.GetResult())
						},
						onDone,
					)
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
				[]string{"grpc:echo:X", "grpc:echo:Y", "grpc:echo:Z"},
			)
		})
	})
}
