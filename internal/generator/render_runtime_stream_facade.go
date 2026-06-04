package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeMethodProjection) {
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
	if method.Stream.CanSend {
		renderRuntimeNativeStreamSend(g, serviceName, streamRegistryName, sessionName, method, facadeName)
	}
	if method.Stream.CanRecv {
		renderRuntimeNativeStreamRecv(g, serviceName, streamRegistryName, sessionName, method, facadeName)
	}
	if method.Stream.CanCloseSend {
		renderRuntimeNativeStreamCloseSend(g, serviceName, streamRegistryName, sessionName, method, facadeName)
	}
	renderRuntimeNativeStreamFinish(g, serviceName, streamRegistryName, sessionName, method, facadeName)
	renderRuntimeNativeStreamCancel(g, serviceName, streamRegistryName, sessionName, method, facadeName)
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeMethodProjection, nativeEnabled bool) {
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
	if method.Stream.CanSend {
		renderRuntimeMessageStreamSend(g, serviceName, streamRegistryName, sessionName, method, facadeName, nativeEnabled)
	}
	if method.Stream.CanRecv {
		renderRuntimeMessageStreamRecv(g, serviceName, streamRegistryName, sessionName, method, facadeName, nativeEnabled)
	}
	if method.Stream.CanCloseSend {
		renderRuntimeMessageStreamCloseSend(g, serviceName, streamRegistryName, sessionName, method, facadeName, nativeEnabled)
	}
	renderRuntimeMessageStreamFinish(g, serviceName, streamRegistryName, sessionName, method, facadeName, nativeEnabled)
	renderRuntimeMessageStreamCancel(g, serviceName, streamRegistryName, sessionName, method, facadeName, nativeEnabled)
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context", method.Native.Args, ") error {")
	g.P("entry, err := rpcruntime.SendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.kind {")
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindGoNative", "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Send(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
	})
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindCGONative", "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Send(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
	})
	renderRuntimeNativeStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
		g.P("if err != nil { return err }")
		g.P("return source.Send(ctx, messageReq)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) (", method.Native.Returns, ") {")
	g.P("entry, err := rpcruntime.RecvStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return ", method.Native.InvalidZero, " }")
	g.P("switch entry.kind {")
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindGoNative", "source", method.Native.InvalidZero, func() {
		g.P("return source.Recv(ctx)")
	})
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindCGONative", "source", method.Native.InvalidZero, func() {
		g.P("return source.Recv(ctx)")
	})
	renderRuntimeNativeStreamMessageSessionCases(g, serviceName, method, "source", method.Native.InvalidZero, func() {
		g.P("messageResp, err := source.Recv(ctx)")
		g.P("if err != nil { return ", method.Native.ErrZero, " }")
		g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
	})
	g.P("default:")
	g.P("return ", method.Native.ErrZero)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("entry, err := rpcruntime.CloseSendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.kind {")
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindGoNative", "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindCGONative", "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	renderRuntimeNativeStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string) {
	if method.Stream.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) (", method.Native.Returns, ") {")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
	}
	g.P("entry, err := rpcruntime.FinishStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	if method.Stream.FinishReturnsResponse {
		g.P("if err != nil { return ", method.Native.InvalidZero, " }")
	} else {
		g.P("if err != nil { return err }")
	}
	g.P("switch entry.kind {")
	invalidReturn := "rpcruntime.ErrStreamInvalidHandle"
	if method.Stream.FinishReturnsResponse {
		invalidReturn = method.Native.InvalidZero
	}
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindGoNative", "source", invalidReturn, func() {
		g.P("return source.Finish(ctx)")
	})
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindCGONative", "source", invalidReturn, func() {
		g.P("return source.Finish(ctx)")
	})
	renderRuntimeNativeStreamMessageSessionCases(g, serviceName, method, "source", invalidReturn, func() {
		if method.Stream.FinishReturnsResponse {
			g.P("messageResp, err := source.Finish(ctx)")
			g.P("if err != nil { return ", method.Native.ErrZero, " }")
			g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
		} else {
			g.P("return source.Finish(ctx)")
		}
	})
	g.P("default:")
	if method.Stream.FinishReturnsResponse {
		g.P("return ", method.Native.ErrZero)
	} else {
		g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.kind)`)
	}
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("entry, err := rpcruntime.CancelStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.kind {")
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindGoNative", "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Cancel(ctx)")
	})
	renderRuntimeNativeStreamNativeSessionCase(g, method, "rpcruntime.ServerKindCGONative", "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Cancel(ctx)")
	})
	renderRuntimeNativeStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Cancel(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string, nativeEnabled bool) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context, req []byte) error {")
	g.P("entry, err := rpcruntime.SendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P(method.Codec.MessageToNativeRequestAssignNames, " := ", method.Codec.MessageToNativeRequest, "(req)")
			g.P("if err != nil { return err }")
			g.P("err = source.Send(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
			g.P("goruntime.KeepAlive(reqOwner)")
			g.P("return err")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Send(ctx, req)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string, nativeEnabled bool) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("entry, err := rpcruntime.RecvStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return nil, err }")
	g.P("switch entry.kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "nil, rpcruntime.ErrStreamInvalidHandle", func() {
			if method.Native.ResultNames == "" {
				g.P("err := source.Recv(ctx)")
			} else {
				g.P(method.Native.ResultNames, ", err := source.Recv(ctx)")
			}
			g.P("if err != nil { return nil, err }")
			g.P("return ", method.Codec.NativeResponseToMessage, "(", method.Native.ResultNames, ")")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", "nil, rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Recv(ctx)")
	})
	g.P("default:")
	g.P(`return nil, fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string, nativeEnabled bool) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("entry, err := rpcruntime.CloseSendStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.CloseSend(ctx)")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string, nativeEnabled bool) {
	if method.Stream.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) ([]byte, error) {")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
	}
	g.P("entry, err := rpcruntime.FinishStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	if method.Stream.FinishReturnsResponse {
		g.P("if err != nil { return nil, err }")
	} else {
		g.P("if err != nil { return err }")
	}
	g.P("switch entry.kind {")
	invalidReturn := "rpcruntime.ErrStreamInvalidHandle"
	if method.Stream.FinishReturnsResponse {
		invalidReturn = "nil, rpcruntime.ErrStreamInvalidHandle"
	}
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", invalidReturn, func() {
			if method.Stream.FinishReturnsResponse {
				if method.Native.ResultNames == "" {
					g.P("err := source.Finish(ctx)")
				} else {
					g.P(method.Native.ResultNames, ", err := source.Finish(ctx)")
				}
				g.P("if err != nil { return nil, err }")
				g.P("return ", method.Codec.NativeResponseToMessage, "(", method.Native.ResultNames, ")")
			} else {
				g.P("return source.Finish(ctx)")
			}
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", invalidReturn, func() {
		g.P("return source.Finish(ctx)")
	})
	g.P("default:")
	if method.Stream.FinishReturnsResponse {
		g.P(`return nil, fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.kind)`)
	} else {
		g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.kind)`)
	}
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, serviceName, streamRegistryName, sessionName string, method runtimeMethodProjection, facadeName string, nativeEnabled bool) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("entry, err := rpcruntime.CancelStreamSession[*", sessionName, "](&", streamRegistryName, ", s.handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.Cancel(ctx)")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Cancel(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamNativeSessionCase(g *protogen.GeneratedFile, method runtimeMethodProjection, kind, sourceName, invalidReturn string, body func()) {
	g.P("case ", kind, ":")
	g.P(sourceName, ", ok := entry.session.(", method.Symbols.NativeSourceSessionType, ")")
	g.P(`if !ok { return `, invalidReturn, ` }`)
	body()
}

func renderRuntimeNativeStreamMessageSessionCases(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, kind := range messageServerKindsForService(serviceName) {
		g.P("case ", kind, ":")
		g.P(sourceName, ", ok := entry.session.(", method.Symbols.MessageSourceSessionType, ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func renderRuntimeMessageStreamNativeSessionCases(g *protogen.GeneratedFile, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, kind := range []string{"rpcruntime.ServerKindGoNative", "rpcruntime.ServerKindCGONative"} {
		g.P("case ", kind, ":")
		g.P(sourceName, ", ok := entry.session.(", method.Symbols.NativeSourceSessionType, ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func renderRuntimeMessageStreamMessageSessionCases(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, kind := range messageServerKindsForService(serviceName) {
		g.P("case ", kind, ":")
		g.P(sourceName, ", ok := entry.session.(", method.Symbols.MessageSourceSessionType, ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func messageServerKindsForService(serviceName string) []string {
	_ = serviceName
	return []string{
		"rpcruntime.ServerKindCGOMessage",
		"rpcruntime.ServerKindConnect",
		"rpcruntime.ServerKindConnectRemote",
		"rpcruntime.ServerKindGRPC",
		"rpcruntime.ServerKindGRPCRemote",
	}
}

func nativeRuntimeStreamFacadeName(serviceName string, method runtimeMethodProjection) string {
	return serviceName + method.Identity.GoName + "NativeStream"
}

func messageRuntimeStreamFacadeName(serviceName string, method runtimeMethodProjection) string {
	return serviceName + method.Identity.GoName + "MessageStream"
}
