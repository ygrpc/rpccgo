package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeAdapterMethod) {
	facadeName := nativeRuntimeStreamFacadeName(serviceName, method)
	sessionName := runtimeFinalNativeSessionName(serviceName, method)
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
	sessionName := runtimeFinalMessageSessionName(serviceName, method)
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
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.EnsureCanSend(); err != nil { return err }")
	g.P("return session.send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	if method.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, method.NativeInvalidZero)
		g.P("return session.finish(ctx)")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, "rpcruntime.ErrStreamInvalidHandle")
		g.P("return session.finish(ctx)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	renderRuntimeLoadSessionWithReturn(g, streamRegistryName, sessionName, method.NativeInvalidZero)
	g.P("return session.recv(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.MarkSendClosed(); err != nil { return err }")
	g.P("return session.closeSend(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	renderRuntimeTakeSessionWithoutFinalize(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.MarkCanceled(); err != nil { return err }")
	g.P("return session.cancel(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context, req []byte) error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.EnsureCanSend(); err != nil { return err }")
	g.P("return session.send(ctx, req)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	if method.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) ([]byte, error) {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, "nil, rpcruntime.ErrStreamInvalidHandle")
		g.P("return session.finish(ctx)")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, "rpcruntime.ErrStreamInvalidHandle")
		g.P("return session.finish(ctx)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	renderRuntimeLoadSessionWithReturn(g, streamRegistryName, sessionName, "nil, rpcruntime.ErrStreamInvalidHandle")
	g.P("return session.recv(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.MarkSendClosed(); err != nil { return err }")
	g.P("return session.closeSend(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	renderRuntimeTakeSessionWithoutFinalize(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.MarkCanceled(); err != nil { return err }")
	g.P("return session.cancel(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeLoadSession(g *protogen.GeneratedFile, registryName, sessionName string) {
	renderRuntimeLoadSessionWithReturn(g, registryName, sessionName, "rpcruntime.ErrStreamInvalidHandle")
}

func renderRuntimeLoadSessionWithReturn(g *protogen.GeneratedFile, registryName, sessionName, invalidReturn string) {
	g.P("value, ok := ", registryName, ".Load(s.handle)")
	g.P("if !ok { return ", invalidReturn, " }")
	g.P("session, ok := value.(*", sessionName, ")")
	g.P("if !ok { return ", invalidReturn, " }")
}

func renderRuntimeTakeSession(g *protogen.GeneratedFile, registryName, sessionName, invalidReturn string) {
	renderRuntimeLoadSessionWithReturn(g, registryName, sessionName, invalidReturn)
	g.P("taken, ok := ", registryName, ".Take(s.handle)")
	g.P("if !ok || taken != session { return ", invalidReturn, " }")
	g.P("if !session.lifecycle.Finalize() { return ", invalidReturn, " }")
}

func renderRuntimeTakeSessionWithoutFinalize(g *protogen.GeneratedFile, registryName, sessionName string) {
	renderRuntimeLoadSession(g, registryName, sessionName)
	g.P("taken, ok := ", registryName, ".Take(s.handle)")
	g.P("if !ok || taken != session { return rpcruntime.ErrStreamInvalidHandle }")
}

func nativeRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "NativeStream"
}

func messageRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "MessageStream"
}
