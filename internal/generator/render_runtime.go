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
	directFmt := directUnary || directConnectStreaming || directGRPCStreaming
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
	activeSlotName := lowerInitial(service.GoName) + "ActiveSlot"
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
	renderMessageServerAdapter(g, service, streamingMethods, messageAdapterName, messageServerAdapterName)

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
	g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P("var ", service.GoName, `UnknownActiveContractErr = errors.New("rpccgo: unknown active server contract")`)
	g.P()

	g.P("func register", service.GoName, "ActiveServer(kind rpcruntime.ServerKind, server ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("snapshot, err := ", activeSlotName, ".Store(kind, rpcruntime.ServerContractNative, server)")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: server}, nil")
	g.P("}")
	g.P()

	g.P("func register", service.GoName, "CGOMessageServer(server ", messageAdapterName, ") (rpcruntime.AdapterSnapshot[", messageAdapterName, "], error) {")
	g.P("snapshot, err := ", activeSlotName, ".Store(rpcruntime.ServerKindCGOMessage, rpcruntime.ServerContractMessage, server)")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: server}, nil")
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

func renderMessageServerAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, serverName, adapterName string) {
	if len(methods) == 0 {
		return
	}
	g.P("type ", adapterName, " struct {")
	g.P("server ", serverName)
	g.P("}")
	g.P()
	for _, method := range methods {
		switch method.SessionKind {
		case SessionKindClient:
			renderMessageServerClientStreamAdapter(g, service.GoName, adapterName, method)
		case SessionKindServer:
			renderMessageServerServerStreamAdapter(g, service.GoName, adapterName, method)
		case SessionKindBidi:
			renderMessageServerBidiStreamAdapter(g, service.GoName, adapterName, method)
		}
	}
}

