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
	runtimeMethods, err := buildRuntimeMethodProjectionsWithMessageTypes(g, service, serviceHasStreamingMethod(service))
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
		g.P(`fmt "fmt"`)
	}
	if nativeServerNeedsRPCRuntime(service) {
		g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
	}
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()
	g.P("var (")
	g.P(errorNames.StreamIsNil, ` = errors.New("rpccgo: native stream is nil")`)
	g.P(errorNames.StreamClosed, ` = errors.New("rpccgo: native stream is closed")`)
	g.P(")")
	g.P()

	renderGoNativeServerInterface(g, service, service.GoName+"NativeServer")
	renderGoNativeStreamInterfaces(g, service)
	streamingMethods := runtimeStreamingMethodProjections(runtimeMethods)
	renderGoNativeStreamTypes(g, service, errorNames)
	for _, method := range streamingMethods {
		renderRuntimeNativeStreamFacade(g, service.GoName, lowerInitial(service.GoName)+"StreamRegistry", method)
	}
	renderUnimplementedGoNativeServer(g, service)
	renderGoNativeRegistration(g, service, service.GoName+"NativeServer", "")
	if err := renderGoNativeServerRegistrations(g, service); err != nil {
		return err
	}
	renderGoNativeStartHelpers(g, service, runtimeMethods, service.GoName+"NativeServer", errorNames)
	return nil
}

func nativeServerNeedsRPCRuntime(service ServicePlan) bool {
	if service.Generation.NativeEnabled {
		return true
	}
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
	if service.DocComment == "" {
		renderDoc(g, serverName, "defines the native Go server contract for "+service.GoName+".")
		g.P("type ", serverName, " interface {")
	} else {
		renderDocLine(g, service.DocComment, "type ", serverName, " interface {")
	}
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
			name := service.GoName + method.GoName + "NativeClientStream"
			renderDoc(g, name, "receives native request values for the "+method.GoName+" client-streaming method.")
			g.P("type ", service.GoName, method.GoName, "NativeClientStream interface {")
			g.P("Recv(ctx context.Context) (", requestReturns, ")")
			g.P("}")
			g.P()
		case StreamingKindServerStreaming:
			name := service.GoName + method.GoName + "NativeServerStream"
			renderDoc(g, name, "sends native response values for the "+method.GoName+" server-streaming method.")
			g.P("type ", service.GoName, method.GoName, "NativeServerStream interface {")
			g.P("FinishRequested() <-chan struct{}")
			g.P("Send(ctx context.Context", responseParams, ") error")
			g.P("}")
			g.P()
		case StreamingKindBidiStreaming:
			name := service.GoName + method.GoName + "NativeBidiStream"
			renderDoc(g, name, "sends and receives native values for the "+method.GoName+" bidi-streaming method.")
			g.P("type ", service.GoName, method.GoName, "NativeBidiStream interface {")
			g.P("FinishRequested() <-chan struct{}")
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
	renderDoc(g, serverName, "provides default unimplemented native server methods for "+service.GoName+".")
	g.P("type ", serverName, " struct{}")
	g.P()
	for _, method := range service.Methods {
		requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.Contract.Native.ResponseFields)
		errExpr := `errors.New("rpccgo: ` + service.GoName + "." + method.GoName + ` native server method is not implemented")`
		renderDoc(g, method.GoName, "returns an unimplemented error for the "+service.GoName+" native "+method.GoName+" method.")
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

func renderGoNativeStartHelpers(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, serverName string, errorNames nativeServerErrorNames) {
	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}

	for _, runtimeMethod := range methods {
		if !runtimeMethod.Stream.Streaming {
			continue
		}
		method, ok := byName[runtimeMethod.Identity.GoName]
		if !ok {
			continue
		}
		switch method.Streaming {
		case StreamingKindClientStreaming:
			renderGoNativeClientStreamStartHelper(g, service, serverName, method, errorNames)
		case StreamingKindServerStreaming:
			renderGoNativeServerStreamStartHelper(g, service, serverName, method, errorNames)
		case StreamingKindBidiStreaming:
			renderGoNativeBidiStreamStartHelper(g, service, serverName, method, errorNames)
		}
	}
	_ = serverName
}

func goNativeStartHelperName(serviceName, methodName string) string {
	return lowerInitial(serviceName) + methodName + "GoNativeStart"
}

func renderGoNativeStreamTypes(g *protogen.GeneratedFile, service ServicePlan, errorNames nativeServerErrorNames) {
	for _, method := range service.Methods {
		renderNativeStreamEnvelopeTypes(g, method)
		renderGoNativeStreamingServerStruct(g, service, method)
		renderGoNativeStreamingServerMethods(g, service, method, errorNames)
	}
}

