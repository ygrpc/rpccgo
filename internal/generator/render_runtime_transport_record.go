package generator

import "google.golang.org/protobuf/compiler/protogen"

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
