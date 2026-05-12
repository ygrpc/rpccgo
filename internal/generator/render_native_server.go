package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeServerSymbols(service); err != nil {
		return err
	}
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}

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
	errorNames := nativeServerErrorNamesFor(service)
	g.P("var (")
	g.P(errorNames.RequestBridgeNotImplemented, ` = errors.New("rpccgo: native request bridge is not implemented")`)
	g.P(errorNames.StreamBridgeNotImplemented, ` = errors.New("rpccgo: native stream bridge is not implemented")`)
	g.P(errorNames.StreamIsNil, ` = errors.New("rpccgo: native stream is nil")`)
	g.P(")")
	g.P()

	serverName := service.GoName + "NativeServer"
	adapterName := lowerInitial(service.GoName) + "GoNativeAdapter"

	renderGoNativeServerInterface(g, service, serverName)
	renderGoNativeStreamInterfaces(g, service)
	renderGoNativeAdapter(g, service, runtimeMethods, serverName, adapterName, errorNames)
	renderGoNativeRegistration(g, service, serverName, adapterName)
	return nil
}

func renderGoNativeServerInterface(g *protogen.GeneratedFile, service ServicePlan, serverName string) {
	g.P("type ", serverName, " interface {")
	for _, method := range service.Methods {
		requestParams := nativeGoRequestParams(g, method.NativeContract.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(method.GoName, "(ctx context.Context", requestParams, ") (", responseReturns, ")")
		case StreamingKindClientStreaming:
			g.P(method.GoName, "(ctx context.Context) (", service.GoName, method.GoName, "NativeClientStream, error)")
		case StreamingKindServerStreaming:
			g.P(method.GoName, "(ctx context.Context", requestParams, ") (", service.GoName, method.GoName, "NativeServerStream, error)")
		case StreamingKindBidiStreaming:
			g.P(method.GoName, "(ctx context.Context) (", service.GoName, method.GoName, "NativeBidiStream, error)")
		}
	}
	g.P("}")
	g.P()
}

