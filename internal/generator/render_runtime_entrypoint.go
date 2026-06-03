package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeEntrypoints(g *protogen.GeneratedFile, serviceName, adapterName, activeName, streamRegistryName string, methods []runtimeMethodProjection) {
	for _, method := range methods {
		if method.Stream.Streaming {
			continue
		}
		g.P("func Invoke", serviceName, "Native", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ") {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return ", method.Native.NoActiveZero, " }")
		g.P("return active.invokeNative", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
		g.P("}")
		g.P()
		g.P("func Invoke", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return nil, rpcruntime.ErrNoActiveServer }")
		g.P("return active.invokeMessage", method.Identity.GoName, "(ctx, req)")
		g.P("}")
		g.P()
	}
	for _, method := range methods {
		if !method.Stream.Streaming {
			continue
		}
		renderRuntimeStartEntrypoints(g, serviceName, activeName, streamRegistryName, method)
	}
}

func renderRuntimeStartEntrypoints(g *protogen.GeneratedFile, serviceName, activeName, streamRegistryName string, method runtimeMethodProjection) {
	if method.Stream.StartAcceptsRequest {
		g.P("func Start", serviceName, "Native", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startNative", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
	} else {
		g.P("func Start", serviceName, "Native", method.Identity.GoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startNative", method.Identity.GoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
	if method.Stream.StartAcceptsRequest {
		g.P("func Start", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startMessage", method.Identity.GoName, "(ctx, req)")
	} else {
		g.P("func Start", serviceName, "Message", method.Identity.GoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startMessage", method.Identity.GoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
}
