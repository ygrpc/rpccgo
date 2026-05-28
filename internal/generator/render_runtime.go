package generator

import (
	"fmt"
	"strings"

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
	directFmt := directUnary || directGRPCStreaming || (directConnectStreaming && (serviceHasServerStreamingMethod(service) || serviceHasBidiStreamingMethod(service)))
	directProto := directUnary || directConnectStreaming || directGRPCStreaming

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if directFmt {
		g.P(`fmt "fmt"`)
	}
	if directProto {
		g.P(`proto "google.golang.org/protobuf/proto"`)
	}
	if directConnectStreaming || directGRPCStreaming {
		g.P(`io "io"`)
		if serviceHasClientStreamingMethod(service) || serviceHasBidiStreamingMethod(service) {
			g.P(`sync "sync"`)
		}
	}
	if directGRPCStreaming {
			g.P(`metadata "google.golang.org/grpc/metadata"`)
		}
		g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	adapterName := service.GoName + "NativeAdapter"
	messageAdapterName := service.GoName + "MessageAdapter"
	activeSlotName := lowerInitial(service.GoName) + "ActiveSlot"
	streamRegistryName := lowerInitial(service.GoName) + "StreamRegistry"

	g.P("type ", adapterName, " interface {")
	for _, method := range runtimeMethods {
		g.P(method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult)
	}
	g.P("}")
	g.P()

	renderRuntimeMessageAdapter(g, service, messageAdapterName, runtimeMethods)

	for _, method := range streamingMethods {
		renderRuntimeSessionInterface(g, method)
		renderRuntimeMessageSessionInterface(g, method)
	}
	for _, method := range streamingMethods {
		renderRuntimeNativeStreamFacade(g, service.GoName, streamRegistryName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, streamRegistryName, method)
	}

	bridgeName := lowerInitial(service.GoName) + "Bridge"
	bridgeTypeName := lowerInitial(service.GoName) + "RuntimeBridge"

	g.P("var ", activeSlotName, " rpcruntime.ActiveServerSlot[any]")
	g.P("var ", streamRegistryName, " rpcruntime.StreamRegistry[*rpcruntime.StreamEntry]")
	g.P("var ", bridgeName, " = ", bridgeTypeName, "{active: &", activeSlotName, ", streams: &", streamRegistryName, "}")
	g.P("var ", service.GoName, `NativeMessageConverterUnavailableErr = errors.New("rpccgo: native/message converter is not enabled")`)
	g.P("var ", service.GoName, `NativeAdapterUnavailableErr = errors.New("rpccgo: native adapter is unavailable")`)
	g.P("var ", service.GoName, `MessageAdapterUnavailableErr = errors.New("rpccgo: message adapter is unavailable")`)
	g.P("var ", service.GoName, `UnknownActiveContractErr = errors.New("rpccgo: unknown active server contract")`)
	g.P()

	g.P("func register", service.GoName, "ActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("snapshot, err := ", activeSlotName, ".Store(kind, rpcruntime.ServerContractNative, adapter)")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil")
	g.P("}")
	g.P()

	g.P("func register", service.GoName, "MessageActiveServer(kind rpcruntime.ServerKind, adapter ", messageAdapterName, ") (rpcruntime.AdapterSnapshot[", messageAdapterName, "], error) {")
	g.P("snapshot, err := ", activeSlotName, ".Store(kind, rpcruntime.ServerContractMessage, adapter)")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil")
	g.P("}")
	g.P()

	renderRuntimeDirectMessageRegistrations(g, service)
	renderRuntimeBridge(g, service, bridgeTypeName, adapterName, messageAdapterName, runtimeMethods, codecEnabled)
	renderRuntimeNativeEntrypoints(g, service.GoName, adapterName, bridgeName, runtimeMethods)
	renderRuntimeMessageEntrypoints(g, service.GoName, messageAdapterName, bridgeName, runtimeMethods)

	return nil
}

