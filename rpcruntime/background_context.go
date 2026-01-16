package rpcruntime

import (
	"context"
	"os"
)

// BackgroundContext returns a context.Context intended to be used by generated CGO entrypoints.
//
// It always carries a protocol selection (see WithProtocol / ProtocolFromContext).
//
// Selection rules:
//   - If env var YGRPC_PROTOCOL is set to "grpc", uses ProtocolGrpc.
//   - If env var YGRPC_PROTOCOL is set to "connectrpc", uses ProtocolConnectRPC.
//   - Otherwise returns context.Background() without a protocol value.
func BackgroundContext() context.Context {
	ctx := context.Background()
	switch os.Getenv("YGRPC_PROTOCOL") {
	case string(ProtocolGrpc):
		return WithProtocol(ctx, ProtocolGrpc)
	case string(ProtocolConnectRPC):
		return WithProtocol(ctx, ProtocolConnectRPC)
	default:
		return ctx
	}
}
