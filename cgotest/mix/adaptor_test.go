// Package cgotest_mix tests the multi-protocol (grpc|connectrpc) adaptor with fallback.
package cgotest_mix

import (
	"connectrpc.com/connect"
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ygrpc/rpccgo/cgotest/testutil"
	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockConnectTestServiceHandler struct {
	pingCalled int32
}

func (m *mockConnectTestServiceHandler) Ping(_ context.Context, req *PingRequest) (*PingResponse, error) {
	atomic.AddInt32(&m.pingCalled, 1)
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type mockGrpcTestServiceServer struct {
	UnimplementedTestServiceServer
}

func (m *mockGrpcTestServiceServer) Ping(_ context.Context, req *PingRequest) (*PingResponse, error) {
	return &PingResponse{Msg: "pong: " + req.GetMsg()}, nil
}

type mockMixConnectStreamServiceHandler struct{}

func (m *mockMixConnectStreamServiceHandler) ClientStreamCall(
	_ context.Context,
	stream *connect.ClientStream[StreamRequest],
) (*StreamResponse, error) {
	joined, err := joinConnectStream(stream)
	if err != nil {
		return nil, err
	}
	return &StreamResponse{Result: "received:" + joined}, nil
}

func (m *mockMixConnectStreamServiceHandler) ServerStreamCall(
	_ context.Context,
	req *StreamRequest,
	stream *connect.ServerStream[StreamResponse],
) error {
	return sendStreamResponses(req.GetData()+"-", stream.Send)
}

func (m *mockMixConnectStreamServiceHandler) BidiStreamCall(
	_ context.Context,
	stream *connect.BidiStream[StreamRequest, StreamResponse],
) error {
	return echoStream(stream.Receive, stream.Send, "echo:")
}

type mockMixGrpcStreamServiceServer struct {
	UnimplementedStreamServiceServer
}

func (m *mockMixGrpcStreamServiceServer) ClientStreamCall(stream StreamService_ClientStreamCallServer) error {
	joined := joinGrpcStream(stream)
	return stream.SendAndClose(&StreamResponse{Result: "received:" + joined})
}

func (m *mockMixGrpcStreamServiceServer) ServerStreamCall(
	req *StreamRequest,
	stream StreamService_ServerStreamCallServer,
) error {
	return sendStreamResponses(req.GetData()+"-", stream.Send)
}

func (m *mockMixGrpcStreamServiceServer) BidiStreamCall(stream StreamService_BidiStreamCallServer) error {
	return echoStream(stream.Recv, stream.Send, "echo:")
}

func registerHandlers(t *testing.T, name string, grpcHandler, connectHandler any) testutil.RegisterFunc {
	return func() func() {
		if grpcHandler != nil {
			_, err := rpcruntime.RegisterGrpcHandler(name, grpcHandler)
			testutil.RequireNoError(t, err)
		}
		if connectHandler != nil {
			_, err := rpcruntime.RegisterConnectHandler(name, connectHandler)
			testutil.RequireNoError(t, err)
		}
		return func() {}
	}
}

func registerStreamHandlers(t *testing.T) testutil.RegisterFunc {
	return registerHandlers(t, StreamService_ServiceName, &mockMixGrpcStreamServiceServer{}, &mockMixConnectStreamServiceHandler{})
}

func pingCall(ctx context.Context, msg string) (string, error) {
	resp, err := TestService_Ping(ctx, &PingRequest{Msg: msg})
	if err != nil {
		return "", err
	}
	return resp.GetMsg(), nil
}

func grpcPingCall(ctx context.Context, msg string) (string, error) {
	ctx = rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)
	return pingCall(ctx, msg)
}

func joinConnectStream(stream *connect.ClientStream[StreamRequest]) (string, error) {
	var builder strings.Builder
	for stream.Receive() {
		builder.WriteString(stream.Msg().GetData())
	}
	if err := stream.Err(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func joinGrpcStream(stream StreamService_ClientStreamCallServer) string {
	var builder strings.Builder
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}
		builder.WriteString(req.GetData())
	}
	return builder.String()
}

func sendStreamResponses(prefix string, send func(*StreamResponse) error) error {
	for _, suffix := range []string{"a", "b", "c"} {
		if err := send(&StreamResponse{Result: prefix + suffix}); err != nil {
			return err
		}
	}
	return nil
}

func echoStream(
	recv func() (*StreamRequest, error),
	send func(*StreamResponse) error,
	prefix string,
) error {
	for {
		req, err := recv()
		if err != nil {
			break
		}
		if err := send(&StreamResponse{Result: prefix + req.GetData()}); err != nil {
			return err
		}
	}
	return nil
}

func TestAllAdaptor_Unary(t *testing.T) {
	testutil.RunUnaryTest(t, registerHandlers(t, TestService_ServiceName, nil, &mockConnectTestServiceHandler{}), pingCall, "hello", "pong: hello")
}

func TestAllAdaptor_ClientStream(t *testing.T) {
	testutil.RunClientStreamTest(t, registerStreamHandlers(t), func(ctx context.Context) (uint64, error) { return StreamService_ClientStreamCallStart(ctx) }, func(handle uint64, data string) error {
		return StreamService_ClientStreamCallSend(handle, &StreamRequest{Data: data})
	}, func(handle uint64) (string, error) {
		resp, err := StreamService_ClientStreamCallFinish(handle)
		if err != nil {
			return "", err
		}
		return resp.GetResult(), nil
	}, []string{"A", "B", "C"}, "received:ABC")
}

func TestAllAdaptor_ServerStream(t *testing.T) {
	testutil.RunServerStreamTest(t, registerStreamHandlers(t), func(ctx context.Context, msg string, onRead func(string) bool) error {
		done := make(chan error, 1)
		onDone := func(err error) { done <- err }
		if err := StreamService_ServerStreamCall(ctx, &StreamRequest{Data: msg}, func(resp *StreamResponse) bool { return onRead(resp.GetResult()) }, onDone); err != nil {
			return err
		}
		return <-done
	}, "test", []string{"test-a", "test-b", "test-c"})
}

func TestAllAdaptor_BidiStream(t *testing.T) {
	testutil.RunBidiStreamTest(t, registerStreamHandlers(t), func(ctx context.Context, onRead func(string) bool, onDone func(error)) (uint64, error) {
		return StreamService_BidiStreamCallStart(ctx, func(resp *StreamResponse) bool { return onRead(resp.GetResult()) }, onDone)
	}, func(handle uint64, data string) error {
		return StreamService_BidiStreamCallSend(handle, &StreamRequest{Data: data})
	}, func(handle uint64) { testutil.RequireNoError(t, StreamService_BidiStreamCallCloseSend(handle)) }, []string{"X", "Y", "Z"}, []string{"echo:X", "echo:Y", "echo:Z"})
}

func TestAllAdaptor_ContextSelection(t *testing.T) {
	t.Run("ExplicitGrpc_NoFallback", func(t *testing.T) {
		mock := &mockConnectTestServiceHandler{}
		_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
		testutil.RequireNoError(t, err)

		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolGrpc)
		_, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		testutil.RequireEqual(t, callErr, rpcruntime.ErrServiceNotRegistered)
		if atomic.LoadInt32(&mock.pingCalled) > 0 {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("NoProtocol_FallbackToConnect", func(t *testing.T) {
		handler := &mockConnectTestServiceHandler{}
		testutil.RunUnaryTest(t, registerHandlers(t, TestService_ServiceName, nil, handler), pingCall, "hello", "pong: hello")
		if atomic.LoadInt32(&handler.pingCalled) == 0 {
			t.Fatalf("expected connect handler to be called")
		}
	})

	t.Run("ExplicitGrpc_UsesGrpc", func(t *testing.T) {
		connectHandler := &mockConnectTestServiceHandler{}
		testutil.RunUnaryTest(t, registerHandlers(t, TestService_ServiceName, &mockGrpcTestServiceServer{}, connectHandler), grpcPingCall, "hello", "pong: hello")
		if atomic.LoadInt32(&connectHandler.pingCalled) > 0 {
			t.Fatalf("expected connect handler not to be called")
		}
	})
}
