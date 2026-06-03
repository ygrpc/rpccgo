package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeMessageBinding(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, currentBindingName, bindingName, adapterExpr string) {
	g.P("binding := &", bindingName, "{}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("binding.invokeMessage", method.MethodGoName, " = ", adapterExpr, ".", method.MethodGoName)
			g.P("binding.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
			g.P("if err != nil { return ", method.NativeErrZero, " }")
			g.P("messageResp, err := ", adapterExpr, ".", method.MethodGoName, "(ctx, messageReq)")
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
			g.P("}")
			continue
		}
		renderRuntimeMessageStreamBinding(g, service, method, adapterExpr)
	}
	g.P(currentBindingName, ".Store(binding)")
	g.P("return nil")
}

func renderRuntimeMessageStreamBinding(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, adapterExpr string) {
	nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
	messageSession := runtimeStreamMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("binding.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx, req)")
	} else {
		g.P("binding.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx)")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("binding.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
		g.P("if err != nil { return nil, err }")
		g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx, messageReq)")
		g.P("if err != nil { return nil, err }")
		renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
	} else {
		g.P("binding.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx)")
		g.P("if err != nil { return nil, err }")
		renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
	}
	g.P("}")
}

func renderRuntimeMessageFinalSessionFromSource(g *protogen.GeneratedFile, sessionName string, method runtimeAdapterMethod, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.CanSend {
		g.P("send: ", sourceExpr, ".Send,")
	}
	if method.CanRecv {
		g.P("recv: ", sourceExpr, ".Recv,")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	g.P("finish: ", sourceExpr, ".Finish,")
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}

func renderRuntimeNativeFinalSessionFromMessageSource(g *protogen.GeneratedFile, service ServicePlan, sessionName string, method runtimeAdapterMethod, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.CanSend {
		g.P("send: func(ctx context.Context", method.NativeArgs, ") error {")
		g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
		g.P("if err != nil { return err }")
		g.P("return ", sourceExpr, ".Send(ctx, messageReq)")
		g.P("},")
	}
	if method.CanRecv {
		g.P("recv: func(ctx context.Context) (", method.NativeReturns, ") {")
		g.P("messageResp, err := ", sourceExpr, ".Recv(ctx)")
		g.P("if err != nil { return ", method.NativeErrZero, " }")
		g.P("return ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
		g.P("},")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	if method.FinishReturnsResponse {
		g.P("finish: func(ctx context.Context) (", method.NativeReturns, ") {")
		g.P("messageResp, err := ", sourceExpr, ".Finish(ctx)")
		g.P("if err != nil { return ", method.NativeErrZero, " }")
		g.P("return ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
		g.P("},")
	} else {
		g.P("finish: ", sourceExpr, ".Finish,")
	}
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}