func renderGoNativeStreamingServerStruct(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	switch method.Streaming {
	case StreamingKindClientStreaming:
		receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeClientStreamingServer"
		g.P("type ", receiver, " struct {")
		g.P("stream rpcruntime.ClientStreamingServer[", method.RenderPlan.Symbols.NativeStreamRequestType, "]")
		g.P("}")
		g.P()
	case StreamingKindServerStreaming:
		receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeServerStreamingServer"
		g.P("type ", receiver, " struct {")
		g.P("stream rpcruntime.ServerStreamingServer[", method.RenderPlan.Symbols.NativeStreamResponseType, "]")
		g.P("}")
		g.P()
	case StreamingKindBidiStreaming:
		receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeBidiStreamingServer"
		g.P("type ", receiver, " struct {")
		g.P("stream rpcruntime.BidiStreamingServer[", method.RenderPlan.Symbols.NativeStreamRequestType, ", ", method.RenderPlan.Symbols.NativeStreamResponseType, "]")
		g.P("}")
		g.P()
	}
}

func renderGoNativeStreamingServerMethods(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerErrorNames) {
	switch method.Streaming {
	case StreamingKindClientStreaming:
		receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeClientStreamingServer"
		renderGoNativeClientStreamingServerRecv(g, receiver, method)
	case StreamingKindServerStreaming:
		receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeServerStreamingServer"
		renderGoNativeServerStreamingServerSend(g, receiver, method, errorNames)
	case StreamingKindBidiStreaming:
		receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeBidiStreamingServer"
		renderGoNativeBidiStreamingServerRecv(g, receiver, method)
		renderGoNativeBidiStreamingServerSend(g, receiver, method, errorNames)
	}
}

func renderGoNativeClientStreamStartHelper(g *protogen.GeneratedFile, service ServicePlan, serverName string, method MethodPlan, errorNames nativeServerErrorNames) {
	clientType := nativeRuntimeStreamingClientInterface(method, "")
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeClientStreamingServer"
	g.P("func ", goNativeStartHelperName(service.GoName, method.GoName), "(ctx context.Context, server ", serverName, ") (", clientType, ", error) {")
	g.P("client, stream, streamCtx := rpcruntime.NewClientStreaming[", method.RenderPlan.Symbols.NativeStreamRequestType, ", ", method.RenderPlan.Symbols.NativeStreamResponseType, "](ctx, rpcruntime.LocalStreamOptions{")
	g.P("RequestBuffer: 16,")
	g.P("StreamClosed: ", errorNames.StreamClosed, ",")
	g.P("})")
	g.P("serverStream := &", receiver, "{stream: stream}")
	g.P("go func() {")
	g.P(renderNativeClientStreamResultLocals(method), "server.", method.GoName, "(streamCtx, serverStream)")
	g.P("stream.Complete(", nativeResponseEnvelopeLiteralFromLocals(method), ", err)")
	g.P("}()")
	g.P("return client, nil")
	g.P("}")
	g.P()

	_ = errorNames
}

func renderGoNativeServerStreamStartHelper(g *protogen.GeneratedFile, service ServicePlan, serverName string, method MethodPlan, errorNames nativeServerErrorNames) {
	clientType := nativeRuntimeStreamingClientInterface(method, "")
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeServerStreamingServer"
	requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	g.P("func ", goNativeStartHelperName(service.GoName, method.GoName), "(ctx context.Context, server ", serverName, requestParams, ") (", clientType, ", error) {")
	g.P("client, stream, streamCtx := rpcruntime.NewServerStreaming[", method.RenderPlan.Symbols.NativeStreamResponseType, "](ctx, rpcruntime.LocalStreamOptions{")
	g.P("ResponseBuffer: 1,")
	g.P("StreamClosed: ", errorNames.StreamClosed, ",")
	g.P("})")
	g.P("serverStream := &", receiver, "{stream: stream}")
	g.P("go func() {")
	if len(method.Contract.Native.RequestFields) == 0 {
		g.P("err := server.", method.GoName, "(streamCtx, serverStream)")
	} else {
		g.P("err := server.", method.GoName, "(streamCtx, ", requestArgs, ", serverStream)")
	}
	g.P("stream.Complete(err)")
	g.P("}()")
	g.P("return client, nil")
	g.P("}")
	g.P()

	_ = errorNames
}

