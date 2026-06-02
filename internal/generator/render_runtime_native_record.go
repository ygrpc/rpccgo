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
			g.P("reqView, err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req)")
			g.P("if err != nil { return nil, err }")
			if method.NativeNames == "" {
				g.P("callErr := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeRequestViewCallSuffix(service, method), ")")
			} else {
				g.P(method.NativeNames, ", callErr := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeRequestViewCallSuffix(service, method), ")")
			}
			g.P("goruntime.KeepAlive(reqView)")
			g.P("if callErr != nil { return nil, callErr }")
			g.P("messageResp, err := ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
			g.P("if err != nil { return nil, err }")
			g.P("return messageResp, nil")
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
		g.P("reqView, err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req)")
		g.P("if err != nil { return nil, err }")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeRequestViewCallSuffix(service, method), ")")
		g.P("goruntime.KeepAlive(reqView)")
		g.P("if err != nil { return nil, err }")
		renderRuntimeMessageFinalSessionFromNativeSource(g, service, messageSession, method, "source", false)
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
		g.P("reqView, err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req)")
		g.P("if err != nil { return err }")
		g.P("err = ", sourceExpr, ".Send(ctx", nativeRequestViewCallSuffix(service, method), ")")
		g.P("goruntime.KeepAlive(reqView)")
		g.P("return err")
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

func nativeRequestViewCallSuffix(service ServicePlan, method runtimeAdapterMethod) string {
	methodPlan := methodForRuntimeService(service, method)
	if len(methodPlan.Contract.Native.RequestFields) == 0 {
		return ""
	}
	args := make([]string, 0, len(methodPlan.Contract.Native.RequestFields))
	for _, field := range methodPlan.Contract.Native.RequestFields {
		args = append(args, "reqView."+lowerInitial(field.GoName))
	}
	return ", " + strings.Join(args, ", ")
}
