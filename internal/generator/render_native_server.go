package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
	if err := validateNativeServerSymbols(service); err != nil {
		return err
	}
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))
	runtimeMethods, err := buildRuntimeMethodProjectionsWithMessageTypes(g, service, false)
	if err != nil {
		return err
	}
	errorNames := nativeServerErrorNamesFor(service)

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	if len(runtimeMethods) > 0 {
		g.P(`context "context"`)
	}
	g.P(`errors "errors"`)
	if nativeServerHasStreamingMethod(service) {
		g.P(`io "io"`)
		if nativeServerHasClientInputStreamingMethod(service) {
			g.P(`sync "sync"`)
		}
	}
	if nativeServerNeedsRPCRuntime(service) {
		g.P(`rpcruntime "rpccgo/rpcruntime"`)
	}
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()
	g.P("var (")
	g.P(errorNames.RequestBridgeNotImplemented, ` = errors.New("rpccgo: native request bridge is not implemented")`)
	g.P(errorNames.StreamBridgeNotImplemented, ` = errors.New("rpccgo: native stream bridge is not implemented")`)
	g.P(errorNames.StreamIsNil, ` = errors.New("rpccgo: native stream is nil")`)
	g.P(errorNames.StreamClosed, ` = errors.New("rpccgo: native stream is closed")`)
	g.P(")")
	g.P()

	renderGoNativeServerInterface(g, service, service.GoName+"NativeServer")
	renderGoNativeStreamInterfaces(g, service)
	renderUnimplementedGoNativeServer(g, service)
	renderGoNativeRegistration(g, service, service.GoName+"NativeServer", "")
	renderGoNativeAdapter(g, service, runtimeMethods, service.GoName+"NativeServer", lowerInitial(service.GoName)+"NativeBinding", lowerInitial(service.GoName)+"NativeActiveBinding", errorNames)
	return nil
}

func nativeServerNeedsRPCRuntime(service ServicePlan) bool {
	for _, method := range service.Methods {
		for _, field := range method.Contract.Native.RequestFields {
			if field.Repeated {
				return true
			}
			switch field.Kind {
			case FieldKindString, FieldKindBytes, FieldKindMessage:
				return true
			}
		}
	}
	return false
}

func renderGoNativeServerInterface(g *protogen.GeneratedFile, service ServicePlan, serverName string) {
	renderDocLine(g, service.DocComment, "type ", serverName, " interface {")
	for _, method := range service.Methods {
		requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
		switch method.Streaming {
		case StreamingKindUnary:
			renderDocLine(g, method.DocComment, method.GoName, "(ctx context.Context", requestParams, ") (", responseReturns, ")")
		case StreamingKindClientStreaming:
			renderDocLine(g, method.DocComment, method.GoName, "(ctx context.Context, stream ", service.GoName, method.GoName, "NativeClientStream) (", responseReturns, ")")
		case StreamingKindServerStreaming:
			renderDocLine(g, method.DocComment, method.GoName, "(ctx context.Context", requestParams, ", stream ", service.GoName, method.GoName, "NativeServerStream) error")
		case StreamingKindBidiStreaming:
			renderDocLine(g, method.DocComment, method.GoName, "(ctx context.Context, stream ", service.GoName, method.GoName, "NativeBidiStream) error")
		}
	}
	g.P("}")
	g.P()
}

