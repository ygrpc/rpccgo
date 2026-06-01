package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	streamingMethods := runtimeStreamingMethods(runtimeMethods)
	codecEnabled := service.CodecEnabled
	directConnectStreaming := service.Adapters.Has(AdapterTokenMessageConnect) && serviceHasStreamingMethod(service)
	directGRPCStreaming := service.Adapters.Has(AdapterTokenMessageGRPC) && serviceHasStreamingMethod(service)
	directUnary := (service.Adapters.Has(AdapterTokenMessageConnect) || service.Adapters.Has(AdapterTokenMessageGRPC)) && serviceHasUnaryMethod(service)
	directFmt := directUnary || directConnectStreaming || directGRPCStreaming
	directProto := directUnary || directConnectStreaming || directGRPCStreaming

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`atomic "sync/atomic"`)
	if directFmt {
		g.P(`fmt "fmt"`)
	}
	if directProto {
		g.P(`proto "google.golang.org/protobuf/proto"`)
	}
	if directConnectStreaming || directGRPCStreaming || nativeServerHasStreamingMethod(service) || serviceHasStreamingMethod(service) {
		g.P(`io "io"`)
		if serviceHasClientStreamingMethod(service) || serviceHasBidiStreamingMethod(service) || nativeServerHasClientInputStreamingMethod(service) {
			g.P(`sync "sync"`)
		}
	}
	if directConnectStreaming {
		g.P(`connect "connectrpc.com/connect"`)
		if serviceHasClientStreamingMethod(service) {
			g.P(`time "time"`)
		}
	}
	if directGRPCStreaming {
		g.P(`grpc "google.golang.org/grpc"`)
		g.P(`metadata "google.golang.org/grpc/metadata"`)
	}
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	adapterName := service.GoName + "NativeServer"
	messageAdapterName := service.GoName + "CGOMessageServer"
	activeName := lowerInitial(service.GoName) + "ActiveServer"
	streamRegistryName := lowerInitial(service.GoName) + "StreamRegistry"

	if !service.NativeFileFamily.NativeServer.Enabled {
		renderGoNativeServerInterface(g, service, adapterName)
		renderGoNativeStreamInterfaces(g, service)
	}
	errorNames := nativeServerErrorNamesFor(service)
	g.P("var (")
	g.P(errorNames.RequestBridgeNotImplemented, ` = errors.New("rpccgo: native request bridge is not implemented")`)
	g.P(errorNames.StreamBridgeNotImplemented, ` = errors.New("rpccgo: native stream bridge is not implemented")`)
	g.P(errorNames.StreamIsNil, ` = errors.New("rpccgo: native stream is nil")`)
	g.P(errorNames.StreamClosed, ` = errors.New("rpccgo: native stream is closed")`)
	g.P(")")
	g.P()
	nativeServerAdapterName := lowerInitial(service.GoName) + "NativeServerAdapter"
	renderGoNativeAdapter(g, service, runtimeMethods, service.GoName+"NativeServer", nativeServerAdapterName, errorNames)
	messageServerAdapterName := lowerInitial(service.GoName) + "MessageServerAdapter"
	renderRuntimeSourceSessionInterfaces(g, service.GoName, streamingMethods)
	renderMessageServerAdapter(g, service, runtimeMethods, messageAdapterName, messageServerAdapterName)

	renderRuntimeActiveServerRecord(g, service, runtimeMethods)
	for _, method := range streamingMethods {
		renderRuntimeFinalSessions(g, service.GoName, method)
		renderRuntimeNativeStreamFacade(g, service.GoName, streamRegistryName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, streamRegistryName, method)
	}

	g.P("var ", activeName, " atomic.Pointer[", lowerInitial(service.GoName), "ActiveServerRecord]")
	g.P("var ", streamRegistryName, " rpcruntime.StreamRegistry")
	g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P("var ", service.GoName, `NativeMessageConverterUnavailableErr = errors.New("rpccgo: native/message converter is not enabled")`)
	g.P()

	renderRuntimeRegistrations(g, service, adapterName, messageAdapterName, runtimeMethods, codecEnabled, activeName)
	renderRuntimeTransportMessageSessions(g, service, streamingMethods)
	renderRuntimeEntrypoints(g, service.GoName, adapterName, activeName, streamRegistryName, runtimeMethods)

	return nil
}

type runtimeAdapterMethod struct {
	SourceFullName        string
	AdapterName           string
	AdapterArgs           string
	AdapterResult         string
	MethodGoName          string
	SessionName           string
	NativeArgs            string
	NativeReturns         string
	NativeZero            string
	NativeErrZero         string
	NativeNoActiveZero    string
	NativeConverterZero   string
	NativeInvalidZero     string
	NativeArgNames        string
	NativeNames           string
	NativeVarDecls        []string
	Streaming             bool
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
}