func renderMessageServerClientStreamAdapter(g *protogen.GeneratedFile, serviceName, adapterName string, method runtimeAdapterMethod) {
	sessionName := methodMessageSessionName(method)
	receiver := lowerInitial(serviceName) + method.MethodGoName + "MessageServerClientStreamSession"
	g.P("func (a *", adapterName, ") Start", method.MethodGoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", receiver, "{ctx: streamCtx, cancel: cancel, requests: make(chan ", receiver, "Request, 16), sendDone: make(chan struct{}), done: make(chan struct{})}")
	g.P("go func() {")
	g.P("defer close(session.done)")
	g.P("session.resp, session.err = a.server.", method.MethodGoName, "(streamCtx, session)")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()
	g.P("type ", receiver, "Request struct {")
	g.P("data []byte")
	g.P("received chan struct{}")
	g.P("}")
	g.P()
	g.P("type ", receiver, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	g.P("requests chan ", receiver, "Request")
	g.P("sendDone chan struct{}")
	g.P("closeSendOnce sync.Once")
	g.P("received chan struct{}")
	g.P("done chan struct{}")
	g.P("resp []byte")
	g.P("err error")
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("select {")
	g.P("case req := <-s.requests:")
	g.P("close(req.received)")
	g.P("return req.data, nil")
	g.P("default:")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return nil, s.ctx.Err()")
	g.P("case req := <-s.requests:")
	g.P("close(req.received)")
	g.P("return req.data, nil")
	g.P("case <-s.sendDone:")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") Send(ctx context.Context, req []byte) error {")
	g.P("select {")
	g.P("case <-s.sendDone:")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("default:")
	g.P("}")
	g.P("queued := ", receiver, "Request{data: append([]byte(nil), req...), received: make(chan struct{})}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("select {")
	g.P("case <-queued.received:")
	g.P("return nil")
	g.P("default:")
	g.P("}")
	g.P("if s.err != nil { return s.err }")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case <-s.sendDone:")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case s.requests <- queued:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("select {")
	g.P("case <-queued.received:")
	g.P("return nil")
	g.P("default:")
	g.P("}")
	g.P("if s.err != nil { return s.err }")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case <-s.sendDone:")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case <-queued.received:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("s.closeSendOnce.Do(func() { close(s.sendDone) })")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case <-s.done:")
	g.P("s.cancel()")
	g.P("return s.resp, s.err")
	g.P("}")
	g.P("}")
	g.P()
	renderMessageServerGeneratedCancel(g, receiver, true)
}

func renderMessageServerServerStreamAdapter(g *protogen.GeneratedFile, serviceName, adapterName string, method runtimeAdapterMethod) {
	sessionName := methodMessageSessionName(method)
	receiver := lowerInitial(serviceName) + method.MethodGoName + "MessageServerServerStreamSession"
	g.P("func (a *", adapterName, ") Start", method.MethodGoName, "(ctx context.Context, req []byte) (", sessionName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", receiver, "{ctx: streamCtx, cancel: cancel, responses: make(chan ", receiver, "Response, 1), done: make(chan struct{})}")
	g.P("req = append([]byte(nil), req...)")
	g.P("go func() {")
	g.P("defer close(session.done)")
	g.P("defer close(session.responses)")
	g.P("session.err = a.server.", method.MethodGoName, "(streamCtx, req, session)")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()
	g.P("type ", receiver, "Response struct {")
	g.P("data []byte")
	g.P("received chan struct{}")
	g.P("}")
	g.P()
	g.P("type ", receiver, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	g.P("responses chan ", receiver, "Response")
	g.P("received chan struct{}")
	g.P("doneRequested bool")
	g.P("done chan struct{}")
	g.P("err error")
	g.P("}")
	g.P()
	renderMessageServerStreamSend(g, receiver)
	renderMessageServerStreamRecv(g, receiver)
	renderMessageServerGeneratedDone(g, receiver)
	renderMessageServerGeneratedCancel(g, receiver, false)
}

func renderMessageServerBidiStreamAdapter(g *protogen.GeneratedFile, serviceName, adapterName string, method runtimeAdapterMethod) {
	sessionName := methodMessageSessionName(method)
	receiver := lowerInitial(serviceName) + method.MethodGoName + "MessageServerBidiStreamSession"
	facadeName := lowerInitial(serviceName) + method.MethodGoName + "MessageServerBidiStreamFacade"
	g.P("func (a *", adapterName, ") Start", method.MethodGoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", receiver, "{ctx: streamCtx, cancel: cancel, requests: make(chan ", receiver, "Request, 16), sendDone: make(chan struct{}), sendDoneReceived: make(chan struct{}), responses: make(chan ", receiver, "Response, 1), done: make(chan struct{})}")
	g.P("go func() {")
	g.P("defer close(session.done)")
	g.P("defer close(session.responses)")
	g.P("session.err = a.server.", method.MethodGoName, "(streamCtx, &", facadeName, "{session: session})")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()
	g.P("type ", receiver, "Request struct {")
	g.P("data []byte")
	g.P("received chan struct{}")
	g.P("}")
	g.P()
	g.P("type ", receiver, "Response struct {")
	g.P("data []byte")
	g.P("received chan struct{}")
	g.P("}")
	g.P()
	g.P("type ", receiver, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	g.P("requests chan ", receiver, "Request")
	g.P("sendDone chan struct{}")
	g.P("sendDoneReceived chan struct{}")
	g.P("sendDoneReceivedOnce sync.Once")
	g.P("closeSendOnce sync.Once")
	g.P("responses chan ", receiver, "Response")
	g.P("received chan struct{}")
	g.P("doneRequested bool")
	g.P("done chan struct{}")
	g.P("err error")
	g.P("}")
	g.P()
	g.P("type ", facadeName, " struct {")
	g.P("session *", receiver)
	g.P("}")
	g.P()
	g.P("func (s *", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("return s.session.recvRequest(ctx)")
	g.P("}")
	g.P()
	g.P("func (s *", facadeName, ") Send(ctx context.Context, resp []byte) error {")
	g.P("return s.session.sendResponse(ctx, resp)")
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") recvRequest(ctx context.Context) ([]byte, error) {")
	g.P("select {")
	g.P("case req := <-s.requests:")
	g.P("close(req.received)")
	g.P("return req.data, nil")
	g.P("default:")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return nil, s.ctx.Err()")
	g.P("case req := <-s.requests:")
	g.P("close(req.received)")
	g.P("return req.data, nil")
	g.P("case <-s.sendDone:")
	g.P("s.sendDoneReceivedOnce.Do(func() { close(s.sendDoneReceived) })")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") sendResponse(ctx context.Context, resp []byte) error {")
	g.P("response := ", receiver, "Response{data: append([]byte(nil), resp...), received: make(chan struct{})}")
	renderMessageServerSendResponseBody(g)
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") Send(ctx context.Context, req []byte) error {")
	g.P("select {")
	g.P("case <-s.sendDone:")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("default:")
	g.P("}")
	g.P("queued := ", receiver, "Request{data: append([]byte(nil), req...), received: make(chan struct{})}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("select {")
	g.P("case <-queued.received:")
	g.P("return nil")
	g.P("default:")
	g.P("}")
	g.P("return nil")
	g.P("case <-s.sendDone:")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case s.requests <- queued:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("select {")
	g.P("case <-queued.received:")
	g.P("return nil")
	g.P("default:")
	g.P("}")
	g.P("return nil")
	g.P("case <-s.sendDone:")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case <-queued.received:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") Recv(ctx context.Context) ([]byte, error) {")
	renderMessageServerStreamRecvBody(g)
	g.P("}")
	g.P()
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	g.P("s.closeSendOnce.Do(func() { close(s.sendDone) })")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil { return s.err }")
	g.P("return nil")
	g.P("case <-s.sendDoneReceived:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	renderMessageServerGeneratedDone(g, receiver)
	renderMessageServerGeneratedCancel(g, receiver, true)
}

func renderMessageServerStreamSend(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Send(ctx context.Context, resp []byte) error {")
	g.P("response := ", receiver, "Response{data: append([]byte(nil), resp...), received: make(chan struct{})}")
	renderMessageServerSendResponseBody(g)
	g.P("}")
	g.P()
}

func renderMessageServerSendResponseBody(g *protogen.GeneratedFile) {
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("if s.doneRequested { return io.EOF }")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("if s.doneRequested { return io.EOF }")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil { return s.err }")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case s.responses <- response:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("if s.doneRequested { return io.EOF }")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("if s.doneRequested { return io.EOF }")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil { return s.err }")
	g.P(`return errors.New("rpccgo: message stream is closed")`)
	g.P("case <-response.received:")
	g.P("if s.ctx.Err() != nil {")
	g.P("if s.doneRequested { return io.EOF }")
	g.P("return s.ctx.Err()")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P("}")
}

func renderMessageServerStreamRecv(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Recv(ctx context.Context) ([]byte, error) {")
	renderMessageServerStreamRecvBody(g)
	g.P("}")
	g.P()
}

func renderMessageServerStreamRecvBody(g *protogen.GeneratedFile) {
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return nil, ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return nil, s.ctx.Err()")
	g.P("case resp, ok := <-s.responses:")
	g.P("if ok {")
	g.P("s.received = resp.received")
	g.P("return resp.data, nil")
	g.P("}")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("<-s.done")
	g.P("if s.err != nil {")
	g.P("err := s.err")
	g.P("s.err = nil")
	g.P("return nil, err")
	g.P("}")
	g.P("return nil, io.EOF")
	g.P("}")
}

func renderMessageServerGeneratedDone(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Done(ctx context.Context) error {")
	g.P("s.doneRequested = true")
	g.P("s.cancel()")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.done:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
}

func renderMessageServerGeneratedCancel(g *protogen.GeneratedFile, receiver string, closeSend bool) {
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("s.cancel()")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.done:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P()
	_ = closeSend
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
		clientName := service.GoName + "Client"
		g.P("func Register", service.GoName, "ConnectHandler(handler ", handlerName, ") (rpcruntime.AdapterSnapshot[", handlerName, "], error) {")
		g.P("snapshot, err := ", lowerInitial(service.GoName), "ActiveSlot.Store(rpcruntime.ServerKindConnectHandler, rpcruntime.ServerContractMessage, handler)")
		g.P("if err != nil {")
		g.P("return rpcruntime.AdapterSnapshot[", handlerName, "]{}, err")
		g.P("}")
		g.P("return rpcruntime.AdapterSnapshot[", handlerName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: handler}, nil")
		g.P("}")
		g.P()
		g.P("func Register", service.GoName, "ConnectRemoteServer(client ", clientName, ") (rpcruntime.AdapterSnapshot[", clientName, "], error) {")
		g.P("snapshot, err := ", lowerInitial(service.GoName), "ActiveSlot.Store(rpcruntime.ServerKindConnectRemote, rpcruntime.ServerContractMessage, client)")
		g.P("if err != nil {")
		g.P("return rpcruntime.AdapterSnapshot[", clientName, "]{}, err")
		g.P("}")
		g.P("return rpcruntime.AdapterSnapshot[", clientName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: client}, nil")
		g.P("}")
		g.P()
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		clientName := service.GoName + "Client"
		g.P("func Register", service.GoName, "GRPCServer(server ", serverName, ") (rpcruntime.AdapterSnapshot[", serverName, "], error) {")
		g.P("snapshot, err := ", lowerInitial(service.GoName), "ActiveSlot.Store(rpcruntime.ServerKindGRPCServer, rpcruntime.ServerContractMessage, server)")
		g.P("if err != nil {")
		g.P("return rpcruntime.AdapterSnapshot[", serverName, "]{}, err")
		g.P("}")
		g.P("return rpcruntime.AdapterSnapshot[", serverName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: server}, nil")
		g.P("}")
		g.P()
		g.P("func Register", service.GoName, "GRPCRemoteServer(client ", clientName, ") (rpcruntime.AdapterSnapshot[", clientName, "], error) {")
		g.P("snapshot, err := ", lowerInitial(service.GoName), "ActiveSlot.Store(rpcruntime.ServerKindGRPCRemote, rpcruntime.ServerContractMessage, client)")
		g.P("if err != nil {")
		g.P("return rpcruntime.AdapterSnapshot[", clientName, "]{}, err")
		g.P("}")
		g.P("return rpcruntime.AdapterSnapshot[", clientName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: client}, nil")
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
	g.P("server, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || server == nil {")
	g.P("err = ", serviceName, "NativeServerUnavailableErr")
	g.P("break")
	g.P("}")
	g.P("adapter := &", lowerInitial(serviceName), "NativeServerAdapter{server: server}")
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
	g.P("server, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || server == nil {")
	g.P("return nil, ", serviceName, "NativeServerUnavailableErr")
	g.P("}")
	if codecEnabled {
		g.P("adapter := &", lowerInitial(serviceName), "NativeServerAdapter{server: server}")
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
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("adapter, ok := snapshot.Adapter.(", messageAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
	g.P("}")
	g.P("return adapter.", method.MethodGoName, "(ctx, req)")
	renderRuntimeBridgeMessageUnaryDirectCases(g, service, method, "req", "return nil, ")
	renderRuntimeBridgeMessageUnaryRemoteCases(g, service, method, "req", "return nil, ")
	g.P("default:")
	g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
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
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("adapter, ok := snapshot.Adapter.(", messageAdapterName, ")")
	g.P("if !ok || adapter == nil {")
	g.P("err = ", serviceName, "MessageServerUnavailableErr")
	g.P("break")
	g.P("}")
	g.P("messageResp, err = adapter.", method.MethodGoName, "(ctx, messageReq)")
	renderRuntimeBridgeNativeUnaryDirectCases(g, service, method)
	renderRuntimeBridgeNativeUnaryRemoteCases(g, service, method)
	g.P("default:")
	g.P("err = ", serviceName, "MessageServerUnavailableErr")
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
		g.P(errPrefix, serviceName, "MessageServerUnavailableErr")
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
		g.P(errPrefix, serviceName, "MessageServerUnavailableErr")
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

func renderRuntimeBridgeMessageUnaryRemoteCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, reqExpr string, errPrefix string) {
	serviceName := service.GoName
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		clientName := service.GoName + "Client"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindConnectRemote:")
		g.P("client, ok := snapshot.Adapter.(", clientName, ")")
		g.P("if !ok || client == nil {")
		g.P(errPrefix, serviceName, "MessageServerUnavailableErr")
		g.P("}")
		g.P("messageReq := new(", reqType, ")")
		g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)`)
		g.P("}")
		g.P("messageResp, err := client.", method.MethodGoName, "(ctx, messageReq)")
		g.P("if err != nil {")
		g.P(errPrefix, "err")
		g.P("}")
		g.P("resp, err := proto.Marshal(messageResp)")
		g.P("if err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: connect remote response protobuf marshal failed: %w", err)`)
		g.P("}")
		g.P("return resp, nil")
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		clientName := service.GoName + "Client"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindGRPCRemote:")
		g.P("client, ok := snapshot.Adapter.(", clientName, ")")
		g.P("if !ok || client == nil {")
		g.P(errPrefix, serviceName, "MessageServerUnavailableErr")
		g.P("}")
		g.P("messageReq := new(", reqType, ")")
		g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: grpc remote request protobuf unmarshal failed: %w", err)`)
		g.P("}")
		g.P("messageResp, err := client.", method.MethodGoName, "(ctx, messageReq)")
		g.P("if err != nil {")
		g.P(errPrefix, "err")
		g.P("}")
		g.P("resp, err := proto.Marshal(messageResp)")
		g.P("if err != nil {")
		g.P(errPrefix, `fmt.Errorf("rpccgo: grpc remote response protobuf marshal failed: %w", err)`)
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
		g.P("err = ", service.GoName, "MessageServerUnavailableErr")
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
		g.P("err = ", service.GoName, "MessageServerUnavailableErr")
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

func renderRuntimeBridgeNativeUnaryRemoteCases(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		clientName := service.GoName + "Client"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindConnectRemote:")
		g.P("client, ok := snapshot.Adapter.(", clientName, ")")
		g.P("if !ok || client == nil {")
		g.P("err = ", service.GoName, "MessageServerUnavailableErr")
		g.P("break")
		g.P("}")
		g.P("directReq := new(", reqType, ")")
		g.P("if err = proto.Unmarshal(messageReq, directReq); err != nil {")
		g.P(`err = fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)`)
		g.P("break")
		g.P("}")
		g.P("directResp, callErr := client.", method.MethodGoName, "(ctx, directReq)")
		g.P("if callErr != nil {")
		g.P("err = callErr")
		g.P("break")
		g.P("}")
		g.P("messageResp, err = proto.Marshal(directResp)")
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		clientName := service.GoName + "Client"
		reqType := qualifiedMethodType(g, methodForRuntimeService(service, method).Request)
		g.P("case rpcruntime.ServerKindGRPCRemote:")
		g.P("client, ok := snapshot.Adapter.(", clientName, ")")
		g.P("if !ok || client == nil {")
		g.P("err = ", service.GoName, "MessageServerUnavailableErr")
		g.P("break")
		g.P("}")
		g.P("directReq := new(", reqType, ")")
		g.P("if err = proto.Unmarshal(messageReq, directReq); err != nil {")
		g.P(`err = fmt.Errorf("rpccgo: grpc remote request protobuf unmarshal failed: %w", err)`)
		g.P("break")
		g.P("}")
		g.P("directResp, callErr := client.", method.MethodGoName, "(ctx, directReq)")
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
	g.P("if snapshot.Kind == rpcruntime.ServerKindCGONative {")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("adapter, ok := snapshot.Adapter.(interface { ", method.AdapterName, "(ctx context.Context) (", method.SessionName, ", error) })")
	case SessionKindServer:
		g.P("adapter, ok := snapshot.Adapter.(interface { ", method.AdapterName, "(ctx context.Context", method.NativeArgs, ") (", method.SessionName, ", error) })")
	}
	g.P("if !ok || adapter == nil {")
	g.P("return 0, ", serviceName, "NativeServerUnavailableErr")
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
	g.P("}")
	g.P("server, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || server == nil {")
	g.P("return 0, ", serviceName, "NativeServerUnavailableErr")
	g.P("}")
	g.P("adapter := &", lowerInitial(serviceName), "NativeServerAdapter{server: server}")
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
	g.P("server, ok := snapshot.Adapter.(", nativeAdapterName, ")")
	g.P("if !ok || server == nil {")
	g.P("return 0, ", serviceName, "NativeServerUnavailableErr")
	g.P("}")
	if codecEnabled {
		g.P("adapter := &", lowerInitial(serviceName), "NativeServerAdapter{server: server}")
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
	g.P("case rpcruntime.ServerKindCGOMessage:")
	g.P("server, ok := snapshot.Adapter.(", messageAdapterName, ")")
	g.P("if !ok || server == nil {")
	g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
	g.P("}")
	g.P("adapter := &", lowerInitial(serviceName), "MessageServerAdapter{server: server}")
	switch method.SessionKind {
	case SessionKindClient, SessionKindBidi:
		g.P("return adapter.Start", method.MethodGoName, "(ctx)")
	case SessionKindServer:
		g.P("return adapter.Start", method.MethodGoName, "(ctx, req)")
	}
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		handlerName := service.GoName + "Handler"
		clientName := service.GoName + "Client"
		g.P("case rpcruntime.ServerKindConnectHandler:")
		g.P("handler, ok := snapshot.Adapter.(", handlerName, ")")
		g.P("if !ok || handler == nil {")
		g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
		g.P("}")
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("return new", connectDirectMessageSessionName(serviceName, method), "(ctx, handler), nil")
		case SessionKindServer:
			g.P("return new", connectDirectMessageSessionName(serviceName, method), "(ctx, handler, req)")
		}
		g.P("case rpcruntime.ServerKindConnectRemote:")
		g.P("client, ok := snapshot.Adapter.(", clientName, ")")
		g.P("if !ok || client == nil {")
		g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
		g.P("}")
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("return new", connectRemoteMessageSessionName(serviceName, method), "(ctx, client)")
		case SessionKindServer:
			g.P("return new", connectRemoteMessageSessionName(serviceName, method), "(ctx, client, req)")
		}
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		clientName := service.GoName + "Client"
		g.P("case rpcruntime.ServerKindGRPCServer:")
		g.P("server, ok := snapshot.Adapter.(", serverName, ")")
		g.P("if !ok || server == nil {")
		g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
		g.P("}")
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("return new", grpcDirectMessageSessionName(serviceName, method), "(ctx, server), nil")
		case SessionKindServer:
			g.P("return new", grpcDirectMessageSessionName(serviceName, method), "(ctx, server, req)")
		}
		g.P("case rpcruntime.ServerKindGRPCRemote:")
		g.P("client, ok := snapshot.Adapter.(", clientName, ")")
		g.P("if !ok || client == nil {")
		g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
		g.P("}")
		switch method.SessionKind {
		case SessionKindClient, SessionKindBidi:
			g.P("return new", grpcRemoteMessageSessionName(serviceName, method), "(ctx, client)")
		case SessionKindServer:
			g.P("return new", grpcRemoteMessageSessionName(serviceName, method), "(ctx, client, req)")
		}
	}
	g.P("default:")
	g.P("return nil, ", serviceName, "MessageServerUnavailableErr")
	g.P("}")
	g.P("}")
	g.P()
	if service.Adapters.Has(AdapterTokenMessageConnect) {
		renderConnectDirectMessageSession(g, service, method)
		renderConnectRemoteMessageSession(g, service, method)
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		renderGRPCDirectMessageSession(g, service, method)
		renderGRPCRemoteMessageSession(g, service, method)
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

func renderConnectRemoteMessageSession(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	methodPlan := methodForRuntimeService(service, method)
	wrapperName := connectRemoteMessageSessionName(service.GoName, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	respType := qualifiedMethodType(g, methodPlan.Response)
	switch method.SessionKind {
	case SessionKindClient:
		clientType := "interface { " + method.MethodGoName + "(context.Context) (*connect.ClientStreamForClientSimple[" + strings.TrimPrefix(reqType, "*") + ", " + strings.TrimPrefix(respType, "*") + "], error) }"
		renderConnectRemoteClientStreamSession(g, method, wrapperName, reqType, respType, clientType)
	case SessionKindServer:
		clientType := "interface { " + method.MethodGoName + "(context.Context, *" + strings.TrimPrefix(reqType, "*") + ") (*connect.ServerStreamForClient[" + strings.TrimPrefix(respType, "*") + "], error) }"
		renderConnectRemoteServerStreamSession(g, method, wrapperName, reqType, respType, clientType)
	case SessionKindBidi:
		clientType := "interface { " + method.MethodGoName + "(context.Context) (*connect.BidiStreamForClientSimple[" + strings.TrimPrefix(reqType, "*") + ", " + strings.TrimPrefix(respType, "*") + "], error) }"
		renderConnectRemoteBidiStreamSession(g, method, wrapperName, reqType, respType, clientType)
	}
}

func renderConnectRemoteClientStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, reqType, respType, clientType string) {
	g.P("func new", wrapperName, "(ctx context.Context, client ", clientType, ") (*", wrapperName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("stream, err := client.", method.MethodGoName, "(streamCtx)")
	g.P("if err != nil {")
	g.P("cancel()")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", wrapperName, "{stream: stream, cancel: cancel}, nil")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("stream *connect.ClientStreamForClientSimple[", reqType, ", ", respType, "]")
	g.P("cancel context.CancelFunc")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("_ = ctx")
	g.P("if s == nil {")
	g.P(`return errors.New("rpccgo: connect remote client stream is nil")`)
	g.P("}")
	g.P("if s.stream == nil {")
	g.P(`return errors.New("rpccgo: connect remote client stream is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return fmt.Errorf("rpccgo: connect remote stream request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return s.stream.Send(request)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil {")
	g.P(`return nil, errors.New("rpccgo: connect remote client stream is nil")`)
	g.P("}")
	g.P("if s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: connect remote client stream is nil")`)
	g.P("}")
	g.P("defer func() { if s.cancel != nil { s.cancel() } }()")
	g.P("resp, err := s.stream.CloseAndReceive()")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("if resp == nil {")
	g.P("return nil, nil")
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect remote stream response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s != nil && s.stream != nil {")
	g.P("closed := make(chan struct{})")
	g.P("go func() {")
	g.P("_, _ = s.stream.CloseAndReceive()")
	g.P("close(closed)")
	g.P("}()")
	g.P("timer := time.NewTimer(100 * time.Millisecond)")
	g.P("select {")
	g.P("case <-closed:")
	g.P("timer.Stop()")
	g.P("return nil")
	g.P("case <-timer.C:")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
	g.P("select {")
	g.P("case <-closed:")
	g.P("case <-time.After(500 * time.Millisecond):")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P("if s != nil && s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderConnectRemoteServerStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, reqType, respType, clientType string) {
	g.P("func new", wrapperName, "(ctx context.Context, client ", clientType, ", req []byte) (*", wrapperName, ", error) {")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("stream, err := client.", method.MethodGoName, "(streamCtx, request)")
	g.P("if err != nil {")
	g.P("cancel()")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", wrapperName, "{stream: stream, cancel: cancel}, nil")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("stream *connect.ServerStreamForClient[", respType, "]")
	g.P("cancel context.CancelFunc")
	g.P("}")
	g.P()
	renderConnectRemoteRecvDoneCancel(g, wrapperName, "server stream")
}

func renderConnectRemoteBidiStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, reqType, respType, clientType string) {
	g.P("func new", wrapperName, "(ctx context.Context, client ", clientType, ") (*", wrapperName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("stream, err := client.", method.MethodGoName, "(streamCtx)")
	g.P("if err != nil {")
	g.P("cancel()")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", wrapperName, "{stream: stream, cancel: cancel}, nil")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("stream *connect.BidiStreamForClientSimple[", reqType, ", ", respType, "]")
	g.P("cancel context.CancelFunc")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("_ = ctx")
	g.P("if s == nil {")
	g.P(`return errors.New("rpccgo: connect remote bidi stream is nil")`)
	g.P("}")
	g.P("if s.stream == nil {")
	g.P(`return errors.New("rpccgo: connect remote bidi stream is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return fmt.Errorf("rpccgo: connect remote bidi request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return s.stream.Send(request)")
	g.P("}")
	g.P()
	renderConnectRemoteBidiRecvDoneCancel(g, wrapperName)
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseRequest()")
	g.P("}")
	g.P()
}

func renderConnectRemoteRecvDoneCancel(g *protogen.GeneratedFile, wrapperName, label string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil {")
	g.P(`return nil, errors.New("rpccgo: connect remote `, label, ` is nil")`)
	g.P("}")
	g.P("if s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: connect remote `, label, ` is nil")`)
	g.P("}")
	g.P("if !s.stream.Receive() {")
	g.P("if err := s.stream.Err(); err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("msg := s.stream.Msg()")
	g.P("if msg == nil {")
	g.P("return nil, nil")
	g.P("}")
	g.P("data, err := proto.Marshal(msg)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect remote stream response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("defer s.cancel()")
	g.P("}")
	g.P("return s.stream.Close()")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderConnectRemoteBidiRecvDoneCancel(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: connect remote bidi stream is nil")`)
	g.P("}")
	g.P("resp, err := s.stream.Receive()")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("if resp == nil {")
	g.P("return nil, nil")
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect remote bidi response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("defer s.cancel()")
	g.P("}")
	g.P("return s.stream.CloseResponse()")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
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

func renderGRPCRemoteMessageSession(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	methodPlan := methodForRuntimeService(service, method)
	wrapperName := grpcRemoteMessageSessionName(service.GoName, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	respType := qualifiedMethodType(g, methodPlan.Response)
	clientName := service.GoName + "Client"
	switch method.SessionKind {
	case SessionKindClient:
		renderGRPCRemoteClientStreamSession(g, method, wrapperName, reqType, respType, clientName)
	case SessionKindServer:
		renderGRPCRemoteServerStreamSession(g, method, wrapperName, reqType, respType, clientName)
	case SessionKindBidi:
		renderGRPCRemoteBidiStreamSession(g, method, wrapperName, reqType, respType, clientName)
	}
}

func renderGRPCRemoteClientStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, reqType, respType, clientName string) {
	g.P("func new", wrapperName, "(ctx context.Context, client ", clientName, ") (*", wrapperName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("stream, err := client.", method.MethodGoName, "(streamCtx)")
	g.P("if err != nil {")
	g.P("cancel()")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", wrapperName, "{stream: stream, cancel: cancel}, nil")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("stream grpc.ClientStreamingClient[", reqType, ", ", respType, "]")
	g.P("cancel context.CancelFunc")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return errors.New("rpccgo: grpc remote client stream is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc remote stream request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return s.stream.Send(request)")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote client stream is nil")`)
	g.P("}")
	g.P("defer func() { if s.cancel != nil { s.cancel() } }()")
	g.P("response, err := s.stream.CloseAndRecv()")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("respData, err := proto.Marshal(response)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: grpc remote stream response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return respData, nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
}

func renderGRPCRemoteServerStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, reqType, respType, clientName string) {
	g.P("func new", wrapperName, "(ctx context.Context, client ", clientName, ", req []byte) (*", wrapperName, ", error) {")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: grpc remote request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("stream, err := client.", method.MethodGoName, "(streamCtx, request)")
	g.P("if err != nil {")
	g.P("cancel()")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", wrapperName, "{stream: stream, cancel: cancel}, nil")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("stream grpc.ServerStreamingClient[", respType, "]")
	g.P("cancel context.CancelFunc")
	g.P("}")
	g.P()
	renderGRPCRemoteRecvDoneCancel(g, wrapperName, "server stream")
}

func renderGRPCRemoteBidiStreamSession(g *protogen.GeneratedFile, method runtimeAdapterMethod, wrapperName, reqType, respType, clientName string) {
	g.P("func new", wrapperName, "(ctx context.Context, client ", clientName, ") (*", wrapperName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("stream, err := client.", method.MethodGoName, "(streamCtx)")
	g.P("if err != nil {")
	g.P("cancel()")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", wrapperName, "{stream: stream, cancel: cancel}, nil")
	g.P("}")
	g.P()
	g.P("type ", wrapperName, " struct {")
	g.P("stream grpc.BidiStreamingClient[", reqType, ", ", respType, "]")
	g.P("cancel context.CancelFunc")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return errors.New("rpccgo: grpc remote bidi stream is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return fmt.Errorf("rpccgo: grpc remote bidi request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return s.stream.Send(request)")
	g.P("}")
	g.P()
	renderGRPCRemoteRecvDoneCancel(g, wrapperName, "bidi stream")
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
}

func renderGRPCRemoteRecvDoneCancel(g *protogen.GeneratedFile, wrapperName, label string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote `, label, ` is nil")`)
	g.P("}")
	g.P("response, err := s.stream.Recv()")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("return nil, err")
	g.P("}")
	g.P("respData, err := proto.Marshal(response)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: grpc remote stream response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return respData, nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s != nil && s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("if s.cancel != nil {")
	g.P("s.cancel()")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
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

	g.P("func Register", serviceName, "CGONativeServer(server ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return register", serviceName, "ActiveServer(rpcruntime.ServerKindCGONative, server)")
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

func grpcRemoteMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "GRPCRemoteMessageStreamSession"
}

func connectRemoteMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "ConnectRemoteMessageStreamSession"
}