func renderGoNativeStreamInterfaces(g *protogen.GeneratedFile, service ServicePlan) {
	for _, method := range service.Methods {
		requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
		requestReturns := nativeGoRequestReturns(g, method.Contract.Native.RequestFields)
		responseParams := nativeGoResponseParams(g, method.Contract.Native.ResponseFields)
		switch method.Streaming {
		case StreamingKindClientStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeClientStream interface {")
			g.P("Recv(ctx context.Context) (", requestReturns, ")")
			g.P("}")
			g.P()
		case StreamingKindServerStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeServerStream interface {")
			g.P("Send(ctx context.Context", responseParams, ") error")
			g.P("}")
			g.P()
		case StreamingKindBidiStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeBidiStream interface {")
			g.P("Recv(ctx context.Context) (", requestReturns, ")")
			g.P("Send(ctx context.Context", responseParams, ") error")
			g.P("}")
			g.P()
		}
		_ = requestParams
		_ = responseReturns
	}
}

func renderUnimplementedGoNativeServer(g *protogen.GeneratedFile, service ServicePlan) {
	serverName := "Unimplemented" + service.GoName + "NativeServer"
	g.P("type ", serverName, " struct{}")
	g.P()
	for _, method := range service.Methods {
		requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
		errExpr := `errors.New("rpccgo: ` + service.GoName + "." + method.GoName + ` native server method is not implemented")`
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("func (", serverName, ") ", method.GoName, "(ctx context.Context", requestParams, ") (", responseReturns, ") {")
			g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, errExpr))
			g.P("}")
		case StreamingKindClientStreaming:
			g.P("func (", serverName, ") ", method.GoName, "(ctx context.Context, stream ", service.GoName, method.GoName, "NativeClientStream) (", responseReturns, ") {")
			g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, errExpr))
			g.P("}")
		case StreamingKindServerStreaming:
			g.P("func (", serverName, ") ", method.GoName, "(ctx context.Context", requestParams, ", stream ", service.GoName, method.GoName, "NativeServerStream) error {")
			g.P("return ", errExpr)
			g.P("}")
		case StreamingKindBidiStreaming:
			g.P("func (", serverName, ") ", method.GoName, "(ctx context.Context, stream ", service.GoName, method.GoName, "NativeBidiStream) error {")
			g.P("return ", errExpr)
			g.P("}")
		}
		g.P()
	}
}

func renderGoNativeAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, serverName, adapterName, bindingName string, errorNames nativeServerErrorNames) {
	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}

	g.P("// ", adapterName, " exposes a native server implementation through the")
	g.P("// method shape used while building a ", bindingName, ".")
	g.P("type ", adapterName, " struct {")
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.Identity.GoName]
		if !ok {
			continue
		}
		fieldName := lowerInitial(method.GoName)
		requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(fieldName, " func(ctx context.Context", requestParams, ") (", responseReturns, ")")
		case StreamingKindClientStreaming:
			g.P(fieldName, " func(ctx context.Context, stream ", service.GoName, method.GoName, "NativeClientStream) (", responseReturns, ")")
		case StreamingKindServerStreaming:
			g.P(fieldName, " func(ctx context.Context", requestParams, ", stream ", service.GoName, method.GoName, "NativeServerStream) error")
		case StreamingKindBidiStreaming:
			g.P(fieldName, " func(ctx context.Context, stream ", service.GoName, method.GoName, "NativeBidiStream) error")
		}
	}
	g.P("}")
	g.P()

	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.Identity.GoName]
		if !ok {
			renderGoNativeFallbackAdapterMethod(g, adapterName, runtimeMethod)
			continue
		}
		switch method.Streaming {
		case StreamingKindUnary:
			renderGoNativeUnaryAdapterMethod(g, adapterName, method, errorNames)
		case StreamingKindClientStreaming:
			renderGoNativeClientStreamAdapterMethod(g, service, adapterName, method, errorNames)
		case StreamingKindServerStreaming:
			renderGoNativeServerStreamAdapterMethod(g, service, adapterName, method, errorNames)
		case StreamingKindBidiStreaming:
			renderGoNativeBidiStreamAdapterMethod(g, service, adapterName, method, errorNames)
		}
	}
	_ = serverName
}

