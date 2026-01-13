package cgotest_mix

import (
	"context"
	"testing"

	"github.com/ygrpc/rpccgo/rpcruntime"
)

type mockConnectTestServiceHandler struct {
	pingCalled bool
	lastMsg    string
}

func (m *mockConnectTestServiceHandler) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	m.pingCalled = true
	m.lastMsg = req.GetMsg()
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
		if mock.pingCalled {
			t.Fatalf("expected connect handler not to be called")
		}
	})

	t.Run("NoProtocol_FallbackToConnect", func(t *testing.T) {
		mock := &mockConnectTestServiceHandler{}
		_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
		if err != nil {
			t.Fatalf("RegisterConnectHandler failed: %v", err)
		}

		ctx := context.Background()
		resp, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		if callErr != nil {
			t.Fatalf("TestService_Ping failed: %v", callErr)
		}
		if !mock.pingCalled {
			t.Fatalf("expected connect handler to be called")
		}
		if mock.lastMsg != "hello" {
			t.Fatalf("expected lastMsg to be 'hello', got %q", mock.lastMsg)
		}
		if resp.GetMsg() != "pong: hello" {
			t.Fatalf("expected response 'pong: hello', got %q", resp.GetMsg())
		}
	})
}