func renderGoNativeBidiStreamStartHelper(g *protogen.GeneratedFile, service ServicePlan, serverName string, method MethodPlan, errorNames nativeServerErrorNames) {
	clientType := nativeRuntimeStreamingClientInterface(method, "")
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeBidiStreamingServer"
	g.P("func ", goNativeStartHelperName(service.GoName, method.GoName), "(ctx context.Context, server ", serverName, ") (", clientType, ", error) {")
	g.P("client, stream, streamCtx := rpcruntime.NewBidiStreaming[", method.RenderPlan.Symbols.NativeStreamRequestType, ", ", method.RenderPlan.Symbols.NativeStreamResponseType, "](ctx, rpcruntime.LocalStreamOptions{")
	g.P("RequestBuffer: 16,")
	g.P("ResponseBuffer: 1,")
	g.P("StreamClosed: ", errorNames.StreamClosed, ",")
	g.P("})")
	g.P("serverStream := &", receiver, "{stream: stream}")
	g.P("go func() {")
	g.P("err := server.", method.GoName, "(streamCtx, serverStream)")
	g.P("stream.Complete(err)")
	g.P("}()")
	g.P("return client, nil")
	g.P("}")
	g.P()
}

func renderGoNativeClientStreamingServerRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan) {
	requestReturns := nativeGoRequestReturns(g, method.Contract.Native.RequestFields)
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", requestReturns, ") {")
	g.P(nativeRequestEnvelopeRecvAssignment(method.Contract.Native.RequestFields), "s.stream.Recv(ctx)")
	g.P("return ", nativeExportedEnvelopeFieldArgs("req", method.Contract.Native.RequestFields), nativeReturnSeparator(method.Contract.Native.RequestFields), "err")
	g.P("}")
	g.P()
}

func renderGoNativeServerStreamingServerSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	g.P("func (s *", receiver, ") FinishRequested() <-chan struct{} {")
	g.P("return s.stream.FinishRequested()")
	g.P("}")
	g.P()
	responseParams := nativeGoResponseParams(g, method.Contract.Native.ResponseFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", responseParams, ") error {")
	g.P("return s.stream.Send(ctx, ", nativeResponseEnvelopeLiteralFromLocals(method), ")")
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGoNativeBidiStreamingServerRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan) {
	g.P("func (s *", receiver, ") FinishRequested() <-chan struct{} {")
	g.P("return s.stream.FinishRequested()")
	g.P("}")
	g.P()
	requestReturns := nativeGoRequestReturns(g, method.Contract.Native.RequestFields)
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", requestReturns, ") {")
	g.P(nativeRequestEnvelopeRecvAssignment(method.Contract.Native.RequestFields), "s.stream.Recv(ctx)")
	g.P("return ", nativeExportedEnvelopeFieldArgs("req", method.Contract.Native.RequestFields), nativeReturnSeparator(method.Contract.Native.RequestFields), "err")
	g.P("}")
	g.P()
}

func renderGoNativeBidiStreamingServerSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseParams := nativeGoResponseParams(g, method.Contract.Native.ResponseFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", responseParams, ") error {")
	g.P("return s.stream.Send(ctx, ", nativeResponseEnvelopeLiteralFromLocals(method), ")")
	g.P("}")
	g.P()
	_ = errorNames
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

func nativeEnvelopeLiteralFromEnvelope(fields []FieldPlan, envelope string) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, lowerInitial(field.GoName)+": "+envelope+"."+field.GoName)
	}
	return strings.Join(parts, ", ")
}

func nativeEnvelopeLiteralFromEnvelopeWithExtra(fields []FieldPlan, envelope, extra string) string {
	literal := nativeEnvelopeLiteralFromEnvelope(fields, envelope)
	if literal == "" {
		return extra
	}
	return literal + ", " + extra
}

func nativeResponseEnvelopeLiteral(prefix string, method MethodPlan) string {
	responseType := method.RenderPlan.Symbols.NativeStreamResponseType
	literal := nativeExportedEnvelopeLiteral(prefix, method.Contract.Native.ResponseFields)
	if literal == "" {
		return responseType + "{}"
	}
	return responseType + "{" + literal + "}"
}

func nativeResponseEnvelopeLiteralFromLocals(method MethodPlan) string {
	responseType := method.RenderPlan.Symbols.NativeStreamResponseType
	literal := nativeExportedEnvelopeLiteralFromLocals(method.Contract.Native.ResponseFields)
	if literal == "" {
		return responseType + "{}"
	}
	return responseType + "{" + literal + "}"
}

func nativeResponseEnvelopeLiteralFromResults(method MethodPlan, qualifier string) string {
	responseType := qualifier + method.RenderPlan.Symbols.NativeStreamResponseType
	literal := nativeExportedEnvelopeLiteralFromResults(method.Contract.Native.ResponseFields)
	if literal == "" {
		return responseType + "{}"
	}
	return responseType + "{" + literal + "}"
}