func renderGoNativeUnaryAdapterMethod(g *protogen.GeneratedFile, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
	responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
	requestArgs := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context", requestParams, ") (", responseReturns, ") {")
	if len(method.Contract.Native.RequestFields) == 0 {
		g.P("return a.", lowerInitial(method.GoName), "(ctx)")
	} else {
		g.P("return a.", lowerInitial(method.GoName), "(ctx, ", requestArgs, ")")
	}
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGoNativeClientStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeClientStreamSession"
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", service.GoName, method.GoName, "NativeClientStream) (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	g.P("return a.", lowerInitial(method.GoName), "(ctx, stream)")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", receiver, "{")
	g.P("ctx: streamCtx,")
	g.P("cancel: cancel,")
	g.P("requests: make(chan ", receiver, "Request, 16),")
	g.P("sendDone: make(chan struct{}),")
	g.P("done: make(chan struct{}),")
	g.P("}")
	g.P("go func() {")
	g.P("defer close(session.done)")
	g.P(renderNativeClientStreamResultAssignment(method), "a.", method.GoName, "(streamCtx, session)")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()

	renderNativeRequestEnvelope(g, receiver+"Request", method.Contract.Native.RequestFields)
	renderNativeClientStreamResult(g, receiver+"Result", method.Contract.Native.ResponseFields)
	g.P("type ", receiver, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	g.P("requests chan ", receiver, "Request")
	g.P("sendDone chan struct{}")
	g.P("closeSendOnce sync.Once")
	g.P("received chan struct{}")
	g.P("done chan struct{}")
	g.P("result ", receiver, "Result")
	g.P("}")
	g.P()
	renderGoNativeClientStreamFacadeRecv(g, receiver, method)
	renderGoNativeClientStreamSend(g, receiver, method, errorNames)
	renderGoNativeClientStreamFinish(g, receiver, method, errorNames)
	renderGeneratedStreamCancel(g, receiver)
}

func renderGoNativeServerStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeServerStreamSession"
	requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context", requestParams, ", stream ", service.GoName, method.GoName, "NativeServerStream) error {")
	if len(method.Contract.Native.RequestFields) == 0 {
		g.P("return a.", lowerInitial(method.GoName), "(ctx, stream)")
	} else {
		g.P("return a.", lowerInitial(method.GoName), "(ctx, ", requestArgs, ", stream)")
	}
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context", requestParams, ") (", sessionName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", receiver, "{")
	g.P("ctx: streamCtx,")
	g.P("cancel: cancel,")
	g.P("responses: make(chan ", receiver, "Response, 1),")
	g.P("done: make(chan struct{}),")
	g.P("}")
	g.P("go func() {")
	g.P("defer close(session.done)")
	g.P("defer close(session.responses)")
	if len(method.Contract.Native.RequestFields) == 0 {
		g.P("session.err = a.", method.GoName, "(streamCtx, session)")
	} else {
		g.P("session.err = a.", method.GoName, "(streamCtx, ", requestArgs, ", session)")
	}
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()

	renderNativeResponseEnvelope(g, receiver+"Response", method.Contract.Native.ResponseFields)
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
	renderGoNativeServerStreamFacadeSend(g, receiver, method, errorNames)
	renderGoNativeServerStreamRecv(g, receiver, method, errorNames)
	renderGeneratedStreamFinish(g, receiver)
	renderGeneratedStreamCancel(g, receiver)
}

func renderGoNativeBidiStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeBidiStreamSession"
	facadeName := lowerInitial(service.GoName) + method.GoName + "GoNativeBidiStreamFacade"
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", service.GoName, method.GoName, "NativeBidiStream) error {")
	g.P("return a.", lowerInitial(method.GoName), "(ctx, stream)")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("streamCtx, cancel := context.WithCancel(ctx)")
	g.P("session := &", receiver, "{")
	g.P("ctx: streamCtx,")
	g.P("cancel: cancel,")
	g.P("requests: make(chan ", receiver, "Request, 16),")
	g.P("sendDone: make(chan struct{}),")
	g.P("sendDoneReceived: make(chan struct{}),")
	g.P("responses: make(chan ", receiver, "Response, 1),")
	g.P("responseReady: make(chan struct{}),")
	g.P("requestReceived: make(chan struct{}),")
	g.P("done: make(chan struct{}),")
	g.P("}")
	g.P("go func() {")
	g.P("defer close(session.done)")
	g.P("defer close(session.responses)")
	g.P("session.err = a.", method.GoName, "(streamCtx, &", facadeName, "{session: session})")
	g.P("}()")
	g.P("return session, nil")
	g.P("}")
	g.P()

	renderNativeRequestEnvelope(g, receiver+"Request", method.Contract.Native.RequestFields)
	renderNativeResponseEnvelope(g, receiver+"Response", method.Contract.Native.ResponseFields)
	g.P("type ", receiver, " struct {")
	g.P("ctx context.Context")
	g.P("cancel context.CancelFunc")
	g.P("requests chan ", receiver, "Request")
	g.P("sendDone chan struct{}")
	g.P("sendDoneReceived chan struct{}")
	g.P("sendDoneReceivedOnce sync.Once")
	g.P("closeSendOnce sync.Once")
	g.P("responses chan ", receiver, "Response")
	g.P("responseReady chan struct{}")
	g.P("responseReadyOnce sync.Once")
	g.P("requestReceived chan struct{}")
	g.P("requestReceivedOnce sync.Once")
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
	renderGoNativeBidiStreamFacadeRecv(g, facadeName, method)
	renderGoNativeBidiStreamFacadeSend(g, facadeName, receiver+"Response", method, errorNames)
	renderGoNativeBidiStreamSend(g, receiver, method, errorNames)
	renderGoNativeBidiStreamRecv(g, receiver, method, errorNames)
	renderGoNativeBidiStreamCloseSend(g, receiver, errorNames)
	renderGeneratedStreamFinish(g, receiver)
	renderGeneratedStreamCancel(g, receiver)
}

