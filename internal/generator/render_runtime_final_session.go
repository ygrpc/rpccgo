package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeFinalSessions(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	nativeName := runtimeFinalNativeSessionName(serviceName, method)
	messageName := runtimeFinalMessageSessionName(serviceName, method)
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
}

func runtimeFinalNativeSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "NativeFinalSession"
}

func runtimeFinalMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "MessageFinalSession"
}
