package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection) {
	bindingName := lowerInitial(service.GoName) + "Binding"
	g.P("// ", bindingName, " is the immutable set of caller-facing method closures")
	g.P("// built after a registration source is accepted.")
	g.P("type ", bindingName, " struct {")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("invokeNative", method.Identity.GoName, " func(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ")")
			g.P("invokeMessage", method.Identity.GoName, " func(ctx context.Context, req []byte) ([]byte, error)")
			continue
		}
		nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
		messageSession := runtimeStreamMessageSessionName(service.GoName, method)
		if method.Stream.StartAcceptsRequest {
			g.P("startNative", method.Identity.GoName, " func(ctx context.Context", method.Native.Args, ") (*", nativeSession, ", error)")
			g.P("startMessage", method.Identity.GoName, " func(ctx context.Context, req []byte) (*", messageSession, ", error)")
			continue
		}
		g.P("startNative", method.Identity.GoName, " func(ctx context.Context) (*", nativeSession, ", error)")
		g.P("startMessage", method.Identity.GoName, " func(ctx context.Context) (*", messageSession, ", error)")
	}
	g.P("}")
	g.P()
}