func renderGoNativeFallbackAdapterMethod(g *protogen.GeneratedFile, adapterName string, method runtimeMethodProjection) {
	g.P("func (a *", adapterName, ") ", method.Symbols.NativeAdapterMethod, "(ctx context.Context)", method.Native.AdapterResult, " {")
	if method.Stream.Streaming {
		g.P(`return nil, errors.New("rpccgo native server method is not implemented")`)
	} else {
		g.P(`return errors.New("rpccgo native server method is not implemented")`)
	}
	g.P("}")
	g.P()
}

func renderGoNativeClientStreamFacadeRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan) {
	requestReturns := nativeGoRequestReturns(g, method.Contract.Native.RequestFields)
	ctxZeroReturns := nativeGoRequestZeroReturns(method.Contract.Native.RequestFields, "ctx.Err()")
	streamCtxZeroReturns := nativeGoRequestZeroReturns(method.Contract.Native.RequestFields, "s.ctx.Err()")
	eofReturns := nativeGoRequestZeroReturns(method.Contract.Native.RequestFields, "io.EOF")
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", requestReturns, ") {")
	g.P("select {")
	g.P("case req := <-s.requests:")
	g.P("close(req.received)")
	g.P("return ", nativeResultReturn("req", method.Contract.Native.RequestFields))
	g.P("default:")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ", ctxZeroReturns)
	g.P("case <-s.ctx.Done():")
	g.P("return ", streamCtxZeroReturns)
	g.P("case req := <-s.requests:")
	g.P("close(req.received)")
	g.P("return ", nativeResultReturn("req", method.Contract.Native.RequestFields))
	g.P("case <-s.sendDone:")
	g.P("return ", eofReturns)
	g.P("}")
	g.P("}")
	g.P()
}

func renderGoNativeClientStreamSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", requestParams, ") error {")
	g.P("select {")
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("default:")
	g.P("}")
	g.P("req := ", receiver, "Request{", nativeEnvelopeLiteralWithExtra(method.Contract.Native.RequestFields, "received: make(chan struct{})"), "}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.result.err != nil {")
	g.P("return s.result.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("case s.requests <- req:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.result.err != nil {")
	g.P("return s.result.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-req.received:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	_ = requestArgs
}

func renderGoNativeClientStreamFinish(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
	zeroReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "ctx.Err()")
	g.P("func (s *", receiver, ") Finish(ctx context.Context) (", responseReturns, ") {")
	g.P("s.closeSendOnce.Do(func() { close(s.sendDone) })")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ", zeroReturns)
	g.P("case <-s.done:")
	g.P("return ", nativeResultReturn("s.result", method.Contract.Native.ResponseFields))
	g.P("}")
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGoNativeServerStreamFacadeSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseParams := nativeGoResponseParams(g, method.Contract.Native.ResponseFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", responseParams, ") error {")
	g.P("resp := ", receiver, "Response{", nativeEnvelopeLiteralWithExtra(method.Contract.Native.ResponseFields, "received: make(chan struct{})"), "}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("if s.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("if s.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil {")
	g.P("return s.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case s.responses <- resp:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("if s.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("if s.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil {")
	g.P("return s.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-resp.received:")
	g.P("if s.ctx.Err() != nil {")
	g.P("if s.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return s.ctx.Err()")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderGoNativeServerStreamRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
	ctxZeroReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "ctx.Err()")
	streamCtxZeroReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "s.ctx.Err()")
	eofReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "io.EOF")
	errReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "s.err")
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", responseReturns, ") {")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ", ctxZeroReturns)
	g.P("case <-s.ctx.Done():")
	g.P("return ", streamCtxZeroReturns)
	g.P("case resp, ok := <-s.responses:")
	g.P("if ok {")
	g.P("s.received = resp.received")
	g.P("return ", nativeResultReturn("resp", method.Contract.Native.ResponseFields))
	g.P("}")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("<-s.done")
	g.P("if s.err != nil {")
	g.P("return ", errReturns)
	g.P("}")
	g.P("return ", eofReturns)
	g.P("}")
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGoNativeBidiStreamFacadeRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan) {
	requestReturns := nativeGoRequestReturns(g, method.Contract.Native.RequestFields)
	ctxZeroReturns := nativeGoRequestZeroReturns(method.Contract.Native.RequestFields, "ctx.Err()")
	streamCtxZeroReturns := nativeGoRequestZeroReturns(method.Contract.Native.RequestFields, "s.session.ctx.Err()")
	eofReturns := nativeGoRequestZeroReturns(method.Contract.Native.RequestFields, "io.EOF")
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", requestReturns, ") {")
	g.P("select {")
	g.P("case req := <-s.session.requests:")
	g.P("close(req.received)")
	g.P("s.session.requestReceivedOnce.Do(func() { close(s.session.requestReceived) })")
	g.P("return ", nativeResultReturn("req", method.Contract.Native.RequestFields))
	g.P("default:")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ", ctxZeroReturns)
	g.P("case <-s.session.ctx.Done():")
	g.P("return ", streamCtxZeroReturns)
	g.P("case req := <-s.session.requests:")
	g.P("close(req.received)")
	g.P("s.session.requestReceivedOnce.Do(func() { close(s.session.requestReceived) })")
	g.P("return ", nativeResultReturn("req", method.Contract.Native.RequestFields))
	g.P("case <-s.session.sendDone:")
	g.P("s.session.sendDoneReceivedOnce.Do(func() { close(s.session.sendDoneReceived) })")
	g.P("return ", eofReturns)
	g.P("}")
	g.P("}")
	g.P()
}

