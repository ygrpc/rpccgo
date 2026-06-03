package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeTransportMessageBinding(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, currentBindingName, bindingName, transportExpr string, projection registrationSourceProjection) error {
	g.P("binding := &", bindingName, "{")
	g.P("}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("binding.invokeMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) ([]byte, error) {")
			renderRuntimeTransportUnaryMessageCall(g, service, method, transportExpr, projection.label, "req")
			g.P("}")
			g.P("binding.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
			g.P("if err != nil { return ", method.NativeErrZero, " }")
			g.P("var messageResp []byte")
			renderRuntimeTransportUnaryNativeMessageCall(g, service, method, transportExpr, projection.label, "messageReq")
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
		if err := renderRuntimeTransportMessageStreamBinding(g, service, method, transportExpr, projection); err != nil {
			return err
		}
	}
	g.P(currentBindingName, ".Store(binding)")
	g.P("return nil")
	return nil
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

func renderRuntimeTransportMessageStreamBinding(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr string, projection registrationSourceProjection) error {
	nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
	messageSession := runtimeStreamMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("binding.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, "ctx", "req")
		if err != nil {
			return err
		}
		renderRuntimeTransportMessageStreamSourceErrCheck(g, hasErr)
	} else {
		g.P("binding.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, "ctx", "")
		if err != nil {
			return err
		}
		renderRuntimeTransportMessageStreamSourceErrCheck(g, hasErr)
	}
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("binding.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
		g.P("if err != nil { return nil, err }")
		hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, "ctx", "messageReq")
		if err != nil {
			return err
		}
		renderRuntimeTransportMessageStreamSourceErrCheck(g, hasErr)
		renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
	} else {
		g.P("binding.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, "ctx", "")
		if err != nil {
			return err
		}
		renderRuntimeTransportMessageStreamSourceErrCheck(g, hasErr)
		renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
	}
	g.P("}")
	return nil
}

func renderRuntimeTransportMessageStreamSource(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr string, projection registrationSourceProjection, ctxExpr, reqExpr string) (bool, error) {
	constructor, hasErr, err := registrationTransportMessageStreamConstructor(service, method, projection)
	if err != nil {
		return false, err
	}
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("source, err := ", constructor, "(", ctxExpr, ", ", transportExpr, ", ", reqExpr, ")")
		return hasErr, nil
	}
	if hasErr {
		g.P("source, err := ", constructor, "(", ctxExpr, ", ", transportExpr, ")")
		return true, nil
	}
	g.P("source := ", constructor, "(", ctxExpr, ", ", transportExpr, ")")
	return false, nil
}

func renderRuntimeTransportMessageStreamSourceErrCheck(g *protogen.GeneratedFile, hasErr bool) {
	if hasErr {
		g.P("if err != nil { return nil, err }")
	}
}
