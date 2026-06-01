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

func renderRuntimeSourceSessionInterfaces(g *protogen.GeneratedFile, serviceName string, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		nativeName := method.SessionName
		messageName := methodMessageSessionName(method)
		g.P("type ", nativeName, " interface {")
		if method.CanSend {
			g.P("Send(ctx context.Context", method.NativeArgs, ") error")
		}
		if method.CanRecv {
			g.P("Recv(ctx context.Context) (", method.NativeReturns, ")")
		}
		if method.CanCloseSend {
			g.P("CloseSend(ctx context.Context) error")
		}
		if method.FinishReturnsResponse {
			g.P("Finish(ctx context.Context) (", method.NativeReturns, ")")
		} else {
			g.P("Finish(ctx context.Context) error")
		}
		g.P("Cancel(ctx context.Context) error")
		g.P("}")
		g.P()
		g.P("type ", messageName, " interface {")
		if method.CanSend {
			g.P("Send(ctx context.Context, req []byte) error")
		}
		if method.CanRecv {
			g.P("Recv(ctx context.Context) ([]byte, error)")
		}
		if method.CanCloseSend {
			g.P("CloseSend(ctx context.Context) error")
		}
		if method.FinishReturnsResponse {
			g.P("Finish(ctx context.Context) ([]byte, error)")
		} else {
			g.P("Finish(ctx context.Context) error")
		}
		g.P("Cancel(ctx context.Context) error")
		g.P("}")
		g.P()
	}
	_ = serviceName
}

