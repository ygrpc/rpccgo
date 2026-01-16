package rpcruntime

import (
	"os"
	"testing"
)

func TestBackgroundContext_DefaultDoesNotSetProtocol(t *testing.T) {
	old := os.Getenv("YGRPC_PROTOCOL")
	t.Cleanup(func() {
		_ = os.Setenv("YGRPC_PROTOCOL", old)
	})
	_ = os.Unsetenv("YGRPC_PROTOCOL")

	ctx := BackgroundContext()
	got, ok := ProtocolFromContext(ctx)
	if ok {
		t.Fatalf("expected ok=false, got ok=true with %q", got)
	}
}

func TestBackgroundContext_RespectsEnv(t *testing.T) {
	old := os.Getenv("YGRPC_PROTOCOL")
	t.Cleanup(func() {
		_ = os.Setenv("YGRPC_PROTOCOL", old)
	})

	_ = os.Setenv("YGRPC_PROTOCOL", string(ProtocolGrpc))
	ctx := BackgroundContext()
	got, ok := ProtocolFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != ProtocolGrpc {
		t.Fatalf("expected %q, got %q", ProtocolGrpc, got)
	}
}
