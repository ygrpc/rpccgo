package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeEntrypoints(g *protogen.GeneratedFile, serviceName, adapterName, activeName, streamRegistryName string, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func Invoke", serviceName, "Native", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return ", method.NativeNoActiveZero, " }")
		g.P("return active.invokeNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("}")
		g.P()
		g.P("func Invoke", serviceName, "Message", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return nil, rpcruntime.ErrNoActiveServer }")
		g.P("return active.invokeMessage", method.MethodGoName, "(ctx, req)")
		g.P("}")
		g.P()
	}
	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		renderRuntimeStartEntrypoints(g, serviceName, activeName, streamRegistryName, method)
	}
}

func renderRuntimeStartEntrypoints(g *protogen.GeneratedFile, serviceName, activeName, streamRegistryName string, method runtimeAdapterMethod) {
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("func Start", serviceName, "Native", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	} else {
		g.P("func Start", serviceName, "Native", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startNative", method.MethodGoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("func Start", serviceName, "Message", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startMessage", method.MethodGoName, "(ctx, req)")
	} else {
		g.P("func Start", serviceName, "Message", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startMessage", method.MethodGoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
}
