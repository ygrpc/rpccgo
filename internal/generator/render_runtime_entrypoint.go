package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeEntrypoints(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, methods []runtimeMethodProjection) error {
	for _, method := range methods {
		if method.Stream.Streaming {
			continue
		}
		if service.Generation.NativeEnabled {
			renderRuntimeUnaryNativeEntrypoint(g, service, serviceIDName, method)
		}
		renderRuntimeUnaryMessageEntrypoint(g, service, serviceIDName, method)
	}
	for _, method := range methods {
		if !method.Stream.Streaming {
			continue
		}
		if service.Generation.NativeEnabled {
			renderRuntimeNativeStartEntrypoint(g, service, serviceIDName, method)
		}
		if err := renderRuntimeMessageStartEntrypoint(g, service, serviceIDName, method); err != nil {
			return err
		}
	}
	return nil
}

func renderRuntimeUnaryNativeEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, method runtimeMethodProjection) {
	name := "Invoke" + service.GoName + "Native" + method.Identity.GoName
	renderDoc(g, name, "invokes the current registered server using the native contract for "+method.Identity.GoName+".")
	g.P("func Invoke", service.GoName, "Native", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ") {")
	g.P("registered, err := rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("switch registered.Kind {")
	if service.Generation.NativeEnabled {
		renderRuntimeUnaryNativeToNativeCase(g, service, method, "rpcruntime.ServerKindGoNative", "go native")
		renderRuntimeUnaryNativeToNativeCase(g, service, method, "rpcruntime.ServerKindCGONative", "cgo native")
	}
	renderRuntimeUnaryNativeToCGOMessageCase(g, service, method)
	renderRuntimeUnaryNativeToTransportCases(g, service, method)
	g.P("default:")
	g.P("return ", method.Native.ErrZero)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeUnaryNativeToNativeCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", service.GoName, "NativeServer)")
	g.P("if !ok { return ", nativeGoZeroReturnsForError(method, "fmt.Errorf(\"rpccgo: "+service.GoName+" "+label+" registered server has invalid type\")"), " }")
	g.P("return server.", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
}

func renderRuntimeUnaryNativeToCGOMessageCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := registered.Server.(", service.GoName, "CGOMessageServer)")
	g.P("if !ok { return ", nativeGoZeroReturnsForError(method, "fmt.Errorf(\"rpccgo: "+service.GoName+" cgo message registered server has invalid type\")"), " }")
	g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("messageResp, err := server.", method.Identity.MessageMethodRef, "(ctx, messageReq)")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
}

func renderRuntimeUnaryNativeToTransportCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	switch service.Generation.MessageTransport {
	case MessageTransportConnect:
		renderRuntimeUnaryNativeToTransportCase(g, service, method, "rpcruntime.ServerKindConnect", service.GoName+"Handler", "connect handler")
		renderRuntimeUnaryNativeToTransportCase(g, service, method, "rpcruntime.ServerKindConnectRemote", service.GoName+"Client", "connect remote")
	case MessageTransportGRPC:
		renderRuntimeUnaryNativeToTransportCase(g, service, method, "rpcruntime.ServerKindGRPC", service.GoName+"Server", "grpc server")
		renderRuntimeUnaryNativeToTransportCase(g, service, method, "rpcruntime.ServerKindGRPCRemote", service.GoName+"Client", "grpc remote")
	}
}

func renderRuntimeUnaryNativeToTransportCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, serverType, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", serverType, ")")
	g.P("if !ok { return ", nativeGoZeroReturnsForError(method, "fmt.Errorf(\"rpccgo: "+service.GoName+" "+label+" registered server has invalid type\")"), " }")
	g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("var messageResp ", runtimeMessageResponseType(method))
	renderRuntimeTransportUnaryNativeMessageCall(g, service, method, "server", label, "messageReq")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
}

func renderRuntimeUnaryMessageEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, method runtimeMethodProjection) {
	name := "Invoke" + service.GoName + "Message" + method.Identity.GoName
	renderDoc(g, name, "invokes the current registered server using the message contract for "+method.Identity.GoName+".")
	g.P("func Invoke", service.GoName, "Message", method.Identity.GoName, "(ctx context.Context, req ", runtimeMessageRequestType(method), ") (", runtimeMessageResponseType(method), ", error) {")
	g.P("if req == nil {")
	g.P(`return nil, errors.New("rpccgo: message request is nil")`)
	g.P("}")
	g.P("registered, err := rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("if err != nil { return nil, err }")
	g.P("switch registered.Kind {")
	if service.Generation.NativeEnabled {
		renderRuntimeUnaryMessageToNativeCase(g, service, method, "rpcruntime.ServerKindGoNative", "go native")
		renderRuntimeUnaryMessageToNativeCase(g, service, method, "rpcruntime.ServerKindCGONative", "cgo native")
	}
	renderRuntimeUnaryMessageToCGOMessageCase(g, service, method)
	renderRuntimeUnaryMessageToTransportCases(g, service, method)
	g.P("default:")
	g.P(`return nil, fmt.Errorf("rpccgo: `, service.GoName, ` registered server kind %d is unsupported for message calls", registered.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeUnaryMessageToNativeCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", service.GoName, "NativeServer)")
	g.P(`if !ok { return nil, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	g.P(method.Codec.MessageToNativeRequestAssignNames, " := ", method.Codec.MessageToNativeRequest, "(req)")
	g.P("if err != nil { return nil, err }")
	if method.Native.ResultNames == "" {
		g.P("err = server.", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
		g.P("goruntime.KeepAlive(reqOwner)")
		g.P("if err != nil { return nil, err }")
		g.P("return ", method.Codec.NativeResponseToMessage, "()")
	} else {
		g.P(method.Native.ResultNames, ", err := server.", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
		g.P("goruntime.KeepAlive(reqOwner)")
		g.P("if err != nil { return nil, err }")
		g.P("return ", method.Codec.NativeResponseToMessage, "(", method.Native.ResultNames, ")")
	}
}

func renderRuntimeUnaryMessageToCGOMessageCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := registered.Server.(", service.GoName, "CGOMessageServer)")
	g.P(`if !ok { return nil, fmt.Errorf("rpccgo: `, service.GoName, ` cgo message registered server has invalid type") }`)
	g.P("resp, err := server.", method.Identity.MessageMethodRef, "(ctx, req)")
	g.P("if err != nil { return nil, err }")
	g.P("if resp == nil {")
	g.P(`return nil, errors.New("rpccgo: message response is nil")`)
	g.P("}")
	g.P("return resp, nil")
}

func renderRuntimeUnaryMessageToTransportCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	switch service.Generation.MessageTransport {
	case MessageTransportConnect:
		renderRuntimeUnaryMessageToTransportCase(g, service, method, "rpcruntime.ServerKindConnect", service.GoName+"Handler", "connect handler")
		renderRuntimeUnaryMessageToTransportCase(g, service, method, "rpcruntime.ServerKindConnectRemote", service.GoName+"Client", "connect remote")
	case MessageTransportGRPC:
		renderRuntimeUnaryMessageToTransportCase(g, service, method, "rpcruntime.ServerKindGRPC", service.GoName+"Server", "grpc server")
		renderRuntimeUnaryMessageToTransportCase(g, service, method, "rpcruntime.ServerKindGRPCRemote", service.GoName+"Client", "grpc remote")
	}
}

func renderRuntimeUnaryMessageToTransportCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, serverType, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", serverType, ")")
	g.P(`if !ok { return nil, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	renderRuntimeTransportUnaryMessageCall(g, service, method, "server", label, "req")
}

func renderRuntimeNativeStartEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, method runtimeMethodProjection) {
	name := runtimeStreamOperationName(service.GoName, "Native", method, "Start")
	renderDoc(g, name, "starts a native contract stream for "+method.Identity.GoName+" on the current registered server.")
	if method.Stream.StartAcceptsRequest {
		g.P("func ", name, "(ctx context.Context", method.Native.Args, ") (rpcruntime.StreamHandle, error) {")
	} else {
		g.P("func ", name, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	}
	g.P("registered, err := rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("if err != nil { return 0, err }")
	g.P("switch registered.Kind {")
	if service.Generation.NativeEnabled {
		renderRuntimeNativeStartNativeCase(g, service, method, "rpcruntime.ServerKindGoNative", "go native")
		renderRuntimeNativeStartNativeCase(g, service, method, "rpcruntime.ServerKindCGONative", "cgo native")
	}
	renderRuntimeNativeStartMessageCase(g, service, method)
	renderRuntimeNativeStartTransportCases(g, service, method)
	g.P("default:")
	g.P(`return 0, fmt.Errorf("rpccgo: `, service.GoName, ` registered server kind %d is unsupported for native stream starts", registered.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStartNativeCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", service.GoName, "NativeServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	if method.Stream.StartAcceptsRequest {
		renderRuntimeStartSourceWithArgs(g, goNativeStartHelperName(service.GoName, method.Identity.GoName), runtimeStartArgs("ctx, server", method.Native.ArgNames), true)
	} else {
		renderRuntimeStartSourceWithArgs(g, goNativeStartHelperName(service.GoName, method.Identity.GoName), "ctx, server", true)
	}
	renderRuntimeCreateNativeStreamHandle(g, kind, "source")
}

func renderRuntimeNativeStartMessageCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := registered.Server.(", service.GoName, "CGOMessageServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, ` cgo message registered server has invalid type") }`)
	if method.Stream.StartAcceptsRequest {
		g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
		g.P("if err != nil { return 0, err }")
		renderRuntimeStartSourceWithArgs(g, cgoMessageStartHelperName(service.GoName, method.Identity.GoName), "ctx, server, messageReq", true)
	} else {
		renderRuntimeStartSourceWithArgs(g, cgoMessageStartHelperName(service.GoName, method.Identity.GoName), "ctx, server", true)
	}
	renderRuntimeCreateNativeStreamHandle(g, "rpcruntime.ServerKindCGOMessage", "source")
}

func renderRuntimeNativeStartTransportCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	switch service.Generation.MessageTransport {
	case MessageTransportConnect:
		renderRuntimeNativeStartTransportCase(g, service, method, "rpcruntime.ServerKindConnect", service.GoName+"Handler", "connect handler")
		renderRuntimeNativeStartTransportCase(g, service, method, "rpcruntime.ServerKindConnectRemote", service.GoName+"Client", "connect remote")
	case MessageTransportGRPC:
		renderRuntimeNativeStartTransportCase(g, service, method, "rpcruntime.ServerKindGRPC", service.GoName+"Server", "grpc server")
		renderRuntimeNativeStartTransportCase(g, service, method, "rpcruntime.ServerKindGRPCRemote", service.GoName+"Client", "grpc remote")
	}
}

func renderRuntimeNativeStartTransportCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, serverType, label string) {
	projection, err := transportProjectionForKind(service, kind)
	if err != nil {
		g.P("case ", kind, ":")
		g.P("return 0, ", err.Error())
		return
	}
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", serverType, ")")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	if method.Stream.StartAcceptsRequest {
		g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
		g.P("if err != nil { return 0, err }")
		renderRuntimeTransportStreamSource(g, service, method, "server", projection, "ctx", "messageReq")
	} else {
		renderRuntimeTransportStreamSource(g, service, method, "server", projection, "ctx", "")
	}
	renderRuntimeCreateNativeStreamHandle(g, kind, "source")
}

func renderRuntimeMessageStartEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, method runtimeMethodProjection) error {
	name := runtimeStreamOperationName(service.GoName, "Message", method, "Start")
	renderDoc(g, name, "starts a message contract stream for "+method.Identity.GoName+" on the current registered server.")
	if method.Stream.StartAcceptsRequest {
		g.P("func ", name, "(ctx context.Context, req ", runtimeMessageRequestType(method), ") (rpcruntime.StreamHandle, error) {")
		g.P("if req == nil {")
		g.P(`return 0, errors.New("rpccgo: message request is nil")`)
		g.P("}")
	} else {
		g.P("func ", name, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	}
	g.P("registered, err := rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("if err != nil { return 0, err }")
	g.P("switch registered.Kind {")
	if service.Generation.NativeEnabled {
		renderRuntimeMessageStartNativeCase(g, service, method, "rpcruntime.ServerKindGoNative", "go native")
		renderRuntimeMessageStartNativeCase(g, service, method, "rpcruntime.ServerKindCGONative", "cgo native")
	}
	renderRuntimeMessageStartMessageCase(g, service, method)
	renderRuntimeMessageStartTransportCases(g, service, method)
	g.P("default:")
	g.P(`return 0, fmt.Errorf("rpccgo: `, service.GoName, ` registered server kind %d is unsupported for message stream starts", registered.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
	return nil
}

func renderRuntimeMessageStartNativeCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", service.GoName, "NativeServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	if method.Stream.StartAcceptsRequest {
		g.P(method.Codec.MessageToNativeRequestAssignNames, " := ", method.Codec.MessageToNativeRequest, "(req)")
		g.P("if err != nil { return 0, err }")
		renderRuntimeStartSourceWithArgs(g, goNativeStartHelperName(service.GoName, method.Identity.GoName), runtimeStartArgs("ctx, server", method.Native.ArgNames), true)
		g.P("goruntime.KeepAlive(reqOwner)")
	} else {
		renderRuntimeStartSourceWithArgs(g, goNativeStartHelperName(service.GoName, method.Identity.GoName), "ctx, server", true)
	}
	renderRuntimeCreateMessageStreamHandle(g, kind, "source")
}

func renderRuntimeMessageStartMessageCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := registered.Server.(", service.GoName, "CGOMessageServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, ` cgo message registered server has invalid type") }`)
	if method.Stream.StartAcceptsRequest {
		renderRuntimeStartSourceWithArgs(g, cgoMessageStartHelperName(service.GoName, method.Identity.GoName), "ctx, server, req", true)
	} else {
		renderRuntimeStartSourceWithArgs(g, cgoMessageStartHelperName(service.GoName, method.Identity.GoName), "ctx, server", true)
	}
	renderRuntimeCreateMessageStreamHandle(g, "rpcruntime.ServerKindCGOMessage", "source")
}

func renderRuntimeMessageStartTransportCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection) {
	switch service.Generation.MessageTransport {
	case MessageTransportConnect:
		renderRuntimeMessageStartTransportCase(g, service, method, "rpcruntime.ServerKindConnect", service.GoName+"Handler", "connect handler")
		renderRuntimeMessageStartTransportCase(g, service, method, "rpcruntime.ServerKindConnectRemote", service.GoName+"Client", "connect remote")
	case MessageTransportGRPC:
		renderRuntimeMessageStartTransportCase(g, service, method, "rpcruntime.ServerKindGRPC", service.GoName+"Server", "grpc server")
		renderRuntimeMessageStartTransportCase(g, service, method, "rpcruntime.ServerKindGRPCRemote", service.GoName+"Client", "grpc remote")
	}
}

func renderRuntimeMessageStartTransportCase(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, kind, serverType, label string) {
	projection, err := transportProjectionForKind(service, kind)
	if err != nil {
		g.P("case ", kind, ":")
		g.P("return 0, ", err.Error())
		return
	}
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", serverType, ")")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	if method.Stream.StartAcceptsRequest {
		renderRuntimeTransportStreamSource(g, service, method, "server", projection, "ctx", "req")
	} else {
		renderRuntimeTransportStreamSource(g, service, method, "server", projection, "ctx", "")
	}
	renderRuntimeCreateMessageStreamHandle(g, kind, "source")
}

