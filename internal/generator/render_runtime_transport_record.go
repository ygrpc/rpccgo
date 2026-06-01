package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeTransportMessageRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, codecEnabled bool, activeName, recordName, transportExpr, label string) {
	g.P("record := &", recordName, "{")
	g.P("}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("record.invokeMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) ([]byte, error) {")
			renderRuntimeTransportUnaryMessageCall(g, service, method, transportExpr, label, "req")
			g.P("}")
			g.P("record.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			if codecEnabled {
				g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				g.P("var messageResp []byte")
				renderRuntimeTransportUnaryNativeMessageCall(g, service, method, transportExpr, label, "messageReq")
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				for _, decl := range method.NativeVarDecls {
					g.P(decl)
				}
				if method.NativeNames == "" {
					g.P("err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
				} else {
					g.P(method.NativeNames, ", err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
				}
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				if method.NativeNames == "" {
					g.P("return nil")
				} else {
					g.P("return ", method.NativeNames, ", nil")
				}
			} else {
				g.P("return ", method.NativeConverterZero)
			}
			g.P("}")
			continue
		}
		renderRuntimeTransportMessageStreamRecord(g, service, method, codecEnabled, transportExpr, label)
	}
	g.P(activeName, ".Store(record)")
	g.P("return nil")
}

func renderRuntimeTransportUnaryMessageCall(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr, label, reqExpr string) {
	methodPlan := methodForRuntimeService(service, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	g.P("messageReq := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: ", label, " request protobuf unmarshal failed: %w\", err)")
	g.P("}")
	g.P("messageResp, err := ", transportExpr, ".", method.MethodGoName, "(ctx, messageReq)")
	g.P("if err != nil { return nil, err }")
	g.P("resp, err := proto.Marshal(messageResp)")
	g.P("if err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: ", label, " response protobuf marshal failed: %w\", err)")
	g.P("}")
	g.P("return resp, nil")
}

func renderRuntimeTransportUnaryNativeMessageCall(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr, label, reqExpr string) {
	methodPlan := methodForRuntimeService(service, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	g.P("directReq := new(", reqType, ")")
	g.P("if err = proto.Unmarshal(", reqExpr, ", directReq); err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("directResp, err := ", transportExpr, ".", method.MethodGoName, "(ctx, directReq)")
	g.P("if err != nil { return ", method.NativeErrZero, " }")
	g.P("messageResp, err = proto.Marshal(directResp)")
}

func renderRuntimeTransportMessageStreamRecord(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, codecEnabled bool, transportExpr, label string) {
	nativeSession := runtimeFinalNativeSessionName(service.GoName, method)
	messageSession := runtimeFinalMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "req")
	} else {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		if codecEnabled {
			g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
			g.P("if err != nil { return nil, err }")
			renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "messageReq")
			g.P("if err != nil { return nil, err }")
			renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	} else {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		if codecEnabled {
			renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "")
			g.P("if err != nil { return nil, err }")
			renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	}
	g.P("}")
}

func renderRuntimeTransportMessageStreamSource(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr, label, ctxExpr, reqExpr string) {
	constructor := ""
	switch label {
	case "connect handler":
		constructor = "new" + connectDirectMessageSessionName(service.GoName, method)
	case "connect remote":
		constructor = "new" + connectRemoteMessageSessionName(service.GoName, method)
	case "grpc server":
		constructor = "new" + grpcDirectMessageSessionName(service.GoName, method)
	case "grpc remote":
		constructor = "new" + grpcRemoteMessageSessionName(service.GoName, method)
	default:
		panic("unknown transport message source")
	}
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("source, err := ", constructor, "(", ctxExpr, ", ", transportExpr, ", ", reqExpr, ")")
		return
	}
	if label == "connect handler" || label == "grpc server" {
		g.P("source := ", constructor, "(", ctxExpr, ", ", transportExpr, ")")
		g.P("var err error")
		return
	}
	g.P("source, err := ", constructor, "(", ctxExpr, ", ", transportExpr, ")")
}
