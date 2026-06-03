package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeAdapterMethod) {
	facadeName := nativeRuntimeStreamFacadeName(serviceName, method)
	sessionName := runtimeStreamNativeSessionName(serviceName, method)
	g.P("type ", facadeName, " struct {")
	g.P("handle rpcruntime.StreamHandle")
	g.P("}")
	g.P()
	g.P("func New", facadeName, "(handle rpcruntime.StreamHandle) ", facadeName, " {")
	g.P("return ", facadeName, "{handle: handle}")
	g.P("}")
	g.P()
	if method.CanSend {
		renderRuntimeNativeStreamSend(g, streamRegistryName, sessionName, method, facadeName)
	}
	if method.CanRecv {
		renderRuntimeNativeStreamRecv(g, streamRegistryName, sessionName, method, facadeName)
	}
	if method.CanCloseSend {
		renderRuntimeNativeStreamCloseSend(g, streamRegistryName, sessionName, facadeName)
	}
	renderRuntimeNativeStreamFinish(g, streamRegistryName, sessionName, method, facadeName)
	renderRuntimeNativeStreamCancel(g, streamRegistryName, sessionName, facadeName)
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeAdapterMethod) {
	facadeName := messageRuntimeStreamFacadeName(serviceName, method)
	sessionName := runtimeStreamMessageSessionName(serviceName, method)
	g.P("type ", facadeName, " struct {")
	g.P("handle rpcruntime.StreamHandle")
	g.P("}")
	g.P()
	g.P("func New", facadeName, "(handle rpcruntime.StreamHandle) ", facadeName, " {")
	g.P("return ", facadeName, "{handle: handle}")
	g.P("}")
	g.P()
	if method.CanSend {
		renderRuntimeMessageStreamSend(g, streamRegistryName, sessionName, facadeName)
	}
	if method.CanRecv {
		renderRuntimeMessageStreamRecv(g, streamRegistryName, sessionName, facadeName)
	}
	if method.CanCloseSend {
		renderRuntimeMessageStreamCloseSend(g, streamRegistryName, sessionName, facadeName)
	}
	renderRuntimeMessageStreamFinish(g, streamRegistryName, sessionName, method, facadeName)
	renderRuntimeMessageStreamCancel(g, streamRegistryName, sessionName, facadeName)
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context", method.NativeArgs, ") error {")
	g.P("session, err := rpcruntime.SendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("return session.send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	if method.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
		g.P("session, err := rpcruntime.FinishStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
		g.P("if err != nil { return ", method.NativeInvalidZero, " }")
		g.P("return session.finish(ctx)")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
		g.P("session, err := rpcruntime.FinishStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
		g.P("if err != nil { return err }")
		g.P("return session.finish(ctx)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	g.P("session, err := rpcruntime.RecvStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return ", method.NativeInvalidZero, " }")
	g.P("return session.recv(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("session, err := rpcruntime.CloseSendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("return session.closeSend(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("session, err := rpcruntime.CancelStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("return session.cancel(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context, req []byte) error {")
	g.P("session, err := rpcruntime.SendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("return session.send(ctx, req)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	if method.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) ([]byte, error) {")
		g.P("session, err := rpcruntime.FinishStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
		g.P("if err != nil { return nil, err }")
		g.P("return session.finish(ctx)")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
		g.P("session, err := rpcruntime.FinishStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
		g.P("if err != nil { return err }")
		g.P("return session.finish(ctx)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("session, err := rpcruntime.RecvStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return nil, err }")
	g.P("return session.recv(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("session, err := rpcruntime.CloseSendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("return session.closeSend(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("session, err := rpcruntime.CancelStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("return session.cancel(ctx)")
	g.P("}")
	g.P()
}

func nativeRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "NativeStream"
}

func messageRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "MessageStream"
}
