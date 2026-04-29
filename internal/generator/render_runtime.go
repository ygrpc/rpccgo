package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	adapterName := service.GoName + "NativeAdapter"
	sessionName := service.GoName + "NativeStreamSession"
	dispatcherName := lowerInitial(service.GoName) + "Dispatcher"

	g.P("type ", adapterName, " interface {")
	renderRuntimeAdapterMethods(g, service)
	g.P("}")
	g.P()

	g.P("type ", sessionName, " interface {")
	g.P("Send(ctx context.Context) error")
	g.P("Finish(ctx context.Context) error")
	g.P("CloseSend(ctx context.Context) error")
	g.P("Cancel(ctx context.Context) error")
	g.P("}")
	g.P()

	g.P("var ", dispatcherName, " rpcruntime.Dispatcher[", adapterName, "]")
	g.P()

	g.P("func register", service.GoName, "ActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return ", dispatcherName, ".Register(kind, rpcruntime.ServerContractNative, adapter)")
	g.P("}")
	g.P()

	g.P("func load", service.GoName, "NativeStream(handle rpcruntime.StreamHandle) (", sessionName, ", bool) {")
	g.P("return rpcruntime.LoadDispatcherStream[", adapterName, ", ", sessionName, "](&", dispatcherName, ", handle)")
	g.P("}")
	g.P()

	g.P("func take", service.GoName, "NativeStream(handle rpcruntime.StreamHandle) (", sessionName, ", bool) {")
	g.P("return rpcruntime.TakeDispatcherStream[", adapterName, ", ", sessionName, "](&", dispatcherName, ", handle)")
	g.P("}")
	g.P()

	g.P("func delete", service.GoName, "NativeStream(handle rpcruntime.StreamHandle) bool {")
	g.P("return rpcruntime.DeleteDispatcherStream[", adapterName, "](&", dispatcherName, ", handle)")
	g.P("}")
}

func renderRuntimeAdapterMethods(g *protogen.GeneratedFile, service ServicePlan) {
	if len(service.Methods) == 0 {
		g.P("DispatchUnary(ctx context.Context) error")
		g.P("StartClientStream(ctx context.Context) (", service.GoName, "NativeStreamSession, error)")
		g.P("StartServerStream(ctx context.Context) (", service.GoName, "NativeStreamSession, error)")
		g.P("StartBidiStream(ctx context.Context) (", service.GoName, "NativeStreamSession, error)")
		return
	}

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(method.GoName, "(ctx context.Context) error")
		case StreamingKindClientStreaming:
			g.P("Start", method.GoName, "(ctx context.Context) (", service.GoName, "NativeStreamSession, error)")
		case StreamingKindServerStreaming:
			g.P("Start", method.GoName, "(ctx context.Context) (", service.GoName, "NativeStreamSession, error)")
		case StreamingKindBidiStreaming:
			g.P("Start", method.GoName, "(ctx context.Context) (", service.GoName, "NativeStreamSession, error)")
		default:
			g.P(method.GoName, "(ctx context.Context) error")
		}
	}
}
