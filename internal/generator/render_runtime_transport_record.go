package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeTransportUnaryMessageCall(g *protogen.GeneratedFile, method runtimeMethodProjection, transportExpr, reqExpr string) {
	g.P("messageResp, err := ", transportExpr, ".", method.Identity.MessageMethodRef, "(ctx, ", reqExpr, ")")
	g.P("if err != nil { return nil, err }")
	g.P("if messageResp == nil {")
	g.P(`return nil, errors.New("rpccgo: message response is nil")`)
	g.P("}")
	g.P("return messageResp, nil")
}

func renderRuntimeTransportUnaryNativeMessageCall(g *protogen.GeneratedFile, method runtimeMethodProjection, transportExpr, reqExpr string) {
	g.P("messageResp, err = ", transportExpr, ".", method.Identity.MessageMethodRef, "(ctx, ", reqExpr, ")")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("if messageResp == nil {")
	g.P(`err = errors.New("rpccgo: message response is nil")`)
	g.P("return ", method.Native.ErrZero)
	g.P("}")
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
