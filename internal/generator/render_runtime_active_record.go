package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeActiveServerRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod) {
	recordName := lowerInitial(service.GoName) + "ActiveServerRecord"
	g.P("type ", recordName, " struct {")
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
