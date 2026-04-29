package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	streamingMethods := runtimeStreamingMethods(runtimeMethods)

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
	dispatcherName := lowerInitial(service.GoName) + "Dispatcher"

	g.P("type ", adapterName, " interface {")
	for _, method := range runtimeMethods {
		g.P(method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult)
	}
	g.P("}")
	g.P()

	for _, method := range streamingMethods {
		renderRuntimeSessionInterface(g, method)
	}

	g.P("var ", dispatcherName, " rpcruntime.Dispatcher[", adapterName, "]")
	g.P()

	g.P("func register", service.GoName, "ActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return ", dispatcherName, ".Register(kind, rpcruntime.ServerContractNative, adapter)")
	g.P("}")
	g.P()

	for _, method := range streamingMethods {
		renderRuntimeStreamHelpers(g, service.GoName, adapterName, dispatcherName, method)
	}

	return nil
}

type runtimeAdapterMethod struct {
	SourceFullName string
	AdapterName    string
	AdapterArgs    string
	AdapterResult  string
	MethodGoName   string
	SessionName    string
	RequestType    string
	ResponseType   string
	StreamingKind  StreamingKind
	Streaming      bool
}

func buildRuntimeAdapterMethods(g *protogen.GeneratedFile, service ServicePlan) ([]runtimeAdapterMethod, error) {
	if len(service.Methods) == 0 {
		return []runtimeAdapterMethod{
			{AdapterName: "DispatchUnary", AdapterResult: " error", MethodGoName: "DispatchUnary", SessionName: service.GoName + "DispatchUnaryNativeStreamSession"},
			{AdapterName: "StartClientStream", AdapterResult: " (" + service.GoName + "ClientStreamNativeStreamSession, error)", MethodGoName: "ClientStream", SessionName: service.GoName + "ClientStreamNativeStreamSession", Streaming: true},
			{AdapterName: "StartServerStream", AdapterResult: " (" + service.GoName + "ServerStreamNativeStreamSession, error)", MethodGoName: "ServerStream", SessionName: service.GoName + "ServerStreamNativeStreamSession", Streaming: true},
			{AdapterName: "StartBidiStream", AdapterResult: " (" + service.GoName + "BidiStreamNativeStreamSession, error)", MethodGoName: "BidiStream", SessionName: service.GoName + "BidiStreamNativeStreamSession", Streaming: true},
		}, nil
	}

	methods := make([]runtimeAdapterMethod, 0, len(service.Methods))
	seen := make(map[string]string, len(service.Methods))
	for _, method := range service.Methods {
		rendered, err := runtimeAdapterMethodFor(g, service, method)
		if err != nil {
			return nil, err
		}
		if previous, exists := seen[rendered.AdapterName]; exists {
			return nil, fmt.Errorf("runtime adapter method %s for %s collides with %s", rendered.AdapterName, method.FullName, previous)
		}
		seen[rendered.AdapterName] = method.FullName
		methods = append(methods, rendered)
	}
	return methods, nil
}

func runtimeAdapterMethodFor(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) (runtimeAdapterMethod, error) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	requestType := nativeRuntimeMessageType(g, method.Request)
	responseType := nativeRuntimeMessageType(g, method.Response)
	rendered := runtimeAdapterMethod{
		SourceFullName: method.FullName,
		MethodGoName:   method.GoName,
		SessionName:    sessionName,
		RequestType:    requestType,
		ResponseType:   responseType,
		StreamingKind:  method.Streaming,
	}
	switch method.Streaming {
	case StreamingKindUnary:
		rendered.AdapterName = method.GoName
		rendered.AdapterArgs = ", req " + requestType
		rendered.AdapterResult = " (" + responseType + ", error)"
	case StreamingKindClientStreaming, StreamingKindServerStreaming, StreamingKindBidiStreaming:
		rendered.AdapterName = "Start" + method.GoName
		rendered.AdapterResult = " (" + sessionName + ", error)"
		rendered.Streaming = true
	default:
		return runtimeAdapterMethod{}, fmt.Errorf("%s has unknown streaming kind %d", method.FullName, method.Streaming)
	}
	return rendered, nil
}

func nativeRuntimeMessageType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}

func runtimeStreamingMethods(methods []runtimeAdapterMethod) []runtimeAdapterMethod {
	streaming := make([]runtimeAdapterMethod, 0, len(methods))
	for _, method := range methods {
		if method.Streaming {
			streaming = append(streaming, method)
		}
	}
	return streaming
}

func renderRuntimeSessionInterface(g *protogen.GeneratedFile, method runtimeAdapterMethod) {
	g.P("type ", method.SessionName, " interface {")
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("Send(ctx context.Context, req ", method.RequestType, ") error")
		g.P("Finish(ctx context.Context) (", method.ResponseType, ", error)")
		g.P("Cancel(ctx context.Context) error")
	case StreamingKindServerStreaming:
		g.P("Recv(ctx context.Context) (", method.ResponseType, ", error)")
		g.P("Cancel(ctx context.Context) error")
	case StreamingKindBidiStreaming:
		g.P("Send(ctx context.Context, req ", method.RequestType, ") error")
		g.P("Recv(ctx context.Context) (", method.ResponseType, ", error)")
		g.P("CloseSend(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	default:
		g.P("Cancel(ctx context.Context) error")
	}
	g.P("}")
	g.P()
}

func renderRuntimeStreamHelpers(g *protogen.GeneratedFile, serviceName, adapterName, dispatcherName string, method runtimeAdapterMethod) {
	g.P("func load", serviceName, method.MethodGoName, "NativeStream(handle rpcruntime.StreamHandle) (", method.SessionName, ", bool) {")
	g.P("return rpcruntime.LoadDispatcherStream[", adapterName, ", ", method.SessionName, "](&", dispatcherName, ", handle)")
	g.P("}")
	g.P()

	g.P("func take", serviceName, method.MethodGoName, "NativeStream(handle rpcruntime.StreamHandle) (", method.SessionName, ", bool) {")
	g.P("return rpcruntime.TakeDispatcherStream[", adapterName, ", ", method.SessionName, "](&", dispatcherName, ", handle)")
	g.P("}")
	g.P()

	g.P("func delete", serviceName, method.MethodGoName, "NativeStream(handle rpcruntime.StreamHandle) bool {")
	g.P("return rpcruntime.DeleteDispatcherStream[", adapterName, "](&", dispatcherName, ", handle)")
	g.P("}")
	g.P()
}
