package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeTransportMessageBinding(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, activeBindingSlotName, messageActiveBindingName, transportExpr string, projection registrationSourceProjection) error {
	g.P("messageBinding := &", messageActiveBindingName, "{}")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("messageBinding.invoke", method.Identity.GoName, " = func(ctx context.Context, req []byte) ([]byte, error) {")
			renderRuntimeTransportUnaryMessageCall(g, service, method, transportExpr, projection.label, "req")
			g.P("}")
			continue
		}
		if err := renderRuntimeTransportMessageStreamBinding(g, service, method, transportExpr, projection); err != nil {
			return err
		}
	}
	g.P(activeBindingSlotName, ".Store(messageBinding)")
	g.P("return nil")
	return nil
}

func renderRuntimeTransportUnaryMessageCall(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, transportExpr, label, reqExpr string) {
	reqType := method.Message.RequestType
	g.P("messageReq := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: ", label, " request protobuf unmarshal failed: %w\", err)")
	g.P("}")
	g.P("messageResp, err := ", transportExpr, ".", method.Identity.MessageMethodRef, "(ctx, messageReq)")
	g.P("if err != nil { return nil, err }")
	g.P("resp, err := proto.Marshal(messageResp)")
	g.P("if err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: ", label, " response protobuf marshal failed: %w\", err)")
	g.P("}")
	g.P("return resp, nil")
}

func renderRuntimeTransportUnaryNativeMessageCall(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, transportExpr, label, reqExpr string) {
	reqType := method.Message.RequestType
	g.P("directReq := new(", reqType, ")")
	g.P("if err = proto.Unmarshal(", reqExpr, ", directReq); err != nil {")
	g.P("return ", method.Native.ErrZero)
	g.P("}")
	g.P("directResp, err := ", transportExpr, ".", method.Identity.MessageMethodRef, "(ctx, directReq)")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("messageResp, err = proto.Marshal(directResp)")
}

func renderRuntimeTransportMessageStreamBinding(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, transportExpr string, projection registrationSourceProjection) error {
	nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
	messageSession := runtimeStreamMessageSessionName(service.GoName, method)
	if method.Stream.StartAcceptsRequest {
		g.P("messageBinding.start", method.Identity.GoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, "ctx", "req")
		if err != nil {
			return err
		}
		renderRuntimeTransportMessageStreamSourceErrCheck(g, hasErr)
	} else {
		g.P("messageBinding.start", method.Identity.GoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, "ctx", "")
		if err != nil {
			return err
		}
		renderRuntimeTransportMessageStreamSourceErrCheck(g, hasErr)
	}
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	_ = nativeSession
	return nil
}

func renderRuntimeTransportMessageStreamSource(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, transportExpr string, projection registrationSourceProjection, ctxExpr, reqExpr string) (bool, error) {
	constructor, hasErr, err := registrationTransportMessageStreamConstructor(service, method, projection)
	if err != nil {
		return false, err
	}
	if method.Stream.StartAcceptsRequest {
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
