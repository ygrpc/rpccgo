package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeMethodProjection) {
	_ = streamRegistryName
	if method.Stream.CanSend {
		renderRuntimeNativeStreamSend(g, serviceName, method)
	}
	if method.Stream.CanRecv {
		renderRuntimeNativeStreamRecv(g, serviceName, method)
	}
	if method.Stream.CanCloseSend {
		renderRuntimeNativeStreamCloseSend(g, serviceName, method)
	}
	renderRuntimeNativeStreamFinish(g, serviceName, method)
	renderRuntimeNativeStreamCancel(g, serviceName, method)
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeMethodProjection, nativeEnabled bool) {
	_ = streamRegistryName
	if method.Stream.CanSend {
		renderRuntimeMessageStreamSend(g, serviceName, method, nativeEnabled)
	}
	if method.Stream.CanRecv {
		renderRuntimeMessageStreamRecv(g, serviceName, method, nativeEnabled)
	}
	if method.Stream.CanCloseSend {
		renderRuntimeMessageStreamCloseSend(g, serviceName, method, nativeEnabled)
	}
	renderRuntimeMessageStreamFinish(g, serviceName, method, nativeEnabled)
	renderRuntimeMessageStreamCancel(g, serviceName, method, nativeEnabled)
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := "Send" + serviceName + "Native" + method.Identity.GoName
	renderDoc(g, name, "sends native request values on an active "+method.Identity.GoName+" stream.")
	g.P("func Send", serviceName, "Native", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle", method.Native.Args, ") error {")
	g.P("entry, err := rpcruntime.SendStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
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
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := "Recv" + serviceName + "Native" + method.Identity.GoName
	renderDoc(g, name, "receives native response values from an active "+method.Identity.GoName+" stream.")
	g.P("func Recv", serviceName, "Native", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) (", method.Native.Returns, ") {")
	g.P("entry, err := rpcruntime.RecvStreamSession(handle)")
	g.P("if err != nil { return ", method.Native.InvalidZero, " }")
	g.P("switch entry.Kind {")
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

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := "CloseSend" + serviceName + "Native" + method.Identity.GoName
	renderDoc(g, name, "closes the native send side of an active "+method.Identity.GoName+" stream.")
	g.P("func CloseSend", serviceName, "Native", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.CloseSendStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
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
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := "Finish" + serviceName + "Native" + method.Identity.GoName
	renderDoc(g, name, "finishes an active native "+method.Identity.GoName+" stream and releases its handle.")
	if method.Stream.FinishReturnsResponse {
		g.P("func Finish", serviceName, "Native", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) (", method.Native.Returns, ") {")
	} else {
		g.P("func Finish", serviceName, "Native", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	}
	g.P("entry, err := rpcruntime.FinishStreamSession(handle)")
	if method.Stream.FinishReturnsResponse {
		g.P("if err != nil { return ", method.Native.InvalidZero, " }")
	} else {
		g.P("if err != nil { return err }")
	}
	g.P("switch entry.Kind {")
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
		g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.Kind)`)
	}
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := "Cancel" + serviceName + "Native" + method.Identity.GoName
	renderDoc(g, name, "cancels an active native "+method.Identity.GoName+" stream and releases its handle.")
	g.P("func Cancel", serviceName, "Native", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.CancelStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
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
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := "Send" + serviceName + "Message" + method.Identity.GoName
	renderDoc(g, name, "sends a message request on an active "+method.Identity.GoName+" stream.")
	g.P("func Send", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle, req ", runtimeMessageRequestType(method), ") error {")
	g.P("if req == nil {")
	g.P(`return errors.New("rpccgo: message request is nil")`)
	g.P("}")
	g.P("entry, err := rpcruntime.SendStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
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
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := "Recv" + serviceName + "Message" + method.Identity.GoName
	renderDoc(g, name, "receives a message response from an active "+method.Identity.GoName+" stream.")
	g.P("func Recv", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) (", runtimeMessageResponseType(method), ", error) {")
	g.P("entry, err := rpcruntime.RecvStreamSession(handle)")
	g.P("if err != nil { return nil, err }")
	g.P("switch entry.Kind {")
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
		g.P("resp, err := source.Recv(ctx)")
		g.P("if err != nil { return nil, err }")
		g.P("if resp == nil {")
		g.P(`return nil, errors.New("rpccgo: message response is nil")`)
		g.P("}")
		g.P("return resp, nil")
	})
	g.P("default:")
	g.P(`return nil, fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := "CloseSend" + serviceName + "Message" + method.Identity.GoName
	renderDoc(g, name, "closes the message send side of an active "+method.Identity.GoName+" stream.")
	g.P("func CloseSend", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.CloseSendStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.CloseSend(ctx)")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := "Finish" + serviceName + "Message" + method.Identity.GoName
	renderDoc(g, name, "finishes an active message "+method.Identity.GoName+" stream and releases its handle.")
	if method.Stream.FinishReturnsResponse {
		g.P("func Finish", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) (", runtimeMessageResponseType(method), ", error) {")
	} else {
		g.P("func Finish", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	}
	g.P("entry, err := rpcruntime.FinishStreamSession(handle)")
	if method.Stream.FinishReturnsResponse {
		g.P("if err != nil { return nil, err }")
	} else {
		g.P("if err != nil { return err }")
	}
	g.P("switch entry.Kind {")
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
		if method.Stream.FinishReturnsResponse {
			g.P("resp, err := source.Finish(ctx)")
			g.P("if err != nil { return nil, err }")
			g.P("if resp == nil {")
			g.P(`return nil, errors.New("rpccgo: message response is nil")`)
			g.P("}")
			g.P("return resp, nil")
		} else {
			g.P("return source.Finish(ctx)")
		}
	})
	g.P("default:")
	if method.Stream.FinishReturnsResponse {
		g.P(`return nil, fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	} else {
		g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	}
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := "Cancel" + serviceName + "Message" + method.Identity.GoName
	renderDoc(g, name, "cancels an active message "+method.Identity.GoName+" stream and releases its handle.")
	g.P("func Cancel", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.CancelStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.Cancel(ctx)")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, serviceName, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Cancel(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamNativeSessionCase(g *protogen.GeneratedFile, method runtimeMethodProjection, kind, sourceName, invalidReturn string, body func()) {
	g.P("case ", kind, ":")
	g.P(sourceName, ", ok := entry.Session.(", method.Symbols.NativeSourceSessionType, ")")
	g.P(`if !ok { return `, invalidReturn, ` }`)
	body()
}

func renderRuntimeNativeStreamMessageSessionCases(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, kind := range messageServerKindsForService(serviceName) {
		g.P("case ", kind, ":")
		g.P(sourceName, ", ok := entry.Session.(", runtimeCGOMessageSessionInterface(method), ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func renderRuntimeMessageStreamNativeSessionCases(g *protogen.GeneratedFile, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, kind := range []string{"rpcruntime.ServerKindGoNative", "rpcruntime.ServerKindCGONative"} {
		g.P("case ", kind, ":")
		g.P(sourceName, ", ok := entry.Session.(", method.Symbols.NativeSourceSessionType, ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func renderRuntimeMessageStreamMessageSessionCases(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, kind := range messageServerKindsForService(serviceName) {
		g.P("case ", kind, ":")
		g.P(sourceName, ", ok := entry.Session.(", runtimeCGOMessageSessionInterface(method), ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func runtimeCGOMessageSessionInterface(method runtimeMethodProjection) string {
	switch method.Stream.Shape {
	case runtimeStreamClient:
		return "rpcruntime.CGOMessageClientStreamSession[" + runtimeMessageRequestType(method) + ", " + runtimeMessageResponseType(method) + "]"
	case runtimeStreamServer:
		return "rpcruntime.CGOMessageServerStreamSession[" + runtimeMessageResponseType(method) + "]"
	case runtimeStreamBidi:
		return "rpcruntime.CGOMessageBidiStreamSession[" + runtimeMessageRequestType(method) + ", " + runtimeMessageResponseType(method) + "]"
	default:
		return "any"
	}
}

func runtimeMessageRequestType(method runtimeMethodProjection) string {
	return "*" + method.Message.RequestType
}

func runtimeMessageResponseType(method runtimeMethodProjection) string {
	return "*" + method.Message.ResponseType
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
