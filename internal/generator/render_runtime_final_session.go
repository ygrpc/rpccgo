package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeStreamSessions(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	nativeName := runtimeStreamNativeSessionName(serviceName, method)
	messageName := runtimeStreamMessageSessionName(serviceName, method)
	g.P("type ", nativeName, " struct {")
	g.P("lifecycle rpcruntime.StreamLifecycle")
	if method.Stream.CanSend {
		g.P("send func(ctx context.Context", method.Native.Args, ") error")
	}
	if method.Stream.CanRecv {
		g.P("recv func(ctx context.Context) (", method.Native.Returns, ")")
	}
	if method.Stream.CanCloseSend {
		g.P("closeSend func(ctx context.Context) error")
	}
	if method.Stream.FinishReturnsResponse {
		g.P("finish func(ctx context.Context) (", method.Native.Returns, ")")
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
	if method.Stream.CanSend {
		g.P("send func(ctx context.Context, req []byte) error")
	}
	if method.Stream.CanRecv {
		g.P("recv func(ctx context.Context) ([]byte, error)")
	}
	if method.Stream.CanCloseSend {
		g.P("closeSend func(ctx context.Context) error")
	}
	if method.Stream.FinishReturnsResponse {
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

func runtimeStreamNativeSessionName(serviceName string, method runtimeMethodProjection) string {
	return lowerInitial(serviceName) + method.Identity.GoName + "NativeStreamSession"
}

func runtimeStreamMessageSessionName(serviceName string, method runtimeMethodProjection) string {
	return lowerInitial(serviceName) + method.Identity.GoName + "MessageStreamSession"
}