func renderGoNativeBidiStreamFacadeSend(g *protogen.GeneratedFile, receiver, responseType string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseParams := nativeGoResponseParams(g, method.Contract.Native.ResponseFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", responseParams, ") error {")
	g.P("resp := ", responseType, "{", nativeEnvelopeLiteralWithExtra(method.Contract.Native.ResponseFields, "received: make(chan struct{})"), "}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("if s.session.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return ctx.Err()")
	g.P("case <-s.session.ctx.Done():")
	g.P("if s.session.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return s.session.ctx.Err()")
	g.P("case <-s.session.done:")
	g.P("if s.session.err != nil {")
	g.P("return s.session.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case s.session.responses <- resp:")
	g.P("s.session.responseReadyOnce.Do(func() { close(s.session.responseReady) })")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("if s.session.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return ctx.Err()")
	g.P("case <-s.session.ctx.Done():")
	g.P("if s.session.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return s.session.ctx.Err()")
	g.P("case <-s.session.done:")
	g.P("if s.session.err != nil {")
	g.P("return s.session.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-resp.received:")
	g.P("if s.session.ctx.Err() != nil {")
	g.P("if s.session.doneRequested {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return s.session.ctx.Err()")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderGoNativeBidiStreamSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", requestParams, ") error {")
	g.P("select {")
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("default:")
	g.P("}")
	g.P("req := ", receiver, "Request{", nativeEnvelopeLiteralWithExtra(method.Contract.Native.RequestFields, "received: make(chan struct{})"), "}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil {")
	g.P("return s.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("case s.requests <- req:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil {")
	g.P("return s.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.responseReady:")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ctx.Err()")
	g.P("case <-s.ctx.Done():")
	g.P("return s.ctx.Err()")
	g.P("case <-s.done:")
	g.P("if s.err != nil {")
	g.P("return s.err")
	g.P("}")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.sendDone:")
	g.P("return ", errorNames.StreamClosed)
	g.P("case <-s.requestReceived:")
	g.P("return nil")
	g.P("case <-req.received:")
	g.P("return nil")
	g.P("}")
	g.P("return nil")
	g.P("case <-req.received:")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	_ = requestArgs
}

func renderGoNativeBidiStreamRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
	ctxZeroReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "ctx.Err()")
	streamCtxZeroReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "s.ctx.Err()")
	eofReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "io.EOF")
	errReturns := nativeGoZeroReturns(method.Contract.Native.ResponseFields, "s.err")
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", responseReturns, ") {")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("select {")
	g.P("case <-ctx.Done():")
	g.P("return ", ctxZeroReturns)
	g.P("case <-s.ctx.Done():")
	g.P("return ", streamCtxZeroReturns)
	g.P("case resp, ok := <-s.responses:")
	g.P("if ok {")
	g.P("s.responseReady = make(chan struct{})")
	g.P("s.responseReadyOnce = sync.Once{}")
	g.P("s.received = resp.received")
	g.P("return ", nativeResultReturn("resp", method.Contract.Native.ResponseFields))
	g.P("}")
	g.P("if s.received != nil {")
	g.P("close(s.received)")
	g.P("s.received = nil")
	g.P("}")
	g.P("<-s.done")
	g.P("if s.err != nil {")
	g.P("return ", errReturns)
	g.P("}")
	g.P("return ", eofReturns)
	g.P("}")
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGoNativeBidiStreamCloseSend(g *protogen.GeneratedFile, receiver string, errorNames nativeServerErrorNames) {
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	g.P("s.closeSendOnce.Do(func() { close(s.sendDone) })")
	g.P("return nil")
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGeneratedStreamCancel(g *protogen.GeneratedFile, receiver string) {
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
}

func renderGeneratedStreamFinish(g *protogen.GeneratedFile, receiver string) {
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
	g.P("if errors.Is(s.err, context.Canceled) || errors.Is(s.err, io.EOF) {")
	g.P("return nil")
	g.P("}")
	g.P("return s.err")
	g.P("}")
	g.P("}")
	g.P()
}

func renderNativeRequestEnvelope(g *protogen.GeneratedFile, name string, fields []FieldPlan) {
	g.P("type ", name, " struct {")
	for _, field := range fields {
		g.P(lowerInitial(field.GoName), " ", nativeGoRequestFieldType(g, field))
	}
	g.P("received chan struct{}")
	g.P("}")
	g.P()
}

func renderNativeResponseEnvelope(g *protogen.GeneratedFile, name string, fields []FieldPlan) {
	g.P("type ", name, " struct {")
	for _, field := range fields {
		g.P(lowerInitial(field.GoName), " ", nativeGoResponseFieldType(g, field))
	}
	g.P("received chan struct{}")
	g.P("}")
	g.P()
}

func renderNativeClientStreamResult(g *protogen.GeneratedFile, name string, fields []FieldPlan) {
	g.P("type ", name, " struct {")
	for _, field := range fields {
		g.P(lowerInitial(field.GoName), " ", nativeGoResponseFieldType(g, field))
	}
	g.P("err error")
	g.P("}")
	g.P()
}

func renderNativeClientStreamResultAssignment(method MethodPlan) string {
	names := nativeEnvelopeFieldNames(method.Contract.Native.ResponseFields)
	if names == "" {
		return "session.result.err = "
	}
	return "session.result." + strings.ReplaceAll(names, ", ", ", session.result.") + ", session.result.err = "
}

func nativeEnvelopeLiteral(fields []FieldPlan) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		parts = append(parts, name+": "+name)
	}
	return strings.Join(parts, ", ")
}

func nativeEnvelopeLiteralWithExtra(fields []FieldPlan, extra string) string {
	literal := nativeEnvelopeLiteral(fields)
	if literal == "" {
		return extra
	}
	return literal + ", " + extra
}

func nativeResultReturn(prefix string, fields []FieldPlan) string {
	parts := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		parts = append(parts, prefix+"."+lowerInitial(field.GoName))
	}
	if prefix == "s.result" {
		parts = append(parts, prefix+".err")
	} else {
		parts = append(parts, "nil")
	}
	return strings.Join(parts, ", ")
}

func nativeEnvelopeFieldNames(fields []FieldPlan) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, lowerInitial(field.GoName))
	}
	return strings.Join(names, ", ")
}

