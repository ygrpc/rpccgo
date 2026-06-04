package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeEntrypoints(g *protogen.GeneratedFile, serviceName, adapterName, nativeActiveName, messageActiveName, streamRegistryName string, methods []runtimeMethodProjection) {
	for _, method := range methods {
		if method.Stream.Streaming {
			continue
		}
		g.P("func Invoke", serviceName, "Native", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ") {")
		g.P("native := ", nativeActiveName, ".Load()")
		g.P("if native == nil { return ", method.Native.NoActiveZero, " }")
		g.P("return native.invoke", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
		g.P("}")
		g.P()
		g.P("func Invoke", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("message := ", messageActiveName, ".Load()")
		g.P("if message == nil { return nil, rpcruntime.ErrNoActiveServer }")
		g.P("return message.invoke", method.Identity.GoName, "(ctx, req)")
		g.P("}")
		g.P()
	}
	for _, method := range methods {
		if !method.Stream.Streaming {
			continue
		}
		renderRuntimeStartEntrypoints(g, serviceName, nativeActiveName, messageActiveName, streamRegistryName, method)
	}
}

func renderRuntimeStartEntrypoints(g *protogen.GeneratedFile, serviceName, nativeActiveName, messageActiveName, streamRegistryName string, method runtimeMethodProjection) {
	if method.Stream.StartAcceptsRequest {
		g.P("func Start", serviceName, "Native", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (rpcruntime.StreamHandle, error) {")
		g.P("native := ", nativeActiveName, ".Load()")
		g.P("if native == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := native.start", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
	} else {
		g.P("func Start", serviceName, "Native", method.Identity.GoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("native := ", nativeActiveName, ".Load()")
		g.P("if native == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := native.start", method.Identity.GoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
	if method.Stream.StartAcceptsRequest {
		g.P("func Start", serviceName, "Message", method.Identity.GoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
		g.P("message := ", messageActiveName, ".Load()")
		g.P("if message == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := message.start", method.Identity.GoName, "(ctx, req)")
	} else {
		g.P("func Start", serviceName, "Message", method.Identity.GoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("message := ", messageActiveName, ".Load()")
		g.P("if message == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := message.start", method.Identity.GoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
}