func renderGoNativeStreamInterfaces(g *protogen.GeneratedFile, service ServicePlan) {
	for _, method := range service.Methods {
		requestParams := nativeGoRequestParams(g, method.NativeContract.RequestFields)
		responseReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
		switch method.Streaming {
		case StreamingKindClientStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeClientStream interface {")
			g.P("Send(ctx context.Context", requestParams, ") error")
			g.P("Finish(ctx context.Context) (", responseReturns, ")")
			g.P("Cancel(ctx context.Context) error")
			g.P("}")
			g.P()
		case StreamingKindServerStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeServerStream interface {")
			g.P("Recv(ctx context.Context) (", responseReturns, ")")
			g.P("Cancel(ctx context.Context) error")
			g.P("}")
			g.P()
		case StreamingKindBidiStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeBidiStream interface {")
			g.P("Send(ctx context.Context", requestParams, ") error")
			g.P("Recv(ctx context.Context) (", responseReturns, ")")
			g.P("CloseSend(ctx context.Context) error")
			g.P("Cancel(ctx context.Context) error")
			g.P("}")
			g.P()
		}
	}
}

func renderGoNativeAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, serverName, adapterName string, errorNames nativeServerErrorNames) {
	g.P("type ", adapterName, " struct {")
	g.P("server ", serverName)
	g.P("}")
	g.P()

	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}

	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
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
}

func renderGoNativeUnaryAdapterMethod(g *protogen.GeneratedFile, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	requestParams := nativeGoRequestParams(g, method.NativeContract.RequestFields)
	responseReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
	requestArgs := nativeGoRequestArgNames(method.NativeContract.RequestFields)
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context", requestParams, ") (", responseReturns, ") {")
	if len(method.NativeContract.RequestFields) == 0 {
		g.P("return a.server.", method.GoName, "(ctx)")
	} else {
		g.P("return a.server.", method.GoName, "(ctx, ", requestArgs, ")")
	}
	g.P("}")
	g.P()
	_ = errorNames
}

func renderGoNativeClientStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("stream, err := a.server.", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("if stream == nil {")
	g.P("return nil, ", errorNames.StreamIsNil)
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoNativeClientStreamSession{stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoNativeClientStreamSession struct {")
	g.P("stream ", service.GoName, method.GoName, "NativeClientStream")
	g.P("}")
	g.P()
	renderGoNativeClientStreamSend(g, lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession", method, errorNames)
	renderGoNativeClientStreamFinish(g, lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession", method, errorNames)
	renderCancelForwarder(g, lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession", errorNames)
}

func renderGoNativeServerStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	requestParams := nativeGoRequestParams(g, method.NativeContract.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.NativeContract.RequestFields)
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context", requestParams, ") (", sessionName, ", error) {")
	if len(method.NativeContract.RequestFields) == 0 {
		g.P("stream, err := a.server.", method.GoName, "(ctx)")
	} else {
		g.P("stream, err := a.server.", method.GoName, "(ctx, ", requestArgs, ")")
	}
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("if stream == nil {")
	g.P("return nil, ", errorNames.StreamIsNil)
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoNativeServerStreamSession{stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoNativeServerStreamSession struct {")
	g.P("stream ", service.GoName, method.GoName, "NativeServerStream")
	g.P("}")
	g.P()
	renderGoNativeServerStreamRecv(g, lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession", method, errorNames)
	renderCancelForwarder(g, lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession", errorNames)
}

func renderGoNativeBidiStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerErrorNames) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("stream, err := a.server.", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("if stream == nil {")
	g.P("return nil, ", errorNames.StreamIsNil)
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoNativeBidiStreamSession{stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoNativeBidiStreamSession struct {")
	g.P("stream ", service.GoName, method.GoName, "NativeBidiStream")
	g.P("}")
	g.P()
	receiver := lowerInitial(service.GoName) + method.GoName + "GoNativeBidiStreamSession"
	renderGoNativeBidiStreamSend(g, receiver, method, errorNames)
	renderGoNativeBidiStreamRecv(g, receiver, method, errorNames)
	renderGoNativeBidiStreamCloseSend(g, receiver, errorNames)
	renderCancelForwarder(g, receiver, errorNames)
}

func renderGoNativeFallbackAdapterMethod(g *protogen.GeneratedFile, adapterName string, method runtimeAdapterMethod) {
	g.P("func (a *", adapterName, ") ", method.AdapterName, "(ctx context.Context)", method.AdapterResult, " {")
	if method.Streaming {
		g.P(`return nil, errors.New("rpccgo native server method is not implemented")`)
	} else {
		g.P(`return errors.New("rpccgo native server method is not implemented")`)
	}
	g.P("}")
	g.P()
}

func renderGoNativeClientStreamSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	requestParams := nativeGoRequestParams(g, method.NativeContract.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.NativeContract.RequestFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", requestParams, ") error {")
	g.P("if s.stream == nil {")
	g.P("return ", errorNames.StreamIsNil)
	g.P("}")
	if len(method.NativeContract.RequestFields) == 0 {
		g.P("return s.stream.Send(ctx)")
	} else {
		g.P("return s.stream.Send(ctx, ", requestArgs, ")")
	}
	g.P("}")
	g.P()
}

func renderGoNativeClientStreamFinish(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
	zeroReturns := nativeGoZeroReturns(method.NativeContract.ResponseFields, errorNames.StreamIsNil)
	g.P("func (s *", receiver, ") Finish(ctx context.Context) (", responseReturns, ") {")
	g.P("if s.stream == nil {")
	g.P("return ", zeroReturns)
	g.P("}")
	g.P("return s.stream.Finish(ctx)")
	g.P("}")
	g.P()
}

func renderGoNativeServerStreamRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
	zeroReturns := nativeGoZeroReturns(method.NativeContract.ResponseFields, errorNames.StreamIsNil)
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", responseReturns, ") {")
	g.P("if s.stream == nil {")
	g.P("return ", zeroReturns)
	g.P("}")
	g.P("return s.stream.Recv(ctx)")
	g.P("}")
	g.P()
}

func renderGoNativeBidiStreamSend(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	requestParams := nativeGoRequestParams(g, method.NativeContract.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.NativeContract.RequestFields)
	g.P("func (s *", receiver, ") Send(ctx context.Context", requestParams, ") error {")
	g.P("if s.stream == nil {")
	g.P("return ", errorNames.StreamIsNil)
	g.P("}")
	if len(method.NativeContract.RequestFields) == 0 {
		g.P("return s.stream.Send(ctx)")
	} else {
		g.P("return s.stream.Send(ctx, ", requestArgs, ")")
	}
	g.P("}")
	g.P()
}

func renderGoNativeBidiStreamRecv(g *protogen.GeneratedFile, receiver string, method MethodPlan, errorNames nativeServerErrorNames) {
	responseReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
	zeroReturns := nativeGoZeroReturns(method.NativeContract.ResponseFields, errorNames.StreamIsNil)
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", responseReturns, ") {")
	g.P("if s.stream == nil {")
	g.P("return ", zeroReturns)
	g.P("}")
	g.P("return s.stream.Recv(ctx)")
	g.P("}")
	g.P()
}

func renderGoNativeBidiStreamCloseSend(g *protogen.GeneratedFile, receiver string, errorNames nativeServerErrorNames) {
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	g.P("if s.stream == nil {")
	g.P("return ", errorNames.StreamIsNil)
	g.P("}")
	g.P("return s.stream.CloseSend(ctx)")
	g.P("}")
	g.P()
}

func renderCancelForwarder(g *protogen.GeneratedFile, receiver string, errorNames nativeServerErrorNames) {
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("if s.stream == nil {")
	g.P("return ", errorNames.StreamIsNil)
	g.P("}")
	g.P("return s.stream.Cancel(ctx)")
	g.P("}")
	g.P()
}

func renderGoNativeRegistration(g *protogen.GeneratedFile, service ServicePlan, serverName, adapterName string) {
	g.P("func Register", service.GoName, "GoNativeServer(server ", serverName, ") (rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter], error) {")
	g.P("if server == nil {")
	g.P(`return rpcruntime.AdapterSnapshot[`, service.GoName, `NativeAdapter]{}, errors.New("rpccgo: `, service.GoName, ` go native server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "ActiveServer(rpcruntime.ServerKindGoNative, &", adapterName, "{server: server})")
	g.P("}")
	g.P()
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
		return "*rpcruntime.RpcRepeat[" + nativeGoScalarType(g, field) + "]"
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
}

func nativeServerErrorNamesFor(service ServicePlan) nativeServerErrorNames {
	prefix := lowerInitial(service.GoName)
	return nativeServerErrorNames{
		RequestBridgeNotImplemented: prefix + "NativeRequestBridgeNotImplemented",
		StreamBridgeNotImplemented:  prefix + "NativeStreamBridgeNotImplemented",
		StreamIsNil:                 prefix + "NativeStreamIsNil",
	}
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