type runtimeAdapterMethod struct {
	SourceFullName string
	AdapterName    string
	AdapterArgs    string
	AdapterResult  string
	MethodGoName   string
	SessionName    string
	NativeArgs     string
	NativeReturns  string
	NativeZero     string
	NativeErrZero  string
	NativeArgNames string
	NativeNames    string
	NativeVarDecls []string
	SessionKind    SessionKind
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
	nativeArgNames := nativeGoRequestArgNames(nativeFields)
	nativeResultNames := nativeGoResponseResultNames(responseFields)
	nativeVarDecls := nativeGoResponseResultVarDecls(g, responseFields)
	rendered := runtimeAdapterMethod{
		SourceFullName: method.FullName,
		MethodGoName:   method.GoName,
		AdapterName:    shape.Symbols.NativeAdapterMethod,
		SessionName:    sessionName,
		NativeArgs:     nativeArgs,
		NativeReturns:  nativeReturns,
		NativeZero:     nativeZero,
		NativeErrZero:  nativeErrZero,
		NativeArgNames: nativeArgNames,
		NativeNames:    nativeResultNames,
		NativeVarDecls: nativeVarDecls,
		SessionKind:    shape.Lifecycle.SessionKind,
		Streaming:      shape.Lifecycle.SessionKind != SessionKindNone,
	}
	if !rendered.Streaming {
		rendered.AdapterArgs = nativeArgs
		rendered.AdapterResult = " (" + nativeReturns + ")"
		return rendered, nil
	}
	rendered.AdapterResult = " (" + sessionName + ", error)"
	if shape.Lifecycle.SessionKind == SessionKindServer {
		rendered.AdapterArgs = nativeArgs
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
	switch method.SessionKind {
	case SessionKindClient:
		g.P("Send(ctx context.Context", method.NativeArgs, ") error")
		g.P("Finish(ctx context.Context) (", method.NativeReturns, ")")
		g.P("Cancel(ctx context.Context) error")
	case SessionKindServer:
		g.P("Recv(ctx context.Context) (", method.NativeReturns, ")")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	case SessionKindBidi:
		g.P("Send(ctx context.Context", method.NativeArgs, ") error")
		g.P("Recv(ctx context.Context) (", method.NativeReturns, ")")
		g.P("CloseSend(ctx context.Context) error")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	default:
		g.P("Cancel(ctx context.Context) error")
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, methods []runtimeAdapterMethod) {
	g.P("type ", adapterName, " interface {")
	for _, method := range methods {
		switch method.SessionKind {
		case SessionKindNone:
			g.P(method.AdapterName, "Message(ctx context.Context, req []byte) ([]byte, error)")
		case SessionKindClient:
			g.P("Start", method.MethodGoName, "Message(ctx context.Context) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		case SessionKindServer:
			g.P("Start", method.MethodGoName, "Message(ctx context.Context, req []byte) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		case SessionKindBidi:
			g.P("Start", method.MethodGoName, "Message(ctx context.Context) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		}
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageSessionInterface(g *protogen.GeneratedFile, method runtimeAdapterMethod) {
	sessionName := methodMessageSessionName(method)
	g.P("type ", sessionName, " interface {")
	switch method.SessionKind {
	case SessionKindClient:
		g.P("Send(ctx context.Context, req []byte) error")
		g.P("Finish(ctx context.Context) ([]byte, error)")
		g.P("Cancel(ctx context.Context) error")
	case SessionKindServer:
		g.P("Recv(ctx context.Context) ([]byte, error)")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	case SessionKindBidi:
		g.P("Send(ctx context.Context, req []byte) error")
		g.P("Recv(ctx context.Context) ([]byte, error)")
		g.P("CloseSend(ctx context.Context) error")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	default:
		g.P("Cancel(ctx context.Context) error")
	}
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeAdapterMethod) {
	facadeName := nativeRuntimeStreamFacadeName(serviceName, method)
	g.P("type ", facadeName, " struct {")
	g.P("handle rpcruntime.StreamHandle")
	g.P("}")
	g.P()
	g.P("func New", facadeName, "(handle rpcruntime.StreamHandle) ", facadeName, " {")
	g.P("return ", facadeName, "{handle: handle}")
	g.P("}")
	g.P()
	switch method.SessionKind {
	case SessionKindClient:
		renderRuntimeNativeStreamSend(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamFinish(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamCancel(g, streamRegistryName, method, facadeName)
	case SessionKindServer:
		renderRuntimeNativeStreamRecv(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamDone(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamCancel(g, streamRegistryName, method, facadeName)
	case SessionKindBidi:
		renderRuntimeNativeStreamSend(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamRecv(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamCloseSend(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamDone(g, streamRegistryName, method, facadeName)
		renderRuntimeNativeStreamCancel(g, streamRegistryName, method, facadeName)
	}
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeAdapterMethod) {
	facadeName := messageRuntimeStreamFacadeName(serviceName, method)
	g.P("type ", facadeName, " struct {")
	g.P("handle rpcruntime.StreamHandle")
	g.P("}")
	g.P()
	g.P("func New", facadeName, "(handle rpcruntime.StreamHandle) ", facadeName, " {")
	g.P("return ", facadeName, "{handle: handle}")
	g.P("}")
	g.P()
	switch method.SessionKind {
	case SessionKindClient:
		renderRuntimeMessageStreamSend(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamFinish(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamCancel(g, streamRegistryName, method, facadeName)
	case SessionKindServer:
		renderRuntimeMessageStreamRecv(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamDone(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamCancel(g, streamRegistryName, method, facadeName)
	case SessionKindBidi:
		renderRuntimeMessageStreamSend(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamRecv(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamCloseSend(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamDone(g, streamRegistryName, method, facadeName)
		renderRuntimeMessageStreamCancel(g, streamRegistryName, method, facadeName)
	}
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context", method.NativeArgs, ") error {")
	g.P("return rpcruntime.StreamRegistrySend[", method.SessionName, "](&", streamRegistryName, ", s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.Send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
	renderRuntimeNativeStreamResultVars(g, method)
	g.P("err := rpcruntime.StreamRegistryFinish[", method.SessionName, "](&", streamRegistryName, ", s.handle, func(session ", method.SessionName, ") error {")
	renderRuntimeNativeStreamCall(g, method, "Finish")
	g.P("})")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	renderRuntimeNativeStreamSuccessReturn(g, method)
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	renderRuntimeNativeStreamResultVars(g, method)
	g.P("err := rpcruntime.StreamRegistryReceive[", method.SessionName, "](&", streamRegistryName, ", s.handle, func(session ", method.SessionName, ") error {")
	renderRuntimeNativeStreamCall(g, method, "Recv")
	g.P("})")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	renderRuntimeNativeStreamSuccessReturn(g, method)
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamResultVars(g *protogen.GeneratedFile, method runtimeAdapterMethod) {
	for _, decl := range method.NativeVarDecls {
		g.P(decl)
	}
}

func renderRuntimeNativeStreamSuccessReturn(g *protogen.GeneratedFile, method runtimeAdapterMethod) {
	if method.NativeNames == "" {
		g.P("return nil")
		return
	}
	g.P("return ", method.NativeNames, ", nil")
}

func renderRuntimeNativeStreamCall(g *protogen.GeneratedFile, method runtimeAdapterMethod, operation string) {
	g.P("var callErr error")
	if method.NativeNames == "" {
		g.P("callErr = session.", operation, "(ctx)")
		g.P("return callErr")
		return
	}
	g.P(method.NativeNames, ", callErr = session.", operation, "(ctx)")
	g.P("return callErr")
}

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("return rpcruntime.StreamRegistryCloseSend[", method.SessionName, "](&", streamRegistryName, ", s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.CloseSend(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamDone(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Done(ctx context.Context) error {")
	g.P("return rpcruntime.StreamRegistryDone[", method.SessionName, "](&", streamRegistryName, ", s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.Done(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("return rpcruntime.StreamRegistryCancel[", method.SessionName, "](&", streamRegistryName, ", s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.Cancel(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Send(ctx context.Context, req []byte) error {")
	g.P("return rpcruntime.StreamRegistrySend[", sessionName, "](&", streamRegistryName, ", s.handle, func(session ", sessionName, ") error {")
	g.P("return session.Send(ctx, req)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("var resp []byte")
	g.P("err := rpcruntime.StreamRegistryFinish[", sessionName, "](&", streamRegistryName, ", s.handle, func(session ", sessionName, ") error {")
	g.P("var callErr error")
	g.P("resp, callErr = session.Finish(ctx)")
	g.P("return callErr")
	g.P("})")
	g.P("return resp, err")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("var resp []byte")
	g.P("err := rpcruntime.StreamRegistryReceive[", sessionName, "](&", streamRegistryName, ", s.handle, func(session ", sessionName, ") error {")
	g.P("var callErr error")
	g.P("resp, callErr = session.Recv(ctx)")
	g.P("return callErr")
	g.P("})")
	g.P("return resp, err")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("return rpcruntime.StreamRegistryCloseSend[", sessionName, "](&", streamRegistryName, ", s.handle, func(session ", sessionName, ") error {")
	g.P("return session.CloseSend(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamDone(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Done(ctx context.Context) error {")
	g.P("return rpcruntime.StreamRegistryDone[", sessionName, "](&", streamRegistryName, ", s.handle, func(session ", sessionName, ") error {")
	g.P("return session.Done(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, streamRegistryName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("return rpcruntime.StreamRegistryCancel[", sessionName, "](&", streamRegistryName, ", s.handle, func(session ", sessionName, ") error {")
	g.P("return session.Cancel(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func nativeRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "NativeStream"
}

func messageRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "MessageStream"
}

func renderRuntimeDirectMessageRegistrations(g *protogen.GeneratedFile, service ServicePlan) {
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		handlerName := service.GoName + "Handler"
		g.P("func Register", service.GoName, "ConnectHandler(handler ", handlerName, ") (rpcruntime.AdapterSnapshot[", handlerName, "], error) {")
		g.P("snapshot, err := ", lowerInitial(service.GoName), "ActiveSlot.Store(rpcruntime.ServerKindConnectHandler, rpcruntime.ServerContractMessage, handler)")
		g.P("if err != nil {")
		g.P("return rpcruntime.AdapterSnapshot[", handlerName, "]{}, err")
		g.P("}")
		g.P("return rpcruntime.AdapterSnapshot[", handlerName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: handler}, nil")
		g.P("}")
		g.P()
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		g.P("func Register", service.GoName, "GRPCServer(server ", serverName, ") (rpcruntime.AdapterSnapshot[", serverName, "], error) {")
		g.P("snapshot, err := ", lowerInitial(service.GoName), "ActiveSlot.Store(rpcruntime.ServerKindGRPCServer, rpcruntime.ServerContractMessage, server)")
		g.P("if err != nil {")
		g.P("return rpcruntime.AdapterSnapshot[", serverName, "]{}, err")
		g.P("}")
		g.P("return rpcruntime.AdapterSnapshot[", serverName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: server}, nil")
		g.P("}")
		g.P()
	}
}

func renderRuntimeBridge(g *protogen.GeneratedFile, service ServicePlan, bridgeTypeName, nativeAdapterName, messageAdapterName string, methods []runtimeAdapterMethod, codecEnabled bool) {
	serviceName := service.GoName
	g.P("type ", bridgeTypeName, " struct {")
	g.P("active *rpcruntime.ActiveServerSlot[any]")
	g.P("streams *rpcruntime.StreamRegistry[*rpcruntime.StreamEntry]")
	g.P("}")
	g.P()

	for _, method := range methods {
		if method.Streaming {
			renderRuntimeBridgeNativeStream(g, serviceName, bridgeTypeName, nativeAdapterName, method, codecEnabled)
			renderRuntimeBridgeMessageStream(g, serviceName, bridgeTypeName, nativeAdapterName, method, codecEnabled)
			renderRuntimeBridgeMessageSessionStarter(g, service, bridgeTypeName, messageAdapterName, method)
			continue
		}
		renderRuntimeBridgeNativeUnary(g, service, bridgeTypeName, nativeAdapterName, messageAdapterName, method, codecEnabled)
		renderRuntimeBridgeMessageUnary(g, service, bridgeTypeName, nativeAdapterName, messageAdapterName, method, codecEnabled)
	}
}

func renderRuntimeBridgeNativeUnary(g *protogen.GeneratedFile, service ServicePlan, bridgeTypeName, nativeAdapterName, messageAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	serviceName := service.GoName
	g.P("func (r ", bridgeTypeName, ") invokeNative", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
	for _, decl := range method.NativeVarDecls {
		g.P(decl)
	}
	g.P("var err error")
	g.P("snapshot, ok := r.active.Load()")
	g.P("if !ok {")
	g.P("err = rpcruntime.ErrNoActiveServer")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractNative:")
	g.P("adapter, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("err = ", serviceName, "NativeAdapterUnavailableErr")
	g.P("break")
	g.P("}")
	if method.NativeNames == "" {
		g.P("err = adapter.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	} else {
		g.P(method.NativeNames, ", err = adapter.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	}
	g.P("case rpcruntime.ServerContractMessage:")
	renderRuntimeBridgeNativeUnaryMessageActiveCall(g, service, messageAdapterName, method, codecEnabled)
	g.P("default:")
	g.P("err = ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	if method.NativeNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", method.NativeNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderRuntimeBridgeMessageUnary(g *protogen.GeneratedFile, service ServicePlan, bridgeTypeName, nativeAdapterName, messageAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	serviceName := service.GoName
	g.P("func (r ", bridgeTypeName, ") invokeMessage", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
	g.P("snapshot, ok := r.active.Load()")
	g.P("if !ok {")
	g.P("return nil, rpcruntime.ErrNoActiveServer")
	g.P("}")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractMessage:")
	renderRuntimeBridgeMessageUnaryActiveCall(g, service, messageAdapterName, method)
	g.P("case rpcruntime.ServerContractNative:")
	g.P("adapter, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("return nil, ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	if codecEnabled {
		g.P("var resp []byte")
		g.P("err := ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
		if method.NativeNames == "" {
			g.P("callErr := adapter.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		} else {
			g.P(method.NativeNames, ", callErr := adapter.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		}
		g.P("if callErr != nil {")
		g.P("return callErr")
		g.P("}")
		g.P("messageResp, err := ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("resp = messageResp")
		g.P("return nil")
		g.P("})")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return resp, nil")
	} else {
		g.P("return nil, ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return nil, ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeBridgeMessageUnaryActiveCall(g *protogen.GeneratedFile, service ServicePlan, messageAdapterName string, method runtimeAdapterMethod) {
	serviceName := service.GoName
	g.P("switch snapshot.Kind {")
	g.P("case rpcruntime.ServerKindCGOMessage, rpcruntime.ServerKindConnectRemote, rpcruntime.ServerKindGRPCRemote:")
	g.P("adapter, ok := snapshot.Adapter.(", messageAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	g.P("return adapter.", method.AdapterName, "Message(ctx, req)")
	renderRuntimeBridgeMessageUnaryDirectCases(g, service, method, "req", "return nil, ")
	g.P("default:")
	g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
}

func renderRuntimeBridgeNativeUnaryMessageActiveCall(g *protogen.GeneratedFile, service ServicePlan, messageAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	serviceName := service.GoName
	if !codecEnabled {
		g.P("err = ", serviceName, "NativeMessageConverterUnavailableErr")
		return
	}
	g.P("messageReq, convertErr := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
	g.P("if convertErr != nil {")
	g.P("err = convertErr")
	g.P("break")
	g.P("}")
	g.P("var messageResp []byte")
	g.P("switch snapshot.Kind {")
	g.P("case rpcruntime.ServerKindCGOMessage, rpcruntime.ServerKindConnectRemote, rpcruntime.ServerKindGRPCRemote:")
	g.P("adapter, ok := snapshot.Adapter.(", messageAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("err = ", serviceName, "MessageAdapterUnavailableErr")
	g.P("break")
	g.P("}")
	g.P("messageResp, err = adapter.", method.AdapterName, "Message(ctx, messageReq)")
	renderRuntimeBridgeNativeUnaryDirectCases(g, service, method)
	g.P("default:")
	g.P("err = ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("break")
	g.P("}")
	if method.NativeNames == "" {
		g.P("err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
	} else {
		g.P(method.NativeNames, ", err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
	}
}

func renderRuntimeBridgeMessageUnaryDirectCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, reqExpr string, errPrefix string) {
	serviceName := service.GoName
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		handlerName := service.GoName + "Handler"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindConnectHandler:")
		g.P("handler, ok := snapshot.Adapter.(", handlerName, ")")
		g.P("if !ok || handler == nil {")
		g.P(errPrefix, serviceName, "MessageAdapterUnavailableErr")
		g.P("}")
		g.P("messageReq := new(", reqType, ")")
		g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: connect handler request protobuf unmarshal failed: %w", err)`)
		g.P("}")
		g.P("messageResp, err := handler.", method.MethodGoName, "(ctx, messageReq)")
		g.P("if err != nil {")
		g.P(errPrefix, "err")
		g.P("}")
		g.P("resp, err := proto.Marshal(messageResp)")
		g.P("if err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: connect handler response protobuf marshal failed: %w", err)`)
		g.P("}")
		g.P("return resp, nil")
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindGRPCServer:")
		g.P("server, ok := snapshot.Adapter.(", serverName, ")")
		g.P("if !ok || server == nil {")
		g.P(errPrefix, serviceName, "MessageAdapterUnavailableErr")
		g.P("}")
		g.P("messageReq := new(", reqType, ")")
		g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: grpc server request protobuf unmarshal failed: %w", err)`)
		g.P("}")
		g.P("messageResp, err := server.", method.MethodGoName, "(ctx, messageReq)")
		g.P("if err != nil {")
		g.P(errPrefix, "err")
		g.P("}")
		g.P("resp, err := proto.Marshal(messageResp)")
		g.P("if err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: grpc server response protobuf marshal failed: %w", err)`)
		g.P("}")
		g.P("return resp, nil")
	}
}

func renderRuntimeBridgeNativeUnaryDirectCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		handlerName := service.GoName + "Handler"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindConnectHandler:")
		g.P("handler, ok := snapshot.Adapter.(", handlerName, ")")
		g.P("if !ok || handler == nil {")
		g.P("err = ", service.GoName, "MessageAdapterUnavailableErr")
		g.P("break")
		g.P("}")
		g.P("directReq := new(", reqType, ")")
		g.P("if err = proto.Unmarshal(messageReq, directReq); err != nil {")
		g.P(`err = fmt.Errorf("rpccgo: connect handler request protobuf unmarshal failed: %w", err)`)
		g.P("break")
		g.P("}")
		g.P("directResp, callErr := handler.", method.MethodGoName, "(ctx, directReq)")
		g.P("if callErr != nil {")
		g.P("err = callErr")
		g.P("break")
		g.P("}")
		g.P("messageResp, err = proto.Marshal(directResp)")
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindGRPCServer:")
		g.P("server, ok := snapshot.Adapter.(", serverName, ")")
		g.P("if !ok || server == nil {")
		g.P("err = ", service.GoName, "MessageAdapterUnavailableErr")
		g.P("break")
		g.P("}")
		g.P("directReq := new(", reqType, ")")
		g.P("if err = proto.Unmarshal(messageReq, directReq); err != nil {")
		g.P(`err = fmt.Errorf("rpccgo: grpc server request protobuf unmarshal failed: %w", err)`)
		g.P("break")
		g.P("}")
		g.P("directResp, callErr := server.", method.MethodGoName, "(ctx, directReq)")
		g.P("if callErr != nil {")
		g.P("err = callErr")
		g.P("break")
		g.P("}")
		g.P("messageResp, err = proto.Marshal(directResp)")
	}
}

func methodForRuntimeService(service ServicePlan, method runtimeAdapterMethod) MethodPlan {
	for _, candidate := range service.Methods {
		if candidate.GoName == method.MethodGoName {
			return candidate
		}
	}
	return MethodPlan{GoName: method.MethodGoName}
}

func renderRuntimeBridgeNativeStream(g *protogen.GeneratedFile, serviceName, bridgeTypeName, nativeAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	switch method.SessionKind {
	case SessionKindClient:
		g.P("func (r ", bridgeTypeName, ") startNative", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	case SessionKindServer:
		g.P("func (r ", bridgeTypeName, ") startNative", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
	case SessionKindBidi:
		g.P("func (r ", bridgeTypeName, ") startNative", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	default:
		return
	}
	g.P("snapshot, ok := r.active.Load()")
	g.P("if !ok {")
	g.P("return 0, rpcruntime.ErrNoActiveServer")
	g.P("}")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractNative:")
	g.P("adapter, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("return 0, ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("session, err := adapter.", method.AdapterName, "(ctx)")
	case SessionKindServer:
		g.P("session, err := adapter.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	}
	g.P("if err != nil {")
	g.P("return 0, err")
	g.P("}")
	g.P("return r.streams.Create(rpcruntime.NewStreamEntry(session))")
	g.P("case rpcruntime.ServerContractMessage:")
	if codecEnabled {
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("messageSession, err := r.start", method.MethodGoName, "MessageSession(ctx, snapshot)")
		case SessionKindServer:
			g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
			g.P("if err != nil {")
			g.P("return 0, err")
			g.P("}")
			g.P("messageSession, err := r.start", method.MethodGoName, "MessageSession(ctx, snapshot, messageReq)")
		}
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return r.streams.Create(rpcruntime.NewStreamEntry(&", messageToNativeStreamWrapperName(serviceName, method), "{message: messageSession}))")
	} else {
		g.P("return 0, ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return 0, ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("}")
	g.P()
	if codecEnabled {
		renderMessageToNativeStreamWrapper(g, serviceName, method)
	}
}

func renderRuntimeBridgeMessageStream(g *protogen.GeneratedFile, serviceName, bridgeTypeName, nativeAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	switch method.SessionKind {
	case SessionKindClient:
		g.P("func (r ", bridgeTypeName, ") startMessage", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	case SessionKindServer:
		g.P("func (r ", bridgeTypeName, ") startMessage", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
	case SessionKindBidi:
		g.P("func (r ", bridgeTypeName, ") startMessage", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	default:
		return
	}
	g.P("snapshot, ok := r.active.Load()")
	g.P("if !ok {")
	g.P("return 0, rpcruntime.ErrNoActiveServer")
	g.P("}")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractMessage:")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("session, err := r.start", method.MethodGoName, "MessageSession(ctx, snapshot)")
	case SessionKindServer:
		g.P("session, err := r.start", method.MethodGoName, "MessageSession(ctx, snapshot, req)")
	}
	g.P("if err != nil {")
	g.P("return 0, err")
	g.P("}")
	g.P("return r.streams.Create(rpcruntime.NewStreamEntry(session))")
	g.P("case rpcruntime.ServerContractNative:")
	g.P("adapter, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("return 0, ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	if codecEnabled {
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("nativeSession, err := adapter.", method.AdapterName, "(ctx)")
			g.P("if err != nil {")
			g.P("return 0, err")
			g.P("}")
			g.P("return r.streams.Create(rpcruntime.NewStreamEntry(&", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}))")
		case SessionKindServer:
			g.P("var session any")
			g.P("err := ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
			g.P("nativeSession, err := adapter.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("session = &", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}")
			g.P("return nil")
			g.P("})")
			g.P("if err != nil {")
			g.P("return 0, err")
			g.P("}")
			g.P("return r.streams.Create(rpcruntime.NewStreamEntry(session))")
		}
	} else {
		g.P("return 0, ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return 0, ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("}")
	g.P()
	if codecEnabled {
		renderNativeToMessageStreamWrapper(g, serviceName, method)
	}
}

func renderRuntimeBridgeMessageSessionStarter(g *protogen.GeneratedFile, service ServicePlan, bridgeTypeName, messageAdapterName string, method runtimeAdapterMethod) {
	serviceName := service.GoName
	sessionName := methodMessageSessionName(method)
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("func (r ", bridgeTypeName, ") start", method.MethodGoName, "MessageSession(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[any]) (", sessionName, ", error) {")
	case SessionKindServer:
		g.P("func (r ", bridgeTypeName, ") start", method.MethodGoName, "MessageSession(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[any], req []byte) (", sessionName, ", error) {")
	default:
		return
	}
	g.P("switch snapshot.Kind {")
	g.P("case rpcruntime.ServerKindCGOMessage, rpcruntime.ServerKindConnectRemote, rpcruntime.ServerKindGRPCRemote:")
	g.P("adapter, ok := snapshot.Adapter.(", messageAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("return adapter.Start", method.MethodGoName, "Message(ctx)")
	case SessionKindServer:
		g.P("return adapter.Start", method.MethodGoName, "Message(ctx, req)")
	}
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		handlerName := service.GoName + "Handler"
		g.P("case rpcruntime.ServerKindConnectHandler:")
		g.P("handler, ok := snapshot.Adapter.(", handlerName, ")")
		g.P("if !ok || handler == nil {")
		g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
		g.P("}")
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("return new", connectDirectMessageSessionName(serviceName, method), "(ctx, handler), nil")
		case SessionKindServer:
			g.P("return new", connectDirectMessageSessionName(serviceName, method), "(ctx, handler, req)")
		}
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		g.P("case rpcruntime.ServerKindGRPCServer:")
		g.P("server, ok := snapshot.Adapter.(", serverName, ")")
		g.P("if !ok || server == nil {")
		g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
		g.P("}")
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("return new", grpcDirectMessageSessionName(serviceName, method), "(ctx, server), nil")
		case SessionKindServer:
			g.P("return new", grpcDirectMessageSessionName(serviceName, method), "(ctx, server, req)")
		}
	}
	g.P("default:")
	g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	g.P("}")
	g.P()
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		renderConnectDirectMessageSession(g, service, method)
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		renderGRPCDirectMessageSession(g, service, method)
	}
}

func renderConnectDirectMessageSession(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	methodPlan := methodForRuntimeService(service, method)
	wrapperName := connectDirectMessageSessionName(service.GoName, method)
	resultName := wrapperName + "Result"
	reqType := qualifiedMethodType(g, methodPlan.Request)
	respType := qualifiedMethodType(g, methodPlan.Response)
	handlerName := service.GoName + "Handler"
	g.P("type ", resultName, " struct {")
	g.P("data []byte")
	g.P("err error")
	g.P("terminal bool")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	if method.SessionKind == SessionKindClient || method.SessionKind == SessionKindBidi {
		g.P("requests chan []byte")
		g.P("closeRequests sync.Once")
	}
	if method.SessionKind == SessionKindClient {
		g.P("result chan ", resultName)
	} else {
		g.P("responses chan ", resultName)
	}
	g.P("}")
	g.P()
	switch method.SessionKind {
	case SessionKindClient:
		renderConnectDirectClientStreamSession(g, method, wrapperName, resultName, reqType, handlerName)
	case SessionKindServer:
		renderConnectDirectServerStreamSession(g, method, wrapperName, resultName, reqType, respType, handlerName)
	case SessionKindBidi:
		renderConnectDirectBidiStreamSession(g, method, wrapperName, resultName, reqType, respType, handlerName)
	}
}

func renderConnectDirectClientStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, resultName, reqType, handlerName string) {
	g.P("func new", wrapperName, "(ctx context.Context, handler ", handlerName, ") *", wrapperName, " {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", wrapperName, "{ctx: streamCtx, cancel: cancel, requests: make(chan []byte), result: make(chan ", resultName, ", 1)}")
	g.P("go func() {")
	g.P("conn := &rpcruntime.ConnectStreamingHandlerConn{ReceiveFunc: func(msg any) error {")
	g.P("data, ok := <-session.requests")
	g.P("if !ok {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return proto.Unmarshal(data, msg.(proto.Message))")
	g.P("}}")
	g.P("resp, err := handler.", method.MethodGoName, "(streamCtx, rpcruntime.NewConnectClientStream[", reqType, "](conn))")
	g.P("if err != nil {")
	g.P("session.result <- ", resultName, "{err: err, terminal: true}")
	g.P("return")
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("session.result <- ", resultName, "{data: data, err: err, terminal: true}")
	g.P("}()")
	g.P("return session")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case s.requests <- req:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("s.closeRequests.Do(func() { close(s.requests) })")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case result := <-s.result:")
	g.P("s.cancel()")
	g.P("return result.data, result.err")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("s.cancel()")
	g.P("s.closeRequests.Do(func() { close(s.requests) })")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderConnectDirectServerStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, resultName, reqType, respType, handlerName string) {
	g.P("func new", wrapperName, "(ctx context.Context, handler ", handlerName, ", req []byte) (*", wrapperName, ", error) {")
	g.P("messageReq := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, messageReq); err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: connect handler stream request protobuf unmarshal failed: %w\", err)")
	g.P("}")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", wrapperName, "{ctx: streamCtx, cancel: cancel, responses: make(chan ", resultName, ", 1)}")
	g.P("go func() {")
	g.P("conn := &rpcruntime.ConnectStreamingHandlerConn{SendFunc: func(msg any) error {")
	g.P("resp, ok := msg.(*", respType, ")")
	g.P("if !ok {")
	g.P("return fmt.Errorf(\"rpccgo: connect handler stream response type mismatch\")")
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("select {")
	g.P("case <-streamCtx.Done():")
	g.P("return streamCtx.Err()")
	g.P("case session.responses <- ", resultName, "{data: data}:")
	g.P("return nil")
	g.P("}")
	g.P("}}")
	g.P("err := handler.", method.MethodGoName, "(streamCtx, messageReq, rpcruntime.NewConnectServerStream[", respType, "](conn))")
	g.P("session.responses <- ", resultName, "{err: err, terminal: true}")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()
	renderConnectDirectRecvDoneCancel(g, wrapperName)
}

func renderConnectDirectBidiStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, resultName, reqType, respType, handlerName string) {
	g.P("func new", wrapperName, "(ctx context.Context, handler ", handlerName, ") *", wrapperName, " {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", wrapperName, "{ctx: streamCtx, cancel: cancel, requests: make(chan []byte), responses: make(chan ", resultName, ", 1)}")
	g.P("go func() {")
	g.P("conn := &rpcruntime.ConnectStreamingHandlerConn{")
	g.P("ReceiveFunc: func(msg any) error {")
	g.P("data, ok := <-session.requests")
	g.P("if !ok {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return proto.Unmarshal(data, msg.(proto.Message))")
	g.P("},")
	g.P("SendFunc: func(msg any) error {")
	g.P("resp, ok := msg.(*", respType, ")")
	g.P("if !ok {")
	g.P("return fmt.Errorf(\"rpccgo: connect handler bidi response type mismatch\")")
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("select {")
	g.P("case <-streamCtx.Done():")
	g.P("return streamCtx.Err()")
	g.P("case session.responses <- ", resultName, "{data: data}:")
	g.P("return nil")
	g.P("}")
	g.P("},")
	g.P("}")
	g.P("err := handler.", method.MethodGoName, "(streamCtx, rpcruntime.NewConnectBidiStream[", reqType, ", ", respType, "](conn))")
	g.P("session.responses <- ", resultName, "{err: err, terminal: true}")
	g.P("}()")
	g.P("return session")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case s.requests <- req:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("s.closeRequests.Do(func() { close(s.requests) })")
	g.P("return nil")
	g.P("}")
	g.P()
	renderConnectDirectRecvDoneCancel(g, wrapperName)
}

func renderConnectDirectRecvDoneCancel(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case result := <-s.responses:")
	g.P("if result.terminal {")
	g.P("s.cancel()")
	g.P("if result.err != nil {")
	g.P("return nil, result.err")
	g.P("}")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("return result.data, result.err")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("s.cancel()")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("s.cancel()")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderGRPCDirectMessageSession(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	methodPlan := methodForRuntimeService(service, method)
	wrapperName := grpcDirectMessageSessionName(service.GoName, method)
	resultName := wrapperName + "Result"
	reqType := qualifiedMethodType(g, methodPlan.Request)
	respType := qualifiedMethodType(g, methodPlan.Response)
	serverName := service.GoName + "Server"
	g.P("type ", resultName, " struct {")
	g.P("data []byte")
	g.P("err error")
	g.P("terminal bool")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	if method.SessionKind == SessionKindClient || method.SessionKind == SessionKindBidi {
		g.P("requests chan []byte")
		g.P("closeRequests sync.Once")
	}
	if method.SessionKind == SessionKindClient {
		g.P("result chan ", resultName)
		g.P("resultOnce sync.Once")
	} else {
		g.P("responses chan ", resultName)
	}
	g.P("header metadata.MD")
	g.P("trailer metadata.MD")
	g.P("}")
	g.P()
	switch method.SessionKind {
	case SessionKindClient:
		renderGRPCDirectClientStreamSession(g, method, wrapperName, resultName, reqType, respType, serverName)
	case SessionKindServer:
		renderGRPCDirectServerStreamSession(g, method, wrapperName, resultName, reqType, respType, serverName)
	case SessionKindBidi:
		renderGRPCDirectBidiStreamSession(g, method, wrapperName, resultName, reqType, respType, serverName)
	}
}

func renderGRPCDirectClientStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, resultName, reqType, respType, serverName string) {
	g.P("func new", wrapperName, "(ctx context.Context, server ", serverName, ") *", wrapperName, " {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", wrapperName, "{ctx: streamCtx, cancel: cancel, requests: make(chan []byte), result: make(chan ", resultName, ", 1)}")
	g.P("go func() {")
	g.P("err := server.", method.MethodGoName, "(session)")
	g.P("if err != nil {")
	g.P("session.deliver(", resultName, "{err: err, terminal: true})")
	g.P("return")
	g.P("}")
	g.P("session.deliver(", resultName, `{err: fmt.Errorf("rpccgo: grpc direct client stream completed without SendAndClose"), terminal: true})`)
	g.P("}()")
	g.P("return session")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") deliver(result ", resultName, ") {")
	g.P("s.resultOnce.Do(func() {")
	g.P("s.result <- result")
	g.P("})")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Recv() (*", reqType, ", error) {")
	g.P("select {")
	g.P("case <-s.ctx.Done():")
	g.P("return nil, s.ctx.Err()")
	g.P("case data, ok := <-s.requests:")
	g.P("if !ok {")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("msg := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(data, msg); err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return msg, nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") RecvMsg(m any) error {")
	g.P("msg, err := s.Recv()")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("typed, ok := m.(*", reqType, ")")
	g.P("if !ok || typed == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct client stream request type mismatch")`)
	g.P("}")
	g.P("*typed = *msg")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SendAndClose(resp *", respType, ") error {")
	g.P("if resp == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct client stream response is nil")`)
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("s.deliver(", resultName, "{data: data, terminal: true})")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SendMsg(m any) error {")
	g.P("typed, ok := m.(*", respType, ")")
	g.P("if !ok || typed == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct client stream response type mismatch")`)
	g.P("}")
	g.P("return s.SendAndClose(typed)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SetHeader(md metadata.MD) error {")
	g.P("if md == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.header == nil {")
	g.P("s.header = md.Copy()")
	g.P("return nil")
	g.P("}")
	g.P("s.header = metadata.Join(s.header, md)")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SendHeader(md metadata.MD) error {")
	g.P("return s.SetHeader(md)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SetTrailer(md metadata.MD) {")
	g.P("if md == nil {")
	g.P("return")
	g.P("}")
	g.P("if s.trailer == nil {")
	g.P("s.trailer = md.Copy()")
	g.P("return")
	g.P("}")
	g.P("s.trailer = metadata.Join(s.trailer, md)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Context() context.Context { return s.ctx }")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case s.requests <- req:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("s.closeRequests.Do(func() { close(s.requests) })")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case result := <-s.result:")
	g.P("s.cancel()")
	g.P("return result.data, result.err")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("s.cancel()")
	g.P("s.closeRequests.Do(func() { close(s.requests) })")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderGRPCDirectServerStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, resultName, reqType, respType, serverName string) {
	g.P("func new", wrapperName, "(ctx context.Context, server ", serverName, ", req []byte) (*", wrapperName, ", error) {")
	g.P("messageReq := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, messageReq); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: grpc direct server stream request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", wrapperName, "{ctx: streamCtx, cancel: cancel, responses: make(chan ", resultName, ", 1)}")
	g.P("go func() {")
	g.P("err := server.", method.MethodGoName, "(messageReq, session)")
	g.P("session.responses <- ", resultName, "{err: err, terminal: true}")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(resp *", respType, ") error {")
	g.P("if resp == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct server stream response is nil")`)
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("select {")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case s.responses <- ", resultName, "{data: data}:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SendMsg(m any) error {")
	g.P("typed, ok := m.(*", respType, ")")
	g.P("if !ok || typed == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct server stream response type mismatch")`)
	g.P("}")
	g.P("return s.Send(typed)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") RecvMsg(m any) error {")
	g.P("return io.EOF")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SetHeader(md metadata.MD) error {")
	g.P("if md == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.header == nil {")
	g.P("s.header = md.Copy()")
	g.P("return nil")
	g.P("}")
	g.P("s.header = metadata.Join(s.header, md)")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SendHeader(md metadata.MD) error {")
	g.P("return s.SetHeader(md)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SetTrailer(md metadata.MD) {")
	g.P("if md == nil {")
	g.P("return")
	g.P("}")
	g.P("if s.trailer == nil {")
	g.P("s.trailer = md.Copy()")
	g.P("return")
	g.P("}")
	g.P("s.trailer = metadata.Join(s.trailer, md)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Context() context.Context { return s.ctx }")
	g.P()
	renderConnectDirectRecvDoneCancel(g, wrapperName)
}

func renderGRPCDirectBidiStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, resultName, reqType, respType, serverName string) {
	streamName := wrapperName + "GRPCStream"
	g.P("type ", streamName, " struct {")
	g.P("session *", wrapperName)
	g.P("}")
	g.P()
	g.P("func new", wrapperName, "(ctx context.Context, server ", serverName, ") *", wrapperName, " {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", wrapperName, "{ctx: streamCtx, cancel: cancel, requests: make(chan []byte), responses: make(chan ", resultName, ", 1)}")
	g.P("go func() {")
	g.P("err := server.", method.MethodGoName, "(&", streamName, "{session: session})")
	g.P("session.responses <- ", resultName, "{err: err, terminal: true}")
	g.P("}()")
	g.P("return session")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") Recv() (*", reqType, ", error) {")
	g.P("select {")
	g.P("case <-s.session.ctx.Done():")
	g.P("return nil, s.session.ctx.Err()")
	g.P("case data, ok := <-s.session.requests:")
	g.P("if !ok {")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("msg := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(data, msg); err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return msg, nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") RecvMsg(m any) error {")
	g.P("msg, err := s.Recv()")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("typed, ok := m.(*", reqType, ")")
	g.P("if !ok || typed == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct bidi request type mismatch")`)
	g.P("}")
	g.P("*typed = *msg")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") Send(resp *", respType, ") error {")
	g.P("if resp == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct bidi response is nil")`)
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("select {")
	g.P("case <-s.session.ctx.Done():")
	g.P("return s.session.ctx.Err()")
	g.P("case s.session.responses <- ", resultName, "{data: data}:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") SendMsg(m any) error {")
	g.P("typed, ok := m.(*", respType, ")")
	g.P("if !ok || typed == nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc direct bidi response type mismatch")`)
	g.P("}")
	g.P("return s.Send(typed)")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") SetHeader(md metadata.MD) error {")
	g.P("return s.session.SetHeader(md)")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") SendHeader(md metadata.MD) error {")
	g.P("return s.session.SendHeader(md)")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") SetTrailer(md metadata.MD) {")
	g.P("s.session.SetTrailer(md)")
	g.P("}")
	g.P()
	g.P("func (s *", streamName, ") Context() context.Context { return s.session.ctx }")
	g.P()
	g.P("func (s *", wrapperName, ") SetHeader(md metadata.MD) error {")
	g.P("if md == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.header == nil {")
	g.P("s.header = md.Copy()")
	g.P("return nil")
	g.P("}")
	g.P("s.header = metadata.Join(s.header, md)")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SendHeader(md metadata.MD) error {")
	g.P("return s.SetHeader(md)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") SetTrailer(md metadata.MD) {")
	g.P("if md == nil {")
	g.P("return")
	g.P("}")
	g.P("if s.trailer == nil {")
	g.P("s.trailer = md.Copy()")
	g.P("return")
	g.P("}")
	g.P("s.trailer = metadata.Join(s.trailer, md)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case s.requests <- req:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("s.closeRequests.Do(func() { close(s.requests) })")
	g.P("return nil")
	g.P("}")
	g.P()
	renderConnectDirectRecvDoneCancel(g, wrapperName)
}

func renderRuntimeNativeEntrypoints(g *protogen.GeneratedFile, serviceName, adapterName, bridgeName string, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func Invoke", serviceName, "Native", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
		g.P("return ", bridgeName, ".invokeNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("}")
		g.P()
	}

	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("func Start", serviceName, "Native", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
			g.P("return ", bridgeName, ".startNative", method.MethodGoName, "(ctx)")
		case SessionKindServer:
			g.P("func Start", serviceName, "Native", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
			g.P("return ", bridgeName, ".startNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		}
		g.P("}")
		g.P()
	}

	g.P("func Register", serviceName, "CGONativeActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return register", serviceName, "ActiveServer(kind, adapter)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageEntrypoints(g *protogen.GeneratedFile, serviceName, adapterName, bridgeName string, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func Invoke", serviceName, "Message", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("return ", bridgeName, ".invokeMessage", method.MethodGoName, "(ctx, req)")
		g.P("}")
		g.P()
	}

	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("func Start", serviceName, "Message", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
			g.P("return ", bridgeName, ".startMessage", method.MethodGoName, "(ctx)")
		case SessionKindServer:
			g.P("func Start", serviceName, "Message", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
			g.P("return ", bridgeName, ".startMessage", method.MethodGoName, "(ctx, req)")
		}
		g.P("}")
		g.P()
	}

	g.P("func Register", serviceName, "CGOMessageActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return register", serviceName, "MessageActiveServer(kind, adapter)")
	g.P("}")
	g.P()
}

func renderNativeToMessageStreamWrapper(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	wrapperName := nativeToMessageStreamWrapperName(serviceName, method)
	g.P("type ", wrapperName, " struct {")
	g.P("native ", method.SessionName)
	g.P("}")
	g.P()
	switch method.SessionKind {
	case SessionKindClient:
		renderNativeToMessageSend(g, serviceName, method, wrapperName)
		renderNativeToMessageFinish(g, serviceName, method, wrapperName)
		renderNativeToMessageCancel(g, wrapperName)
	case SessionKindServer:
		renderNativeToMessageRecv(g, serviceName, method, wrapperName)
		renderNativeToMessageDone(g, wrapperName)
		renderNativeToMessageCancel(g, wrapperName)
	case SessionKindBidi:
		renderNativeToMessageSend(g, serviceName, method, wrapperName)
		renderNativeToMessageRecv(g, serviceName, method, wrapperName)
		renderNativeToMessageCloseSend(g, wrapperName)
		renderNativeToMessageDone(g, wrapperName)
		renderNativeToMessageCancel(g, wrapperName)
	}
}

func renderNativeToMessageSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("return ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
	g.P("return s.native.Send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("})")
	g.P("}")
	g.P()
}

func renderNativeToMessageFinish(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) ([]byte, error) {")
	if method.NativeNames == "" {
		g.P("err := s.native.Finish(ctx)")
	} else {
		g.P(method.NativeNames, ", err := s.native.Finish(ctx)")
	}
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
	g.P("}")
	g.P()
}

func renderNativeToMessageRecv(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) ([]byte, error) {")
	if method.NativeNames == "" {
		g.P("err := s.native.Recv(ctx)")
	} else {
		g.P(method.NativeNames, ", err := s.native.Recv(ctx)")
	}
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
	g.P("}")
	g.P()
}

func renderNativeToMessageCloseSend(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("return s.native.CloseSend(ctx)")
	g.P("}")
	g.P()
}

func renderNativeToMessageDone(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("return s.native.Done(ctx)")
	g.P("}")
	g.P()
}

func renderNativeToMessageCancel(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("return s.native.Cancel(ctx)")
	g.P("}")
	g.P()
}

func renderMessageToNativeStreamWrapper(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	wrapperName := messageToNativeStreamWrapperName(serviceName, method)
	g.P("type ", wrapperName, " struct {")
	g.P("message ", methodMessageSessionName(method))
	g.P("}")
	g.P()
	switch method.SessionKind {
	case SessionKindClient:
		renderMessageToNativeSend(g, serviceName, method, wrapperName)
		renderMessageToNativeFinish(g, serviceName, method, wrapperName)
		renderMessageToNativeCancel(g, wrapperName)
	case SessionKindServer:
		renderMessageToNativeRecv(g, serviceName, method, wrapperName)
		renderMessageToNativeDone(g, wrapperName)
		renderMessageToNativeCancel(g, wrapperName)
	case SessionKindBidi:
		renderMessageToNativeSend(g, serviceName, method, wrapperName)
		renderMessageToNativeRecv(g, serviceName, method, wrapperName)
		renderMessageToNativeCloseSend(g, wrapperName)
		renderMessageToNativeDone(g, wrapperName)
		renderMessageToNativeCancel(g, wrapperName)
	}
}

func renderMessageToNativeSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Send(ctx context.Context", method.NativeArgs, ") error {")
	g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("return s.message.Send(ctx, messageReq)")
	g.P("}")
	g.P()
}

func renderMessageToNativeFinish(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
	g.P("messageResp, err := s.message.Finish(ctx)")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("return ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
	g.P("}")
	g.P()
}

func renderMessageToNativeRecv(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	g.P("messageResp, err := s.message.Recv(ctx)")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("return ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
	g.P("}")
	g.P()
}

func renderMessageToNativeCloseSend(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("return s.message.CloseSend(ctx)")
	g.P("}")
	g.P()
}

func renderMessageToNativeDone(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("return s.message.Done(ctx)")
	g.P("}")
	g.P()
}

func renderMessageToNativeCancel(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("return s.message.Cancel(ctx)")
	g.P("}")
	g.P()
}

func methodMessageSessionName(method runtimeAdapterMethod) string {
	return strings.Replace(method.SessionName, "NativeStreamSession", "MessageStreamSession", 1)
}

func nativeToMessageStreamWrapperName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "NativeToMessageStreamSession"
}

func messageToNativeStreamWrapperName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "MessageToNativeStreamSession"
}

func grpcDirectMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "GRPCDirectMessageStreamSession"
}

func connectDirectMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "ConnectDirectMessageStreamSession"
}