func nativeExportedEnvelopeLiteral(prefix string, fields []FieldPlan) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.GoName+": "+prefix+"."+lowerInitial(field.GoName))
	}
	return strings.Join(parts, ", ")
}

func nativeExportedEnvelopeLiteralFromLocals(fields []FieldPlan) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.GoName+": "+lowerInitial(field.GoName))
	}
	return strings.Join(parts, ", ")
}

func nativeExportedEnvelopeLiteralFromResults(fields []FieldPlan) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.GoName+": "+lowerInitial(field.GoName)+"Result")
	}
	return strings.Join(parts, ", ")
}

func nativeExportedEnvelopeFieldArgs(prefix string, fields []FieldPlan) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, prefix+"."+field.GoName)
	}
	return strings.Join(parts, ", ")
}

func renderNativeClientStreamResultLocals(method MethodPlan) string {
	names := nativeEnvelopeFieldNames(method.Contract.Native.ResponseFields)
	if names == "" {
		return "err := "
	}
	return names + ", err := "
}

func nativeRequestEnvelopeRecvAssignment(fields []FieldPlan) string {
	if len(fields) == 0 {
		return "_, err := "
	}
	return "req, err := "
}

func nativeReturnSeparator(fields []FieldPlan) string {
	if len(fields) == 0 {
		return ""
	}
	return ", "
}

func nativeRuntimeStreamingClientInterface(method MethodPlan, qualifier string) string {
	requestType := qualifier + method.RenderPlan.Symbols.NativeStreamRequestType
	responseType := qualifier + method.RenderPlan.Symbols.NativeStreamResponseType
	switch method.Streaming {
	case StreamingKindClientStreaming:
		return "rpcruntime.ClientStreamingClient[" + requestType + ", " + responseType + "]"
	case StreamingKindServerStreaming:
		return "rpcruntime.ServerStreamingClient[" + responseType + "]"
	case StreamingKindBidiStreaming:
		return "rpcruntime.BidiStreamingClient[" + requestType + ", " + responseType + "]"
	default:
		return "any"
	}
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
	renderDoc(g, "Register"+service.GoName+"GoNativeServer", "registers a Go native server as the current server for "+service.GoName+".")
	g.P("func Register", service.GoName, "GoNativeServer(server ", serverName, ") error {")
	g.P("if server == nil {")
	g.P("_ = register", service.GoName, "GoNativeServer(server)")
	g.P(`return errors.New("rpccgo: `, service.GoName, ` go native server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "GoNativeServer(server)")
	g.P("}")
	g.P()
	_ = adapterName
}

func renderGoNativeServerRegistrations(g *protogen.GeneratedFile, service ServicePlan) error {
	serviceIDName := lowerInitial(service.GoName) + "ServiceID"
	for _, source := range []RegistrationSourcePlan{
		{
			Origin:    RegistrationOriginGo,
			Contract:  RegistrationContractNative,
			Transport: RegistrationTransportNone,
			Mode:      RegistrationModeLocal,
		},
		{
			Origin:    RegistrationOriginCGO,
			Contract:  RegistrationContractNative,
			Transport: RegistrationTransportNone,
			Mode:      RegistrationModeLocal,
		},
	} {
		projection, err := ProjectRegistrationSource(service, source)
		if err != nil {
			return err
		}
		renderRuntimeServerRegistration(g, serviceIDName, projection)
	}
	return nil
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
	StreamIsNil  string
	StreamClosed string
}

func nativeServerErrorNamesFor(service ServicePlan) nativeServerErrorNames {
	prefix := lowerInitial(service.GoName)
	return nativeServerErrorNames{
		StreamIsNil:  prefix + "NativeStreamIsNil",
		StreamClosed: prefix + "NativeStreamClosed",
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
			if err := addGenerated(lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamingServer", method.FullName+" client-streaming server"); err != nil {
				return err
			}
		case StreamingKindServerStreaming:
			if err := addGenerated(service.GoName+method.GoName+"NativeServerStream", method.FullName+" server stream interface"); err != nil {
				return err
			}
			if err := addGenerated(lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamingServer", method.FullName+" server-streaming server"); err != nil {
				return err
			}
		case StreamingKindBidiStreaming:
			if err := addGenerated(service.GoName+method.GoName+"NativeBidiStream", method.FullName+" bidi stream interface"); err != nil {
				return err
			}
			if err := addGenerated(lowerInitial(service.GoName)+method.GoName+"GoNativeBidiStreamingServer", method.FullName+" bidi-streaming server"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s has unknown native server streaming kind %d", method.FullName, method.Streaming)
		}
	}
	return nil
}
