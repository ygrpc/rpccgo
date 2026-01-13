package cgotest_connect_suffix

import (
	"context"
	"testing"

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
	mock := &mockConnectSuffixHandler{}
	_, err := rpcruntime.RegisterConnectHandler(TestService_ServiceName, mock)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	t.Run("NoProtocol_DefaultConnect", func(t *testing.T) {
		resp, callErr := TestService_Ping(context.Background(), &PingRequest{Msg: "hello"})
		if callErr != nil {
			t.Fatalf("TestService_Ping failed: %v", callErr)
		}
		if !mock.pingCalled {
			t.Fatalf("expected handler to be called")
		}
		if resp.GetMsg() != "pong: hello" {
			t.Fatalf("expected response 'pong: hello', got %q", resp.GetMsg())
		}
	})

	t.Run("ExplicitConnectRPC", func(t *testing.T) {
		ctx := rpcruntime.WithProtocol(context.Background(), rpcruntime.ProtocolConnectRPC)
		resp, callErr := TestService_Ping(ctx, &PingRequest{Msg: "hello"})
		if callErr != nil {
			t.Fatalf("TestService_Ping failed: %v", callErr)
		}
		if resp.GetMsg() != "pong: hello" {
			t.Fatalf("expected response 'pong: hello', got %q", resp.GetMsg())
		}
	})
}
