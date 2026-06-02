package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeNativeRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, activeName, recordName, adapterExpr string) {
	g.P("record := &", recordName, "{}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("record.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			g.P("return ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			g.P("}")
			g.P("record.invokeMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) ([]byte, error) {")
			g.P("var resp []byte")
			g.P("err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
			if method.NativeNames == "" {
				g.P("callErr := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			} else {
				g.P(method.NativeNames, ", callErr := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			}
			g.P("if callErr != nil { return callErr }")
			g.P("messageResp, err := ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
			g.P("if err != nil { return err }")
			g.P("resp = messageResp")
			g.P("return nil")
			g.P("})")
			g.P("if err != nil { return nil, err }")
			g.P("return resp, nil")
			g.P("}")
			continue
		}
		renderRuntimeNativeStreamRecord(g, service, method, adapterExpr)
	}
	g.P(activeName, ".Store(record)")
	g.P("return nil")
}

func renderRuntimeNativeStreamRecord(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, adapterExpr string) {
	nativeSession := runtimeFinalNativeSessionName(service.GoName, method)
	messageSession := runtimeFinalMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	} else {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx)")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeNativeFinalSessionFromSource(g, nativeSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		g.P("var final *", messageSession)
		g.P("err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("if err != nil { return err }")
		renderRuntimeMessageFinalSessionFromNativeSource(g, service, messageSession, method, "source", true)
		g.P("return nil")
		g.P("})")
		g.P("if err != nil { return nil, err }")
		g.P("return final, nil")
	} else {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx)")
		g.P("if err != nil { return nil, err }")
		renderRuntimeMessageFinalSessionFromNativeSource(g, service, messageSession, method, "source", false)
	}
	g.P("}")
}

func renderRuntimeNativeFinalSessionFromSource(g *protogen.GeneratedFile, sessionName string, method runtimeAdapterMethod, sourceExpr string) {
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

func renderRuntimeMessageFinalSessionFromNativeSource(g *protogen.GeneratedFile, service ServicePlan, sessionName string, method runtimeAdapterMethod, sourceExpr string, assign bool) {
	target := "return "
	if assign {
		target = "final = "
	}
	g.P(target, "&", sessionName, "{")
	if method.CanSend {
		g.P("send: func(ctx context.Context, req []byte) error {")
		g.P("return ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
		g.P("return ", sourceExpr, ".Send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("})")
		g.P("},")
	}
	if method.CanRecv {
		g.P("recv: func(ctx context.Context) ([]byte, error) {")
		if method.NativeNames == "" {
			g.P("err := ", sourceExpr, ".Recv(ctx)")
		} else {
			g.P(method.NativeNames, ", err := ", sourceExpr, ".Recv(ctx)")
		}
		g.P("if err != nil { return nil, err }")
		g.P("return ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
		g.P("},")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	if method.FinishReturnsResponse {
		g.P("finish: func(ctx context.Context) ([]byte, error) {")
		if method.NativeNames == "" {
			g.P("err := ", sourceExpr, ".Finish(ctx)")
		} else {
			g.P(method.NativeNames, ", err := ", sourceExpr, ".Finish(ctx)")
		}
		g.P("if err != nil { return nil, err }")
		g.P("return ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
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