func runtimeStartArgs(prefix, nativeArgNames string) string {
	if nativeArgNames == "" {
		return prefix
	}
	return prefix + ", " + nativeArgNames
}

func renderRuntimeStartSourceWithArgs(g *protogen.GeneratedFile, startExpr, args string, withErr bool) {
	if withErr {
		g.P("source, err := ", startExpr, "(", args, ")")
		g.P("if err != nil { return 0, err }")
		return
	}
	g.P("source := ", startExpr, "(", args, ")")
}

func renderRuntimeTransportStreamSource(g *protogen.GeneratedFile, service ServicePlan, method runtimeMethodProjection, transportExpr string, projection registrationSourceProjection, ctxExpr, reqExpr string) {
	hasErr, err := renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, projection, ctxExpr, reqExpr)
	if err != nil {
		g.P(`return 0, fmt.Errorf("rpccgo: `, projection.label, ` stream constructor is unavailable")`)
		return
	}
	if hasErr {
		g.P("if err != nil { return 0, err }")
	}
}

func renderRuntimeCreateNativeStreamHandle(g *protogen.GeneratedFile, kind, sourceExpr string) {
	g.P("return rpcruntime.CreateStreamSession(", kind, ", ", sourceExpr, ")")
}

func renderRuntimeCreateMessageStreamHandle(g *protogen.GeneratedFile, kind, sourceExpr string) {
	g.P("return rpcruntime.CreateStreamSession(", kind, ", ", sourceExpr, ")")
}

func nativeGoZeroReturnsForError(method runtimeMethodProjection, errExpr string) string {
	if method.Native.ErrZero == "err" {
		return errExpr
	}
	return strings.TrimSuffix(method.Native.ErrZero, "err") + errExpr
}

func transportProjectionForKind(service ServicePlan, kind string) (registrationSourceProjection, error) {
	for _, source := range registrationSourcesForService(service) {
		projection, err := ProjectRegistrationSource(service, source)
		if err != nil {
			return registrationSourceProjection{}, err
		}
		if projection.serverKind == kind {
			return projection, nil
		}
	}
	return registrationSourceProjection{}, errUnknownTransportKind
}

var errUnknownTransportKind = generatorStringError("unknown transport server kind")

type generatorStringError string

func (e generatorStringError) Error() string { return string(e) }
