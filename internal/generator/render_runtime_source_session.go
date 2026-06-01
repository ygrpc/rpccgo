package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeSourceSessionInterfaces(g *protogen.GeneratedFile, serviceName string, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		nativeName := method.SessionName
		messageName := methodMessageSessionName(method)
		g.P("type ", nativeName, " interface {")
		if method.CanSend {
			g.P("Send(ctx context.Context", method.NativeArgs, ") error")
		}
		if method.CanRecv {
			g.P("Recv(ctx context.Context) (", method.NativeReturns, ")")
		}
		if method.CanCloseSend {
			g.P("CloseSend(ctx context.Context) error")
		}
		if method.FinishReturnsResponse {
			g.P("Finish(ctx context.Context) (", method.NativeReturns, ")")
		} else {
			g.P("Finish(ctx context.Context) error")
		}
		g.P("Cancel(ctx context.Context) error")
		g.P("}")
		g.P()
		g.P("type ", messageName, " interface {")
		if method.CanSend {
			g.P("Send(ctx context.Context, req []byte) error")
		}
		if method.CanRecv {
			g.P("Recv(ctx context.Context) ([]byte, error)")
		}
		if method.CanCloseSend {
			g.P("CloseSend(ctx context.Context) error")
		}
		if method.FinishReturnsResponse {
			g.P("Finish(ctx context.Context) ([]byte, error)")
		} else {
			g.P("Finish(ctx context.Context) error")
		}
		g.P("Cancel(ctx context.Context) error")
		g.P("}")
		g.P()
	}
	_ = serviceName
}

func methodMessageSessionName(method runtimeAdapterMethod) string {
	return strings.Replace(method.SessionName, "NativeStreamSession", "MessageStreamSession", 1)
}
