package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	if method.Stream.CanSend {
		renderRuntimeNativeStreamSend(g, serviceName, method)
	}
	if method.Stream.CanRecv {
		renderRuntimeNativeStreamRecv(g, serviceName, method)
	}
	if method.Stream.CanCloseSend {
		renderRuntimeNativeStreamCloseSend(g, serviceName, method)
	}
	if method.Stream.CanSend {
		renderRuntimeNativeStreamFinish(g, serviceName, method)
	}
	renderRuntimeNativeStreamCancel(g, serviceName, method)
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	if method.Stream.CanSend {
		renderRuntimeMessageStreamSend(g, serviceName, method, nativeEnabled)
	}
	if method.Stream.CanRecv {
		renderRuntimeMessageStreamRecv(g, serviceName, method, nativeEnabled)
	}
	if method.Stream.CanCloseSend {
		renderRuntimeMessageStreamCloseSend(g, serviceName, method, nativeEnabled)
	}
	if method.Stream.CanSend {
		renderRuntimeMessageStreamFinish(g, serviceName, method, nativeEnabled)
	}
	renderRuntimeMessageStreamCancel(g, serviceName, method, nativeEnabled)
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := runtimeStreamOperationName(serviceName, "Native", method, "Send")
	renderDoc(g, name, "sends native request values on an active "+method.Identity.GoName+" stream.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle", method.Native.Args, ") error {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	for _, route := range method.Routes.NativeServers {
		renderRuntimeNativeStreamNativeSessionCase(g, method, route, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.Send(ctx, ", runtimeNativeRequestLiteral(method), ")")
		})
	}
	renderRuntimeNativeStreamMessageSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
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
	name := runtimeStreamOperationName(serviceName, "Native", method, "Recv")
	renderDoc(g, name, "receives native response values from an active "+method.Identity.GoName+" stream.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) (", method.Native.Returns, ") {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return ", method.Native.InvalidZero, " }")
	g.P("switch entry.Kind {")
	for _, route := range method.Routes.NativeServers {
		renderRuntimeNativeStreamNativeSessionCase(g, method, route, "source", method.Native.InvalidZero, func() {
			if method.Native.ResultNames == "" {
				g.P("_, err := source.Recv(ctx)")
			} else {
				g.P("resp, err := source.Recv(ctx)")
			}
			if method.Stream.Shape == runtimeStreamServer {
				g.P("if errors.Is(err, io.EOF) {")
				g.P("if finisher, ok := any(source).(interface{ Finish(context.Context) error }); ok {")
				g.P("if finishErr := finisher.Finish(ctx); finishErr != nil { _, _ = rpcruntime.RemoveStreamSession(handle); return ", method.Native.ErrZero, " }")
				g.P("}")
				g.P("_, _ = rpcruntime.RemoveStreamSession(handle)")
				g.P("}")
			}
			g.P("if err != nil { return ", method.Native.ErrZero, " }")
			g.P("return ", runtimeNativeResponseReturn("resp", method))
		})
	}
	renderRuntimeNativeStreamMessageSessionCases(g, method, "source", method.Native.InvalidZero, func() {
		g.P("messageResp, err := source.Recv(ctx)")
		if method.Stream.Shape == runtimeStreamServer {
			g.P("if errors.Is(err, io.EOF) {")
			g.P("if finisher, ok := any(source).(interface{ Finish(context.Context) error }); ok {")
			g.P("if finishErr := finisher.Finish(ctx); finishErr != nil { _, _ = rpcruntime.RemoveStreamSession(handle); return ", method.Native.ErrZero, " }")
			g.P("}")
			g.P("_, _ = rpcruntime.RemoveStreamSession(handle)")
			g.P("}")
		}
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
	name := runtimeStreamOperationName(serviceName, "Native", method, "CloseSend")
	renderDoc(g, name, "closes the native send side of an active "+method.Identity.GoName+" stream.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	for _, route := range method.Routes.NativeServers {
		renderRuntimeNativeStreamNativeSessionCase(g, method, route, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.CloseSend(ctx)")
		})
	}
	renderRuntimeNativeStreamMessageSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	name := runtimeStreamOperationName(serviceName, "Native", method, "Finish")
	renderDoc(g, name, "finishes an active native "+method.Identity.GoName+" stream and releases its handle.")
	if method.Stream.FinishReturnsResponse {
		g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) (", method.Native.Returns, ") {")
	} else {
		g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	}
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
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
	for _, route := range method.Routes.NativeServers {
		renderRuntimeNativeStreamNativeSessionCase(g, method, route, "source", invalidReturn, func() {
			if method.Stream.FinishReturnsResponse {
				if method.Native.ResultNames == "" {
					g.P("_, err := source.Finish(ctx)")
				} else {
					g.P("resp, err := source.Finish(ctx)")
				}
				g.P("if err != nil { return ", method.Native.ErrZero, " }")
				g.P("if _, err = rpcruntime.RemoveStreamSession(handle); err != nil { return ", invalidReturn, " }")
				g.P("return ", runtimeNativeResponseReturn("resp", method))
			} else {
				g.P("if err := source.Finish(ctx); err != nil { return err }")
				g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
				g.P("return err")
			}
		})
	}
	renderRuntimeNativeStreamMessageSessionCases(g, method, "source", invalidReturn, func() {
		if method.Stream.FinishReturnsResponse {
			g.P("messageResp, err := source.Finish(ctx)")
			g.P("if err != nil { return ", method.Native.ErrZero, " }")
			g.P("if _, err = rpcruntime.RemoveStreamSession(handle); err != nil { return ", invalidReturn, " }")
			g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
		} else {
			g.P("if err := source.Finish(ctx); err != nil { return err }")
			g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
			g.P("return err")
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
	name := runtimeStreamOperationName(serviceName, "Native", method, "Cancel")
	renderDoc(g, name, "cancels an active native "+method.Identity.GoName+" stream and releases its handle.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	for _, route := range method.Routes.NativeServers {
		renderRuntimeNativeStreamNativeSessionCase(g, method, route, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("if err := source.Cancel(ctx); err != nil { return err }")
			g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
			g.P("return err")
		})
	}
	renderRuntimeNativeStreamMessageSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("if err := source.Cancel(ctx); err != nil { return err }")
		g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
		g.P("return err")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` native stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := runtimeStreamOperationName(serviceName, "Message", method, "Send")
	renderDoc(g, name, "sends a message request on an active "+method.Identity.GoName+" stream.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle, req ", runtimeMessageRequestType(method), ") error {")
	g.P("if req == nil {")
	g.P(`return errors.New("rpccgo: message request is nil")`)
	g.P("}")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P(method.Codec.MessageToNativeRequestAssignNames, " := ", method.Codec.MessageToNativeRequest, "(req)")
			g.P("if err != nil { return err }")
			g.P("err = source.Send(ctx, ", runtimeNativeRequestLiteral(method), ")")
			g.P("goruntime.KeepAlive(reqOwner)")
			g.P("return err")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.Send(ctx, req)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := runtimeStreamOperationName(serviceName, "Message", method, "Recv")
	renderDoc(g, name, "receives a message response from an active "+method.Identity.GoName+" stream.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) (", runtimeMessageResponseType(method), ", error) {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return nil, err }")
	g.P("switch entry.Kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "nil, rpcruntime.ErrStreamInvalidHandle", func() {
			if method.Native.ResultNames == "" {
				g.P("_, err := source.Recv(ctx)")
			} else {
				g.P("resp, err := source.Recv(ctx)")
			}
			if method.Stream.Shape == runtimeStreamServer {
				g.P("if errors.Is(err, io.EOF) {")
				g.P("if finisher, ok := any(source).(interface{ Finish(context.Context) error }); ok {")
				g.P("if finishErr := finisher.Finish(ctx); finishErr != nil { _, _ = rpcruntime.RemoveStreamSession(handle); return nil, finishErr }")
				g.P("}")
				g.P("_, _ = rpcruntime.RemoveStreamSession(handle)")
				g.P("}")
			}
			g.P("if err != nil { return nil, err }")
			g.P("return ", method.Codec.NativeResponseToMessage, "(", runtimeNativeResponseFieldArgs("resp", method), ")")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, method, "source", "nil, rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("resp, err := source.Recv(ctx)")
		if method.Stream.Shape == runtimeStreamServer {
			g.P("if errors.Is(err, io.EOF) {")
			g.P("if finisher, ok := any(source).(interface{ Finish(context.Context) error }); ok {")
			g.P("if finishErr := finisher.Finish(ctx); finishErr != nil { _, _ = rpcruntime.RemoveStreamSession(handle); return nil, finishErr }")
			g.P("}")
			g.P("_, _ = rpcruntime.RemoveStreamSession(handle)")
			g.P("}")
		}
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
	name := runtimeStreamOperationName(serviceName, "Message", method, "CloseSend")
	renderDoc(g, name, "closes the message send side of an active "+method.Identity.GoName+" stream.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("return source.CloseSend(ctx)")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("return source.CloseSend(ctx)")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection, nativeEnabled bool) {
	name := runtimeStreamOperationName(serviceName, "Message", method, "Finish")
	renderDoc(g, name, "finishes an active message "+method.Identity.GoName+" stream and releases its handle.")
	if method.Stream.FinishReturnsResponse {
		g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) (", runtimeMessageResponseType(method), ", error) {")
	} else {
		g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	}
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
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
					g.P("_, err := source.Finish(ctx)")
				} else {
					g.P("resp, err := source.Finish(ctx)")
				}
				g.P("if err != nil { return nil, err }")
				g.P("if _, err = rpcruntime.RemoveStreamSession(handle); err != nil { return ", invalidReturn, " }")
				g.P("return ", method.Codec.NativeResponseToMessage, "(", runtimeNativeResponseFieldArgs("resp", method), ")")
			} else {
				g.P("if err := source.Finish(ctx); err != nil { return err }")
				g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
				g.P("return err")
			}
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, method, "source", invalidReturn, func() {
		if method.Stream.FinishReturnsResponse {
			g.P("resp, err := source.Finish(ctx)")
			g.P("if err != nil { return nil, err }")
			g.P("if resp == nil {")
			g.P(`return nil, errors.New("rpccgo: message response is nil")`)
			g.P("}")
			g.P("if _, err = rpcruntime.RemoveStreamSession(handle); err != nil { return ", invalidReturn, " }")
			g.P("return resp, nil")
		} else {
			g.P("if err := source.Finish(ctx); err != nil { return err }")
			g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
			g.P("return err")
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
	name := runtimeStreamOperationName(serviceName, "Message", method, "Cancel")
	renderDoc(g, name, "cancels an active message "+method.Identity.GoName+" stream and releases its handle.")
	g.P("func ", name, "(ctx context.Context, handle rpcruntime.StreamHandle) error {")
	g.P("entry, err := rpcruntime.LoadStreamSession(handle)")
	g.P("if err != nil { return err }")
	g.P("switch entry.Kind {")
	if nativeEnabled {
		renderRuntimeMessageStreamNativeSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
			g.P("if err := source.Cancel(ctx); err != nil { return err }")
			g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
			g.P("return err")
		})
	}
	renderRuntimeMessageStreamMessageSessionCases(g, method, "source", "rpcruntime.ErrStreamInvalidHandle", func() {
		g.P("if err := source.Cancel(ctx); err != nil { return err }")
		g.P("_, err = rpcruntime.RemoveStreamSession(handle)")
		g.P("return err")
	})
	g.P("default:")
	g.P(`return fmt.Errorf("rpccgo: `, serviceName, ` message stream session kind %d is unsupported", entry.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamNativeSessionCase(g *protogen.GeneratedFile, method runtimeMethodProjection, route runtimeServerRouteProjection, sourceName, invalidReturn string, body func()) {
	g.P("case ", route.Kind, ":")
	g.P(sourceName, ", ok := entry.Session.(", runtimeNativeStreamingClientInterface(method), ")")
	g.P(`if !ok { return `, invalidReturn, ` }`)
	body()
}

func renderRuntimeNativeStreamMessageSessionCases(g *protogen.GeneratedFile, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, route := range method.Routes.MessageServers {
		g.P("case ", route.Kind, ":")
		g.P(sourceName, ", ok := entry.Session.(", runtimeMessageStreamingClientInterface(method), ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func renderRuntimeMessageStreamNativeSessionCases(g *protogen.GeneratedFile, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, route := range method.Routes.NativeServers {
		g.P("case ", route.Kind, ":")
		g.P(sourceName, ", ok := entry.Session.(", runtimeNativeStreamingClientInterface(method), ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func runtimeNativeStreamingClientInterface(method runtimeMethodProjection) string {
	switch method.Stream.Shape {
	case runtimeStreamClient:
		return "rpcruntime.ClientStreamingClient[" + method.Symbols.NativeStreamRequestType + ", " + method.Symbols.NativeStreamResponseType + "]"
	case runtimeStreamServer:
		return "rpcruntime.ServerStreamingClient[" + method.Symbols.NativeStreamResponseType + "]"
	case runtimeStreamBidi:
		return "rpcruntime.BidiStreamingClient[" + method.Symbols.NativeStreamRequestType + ", " + method.Symbols.NativeStreamResponseType + "]"
	default:
		return "any"
	}
}

func runtimeNativeRequestLiteral(method runtimeMethodProjection) string {
	if method.Native.ArgNames == "" {
		return method.Symbols.NativeStreamRequestType + "{}"
	}
	parts := strings.Split(method.Native.ArgNames, ", ")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		fields = append(fields, upperInitial(part)+": "+part)
	}
	return method.Symbols.NativeStreamRequestType + "{" + strings.Join(fields, ", ") + "}"
}

func runtimeNativeResponseReturn(prefix string, method runtimeMethodProjection) string {
	if method.Native.ResultNames == "" {
		return "nil"
	}
	parts := strings.Split(method.Native.ResultNames, ", ")
	values := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		name := strings.TrimSuffix(part, "Result")
		values = append(values, prefix+"."+upperInitial(name))
	}
	values = append(values, "nil")
	return strings.Join(values, ", ")
}

func runtimeNativeResponseFieldArgs(prefix string, method runtimeMethodProjection) string {
	if method.Native.ResultNames == "" {
		return ""
	}
	parts := strings.Split(method.Native.ResultNames, ", ")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSuffix(part, "Result")
		values = append(values, prefix+"."+upperInitial(name))
	}
	return strings.Join(values, ", ")
}

func renderRuntimeMessageStreamMessageSessionCases(g *protogen.GeneratedFile, method runtimeMethodProjection, sourceName, invalidReturn string, body func()) {
	for _, route := range method.Routes.MessageServers {
		g.P("case ", route.Kind, ":")
		g.P(sourceName, ", ok := entry.Session.(", runtimeMessageStreamingClientInterface(method), ")")
		g.P(`if !ok { return `, invalidReturn, ` }`)
		body()
	}
}

func runtimeStreamOperationName(serviceName, contract string, method runtimeMethodProjection, operation string) string {
	return serviceName + contract + method.Identity.GoName + operation
}

func runtimeMessageStreamingClientInterface(method runtimeMethodProjection) string {
	switch method.Stream.Shape {
	case runtimeStreamClient:
		return "rpcruntime.ClientStreamingClient[" + runtimeMessageRequestType(method) + ", " + runtimeMessageResponseType(method) + "]"
	case runtimeStreamServer:
		return "rpcruntime.ServerStreamingClient[" + runtimeMessageResponseType(method) + "]"
	case runtimeStreamBidi:
		return "rpcruntime.BidiStreamingClient[" + runtimeMessageRequestType(method) + ", " + runtimeMessageResponseType(method) + "]"
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
