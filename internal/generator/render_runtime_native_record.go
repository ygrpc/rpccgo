package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeNativeBinding(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, activeBindingSlotName, nativeActiveBindingName, adapterExpr string) {
	g.P("nativeBinding := &", nativeActiveBindingName, "{}")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("nativeBinding.invoke", method.Identity.GoName, " = func(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ") {")
			g.P("return ", adapterExpr, ".", method.Symbols.NativeAdapterMethod, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
			g.P("}")
			continue
		}
		renderRuntimeNativeStreamBinding(g, service, method, adapterExpr)
	}
	g.P(activeBindingSlotName, ".Store(nativeBinding)")
	g.P("return nil")
}

func renderRuntimeNativeStreamBinding(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, adapterExpr string) {
	nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
	messageSession := runtimeStreamMessageSessionName(service.GoName, method)
	if method.Stream.StartAcceptsRequest {
		g.P("nativeBinding.start", method.Identity.GoName, " = func(ctx context.Context", method.Native.Args, ") (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.Symbols.NativeAdapterMethod, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
	} else {
		g.P("nativeBinding.start", method.Identity.GoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.Symbols.NativeAdapterMethod, "(ctx)")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeNativeFinalSessionFromSource(g, nativeSession, method, "source")
	g.P("}")
	_ = messageSession
}

func renderRuntimeNativeFinalSessionFromSource(g *protogen.GeneratedFile, sessionName string, method runtimeMethodProjection, sourceExpr string) {
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

func renderRuntimeMessageFinalSessionFromNativeSource(g *protogen.GeneratedFile, service ServicePlan, sessionName string, method runtimeMethodProjection, sourceExpr string, assign bool) {
	target := "return "
	if assign {
		target = "final = "
	}
	g.P(target, "&", sessionName, "{")
	if method.Stream.CanSend {
		g.P("send: func(ctx context.Context, req []byte) error {")
		g.P(method.Codec.MessageToNativeRequestAssignNames, " := ", method.Codec.MessageToNativeRequest, "(req)")
		g.P("if err != nil { return err }")
		g.P("err = ", sourceExpr, ".Send(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
		g.P("goruntime.KeepAlive(reqOwner)")
		g.P("return err")
		g.P("},")
	}
	if method.Stream.CanRecv {
		g.P("recv: func(ctx context.Context) ([]byte, error) {")
		if method.Native.ResultNames == "" {
			g.P("err := ", sourceExpr, ".Recv(ctx)")
		} else {
			g.P(method.Native.ResultNames, ", err := ", sourceExpr, ".Recv(ctx)")
		}
		g.P("if err != nil { return nil, err }")
		g.P("return ", method.Codec.NativeResponseToMessage, "(", method.Native.ResultNames, ")")
		g.P("},")
	}
	if method.Stream.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	if method.Stream.FinishReturnsResponse {
		g.P("finish: func(ctx context.Context) ([]byte, error) {")
		if method.Native.ResultNames == "" {
			g.P("err := ", sourceExpr, ".Finish(ctx)")
		} else {
			g.P(method.Native.ResultNames, ", err := ", sourceExpr, ".Finish(ctx)")
		}
		g.P("if err != nil { return nil, err }")
		g.P("return ", method.Codec.NativeResponseToMessage, "(", method.Native.ResultNames, ")")
		g.P("},")
	} else {
		g.P("finish: ", sourceExpr, ".Finish,")
	}
	g.P("cancel: ", sourceExpr, ".Cancel,")
	if assign {
		g.P("}")
	} else {
		g.P("}, nil")
	}
}

func codecMessageToNativeRequestAssignNames(fields []FieldPlan, ownerName, errName string) string {
	names := make([]string, 0, len(fields)+2)
	for _, field := range fields {
		names = append(names, lowerInitial(field.GoName))
	}
	names = append(names, ownerName, errName)
	return strings.Join(names, ", ")
}
