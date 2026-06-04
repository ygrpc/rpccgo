package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeEntrypoints(g *protogen.GeneratedFile, service ServicePlan, serviceIDName, streamRegistryName string, methods []runtimeMethodProjection) error {
	for _, method := range methods {
		if method.Stream.Streaming {
			continue
		}
		renderRuntimeUnaryNativeEntrypoint(g, service, serviceIDName, method)
		renderRuntimeUnaryMessageEntrypoint(g, service, serviceIDName, method)
	}
	for _, method := range methods {
		if !method.Stream.Streaming {
			continue
		}
		renderRuntimeNativeStartEntrypoint(g, service, serviceIDName, streamRegistryName, method)
		if err := renderRuntimeMessageStartEntrypoint(g, service, serviceIDName, streamRegistryName, method); err != nil {
			return err
		}
	}
	return nil
}

func renderRuntimeUnaryNativeEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, method runtimeMethodProjection) {
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
	g.P("var messageResp []byte")
	renderRuntimeTransportUnaryNativeMessageCall(g, service, method, "server", label, "messageReq")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("return ", method.Codec.MessageToNativeResponse, "(messageResp)")
}

func renderRuntimeUnaryMessageEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string, method runtimeMethodProjection) {
	g.P("func Invoke", service.GoName, "Message", method.Identity.GoName, "(ctx context.Context, req []byte) ([]byte, error) {")
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
	g.P("return server.", method.Identity.MessageMethodRef, "(ctx, req)")
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

func renderRuntimeNativeStartEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName, streamRegistryName string, method runtimeMethodProjection) {
	if method.Stream.StartAcceptsRequest {
		g.P("func Start", service.GoName, "Native", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (rpcruntime.StreamHandle, error) {")
	} else {
		g.P("func Start", service.GoName, "Native", method.Identity.GoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	}
	g.P("registered, err := rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("if err != nil { return 0, err }")
	g.P("switch registered.Kind {")
	if service.Generation.NativeEnabled {
		renderRuntimeNativeStartNativeCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindGoNative", "go native")
		renderRuntimeNativeStartNativeCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindCGONative", "cgo native")
	}
	renderRuntimeNativeStartMessageCase(g, service, streamRegistryName, method)
	renderRuntimeNativeStartTransportCases(g, service, streamRegistryName, method)
	g.P("default:")
	g.P(`return 0, fmt.Errorf("rpccgo: `, service.GoName, ` registered server kind %d is unsupported for native stream starts", registered.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStartNativeCase(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection, kind, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", service.GoName, "NativeServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	g.P("entry := &", lowerInitial(service.GoName), "GoNativeEntry{server: server}")
	renderRuntimeStartSource(g, method, "entry.Start"+method.Identity.GoName, true)
	renderRuntimeCreateNativeStreamHandle(g, service, streamRegistryName, method, kind, "source")
}

func renderRuntimeNativeStartMessageCase(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection) {
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := registered.Server.(", service.GoName, "CGOMessageServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, ` cgo message registered server has invalid type") }`)
	g.P("entry := &", lowerInitial(service.GoName), "CGOMessageEntry{server: server}")
	if method.Stream.StartAcceptsRequest {
		g.P("messageReq, err := ", method.Codec.NativeRequestToMessage, "(", method.Native.ArgNames, ")")
		g.P("if err != nil { return 0, err }")
		renderRuntimeStartSourceWithArgs(g, "entry.Start"+method.Identity.GoName, "ctx, messageReq", true)
	} else {
		renderRuntimeStartSourceWithArgs(g, "entry.Start"+method.Identity.GoName, "ctx", true)
	}
	renderRuntimeCreateNativeStreamHandle(g, service, streamRegistryName, method, "rpcruntime.ServerKindCGOMessage", "source")
}

func renderRuntimeNativeStartTransportCases(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection) {
	switch service.Generation.MessageTransport {
	case MessageTransportConnect:
		renderRuntimeNativeStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindConnect", service.GoName+"Handler", "connect handler")
		renderRuntimeNativeStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindConnectRemote", service.GoName+"Client", "connect remote")
	case MessageTransportGRPC:
		renderRuntimeNativeStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindGRPC", service.GoName+"Server", "grpc server")
		renderRuntimeNativeStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindGRPCRemote", service.GoName+"Client", "grpc remote")
	}
}

func renderRuntimeNativeStartTransportCase(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection, kind, serverType, label string) {
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
	renderRuntimeCreateNativeStreamHandle(g, service, streamRegistryName, method, kind, "source")
}

func renderRuntimeMessageStartEntrypoint(g *protogen.GeneratedFile, service ServicePlan, serviceIDName, streamRegistryName string, method runtimeMethodProjection) error {
	if method.Stream.StartAcceptsRequest {
		g.P("func Start", service.GoName, "Message", method.Identity.GoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
	} else {
		g.P("func Start", service.GoName, "Message", method.Identity.GoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	}
	g.P("registered, err := rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("if err != nil { return 0, err }")
	g.P("switch registered.Kind {")
	if service.Generation.NativeEnabled {
		renderRuntimeMessageStartNativeCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindGoNative", "go native")
		renderRuntimeMessageStartNativeCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindCGONative", "cgo native")
	}
	renderRuntimeMessageStartMessageCase(g, service, streamRegistryName, method)
	renderRuntimeMessageStartTransportCases(g, service, streamRegistryName, method)
	g.P("default:")
	g.P(`return 0, fmt.Errorf("rpccgo: `, service.GoName, ` registered server kind %d is unsupported for message stream starts", registered.Kind)`)
	g.P("}")
	g.P("}")
	g.P()
	return nil
}

func renderRuntimeMessageStartNativeCase(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection, kind, label string) {
	g.P("case ", kind, ":")
	g.P("server, ok := registered.Server.(", service.GoName, "NativeServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, " ", label, ` registered server has invalid type") }`)
	g.P("entry := &", lowerInitial(service.GoName), "GoNativeEntry{server: server}")
	if method.Stream.StartAcceptsRequest {
		g.P(method.Codec.MessageToNativeRequestAssignNames, " := ", method.Codec.MessageToNativeRequest, "(req)")
		g.P("if err != nil { return 0, err }")
		renderRuntimeStartSource(g, method, "entry.Start"+method.Identity.GoName, true)
		g.P("goruntime.KeepAlive(reqOwner)")
	} else {
		renderRuntimeStartSource(g, method, "entry.Start"+method.Identity.GoName, true)
	}
	renderRuntimeCreateMessageStreamHandle(g, service, streamRegistryName, method, kind, "source")
}

func renderRuntimeMessageStartMessageCase(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection) {
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := registered.Server.(", service.GoName, "CGOMessageServer)")
	g.P(`if !ok { return 0, fmt.Errorf("rpccgo: `, service.GoName, ` cgo message registered server has invalid type") }`)
	g.P("entry := &", lowerInitial(service.GoName), "CGOMessageEntry{server: server}")
	if method.Stream.StartAcceptsRequest {
		renderRuntimeStartSourceWithArgs(g, "entry.Start"+method.Identity.GoName, "ctx, req", true)
	} else {
		renderRuntimeStartSourceWithArgs(g, "entry.Start"+method.Identity.GoName, "ctx", true)
	}
	renderRuntimeCreateMessageStreamHandle(g, service, streamRegistryName, method, "rpcruntime.ServerKindCGOMessage", "source")
}

func renderRuntimeMessageStartTransportCases(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection) {
	switch service.Generation.MessageTransport {
	case MessageTransportConnect:
		renderRuntimeMessageStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindConnect", service.GoName+"Handler", "connect handler")
		renderRuntimeMessageStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindConnectRemote", service.GoName+"Client", "connect remote")
	case MessageTransportGRPC:
		renderRuntimeMessageStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindGRPC", service.GoName+"Server", "grpc server")
		renderRuntimeMessageStartTransportCase(g, service, streamRegistryName, method, "rpcruntime.ServerKindGRPCRemote", service.GoName+"Client", "grpc remote")
	}
}

func renderRuntimeMessageStartTransportCase(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection, kind, serverType, label string) {
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
	renderRuntimeCreateMessageStreamHandle(g, service, streamRegistryName, method, kind, "source")
}

func renderRuntimeStartSource(g *protogen.GeneratedFile, method runtimeMethodProjection, startExpr string, withErr bool) {
	args := "ctx"
	if method.Stream.StartAcceptsRequest {
		args += nativeGoCallSuffix(method.Native.ArgNames)
	}
	renderRuntimeStartSourceWithArgs(g, startExpr, args, withErr)
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

func renderRuntimeCreateNativeStreamHandle(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection, kind, sourceExpr string) {
	sessionName := runtimeStreamNativeSessionName(service.GoName, method)
	g.P("return ", streamRegistryName, ".Create(&", sessionName, "{kind: ", kind, ", session: ", sourceExpr, "})")
}

func renderRuntimeCreateMessageStreamHandle(g *protogen.GeneratedFile, service ServicePlan, streamRegistryName string, method runtimeMethodProjection, kind, sourceExpr string) {
	sessionName := runtimeStreamMessageSessionName(service.GoName, method)
	g.P("return ", streamRegistryName, ".Create(&", sessionName, "{kind: ", kind, ", session: ", sourceExpr, "})")
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
