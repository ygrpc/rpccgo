package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeStreamSessions(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	nativeName := runtimeStreamNativeSessionName(serviceName, method)
	messageName := runtimeStreamMessageSessionName(serviceName, method)
	g.P("type ", nativeName, " struct {")
	g.P("lifecycle rpcruntime.StreamLifecycle")
	if method.CanSend {
		g.P("send func(ctx context.Context", method.NativeArgs, ") error")
	}
	if method.CanRecv {
		g.P("recv func(ctx context.Context) (", method.NativeReturns, ")")
	}
	if method.CanCloseSend {
		g.P("closeSend func(ctx context.Context) error")
	}
	if method.FinishReturnsResponse {
		g.P("finish func(ctx context.Context) (", method.NativeReturns, ")")
	} else {
		g.P("finish func(ctx context.Context) error")
	}
	g.P("cancel func(ctx context.Context) error")
	g.P("}")
	g.P()
	g.P("func (s *", nativeName, ") StreamLifecycle() *rpcruntime.StreamLifecycle {")
	g.P("return &s.lifecycle")
	g.P("}")
	g.P()
	g.P("type ", messageName, " struct {")
	g.P("lifecycle rpcruntime.StreamLifecycle")
	if method.CanSend {
		g.P("send func(ctx context.Context, req []byte) error")
	}
	if method.CanRecv {
		g.P("recv func(ctx context.Context) ([]byte, error)")
	}
	if method.CanCloseSend {
		g.P("closeSend func(ctx context.Context) error")
	}
	if method.FinishReturnsResponse {
		g.P("finish func(ctx context.Context) ([]byte, error)")
	} else {
		g.P("finish func(ctx context.Context) error")
	}
	g.P("cancel func(ctx context.Context) error")
	g.P("}")
	g.P()
	g.P("func (s *", messageName, ") StreamLifecycle() *rpcruntime.StreamLifecycle {")
	g.P("return &s.lifecycle")
	g.P("}")
	g.P()
}

func runtimeStreamNativeSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "NativeStreamSession"
}

func runtimeStreamMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "MessageStreamSession"
}
