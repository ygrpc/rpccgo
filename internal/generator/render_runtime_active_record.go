package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod) {
	bindingName := lowerInitial(service.GoName) + "Binding"
	g.P("// ", bindingName, " is the immutable set of caller-facing method closures")
	g.P("// built after a registration source is accepted.")
	g.P("type ", bindingName, " struct {")
	for _, method := range methods {
		if !method.Streaming {
			g.P("invokeNative", method.MethodGoName, " func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ")")
			g.P("invokeMessage", method.MethodGoName, " func(ctx context.Context, req []byte) ([]byte, error)")
			continue
		}
		nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
		messageSession := runtimeStreamMessageSessionName(service.GoName, method)
		if runtimeStreamShapeFor(method) == runtimeStreamServer {
			g.P("startNative", method.MethodGoName, " func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error)")
			g.P("startMessage", method.MethodGoName, " func(ctx context.Context, req []byte) (*", messageSession, ", error)")
			continue
		}
		g.P("startNative", method.MethodGoName, " func(ctx context.Context) (*", nativeSession, ", error)")
		g.P("startMessage", method.MethodGoName, " func(ctx context.Context) (*", messageSession, ", error)")
	}
	g.P("}")
	g.P()
}
