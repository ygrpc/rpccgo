package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeBindingTypes(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection) {
	nativeBindingName := lowerInitial(service.GoName) + "NativeActiveBinding"
	messageBindingName := lowerInitial(service.GoName) + "MessageActiveBinding"
	renderRuntimeNativeBindingType(g, service, methods, nativeBindingName)
	renderRuntimeMessageBindingType(g, service, methods, messageBindingName)
}

func renderRuntimeNativeBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, bindingName string) {
	g.P("// ", bindingName, " is the immutable native active closure set.")
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
	g.P("// ", bindingName, " is the immutable message active closure set.")
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