func renderGoNativeRegistration(g *protogen.GeneratedFile, service ServicePlan, serverName, adapterName string) {
	g.P("func Register", service.GoName, "GoNativeServer(server ", serverName, ") error {")
	g.P("if server == nil {")
	g.P(`return errors.New("rpccgo: `, service.GoName, ` go native server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "GoNativeServer(server)")
	g.P("}")
	g.P()
	_ = adapterName
}

func nativeGoMessageType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}

func nativeGoRequestParams(g *protogen.GeneratedFile, fields []FieldPlan) string {
	if len(fields) == 0 {
		return ""
	}
	params := make([]string, 0, len(fields))
	for _, field := range fields {
		params = append(params, lowerInitial(field.GoName)+" "+nativeGoRequestFieldType(g, field))
	}
	return ", " + strings.Join(params, ", ")
}

func nativeGoResponseReturns(g *protogen.GeneratedFile, fields []FieldPlan) string {
	returns := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		returns = append(returns, nativeGoResponseFieldType(g, field))
	}
	returns = append(returns, "error")
	return strings.Join(returns, ", ")
}

func nativeGoZeroReturns(fields []FieldPlan, errExpr string) string {
	values := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		values = append(values, nativeGoZeroValue(field))
	}
	values = append(values, errExpr)
	return strings.Join(values, ", ")
}

func nativeGoRequestZeroReturns(fields []FieldPlan, errExpr string) string {
	values := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		values = append(values, nativeGoRequestZeroValue(field))
	}
	values = append(values, errExpr)
	return strings.Join(values, ", ")
}

func nativeGoRequestArgNames(fields []FieldPlan) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, lowerInitial(field.GoName))
	}
	return strings.Join(names, ", ")
}

func nativeGoResponseValueNames(fields []FieldPlan) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, lowerInitial(field.GoName))
	}
	return strings.Join(names, ", ")
}

func nativeGoResponseParams(g *protogen.GeneratedFile, fields []FieldPlan) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, lowerInitial(field.GoName)+" "+nativeGoResponseFieldType(g, field))
	}
	return ", " + strings.Join(parts, ", ")
}

func nativeGoResponseVarDecls(g *protogen.GeneratedFile, fields []FieldPlan) []string {
	decls := make([]string, 0, len(fields))
	for _, field := range fields {
		decls = append(decls, "var "+lowerInitial(field.GoName)+" "+nativeGoResponseFieldType(g, field))
	}
	return decls
}

func nativeGoResponseResultNames(fields []FieldPlan) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, lowerInitial(field.GoName)+"Result")
	}
	return strings.Join(names, ", ")
}

func nativeGoResponseResultVarDecls(g *protogen.GeneratedFile, fields []FieldPlan) []string {
	decls := make([]string, 0, len(fields))
	for _, field := range fields {
		decls = append(decls, "var "+lowerInitial(field.GoName)+"Result "+nativeGoResponseFieldType(g, field))
	}
	return decls
}

func nativeGoCallSuffix(args string) string {
	if args == "" {
		return ""
	}
	return ", " + args
}

func nativeGoRequestFieldType(g *protogen.GeneratedFile, field FieldPlan) string {
	if field.Repeated {
		if field.Kind == FieldKindBool {
			return "*rpcruntime.RpcBoolRepeat"
		}
		return "*rpcruntime.RpcRepeat[" + nativeGoRequestRepeatElemType(g, field) + "]"
	}
	switch field.Kind {
	case FieldKindString:
		return "*rpcruntime.RpcString"
	case FieldKindBytes, FieldKindMessage:
		return "*rpcruntime.RpcBytes"
	default:
		return nativeGoScalarType(g, field)
	}
}

func nativeGoRequestRepeatElemType(g *protogen.GeneratedFile, field FieldPlan) string {
	if field.Kind == FieldKindEnum {
		return "int32"
	}
	return nativeGoScalarType(g, field)
}

func nativeGoResponseFieldType(g *protogen.GeneratedFile, field FieldPlan) string {
	if field.Repeated {
		return "[]" + nativeGoScalarType(g, field)
	}
	switch field.Kind {
	case FieldKindBytes, FieldKindMessage:
		return "[]byte"
	default:
		return nativeGoScalarType(g, field)
	}
}

func nativeGoScalarType(g *protogen.GeneratedFile, field FieldPlan) string {
	switch field.Kind {
	case FieldKindSignedInt32:
		return "int32"
	case FieldKindSignedInt64:
		return "int64"
	case FieldKindUnsignedInt32:
		return "uint32"
	case FieldKindUnsignedInt64:
		return "uint64"
	case FieldKindFloat:
		return "float32"
	case FieldKindDouble:
		return "float64"
	case FieldKindBool:
		return "bool"
	case FieldKindString:
		return "string"
	case FieldKindBytes, FieldKindMessage:
		return "[]byte"
	case FieldKindEnum:
		return nativeGoEnumType(g, field)
	default:
		return "any"
	}
}

func nativeGoZeroValue(field FieldPlan) string {
	if field.Repeated || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage {
		return "nil"
	}
	switch field.Kind {
	case FieldKindBool:
		return "false"
	case FieldKindString:
		return `""`
	default:
		return "0"
	}
}

func nativeGoMessagePackagePrefix(g *protogen.GeneratedFile, message MethodIOPlan) string {
	qualified := g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
	if strings.HasSuffix(qualified, "."+message.GoName) {
		return qualified[:len(qualified)-len(message.GoName)]
	}
	return ""
}

type nativeServerErrorNames struct {
	RequestBridgeNotImplemented string
	StreamBridgeNotImplemented  string
	StreamIsNil                 string
	StreamClosed                string
}

func nativeServerErrorNamesFor(service ServicePlan) nativeServerErrorNames {
	prefix := lowerInitial(service.GoName)
	return nativeServerErrorNames{
		RequestBridgeNotImplemented: prefix + "NativeRequestBridgeNotImplemented",
		StreamBridgeNotImplemented:  prefix + "NativeStreamBridgeNotImplemented",
		StreamIsNil:                 prefix + "NativeStreamIsNil",
		StreamClosed:                prefix + "NativeStreamClosed",
	}
}

func nativeServerHasStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			return true
		}
	}
	return false
}

func nativeServerHasClientInputStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindClientStreaming || method.Streaming == StreamingKindBidiStreaming {
			return true
		}
	}
	return false
}

func validateNativeServerSymbols(service ServicePlan) error {
	seen := make(map[string]string)
	messageTypes := make(map[string]string)
	for _, method := range service.Methods {
		if method.Request.GoName != "" {
			messageTypes[method.Request.GoName] = method.FullName + " request"
		}
		if method.Response.GoName != "" {
			messageTypes[method.Response.GoName] = method.FullName + " response"
		}
	}

	addGenerated := func(symbol, source string) error {
		if symbol == "" {
			return nil
		}
		if previous, exists := seen[symbol]; exists {
			return fmt.Errorf("native server symbol %s for %s collides with %s", symbol, source, previous)
		}
		if messageSource, exists := messageTypes[symbol]; exists {
			return fmt.Errorf("native server symbol %s for %s collides with protobuf message type from %s", symbol, source, messageSource)
		}
		seen[symbol] = source
		return nil
	}

	if err := addGenerated(service.GoName+"NativeServer", service.FullName+" native server interface"); err != nil {
		return err
	}
	if err := addGenerated("Unimplemented"+service.GoName+"NativeServer", service.FullName+" unimplemented native server helper"); err != nil {
		return err
	}
	if err := addGenerated(lowerInitial(service.GoName)+"GoNativeAdapter", service.FullName+" go native adapter"); err != nil {
		return err
	}
	if err := addGenerated("Register"+service.GoName+"GoNativeServer", service.FullName+" go native registration"); err != nil {
		return err
	}

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
		case StreamingKindClientStreaming:
			if err := addGenerated(service.GoName+method.GoName+"NativeClientStream", method.FullName+" client stream interface"); err != nil {
				return err
			}
			if err := addGenerated(lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession", method.FullName+" client stream session"); err != nil {
				return err
			}
		case StreamingKindServerStreaming:
			if err := addGenerated(service.GoName+method.GoName+"NativeServerStream", method.FullName+" server stream interface"); err != nil {
				return err
			}
			if err := addGenerated(lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession", method.FullName+" server stream session"); err != nil {
				return err
			}
		case StreamingKindBidiStreaming:
			if err := addGenerated(service.GoName+method.GoName+"NativeBidiStream", method.FullName+" bidi stream interface"); err != nil {
				return err
			}
			if err := addGenerated(lowerInitial(service.GoName)+method.GoName+"GoNativeBidiStreamSession", method.FullName+" bidi stream session"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s has unknown native server streaming kind %d", method.FullName, method.Streaming)
		}
	}
	return nil
}