func buildRuntimeAdapterMethods(g *protogen.GeneratedFile, service ServicePlan) ([]runtimeAdapterMethod, error) {
	if len(service.Methods) == 0 {
		return []runtimeAdapterMethod{
			{AdapterName: "DispatchUnary", AdapterResult: " error", MethodGoName: "DispatchUnary", SessionName: service.GoName + "DispatchUnaryNativeStreamSession"},
			{AdapterName: "StartClientStream", AdapterResult: " (" + service.GoName + "ClientStreamNativeStreamSession, error)", MethodGoName: "ClientStream", SessionName: service.GoName + "ClientStreamNativeStreamSession", Streaming: true, CanSend: true, FinishReturnsResponse: true},
			{AdapterName: "StartServerStream", AdapterResult: " (" + service.GoName + "ServerStreamNativeStreamSession, error)", MethodGoName: "ServerStream", SessionName: service.GoName + "ServerStreamNativeStreamSession", Streaming: true, CanRecv: true},
			{AdapterName: "StartBidiStream", AdapterResult: " (" + service.GoName + "BidiStreamNativeStreamSession, error)", MethodGoName: "BidiStream", SessionName: service.GoName + "BidiStreamNativeStreamSession", Streaming: true, CanSend: true, CanRecv: true, CanCloseSend: true},
		}, nil
	}

	methods := make([]runtimeAdapterMethod, 0, len(service.Methods))
	seen := make(map[string]string, len(service.Methods))
	for _, method := range service.Methods {
		rendered, err := runtimeAdapterMethodFor(g, method)
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

func runtimeAdapterMethodFor(g *protogen.GeneratedFile, method MethodPlan) (runtimeAdapterMethod, error) {
	if err := ValidateMethodRenderPlan(method); err != nil {
		return runtimeAdapterMethod{}, err
	}
	shape := method.RenderPlan
	nativeFields := method.Contract.Native.RequestFields
	responseFields := method.Contract.Native.ResponseFields
	sessionName := shape.Symbols.NativeSessionType
	nativeArgs := nativeGoRequestParams(g, nativeFields)
	nativeReturns := nativeGoResponseReturns(g, responseFields)
	nativeZero := nativeGoZeroReturns(responseFields, `errors.New("rpccgo native server method is not implemented")`)
	nativeErrZero := nativeGoZeroReturns(responseFields, "err")
	nativeNoActiveZero := nativeGoZeroReturns(responseFields, "rpcruntime.ErrNoActiveServer")
	nativeConverterZero := nativeGoZeroReturns(responseFields, shape.Errors.NativeMessageConverterErr)
	nativeInvalidZero := nativeGoZeroReturns(responseFields, "rpcruntime.ErrStreamInvalidHandle")
	nativeArgNames := nativeGoRequestArgNames(nativeFields)
	nativeResultNames := nativeGoResponseResultNames(responseFields)
	nativeVarDecls := nativeGoResponseResultVarDecls(g, responseFields)
	rendered := runtimeAdapterMethod{
		SourceFullName:        method.FullName,
		MethodGoName:          method.GoName,
		AdapterName:           shape.Symbols.NativeAdapterMethod,
		SessionName:           sessionName,
		NativeArgs:            nativeArgs,
		NativeReturns:         nativeReturns,
		NativeZero:            nativeZero,
		NativeErrZero:         nativeErrZero,
		NativeNoActiveZero:    nativeNoActiveZero,
		NativeConverterZero:   nativeConverterZero,
		NativeInvalidZero:     nativeInvalidZero,
		NativeArgNames:        nativeArgNames,
		NativeNames:           nativeResultNames,
		NativeVarDecls:        nativeVarDecls,
		Streaming:             shape.Lifecycle.Streaming,
		CanSend:               shape.Lifecycle.CanSend,
		CanRecv:               shape.Lifecycle.CanRecv,
		CanCloseSend:          shape.Lifecycle.CanCloseSend,
		FinishReturnsResponse: shape.Lifecycle.FinishReturnsResponse,
	}
	if !rendered.Streaming {
		rendered.AdapterArgs = nativeArgs
		rendered.AdapterResult = " (" + nativeReturns + ")"
		return rendered, nil
	}
	rendered.AdapterResult = " (" + sessionName + ", error)"
	if rendered.CanRecv && !rendered.CanSend {
		rendered.AdapterArgs = nativeArgs
	}
	return rendered, nil
}

type runtimeStreamShape int

const (
	runtimeStreamUnary runtimeStreamShape = iota
	runtimeStreamClient
	runtimeStreamServer
	runtimeStreamBidi
)

func runtimeStreamShapeFor(method runtimeAdapterMethod) runtimeStreamShape {
	switch {
	case !method.Streaming:
		return runtimeStreamUnary
	case method.CanSend && method.FinishReturnsResponse:
		return runtimeStreamClient
	case method.CanRecv && !method.CanSend:
		return runtimeStreamServer
	case method.CanSend && method.CanRecv && method.CanCloseSend:
		return runtimeStreamBidi
	default:
		panic("invalid runtime stream capabilities")
	}
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

func renderRuntimeActiveServerRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod) {
	recordName := lowerInitial(service.GoName) + "ActiveServerRecord"
	g.P("type ", recordName, " struct {")
	for _, method := range methods {
		if !method.Streaming {
			g.P("invokeNative", method.MethodGoName, " func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ")")
			g.P("invokeMessage", method.MethodGoName, " func(ctx context.Context, req []byte) ([]byte, error)")
			continue
		}
		nativeSession := runtimeFinalNativeSessionName(service.GoName, method)
		messageSession := runtimeFinalMessageSessionName(service.GoName, method)
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

func methodForRuntimeService(service ServicePlan, method runtimeAdapterMethod) MethodPlan {
	for _, candidate := range service.Methods {
		if candidate.GoName == method.MethodGoName {
			return candidate
		}
	}
	return MethodPlan{GoName: method.MethodGoName}
}
