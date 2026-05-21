package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	streamingMethods := runtimeStreamingMethods(runtimeMethods)
	codecEnabled := service.CodecEnabled

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	adapterName := service.GoName + "NativeAdapter"
	messageAdapterName := service.GoName + "MessageAdapter"
	activeAdapterName := service.GoName + "ActiveAdapter"
	dispatcherName := lowerInitial(service.GoName) + "Dispatcher"

	g.P("type ", adapterName, " interface {")
	for _, method := range runtimeMethods {
		g.P(method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult)
	}
	g.P("}")
	g.P()

	renderRuntimeMessageAdapter(g, service, messageAdapterName, runtimeMethods)
	renderRuntimeActiveAdapter(g, activeAdapterName, adapterName, messageAdapterName)

	for _, method := range streamingMethods {
		renderRuntimeSessionInterface(g, method)
		renderRuntimeMessageSessionInterface(g, method)
	}
	for _, method := range streamingMethods {
		renderRuntimeNativeStreamFacade(g, service.GoName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, method)
	}

	routerName := lowerInitial(service.GoName) + "Router"
	routerTypeName := lowerInitial(service.GoName) + "ActiveRouter"

	g.P("var ", dispatcherName, " rpcruntime.Dispatcher[", activeAdapterName, "]")
	g.P("var ", routerName, " = ", routerTypeName, "{dispatcher: &", dispatcherName, "}")
	g.P("var ", service.GoName, `NativeMessageConverterUnavailableErr = errors.New("rpccgo: native/message converter is not enabled")`)
	g.P("var ", service.GoName, `NativeAdapterUnavailableErr = errors.New("rpccgo: native adapter is unavailable")`)
	g.P("var ", service.GoName, `MessageAdapterUnavailableErr = errors.New("rpccgo: message adapter is unavailable")`)
	g.P("var ", service.GoName, `UnknownActiveContractErr = errors.New("rpccgo: unknown active server contract")`)
	g.P()

	g.P("func ", service.GoName, "DispatcherForRuntime() *rpcruntime.Dispatcher[", activeAdapterName, "] {")
	g.P("return &", dispatcherName)
	g.P("}")
	g.P()

	g.P("func register", service.GoName, "ActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("snapshot, err := ", dispatcherName, ".Register(kind, rpcruntime.ServerContractNative, ", activeAdapterName, "{Native: adapter})")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil")
	g.P("}")
	g.P()

	g.P("func register", service.GoName, "MessageActiveServer(kind rpcruntime.ServerKind, adapter ", messageAdapterName, ") (rpcruntime.AdapterSnapshot[", messageAdapterName, "], error) {")
	g.P("snapshot, err := ", dispatcherName, ".Register(kind, rpcruntime.ServerContractMessage, ", activeAdapterName, "{Message: adapter})")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil")
	g.P("}")
	g.P()

	renderRuntimeActiveRouter(g, service.GoName, routerTypeName, activeAdapterName, runtimeMethods, codecEnabled)
	renderRuntimeCGOBridge(g, service.GoName, adapterName, activeAdapterName, dispatcherName, runtimeMethods, codecEnabled)
	renderRuntimeMessageCGOBridge(g, service.GoName, messageAdapterName, activeAdapterName, dispatcherName, runtimeMethods, codecEnabled)

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
	shape := method.RenderShape
	nativeFields := shape.Conversion.MessageToNative.Native.Request
	responseFields := shape.Conversion.MessageToNative.Native.Response
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
		SessionKind:    shape.Session.Kind,
		Streaming:      shape.Session.Kind != SessionKindNone,
	}
	if !rendered.Streaming {
		rendered.AdapterArgs = nativeArgs
		rendered.AdapterResult = " (" + nativeReturns + ")"
		return rendered, nil
	}
	rendered.AdapterResult = " (" + sessionName + ", error)"
	if hasRenderOperation(shape.Session, SessionOperationStart) && shape.Session.Kind == SessionKindServer {
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

func hasRenderOperation(session SessionRenderPlan, kind SessionOperationKind) bool {
	for _, op := range session.Operations {
		if op.Kind == kind && op.Enabled {
			return true
		}
	}
	return false
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

func renderRuntimeActiveAdapter(g *protogen.GeneratedFile, activeAdapterName, nativeAdapterName, messageAdapterName string) {
	g.P("type ", activeAdapterName, " struct {")
	g.P("Native ", nativeAdapterName)
	g.P("Message ", messageAdapterName)
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

func renderRuntimeNativeStreamFacade(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
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
		renderRuntimeNativeStreamSend(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamFinish(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamCancel(g, serviceName, method, facadeName)
	case SessionKindServer:
		renderRuntimeNativeStreamRecv(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamDone(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamCancel(g, serviceName, method, facadeName)
	case SessionKindBidi:
		renderRuntimeNativeStreamSend(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamRecv(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamCloseSend(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamDone(g, serviceName, method, facadeName)
		renderRuntimeNativeStreamCancel(g, serviceName, method, facadeName)
	}
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
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
		renderRuntimeMessageStreamSend(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamFinish(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamCancel(g, serviceName, method, facadeName)
	case SessionKindServer:
		renderRuntimeMessageStreamRecv(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamDone(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamCancel(g, serviceName, method, facadeName)
	case SessionKindBidi:
		renderRuntimeMessageStreamSend(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamRecv(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamCloseSend(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamDone(g, serviceName, method, facadeName)
		renderRuntimeMessageStreamCancel(g, serviceName, method, facadeName)
	}
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context", method.NativeArgs, ") error {")
	g.P("return rpcruntime.DispatcherStreamSend[", serviceName, "ActiveAdapter, ", method.SessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.Send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
	renderRuntimeNativeStreamResultVars(g, method)
	g.P("err := rpcruntime.DispatcherStreamFinish[", serviceName, "ActiveAdapter, ", method.SessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", method.SessionName, ") error {")
	renderRuntimeNativeStreamCall(g, method, "Finish")
	g.P("})")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	renderRuntimeNativeStreamSuccessReturn(g, method)
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	renderRuntimeNativeStreamResultVars(g, method)
	g.P("err := rpcruntime.DispatcherStreamReceive[", serviceName, "ActiveAdapter, ", method.SessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", method.SessionName, ") error {")
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

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("return rpcruntime.DispatcherStreamCloseSend[", serviceName, "ActiveAdapter, ", method.SessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.CloseSend(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamDone(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Done(ctx context.Context) error {")
	g.P("return rpcruntime.DispatcherStreamDone[", serviceName, "ActiveAdapter, ", method.SessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.Done(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("return rpcruntime.DispatcherStreamCancel[", serviceName, "ActiveAdapter, ", method.SessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", method.SessionName, ") error {")
	g.P("return session.Cancel(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Send(ctx context.Context, req []byte) error {")
	g.P("return rpcruntime.DispatcherStreamSend[", serviceName, "ActiveAdapter, ", sessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", sessionName, ") error {")
	g.P("return session.Send(ctx, req)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("var resp []byte")
	g.P("err := rpcruntime.DispatcherStreamFinish[", serviceName, "ActiveAdapter, ", sessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", sessionName, ") error {")
	g.P("var callErr error")
	g.P("resp, callErr = session.Finish(ctx)")
	g.P("return callErr")
	g.P("})")
	g.P("return resp, err")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("var resp []byte")
	g.P("err := rpcruntime.DispatcherStreamReceive[", serviceName, "ActiveAdapter, ", sessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", sessionName, ") error {")
	g.P("var callErr error")
	g.P("resp, callErr = session.Recv(ctx)")
	g.P("return callErr")
	g.P("})")
	g.P("return resp, err")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	g.P("return rpcruntime.DispatcherStreamCloseSend[", serviceName, "ActiveAdapter, ", sessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", sessionName, ") error {")
	g.P("return session.CloseSend(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamDone(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Done(ctx context.Context) error {")
	g.P("return rpcruntime.DispatcherStreamDone[", serviceName, "ActiveAdapter, ", sessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", sessionName, ") error {")
	g.P("return session.Done(ctx)")
	g.P("})")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, facadeName string) {
	sessionName := methodMessageSessionName(method)
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	g.P("return rpcruntime.DispatcherStreamCancel[", serviceName, "ActiveAdapter, ", sessionName, "](", serviceName, "DispatcherForRuntime(), s.handle, func(session ", sessionName, ") error {")
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

func renderRuntimeActiveRouter(g *protogen.GeneratedFile, serviceName, routerTypeName, activeAdapterName string, methods []runtimeAdapterMethod, codecEnabled bool) {
	g.P("type ", routerTypeName, " struct {")
	g.P("dispatcher *rpcruntime.Dispatcher[", activeAdapterName, "]")
	g.P("}")
	g.P()

	for _, method := range methods {
		if method.Streaming {
			renderRuntimeActiveRouterNativeStream(g, serviceName, routerTypeName, activeAdapterName, method, codecEnabled)
			renderRuntimeActiveRouterMessageStream(g, serviceName, routerTypeName, activeAdapterName, method, codecEnabled)
			continue
		}
		renderRuntimeActiveRouterNativeUnary(g, serviceName, routerTypeName, activeAdapterName, method, codecEnabled)
		renderRuntimeActiveRouterMessageUnary(g, serviceName, routerTypeName, activeAdapterName, method, codecEnabled)
	}
}

func renderRuntimeActiveRouterNativeUnary(g *protogen.GeneratedFile, serviceName, routerTypeName, activeAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	g.P("func (r ", routerTypeName, ") invokeNative", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
	for _, decl := range method.NativeVarDecls {
		g.P(decl)
	}
	g.P("err := r.dispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) error {")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractNative:")
	g.P("if snapshot.Adapter.Native == nil {")
	g.P("return ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	if method.NativeNames == "" {
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	} else {
		g.P("var callErr error")
		g.P(method.NativeNames, ", callErr = snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("return callErr")
	}
	g.P("case rpcruntime.ServerContractMessage:")
	g.P("if snapshot.Adapter.Message == nil {")
	g.P("return ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	if codecEnabled {
		g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("messageResp, err := snapshot.Adapter.Message.", method.AdapterName, "Message(ctx, messageReq)")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		if method.NativeNames == "" {
			g.P("return ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
		} else {
			g.P("var callErr error")
			g.P(method.NativeNames, ", callErr = ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
			g.P("return callErr")
		}
	} else {
		g.P("return ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("})")
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

func renderRuntimeActiveRouterMessageUnary(g *protogen.GeneratedFile, serviceName, routerTypeName, activeAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	g.P("func (r ", routerTypeName, ") invokeMessage", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
	g.P("var resp []byte")
	g.P("err := r.dispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) error {")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractMessage:")
	g.P("if snapshot.Adapter.Message == nil {")
	g.P("return ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	g.P("var callErr error")
	g.P("resp, callErr = snapshot.Adapter.Message.", method.AdapterName, "Message(ctx, req)")
	g.P("return callErr")
	g.P("case rpcruntime.ServerContractNative:")
	g.P("if snapshot.Adapter.Native == nil {")
	g.P("return ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	if codecEnabled {
		g.P("return ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
		if method.NativeNames == "" {
			g.P("err := snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		} else {
			g.P(method.NativeNames, ", err := snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		}
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("messageResp, err := ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("resp = messageResp")
		g.P("return nil")
		g.P("})")
	} else {
		g.P("return ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("})")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderRuntimeActiveRouterNativeStream(g *protogen.GeneratedFile, serviceName, routerTypeName, activeAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	switch method.SessionKind {
	case SessionKindClient:
		g.P("func (r ", routerTypeName, ") startNative", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	case SessionKindServer:
		g.P("func (r ", routerTypeName, ") startNative", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
	case SessionKindBidi:
		g.P("func (r ", routerTypeName, ") startNative", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	default:
		return
	}
	g.P("return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractNative:")
	g.P("if snapshot.Adapter.Native == nil {")
	g.P("return nil, ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
	case SessionKindServer:
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	}
	g.P("case rpcruntime.ServerContractMessage:")
	g.P("if snapshot.Adapter.Message == nil {")
	g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	if codecEnabled {
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("messageSession, err := snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("return &", messageToNativeStreamWrapperName(serviceName, method), "{message: messageSession}, nil")
		case SessionKindServer:
			g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("messageSession, err := snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx, messageReq)")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("return &", messageToNativeStreamWrapperName(serviceName, method), "{message: messageSession}, nil")
		}
	} else {
		g.P("return nil, ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return nil, ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("})")
	g.P("}")
	g.P()
	if codecEnabled {
		renderMessageToNativeStreamWrapper(g, serviceName, method)
	}
}

func renderRuntimeActiveRouterMessageStream(g *protogen.GeneratedFile, serviceName, routerTypeName, activeAdapterName string, method runtimeAdapterMethod, codecEnabled bool) {
	switch method.SessionKind {
	case SessionKindClient:
		g.P("func (r ", routerTypeName, ") startMessage", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	case SessionKindServer:
		g.P("func (r ", routerTypeName, ") startMessage", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
	case SessionKindBidi:
		g.P("func (r ", routerTypeName, ") startMessage", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
	default:
		return
	}
	g.P("return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
	g.P("switch snapshot.Contract {")
	g.P("case rpcruntime.ServerContractMessage:")
	g.P("if snapshot.Adapter.Message == nil {")
	g.P("return nil, ", serviceName, "MessageAdapterUnavailableErr")
	g.P("}")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
	case SessionKindServer:
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx, req)")
	}
	g.P("case rpcruntime.ServerContractNative:")
	g.P("if snapshot.Adapter.Native == nil {")
	g.P("return nil, ", serviceName, "NativeAdapterUnavailableErr")
	g.P("}")
	if codecEnabled {
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("nativeSession, err := snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("return &", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}, nil")
		case SessionKindServer:
			g.P("var session any")
			g.P("err := ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
			g.P("nativeSession, err := snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("session = &", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}")
			g.P("return nil")
			g.P("})")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("return session, nil")
		}
	} else {
		g.P("return nil, ", serviceName, "NativeMessageConverterUnavailableErr")
	}
	g.P("default:")
	g.P("return nil, ", serviceName, "UnknownActiveContractErr")
	g.P("}")
	g.P("})")
	g.P("}")
	g.P()
	if codecEnabled {
		renderNativeToMessageStreamWrapper(g, serviceName, method)
	}
}

func renderRuntimeCGOBridge(g *protogen.GeneratedFile, serviceName, adapterName, _ string, _ string, methods []runtimeAdapterMethod, _ bool) {
	bridgeName := serviceName + "CGONativeClientBridge"
	routerName := lowerInitial(serviceName) + "Router"
	g.P("type ", bridgeName, " struct{}")
	g.P()

	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func (", bridgeName, ") ", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
		g.P("return ", routerName, ".invokeNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("}")
		g.P()
	}

	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
			g.P("return ", routerName, ".startNative", method.MethodGoName, "(ctx)")
		case SessionKindServer:
			g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
			g.P("return ", routerName, ".startNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		}
		g.P("}")
		g.P()
	}

	g.P("func New", serviceName, "CGONativeClientBridge() ", bridgeName, " {")
	g.P("return ", bridgeName, "{}")
	g.P("}")
	g.P()

	g.P("func Register", serviceName, "CGONativeActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return register", serviceName, "ActiveServer(kind, adapter)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageCGOBridge(g *protogen.GeneratedFile, serviceName, adapterName, _ string, _ string, methods []runtimeAdapterMethod, _ bool) {
	bridgeName := serviceName + "CGOMessageClientBridge"
	routerName := lowerInitial(serviceName) + "Router"
	g.P("type ", bridgeName, " struct{}")
	g.P()

	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func (", bridgeName, ") ", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("return ", routerName, ".invokeMessage", method.MethodGoName, "(ctx, req)")
		g.P("}")
		g.P()
	}

	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
			g.P("return ", routerName, ".startMessage", method.MethodGoName, "(ctx)")
		case SessionKindServer:
			g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
			g.P("return ", routerName, ".startMessage", method.MethodGoName, "(ctx, req)")
		}
		g.P("}")
		g.P()
	}

	g.P("func New", serviceName, "CGOMessageClientBridge() ", bridgeName, " {")
	g.P("return ", bridgeName, "{}")
	g.P("}")
	g.P()

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
