package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeMessageBinding(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, activeBindingSlotName, messageActiveBindingName, adapterExpr string) {
	g.P("messageBinding := &", messageActiveBindingName, "{}")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("messageBinding.invoke", method.Identity.GoName, " = ", adapterExpr, ".", method.Identity.MessageMethodRef)
			continue
		}
		renderRuntimeMessageStreamBinding(g, service, method, adapterExpr)
	}
	g.P(activeBindingSlotName, ".Store(messageBinding)")
	g.P("return nil")
}

func renderRuntimeMessageStreamBinding(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, adapterExpr string) {
	nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
	messageSession := runtimeStreamMessageSessionName(service.GoName, method)
	if method.Stream.StartAcceptsRequest {
		g.P("messageBinding.start", method.Identity.GoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.Identity.MessageMethodRef, "(ctx, req)")
	} else {
		g.P("messageBinding.start", method.Identity.GoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.Identity.MessageMethodRef, "(ctx)")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	_ = nativeSession
}

func renderRuntimeMessageFinalSessionFromSource(g *protogen.GeneratedFile, sessionName string, method runtimeMethodProjection, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.Stream.CanSend {
		g.P("send: ", sourceExpr, ".Send,")
	}
	if method.Stream.CanRecv {
		g.P("recv: ", sourceExpr, ".Recv,")
	}
	if method.Stream.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	g.P("finish: ", sourceExpr, ".Finish,")
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}

func renderRuntimeNativeFinalSessionFromMessageSource(g *protogen.GeneratedFile, service ServicePlan, sessionName string, method runtimeMethodProjection, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.Stream.CanSend {
		g.P("send: func(ctx context.Context", method.Native.Args, ") error {")
		g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
		g.P("if err != nil { return err }")
		g.P("return ", sourceExpr, ".Send(ctx, messageReq)")
		g.P("},")
	}
	if method.Stream.CanRecv {
		g.P("recv: func(ctx context.Context) (", method.Native.Returns, ") {")
		g.P("messageResp, err := ", sourceExpr, ".Recv(ctx)")
		g.P("if err != nil { return ", method.Native.ErrZero, " }")
		g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
		g.P("},")
	}
	if method.Stream.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	if method.Stream.FinishReturnsResponse {
		g.P("finish: func(ctx context.Context) (", method.Native.Returns, ") {")
		g.P("messageResp, err := ", sourceExpr, ".Finish(ctx)")
		g.P("if err != nil { return ", method.Native.ErrZero, " }")
		g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
		g.P("},")
	} else {
		g.P("finish: ", sourceExpr, ".Finish,")
	}
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}