func renderRuntimeFinalSessions(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	nativeName := runtimeFinalNativeSessionName(serviceName, method)
	messageName := runtimeFinalMessageSessionName(serviceName, method)
	g.P("type ", nativeName, " struct {")
	g.P("lifecycle rpcruntime.StreamLifecycle")
	if method.CanSend {
		g.P("send func(ctx context.Context", method.NativeArgs, ") error")
	}
	if method.CanRecv {
		g.P("recv func(ctx context.Context) (", method.NativeReturns, ")")
	}
	if method.CanCloseSend {
		g.P("closeSend func(ctx context.Context) error")
	}
	if method.FinishReturnsResponse {
		g.P("finish func(ctx context.Context) (", method.NativeReturns, ")")
	} else {
		g.P("finish func(ctx context.Context) error")
	}
	g.P("cancel func(ctx context.Context) error")
	g.P("}")
	g.P()
	g.P("type ", messageName, " struct {")
	g.P("lifecycle rpcruntime.StreamLifecycle")
	if method.CanSend {
		g.P("send func(ctx context.Context, req []byte) error")
	}
	if method.CanRecv {
		g.P("recv func(ctx context.Context) ([]byte, error)")
	}
	if method.CanCloseSend {
		g.P("closeSend func(ctx context.Context) error")
	}
	if method.FinishReturnsResponse {
		g.P("finish func(ctx context.Context) ([]byte, error)")
	} else {
		g.P("finish func(ctx context.Context) error")
	}
	g.P("cancel func(ctx context.Context) error")
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
		if !method.Streaming {
			g.P("func (a *", adapterName, ") ", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
			g.P("return a.server.", method.MethodGoName, "(ctx, req)")
			g.P("}")
			g.P()
			continue
		}
		switch runtimeStreamShapeFor(method) {
		case runtimeStreamClient:
			renderMessageServerClientStreamAdapter(g, service.GoName, adapterName, method)
		case runtimeStreamServer:
			renderMessageServerServerStreamAdapter(g, service.GoName, adapterName, method)
		case runtimeStreamBidi:
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
	renderMessageServerGeneratedFinish(g, receiver)
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
	renderMessageServerGeneratedFinish(g, receiver)
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

func renderMessageServerGeneratedFinish(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Finish(ctx context.Context) error {")
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
	sessionName := runtimeFinalNativeSessionName(serviceName, method)
	g.P("type ", facadeName, " struct {")
	g.P("handle rpcruntime.StreamHandle")
	g.P("}")
	g.P()
	g.P("func New", facadeName, "(handle rpcruntime.StreamHandle) ", facadeName, " {")
	g.P("return ", facadeName, "{handle: handle}")
	g.P("}")
	g.P()
	if method.CanSend {
		renderRuntimeNativeStreamSend(g, streamRegistryName, sessionName, method, facadeName)
	}
	if method.CanRecv {
		renderRuntimeNativeStreamRecv(g, streamRegistryName, sessionName, method, facadeName)
	}
	if method.CanCloseSend {
		renderRuntimeNativeStreamCloseSend(g, streamRegistryName, sessionName, facadeName)
	}
	renderRuntimeNativeStreamFinish(g, streamRegistryName, sessionName, method, facadeName)
	renderRuntimeNativeStreamCancel(g, streamRegistryName, sessionName, facadeName)
}

func renderRuntimeMessageStreamFacade(g *protogen.GeneratedFile, serviceName, streamRegistryName string, method runtimeAdapterMethod) {
	facadeName := messageRuntimeStreamFacadeName(serviceName, method)
	sessionName := runtimeFinalMessageSessionName(serviceName, method)
	g.P("type ", facadeName, " struct {")
	g.P("handle rpcruntime.StreamHandle")
	g.P("}")
	g.P()
	g.P("func New", facadeName, "(handle rpcruntime.StreamHandle) ", facadeName, " {")
	g.P("return ", facadeName, "{handle: handle}")
	g.P("}")
	g.P()
	if method.CanSend {
		renderRuntimeMessageStreamSend(g, streamRegistryName, sessionName, facadeName)
	}
	if method.CanRecv {
		renderRuntimeMessageStreamRecv(g, streamRegistryName, sessionName, facadeName)
	}
	if method.CanCloseSend {
		renderRuntimeMessageStreamCloseSend(g, streamRegistryName, sessionName, facadeName)
	}
	renderRuntimeMessageStreamFinish(g, streamRegistryName, sessionName, method, facadeName)
	renderRuntimeMessageStreamCancel(g, streamRegistryName, sessionName, facadeName)
}

func renderRuntimeNativeStreamSend(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context", method.NativeArgs, ") error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.EnsureCanSend(); err != nil { return err }")
	g.P("return session.send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamFinish(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	if method.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, method.NativeInvalidZero)
		g.P("return session.finish(ctx)")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, "rpcruntime.ErrStreamInvalidHandle")
		g.P("return session.finish(ctx)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamRecv(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	renderRuntimeLoadSessionWithReturn(g, streamRegistryName, sessionName, method.NativeInvalidZero)
	g.P("return session.recv(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.MarkSendClosed(); err != nil { return err }")
	g.P("return session.closeSend(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeNativeStreamCancel(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	renderRuntimeTakeSessionWithoutFinalize(g, streamRegistryName, sessionName)
	g.P("return session.lifecycle.Cancel(func() error { return session.cancel(ctx) })")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Send(ctx context.Context, req []byte) error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.EnsureCanSend(); err != nil { return err }")
	g.P("return session.send(ctx, req)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamFinish(g *protogen.GeneratedFile, streamRegistryName, sessionName string, method runtimeAdapterMethod, facadeName string) {
	if method.FinishReturnsResponse {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) ([]byte, error) {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, "nil, rpcruntime.ErrStreamInvalidHandle")
		g.P("return session.finish(ctx)")
	} else {
		g.P("func (s ", facadeName, ") Finish(ctx context.Context) error {")
		renderRuntimeTakeSession(g, streamRegistryName, sessionName, "rpcruntime.ErrStreamInvalidHandle")
		g.P("return session.finish(ctx)")
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamRecv(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Recv(ctx context.Context) ([]byte, error) {")
	renderRuntimeLoadSessionWithReturn(g, streamRegistryName, sessionName, "nil, rpcruntime.ErrStreamInvalidHandle")
	g.P("return session.recv(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCloseSend(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") CloseSend(ctx context.Context) error {")
	renderRuntimeLoadSession(g, streamRegistryName, sessionName)
	g.P("if err := session.lifecycle.MarkSendClosed(); err != nil { return err }")
	g.P("return session.closeSend(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageStreamCancel(g *protogen.GeneratedFile, streamRegistryName, sessionName, facadeName string) {
	g.P("func (s ", facadeName, ") Cancel(ctx context.Context) error {")
	renderRuntimeTakeSessionWithoutFinalize(g, streamRegistryName, sessionName)
	g.P("return session.lifecycle.Cancel(func() error { return session.cancel(ctx) })")
	g.P("}")
	g.P()
}

func renderRuntimeLoadSession(g *protogen.GeneratedFile, registryName, sessionName string) {
	renderRuntimeLoadSessionWithReturn(g, registryName, sessionName, "rpcruntime.ErrStreamInvalidHandle")
}

func renderRuntimeLoadSessionWithReturn(g *protogen.GeneratedFile, registryName, sessionName, invalidReturn string) {
	g.P("value, ok := ", registryName, ".Load(s.handle)")
	g.P("if !ok { return ", invalidReturn, " }")
	g.P("session, ok := value.(*", sessionName, ")")
	g.P("if !ok { return ", invalidReturn, " }")
}

func renderRuntimeTakeSession(g *protogen.GeneratedFile, registryName, sessionName, invalidReturn string) {
	renderRuntimeLoadSessionWithReturn(g, registryName, sessionName, invalidReturn)
	g.P("taken, ok := ", registryName, ".Take(s.handle)")
	g.P("if !ok || taken != session { return ", invalidReturn, " }")
	g.P("if !session.lifecycle.Finalize() { return ", invalidReturn, " }")
}

func renderRuntimeTakeSessionWithoutFinalize(g *protogen.GeneratedFile, registryName, sessionName string) {
	renderRuntimeLoadSession(g, registryName, sessionName)
	g.P("taken, ok := ", registryName, ".Take(s.handle)")
	g.P("if !ok || taken != session { return rpcruntime.ErrStreamInvalidHandle }")
}

func nativeRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "NativeStream"
}

func messageRuntimeStreamFacadeName(serviceName string, method runtimeAdapterMethod) string {
	return serviceName + method.MethodGoName + "MessageStream"
}

func runtimeFinalNativeSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "NativeFinalSession"
}

func runtimeFinalMessageSessionName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "MessageFinalSession"
}

func renderRuntimeRegistrations(g *protogen.GeneratedFile, service ServicePlan, nativeAdapterName, messageAdapterName string, methods []runtimeAdapterMethod, codecEnabled bool, activeName string) {
	serviceName := service.GoName
	recordName := lowerInitial(serviceName) + "ActiveServerRecord"
	nativeAdapter := lowerInitial(serviceName) + "NativeServerAdapter"
	messageAdapter := lowerInitial(serviceName) + "MessageServerAdapter"

	g.P("func register", serviceName, "GoNativeServer(server ", nativeAdapterName, ") error {")
	g.P("if server == nil { return ", serviceName, "NativeServerUnavailableErr }")
	g.P("adapter := &", nativeAdapter, "{server: server}")
	renderRuntimeNativeRecord(g, service, methods, codecEnabled, activeName, recordName, "adapter")
	g.P("}")
	g.P()

	g.P("func Register", serviceName, "CGONativeServer(server ", nativeAdapterName, ") error {")
	g.P("return register", serviceName, "GoNativeServer(server)")
	g.P("}")
	g.P()

	g.P("func register", serviceName, "CGOMessageServer(server ", messageAdapterName, ") error {")
	g.P("if server == nil { return ", serviceName, "MessageServerUnavailableErr }")
	g.P("adapter := &", messageAdapter, "{server: server}")
	renderRuntimeMessageRecord(g, service, methods, codecEnabled, activeName, recordName, "adapter")
	g.P("}")
	g.P()

	if service.Adapters.Has(AdapterTokenMessageConnect) {
		handlerName := service.GoName + "Handler"
		clientName := service.GoName + "Client"
		g.P("func Register", service.GoName, "ConnectHandler(handler ", handlerName, ") error {")
		g.P("if handler == nil { return ", serviceName, "MessageServerUnavailableErr }")
		renderRuntimeTransportMessageRecord(g, service, methods, codecEnabled, activeName, recordName, "handler", "connect handler")
		g.P("}")
		g.P()
		g.P("func Register", service.GoName, "ConnectRemoteServer(client ", clientName, ") error {")
		g.P("if client == nil { return ", serviceName, "MessageServerUnavailableErr }")
		renderRuntimeTransportMessageRecord(g, service, methods, codecEnabled, activeName, recordName, "client", "connect remote")
		g.P("}")
		g.P()
	}
	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		serverName := service.GoName + "Server"
		clientName := service.GoName + "Client"
		g.P("func Register", service.GoName, "GRPCServer(server ", serverName, ") error {")
		g.P("if server == nil { return ", serviceName, "MessageServerUnavailableErr }")
		renderRuntimeTransportMessageRecord(g, service, methods, codecEnabled, activeName, recordName, "server", "grpc server")
		g.P("}")
		g.P()
		g.P("func Register", service.GoName, "GRPCRemoteServer(client ", clientName, ") error {")
		g.P("if client == nil { return ", serviceName, "MessageServerUnavailableErr }")
		renderRuntimeTransportMessageRecord(g, service, methods, codecEnabled, activeName, recordName, "client", "grpc remote")
		g.P("}")
		g.P()
	}
}

func renderRuntimeTransportMessageRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, codecEnabled bool, activeName, recordName, transportExpr, label string) {
	g.P("record := &", recordName, "{")
	g.P("}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("record.invokeMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) ([]byte, error) {")
			renderRuntimeTransportUnaryMessageCall(g, service, method, transportExpr, label, "req")
			g.P("}")
			g.P("record.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			if codecEnabled {
				g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				g.P("var messageResp []byte")
				renderRuntimeTransportUnaryNativeMessageCall(g, service, method, transportExpr, label, "messageReq")
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				for _, decl := range method.NativeVarDecls {
					g.P(decl)
				}
				if method.NativeNames == "" {
					g.P("err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
				} else {
					g.P(method.NativeNames, ", err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
				}
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				if method.NativeNames == "" {
					g.P("return nil")
				} else {
					g.P("return ", method.NativeNames, ", nil")
				}
			} else {
				g.P("return ", method.NativeConverterZero)
			}
			g.P("}")
			continue
		}
		renderRuntimeTransportMessageStreamRecord(g, service, method, codecEnabled, transportExpr, label)
	}
	g.P(activeName, ".Store(record)")
	g.P("return nil")
}

func renderRuntimeTransportUnaryMessageCall(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr, label, reqExpr string) {
	methodPlan := methodForRuntimeService(service, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	g.P("messageReq := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(", reqExpr, ", messageReq); err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: ", label, " request protobuf unmarshal failed: %w\", err)")
	g.P("}")
	g.P("messageResp, err := ", transportExpr, ".", method.MethodGoName, "(ctx, messageReq)")
	g.P("if err != nil { return nil, err }")
	g.P("resp, err := proto.Marshal(messageResp)")
	g.P("if err != nil {")
	g.P("return nil, fmt.Errorf(\"rpccgo: ", label, " response protobuf marshal failed: %w\", err)")
	g.P("}")
	g.P("return resp, nil")
}

func renderRuntimeTransportUnaryNativeMessageCall(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr, label, reqExpr string) {
	methodPlan := methodForRuntimeService(service, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	g.P("directReq := new(", reqType, ")")
	g.P("if err = proto.Unmarshal(", reqExpr, ", directReq); err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("directResp, err := ", transportExpr, ".", method.MethodGoName, "(ctx, directReq)")
	g.P("if err != nil { return ", method.NativeErrZero, " }")
	g.P("messageResp, err = proto.Marshal(directResp)")
}

func renderRuntimeTransportMessageStreamRecord(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, codecEnabled bool, transportExpr, label string) {
	nativeSession := runtimeFinalNativeSessionName(service.GoName, method)
	messageSession := runtimeFinalMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "req")
	} else {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		if codecEnabled {
			g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
			g.P("if err != nil { return nil, err }")
			renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "messageReq")
			g.P("if err != nil { return nil, err }")
			renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	} else {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		if codecEnabled {
			renderRuntimeTransportMessageStreamSource(g, service, method, transportExpr, label, "ctx", "")
			g.P("if err != nil { return nil, err }")
			renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	}
	g.P("}")
}

func renderRuntimeTransportMessageStreamSource(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, transportExpr, label, ctxExpr, reqExpr string) {
	constructor := ""
	switch label {
	case "connect handler":
		constructor = "new" + connectDirectMessageSessionName(service.GoName, method)
	case "connect remote":
		constructor = "new" + connectRemoteMessageSessionName(service.GoName, method)
	case "grpc server":
		constructor = "new" + grpcDirectMessageSessionName(service.GoName, method)
	case "grpc remote":
		constructor = "new" + grpcRemoteMessageSessionName(service.GoName, method)
	default:
		panic("unknown transport message source")
	}
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("source, err := ", constructor, "(", ctxExpr, ", ", transportExpr, ", ", reqExpr, ")")
		return
	}
	if label == "connect handler" || label == "grpc server" {
		g.P("source := ", constructor, "(", ctxExpr, ", ", transportExpr, ")")
		g.P("var err error")
		return
	}
	g.P("source, err := ", constructor, "(", ctxExpr, ", ", transportExpr, ")")
}

func renderRuntimeTransportMessageSessions(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		if service.Adapters.Has(AdapterTokenMessageConnect) {
			renderConnectDirectMessageSession(g, service, method)
			renderConnectRemoteMessageSession(g, service, method)
		}
		if service.Adapters.Has(AdapterTokenMessageGRPC) {
			renderGRPCDirectMessageSession(g, service, method)
			renderGRPCRemoteMessageSession(g, service, method)
		}
	}
}

func renderRuntimeNativeRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, codecEnabled bool, activeName, recordName, adapterExpr string) {
	g.P("record := &", recordName, "{}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("record.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			g.P("return ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			g.P("}")
			g.P("record.invokeMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) ([]byte, error) {")
			if codecEnabled {
				g.P("var resp []byte")
				g.P("err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
				if method.NativeNames == "" {
					g.P("callErr := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
				} else {
					g.P(method.NativeNames, ", callErr := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
				}
				g.P("if callErr != nil { return callErr }")
				g.P("messageResp, err := ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
				g.P("if err != nil { return err }")
				g.P("resp = messageResp")
				g.P("return nil")
				g.P("})")
				g.P("if err != nil { return nil, err }")
				g.P("return resp, nil")
			} else {
				g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
			}
			g.P("}")
			continue
		}
		renderRuntimeNativeStreamRecord(g, service, method, codecEnabled, adapterExpr)
	}
	g.P(activeName, ".Store(record)")
	g.P("return nil")
}

func renderRuntimeNativeStreamRecord(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, codecEnabled bool, adapterExpr string) {
	nativeSession := runtimeFinalNativeSessionName(service.GoName, method)
	messageSession := runtimeFinalMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	} else {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx)")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeNativeFinalSessionFromSource(g, nativeSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		if codecEnabled {
			g.P("var final *", messageSession)
			g.P("err := ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
			g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			g.P("if err != nil { return err }")
			renderRuntimeMessageFinalSessionFromNativeSource(g, service, messageSession, method, "source", true)
			g.P("return nil")
			g.P("})")
			g.P("if err != nil { return nil, err }")
			g.P("return final, nil")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	} else {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		if codecEnabled {
			g.P("source, err := ", adapterExpr, ".", method.AdapterName, "(ctx)")
			g.P("if err != nil { return nil, err }")
			renderRuntimeMessageFinalSessionFromNativeSource(g, service, messageSession, method, "source", false)
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	}
	g.P("}")
}

func renderRuntimeNativeFinalSessionFromSource(g *protogen.GeneratedFile, sessionName string, method runtimeAdapterMethod, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.CanSend {
		g.P("send: ", sourceExpr, ".Send,")
	}
	if method.CanRecv {
		g.P("recv: ", sourceExpr, ".Recv,")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	g.P("finish: ", sourceExpr, ".Finish,")
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}

func renderRuntimeMessageFinalSessionFromNativeSource(g *protogen.GeneratedFile, service ServicePlan, sessionName string, method runtimeAdapterMethod, sourceExpr string, assign bool) {
	target := "return "
	if assign {
		target = "final = "
	}
	g.P(target, "&", sessionName, "{")
	if method.CanSend {
		g.P("send: func(ctx context.Context, req []byte) error {")
		g.P("return ", codecMessageToNativeRequestName(service, methodForRuntimeService(service, method)), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
		g.P("return ", sourceExpr, ".Send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("})")
		g.P("},")
	}
	if method.CanRecv {
		g.P("recv: func(ctx context.Context) ([]byte, error) {")
		if method.NativeNames == "" {
			g.P("err := ", sourceExpr, ".Recv(ctx)")
		} else {
			g.P(method.NativeNames, ", err := ", sourceExpr, ".Recv(ctx)")
		}
		g.P("if err != nil { return nil, err }")
		g.P("return ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
		g.P("},")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	if method.FinishReturnsResponse {
		g.P("finish: func(ctx context.Context) ([]byte, error) {")
		if method.NativeNames == "" {
			g.P("err := ", sourceExpr, ".Finish(ctx)")
		} else {
			g.P(method.NativeNames, ", err := ", sourceExpr, ".Finish(ctx)")
		}
		g.P("if err != nil { return nil, err }")
		g.P("return ", codecNativeResponseToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeNames, ")")
		g.P("},")
	} else {
		g.P("finish: ", sourceExpr, ".Finish,")
	}
	g.P("cancel: ", sourceExpr, ".Cancel,")
	if assign {
		g.P("}")
	} else {
		g.P("}, nil")
	}
}

func renderRuntimeMessageRecord(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, codecEnabled bool, activeName, recordName, adapterExpr string) {
	g.P("record := &", recordName, "{}")
	for _, method := range methods {
		if !method.Streaming {
			g.P("record.invokeMessage", method.MethodGoName, " = ", adapterExpr, ".", method.MethodGoName)
			g.P("record.invokeNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
			if codecEnabled {
				g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				g.P("messageResp, err := ", adapterExpr, ".", method.MethodGoName, "(ctx, messageReq)")
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				for _, decl := range method.NativeVarDecls {
					g.P(decl)
				}
				if method.NativeNames == "" {
					g.P("err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
				} else {
					g.P(method.NativeNames, ", err = ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
				}
				g.P("if err != nil { return ", method.NativeErrZero, " }")
				if method.NativeNames == "" {
					g.P("return nil")
				} else {
					g.P("return ", method.NativeNames, ", nil")
				}
			} else {
				g.P("return ", method.NativeConverterZero)
			}
			g.P("}")
			continue
		}
		renderRuntimeMessageStreamRecord(g, service, method, codecEnabled, adapterExpr)
	}
	g.P(activeName, ".Store(record)")
	g.P("return nil")
}

func renderRuntimeMessageStreamRecord(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod, codecEnabled bool, adapterExpr string) {
	nativeSession := runtimeFinalNativeSessionName(service.GoName, method)
	messageSession := runtimeFinalMessageSessionName(service.GoName, method)
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context, req []byte) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx, req)")
	} else {
		g.P("record.startMessage", method.MethodGoName, " = func(ctx context.Context) (*", messageSession, ", error) {")
		g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx)")
	}
	g.P("if err != nil { return nil, err }")
	renderRuntimeMessageFinalSessionFromSource(g, messageSession, method, "source")
	g.P("}")
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context", method.NativeArgs, ") (*", nativeSession, ", error) {")
		if codecEnabled {
			g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
			g.P("if err != nil { return nil, err }")
			g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx, messageReq)")
			g.P("if err != nil { return nil, err }")
			renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	} else {
		g.P("record.startNative", method.MethodGoName, " = func(ctx context.Context) (*", nativeSession, ", error) {")
		if codecEnabled {
			g.P("source, err := ", adapterExpr, ".Start", method.MethodGoName, "(ctx)")
			g.P("if err != nil { return nil, err }")
			renderRuntimeNativeFinalSessionFromMessageSource(g, service, nativeSession, method, "source")
		} else {
			g.P("return nil, ", service.GoName, "NativeMessageConverterUnavailableErr")
		}
	}
	g.P("}")
}

func renderRuntimeMessageFinalSessionFromSource(g *protogen.GeneratedFile, sessionName string, method runtimeAdapterMethod, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.CanSend {
		g.P("send: ", sourceExpr, ".Send,")
	}
	if method.CanRecv {
		g.P("recv: ", sourceExpr, ".Recv,")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	g.P("finish: ", sourceExpr, ".Finish,")
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}

func renderRuntimeNativeFinalSessionFromMessageSource(g *protogen.GeneratedFile, service ServicePlan, sessionName string, method runtimeAdapterMethod, sourceExpr string) {
	g.P("return &", sessionName, "{")
	if method.CanSend {
		g.P("send: func(ctx context.Context", method.NativeArgs, ") error {")
		g.P("messageReq, err := ", codecNativeRequestToMessageName(service, methodForRuntimeService(service, method)), "(", method.NativeArgNames, ")")
		g.P("if err != nil { return err }")
		g.P("return ", sourceExpr, ".Send(ctx, messageReq)")
		g.P("},")
	}
	if method.CanRecv {
		g.P("recv: func(ctx context.Context) (", method.NativeReturns, ") {")
		g.P("messageResp, err := ", sourceExpr, ".Recv(ctx)")
		g.P("if err != nil { return ", method.NativeErrZero, " }")
		g.P("return ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
		g.P("},")
	}
	if method.CanCloseSend {
		g.P("closeSend: ", sourceExpr, ".CloseSend,")
	}
	if method.FinishReturnsResponse {
		g.P("finish: func(ctx context.Context) (", method.NativeReturns, ") {")
		g.P("messageResp, err := ", sourceExpr, ".Finish(ctx)")
		g.P("if err != nil { return ", method.NativeErrZero, " }")
		g.P("return ", codecMessageToNativeResponseName(service, methodForRuntimeService(service, method)), "(messageResp)")
		g.P("},")
	} else {
		g.P("finish: ", sourceExpr, ".Finish,")
	}
	g.P("cancel: ", sourceExpr, ".Cancel,")
	g.P("}, nil")
}

func renderRuntimeEntrypoints(g *protogen.GeneratedFile, serviceName, adapterName, activeName, streamRegistryName string, methods []runtimeAdapterMethod) {
	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func Invoke", serviceName, "Native", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return ", method.NativeNoActiveZero, " }")
		g.P("return active.invokeNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("}")
		g.P()
		g.P("func Invoke", serviceName, "Message", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return nil, rpcruntime.ErrNoActiveServer }")
		g.P("return active.invokeMessage", method.MethodGoName, "(ctx, req)")
		g.P("}")
		g.P()
	}
	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		renderRuntimeStartEntrypoints(g, serviceName, activeName, streamRegistryName, method)
	}
}

func renderRuntimeStartEntrypoints(g *protogen.GeneratedFile, serviceName, activeName, streamRegistryName string, method runtimeAdapterMethod) {
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("func Start", serviceName, "Native", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startNative", method.MethodGoName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	} else {
		g.P("func Start", serviceName, "Native", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startNative", method.MethodGoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
	g.P("}")
	g.P()
	if runtimeStreamShapeFor(method) == runtimeStreamServer {
		g.P("func Start", serviceName, "Message", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startMessage", method.MethodGoName, "(ctx, req)")
	} else {
		g.P("func Start", serviceName, "Message", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("active := ", activeName, ".Load()")
		g.P("if active == nil { return 0, rpcruntime.ErrNoActiveServer }")
		g.P("session, err := active.startMessage", method.MethodGoName, "(ctx)")
	}
	g.P("if err != nil { return 0, err }")
	g.P("handle, err := ", streamRegistryName, ".Create(session)")
	g.P("if err != nil { _ = session.cancel(ctx); return 0, err }")
	g.P("return handle, nil")
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
	if method.CanSend {
		g.P("requests chan []byte")
		g.P("closeRequests sync.Once")
	}
	if method.FinishReturnsResponse {
		g.P("result chan ", resultName)
	} else {
		g.P("responses chan ", resultName)
	}
	g.P("}")
	g.P()
	switch runtimeStreamShapeFor(method) {
	case runtimeStreamClient:
		renderConnectDirectClientStreamSession(g, method, wrapperName, resultName, reqType, handlerName)
	case runtimeStreamServer:
		renderConnectDirectServerStreamSession(g, method, wrapperName, resultName, reqType, respType, handlerName)
	case runtimeStreamBidi:
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
	renderConnectDirectRecvFinishCancel(g, wrapperName)
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
	renderConnectDirectRecvFinishCancel(g, wrapperName)
}

func renderConnectDirectRecvFinishCancel(g *protogen.GeneratedFile, wrapperName string) {
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
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) error {")
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
	switch runtimeStreamShapeFor(method) {
	case runtimeStreamClient:
		clientType := "interface { " + method.MethodGoName + "(context.Context) (*connect.ClientStreamForClientSimple[" + strings.TrimPrefix(reqType, "*") + ", " + strings.TrimPrefix(respType, "*") + "], error) }"
		renderConnectRemoteClientStreamSession(g, method, wrapperName, reqType, respType, clientType)
	case runtimeStreamServer:
		clientType := "interface { " + method.MethodGoName + "(context.Context, *" + strings.TrimPrefix(reqType, "*") + ") (*connect.ServerStreamForClient[" + strings.TrimPrefix(respType, "*") + "], error) }"
		renderConnectRemoteServerStreamSession(g, method, wrapperName, reqType, respType, clientType)
	case runtimeStreamBidi:
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
	renderConnectRemoteRecvFinishCancel(g, wrapperName, "server stream")
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
	renderConnectRemoteBidiRecvFinishCancel(g, wrapperName)
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseRequest()")
	g.P("}")
	g.P()
}

func renderConnectRemoteRecvFinishCancel(g *protogen.GeneratedFile, wrapperName, label string) {
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
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) error {")
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

func renderConnectRemoteBidiRecvFinishCancel(g *protogen.GeneratedFile, wrapperName string) {
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
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) error {")
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
	if method.CanSend {
		g.P("requests chan []byte")
		g.P("closeRequests sync.Once")
	}
	if method.FinishReturnsResponse {
		g.P("result chan ", resultName)
		g.P("resultOnce sync.Once")
	} else {
		g.P("responses chan ", resultName)
	}
	g.P("header metadata.MD")
	g.P("trailer metadata.MD")
	g.P("}")
	g.P()
	switch runtimeStreamShapeFor(method) {
	case runtimeStreamClient:
		renderGRPCDirectClientStreamSession(g, method, wrapperName, resultName, reqType, respType, serverName)
	case runtimeStreamServer:
		renderGRPCDirectServerStreamSession(g, method, wrapperName, resultName, reqType, respType, serverName)
	case runtimeStreamBidi:
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
	renderConnectDirectRecvFinishCancel(g, wrapperName)
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
	renderConnectDirectRecvFinishCancel(g, wrapperName)
}

func renderGRPCRemoteMessageSession(g *protogen.GeneratedFile, service ServicePlan, method runtimeAdapterMethod) {
	methodPlan := methodForRuntimeService(service, method)
	wrapperName := grpcRemoteMessageSessionName(service.GoName, method)
	reqType := qualifiedMethodType(g, methodPlan.Request)
	respType := qualifiedMethodType(g, methodPlan.Response)
	clientName := service.GoName + "Client"
	switch runtimeStreamShapeFor(method) {
	case runtimeStreamClient:
		renderGRPCRemoteClientStreamSession(g, method, wrapperName, reqType, respType, clientName)
	case runtimeStreamServer:
		renderGRPCRemoteServerStreamSession(g, method, wrapperName, reqType, respType, clientName)
	case runtimeStreamBidi:
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
	renderGRPCRemoteRecvFinishCancel(g, wrapperName, "server stream")
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
	renderGRPCRemoteRecvFinishCancel(g, wrapperName, "bidi stream")
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
}

func renderGRPCRemoteRecvFinishCancel(g *protogen.GeneratedFile, wrapperName, label string) {
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
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) error {")
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

func methodMessageSessionName(method runtimeAdapterMethod) string {
	return strings.Replace(method.SessionName, "NativeStreamSession", "MessageStreamSession", 1)
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
