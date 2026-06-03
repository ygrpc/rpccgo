package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeActiveServerType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection) {
	activeServerName := lowerInitial(service.GoName) + "ActiveServer"
	nativeBindingName := lowerInitial(service.GoName) + "NativeCallerBinding"
	messageBindingName := lowerInitial(service.GoName) + "MessageCallerBinding"
	g.P("// ", activeServerName, " is the immutable active server record")
	g.P("// built after a registration source is accepted.")
	g.P("type ", activeServerName, " struct {")
	g.P("native *", nativeBindingName)
	g.P("message *", messageBindingName)
	g.P("}")
	g.P()
	renderRuntimeNativeBindingType(g, service, methods, nativeBindingName)
	renderRuntimeMessageBindingType(g, service, methods, messageBindingName)
}

func renderRuntimeNativeBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, bindingName string) {
	g.P("// ", bindingName, " is the immutable native caller-facing closure set.")
	g.P("type ", bindingName, " struct {")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("invoke", method.Identity.GoName, " func(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ")")
			continue
		}
		nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
		if method.Stream.StartAcceptsRequest {
			g.P("start", method.Identity.GoName, " func(ctx context.Context", method.Native.Args, ") (*", nativeSession, ", error)")
			continue
		}
		g.P("start", method.Identity.GoName, " func(ctx context.Context) (*", nativeSession, ", error)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, bindingName string) {
	g.P("// ", bindingName, " is the immutable message caller-facing closure set.")
	g.P("type ", bindingName, " struct {")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("invoke", method.Identity.GoName, " func(ctx context.Context, req []byte) ([]byte, error)")
			continue
		}
		messageSession := runtimeStreamMessageSessionName(service.GoName, method)
		if method.Stream.StartAcceptsRequest {
			g.P("start", method.Identity.GoName, " func(ctx context.Context, req []byte) (*", messageSession, ", error)")
			continue
		}
		g.P("start", method.Identity.GoName, " func(ctx context.Context) (*", messageSession, ", error)")
	}
	g.P("}")
	g.P()
}
