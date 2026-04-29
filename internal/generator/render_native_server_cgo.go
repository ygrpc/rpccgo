package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeServerCGOSymbols(service); err != nil {
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
	g.P(`fmt "fmt"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	if nativeServerCGONeedsUnsafe(service) {
		g.P(`unsafe "unsafe"`)
	}
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	errorNames := nativeServerCGOErrorNamesFor(service)
	g.P("var (")
	g.P(errorNames.CallbacksNil, ` = errors.New("rpccgo: `, service.GoName, ` cgo native server callbacks are nil")`)
	g.P(errorNames.UnaryCallbackMissing, ` = errors.New("rpccgo: `, service.GoName, ` cgo native server unary callback is missing")`)
	g.P(errorNames.UnsupportedField, ` = errors.New("rpccgo: cgo native server field bridge is not implemented")`)
	g.P(errorNames.StreamNotImplemented, ` = errors.New("rpccgo: cgo native server streaming is not implemented")`)
	g.P(")")
	g.P()

	callbacksName := service.GoName + "CGONativeServerCallbacks"
	adapterName := lowerInitial(service.GoName) + "CGONativeAdapter"
	renderCGONativeServerCallbackTable(g, service, callbacksName)
	renderCGONativeServerAdapter(g, service, runtimeMethods, callbacksName, adapterName, errorNames)
	renderCGONativeServerRegistration(g, service, callbacksName, adapterName, errorNames)
	return nil
}

func renderCGONativeServerCallbackTable(g *protogen.GeneratedFile, service ServicePlan, callbacksName string) {
	g.P("type ", callbacksName, " struct {")
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P(method.GoName, " func(ctx context.Context, input *", nativeCGOServerRequestName(service, method), ", output *", nativeCGOServerResponseName(service, method), ") int32")
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, callbacksName, adapterName string, errorNames nativeServerCGOErrorNames) {
	g.P("type ", adapterName, " struct {")
	g.P("callbacks *", callbacksName)
	g.P("}")
	g.P()

	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
		if !ok || method.Streaming != StreamingKindUnary {
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod, errorNames)
			continue
		}
		renderCGONativeServerUnaryAdapter(g, service, adapterName, method, errorNames)
	}

	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		renderCGONativeServerRequestType(g, service, method)
		renderCGONativeServerResponseType(g, service, method)
		renderCGONativeServerRequestEncoder(g, service, method, errorNames)
		renderCGONativeServerResponseDecoder(g, service, method, errorNames)
	}
	renderCGONativeErrorIDHelper(g, service)
}

func renderCGONativeServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if a == nil || a.callbacks == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("callback := a.callbacks.", method.GoName)
	g.P("if callback == nil {")
	g.P("return nil, ", errorNames.UnaryCallbackMissing)
	g.P("}")
	g.P("input, cleanup, err := ", nativeCGOServerRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("output := &", nativeCGOServerResponseName(service, method), "{}")
	g.P("errID := callback(ctx, input, output)")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return ", nativeCGOServerResponseDecoderName(service, method), "(output)")
	g.P("}")
	g.P()
}

func renderCGONativeServerStreamingFallback(g *protogen.GeneratedFile, adapterName string, method runtimeAdapterMethod, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult, " {")
	if method.Streaming {
		g.P("return nil, ", errorNames.StreamNotImplemented)
	} else {
		g.P("return nil, ", errorNames.UnaryCallbackMissing)
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestType(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("type ", nativeCGOServerRequestName(service, method), " struct {")
	renderNativeClientFields(g, method.NativeContract.RequestFields, true)
	g.P("}")
	g.P()
}

func renderCGONativeServerResponseType(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("type ", nativeCGOServerResponseName(service, method), " struct {")
	renderNativeClientFields(g, method.NativeContract.ResponseFields, false)
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	requestName := nativeCGOServerRequestName(service, method)
	g.P("func ", nativeCGOServerRequestEncoderName(service, method), "(req ", nativeGoMessageType(g, method.Request), ") (*", requestName, ", func(), error) {")
	g.P("if req == nil {")
	g.P(`return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")`)
	g.P("}")
	g.P("input := &", requestName, "{}")
	g.P("var pinned []uintptr")
	g.P("cleanup := func() {")
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
	g.P("}")
	for _, field := range method.NativeContract.RequestFields {
		renderCGONativeServerRequestFieldEncode(g, field, errorNames)
	}
	g.P("return input, cleanup, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestFieldEncode(g *protogen.GeneratedFile, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("if req.", field.GoName, " {")
		g.P("input.", field.GoName, " = 1")
		g.P("}")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble:
			g.P("input.", field.GoName, " = req.", field.GoName)
		case FieldKindEnum:
			g.P("input.", field.GoName, " = int32(req.", field.GoName, ")")
		case FieldKindString:
			g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(req.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P("_, ", field.GoName, "Ptr, err := rpcruntime.PinString(req.", field.GoName, ")")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P("if ", field.GoName, "Ptr != 0 {")
			g.P("pinned = append(pinned, ", field.GoName, "Ptr)")
			g.P("}")
			g.P("input.", field.GoName, "Ptr = ", field.GoName, "Ptr")
			g.P("input.", field.GoName, "Len = ", field.GoName, "Len")
		case FieldKindBytes:
			g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(req.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P(field.GoName, "Ptr, err := rpcruntime.PinBytes(req.", field.GoName, ")")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P("if ", field.GoName, "Ptr != 0 {")
			g.P("pinned = append(pinned, ", field.GoName, "Ptr)")
			g.P("}")
			g.P("input.", field.GoName, "Ptr = ", field.GoName, "Ptr")
			g.P("input.", field.GoName, "Len = ", field.GoName, "Len")
		default:
			g.P("cleanup()")
			g.P("return nil, func() {}, ", errorNames.UnsupportedField)
		}
	default:
		g.P("cleanup()")
		g.P("return nil, func() {}, ", errorNames.UnsupportedField)
	}
}

func renderCGONativeServerResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	responseName := nativeCGOServerResponseName(service, method)
	g.P("func ", nativeCGOServerResponseDecoderName(service, method), "(output *", responseName, ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if output == nil {")
	g.P(`return nil, errors.New("rpccgo: cgo native server response output is nil")`)
	g.P("}")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.ResponseFields {
		renderCGONativeServerResponseFieldDecode(g, field, errorNames)
	}
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerResponseFieldDecode(g *protogen.GeneratedFile, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("resp.", field.GoName, " = output.", field.GoName, " != 0")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble:
			g.P("resp.", field.GoName, " = output.", field.GoName)
		case FieldKindEnum:
			g.P("resp.", field.GoName, " = ", nativeGoEnumType(g, field), "(output.", field.GoName, ")")
		case FieldKindString:
			renderCGONativeServerResponseTextDecode(g, field, "String", "SafeString")
		case FieldKindBytes:
			renderCGONativeServerResponseTextDecode(g, field, "Bytes", "SafeBytes")
		default:
			g.P("return nil, ", errorNames.UnsupportedField)
		}
	default:
		g.P("return nil, ", errorNames.UnsupportedField)
	}
}

func renderCGONativeServerResponseTextDecode(g *protogen.GeneratedFile, field FieldPlan, wrapper, safeMethod string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(output.", field.GoName, "Len); err != nil {")
	g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
	g.P("}")
	g.P(field.GoName, " := rpcruntime.NewRpc", wrapper, "((*byte)(unsafe.Pointer(output.", field.GoName, "Ptr)), output.", field.GoName, "Len, output.", field.GoName, "Ownership > 0)")
	g.P("resp.", field.GoName, " = ", field.GoName, ".", safeMethod, "()")
	g.P("if err := ", field.GoName, ".Release(); err != nil {")
	g.P("return nil, err")
	g.P("}")
}

func renderCGONativeServerRegistration(g *protogen.GeneratedFile, service ServicePlan, callbacksName, adapterName string, errorNames nativeServerCGOErrorNames) {
	g.P("func Register", service.GoName, "CGONativeServer(callbacks *", callbacksName, ") (rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter], error) {")
	g.P("if callbacks == nil {")
	g.P("return rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter]{}, ", errorNames.CallbacksNil)
	g.P("}")
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P("if callbacks.", method.GoName, " == nil {")
		g.P("return rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter]{}, ", errorNames.UnaryCallbackMissing)
		g.P("}")
	}
	g.P("return register", service.GoName, "ActiveServer(rpcruntime.ServerKindCGONative, &", adapterName, "{callbacks: callbacks})")
	g.P("}")
	g.P()
}

func renderCGONativeErrorIDHelper(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("func ", nativeCGOServerErrorIDHelperName(service), "(errID int32) error {")
	g.P("if errID == 0 {")
	g.P("return nil")
	g.P("}")
	g.P("text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))")
	g.P("if ok {")
	g.P("if ptr != 0 {")
	g.P("defer rpcruntime.Release(ptr)")
	g.P("}")
	g.P("return errors.New(string(text))")
	g.P("}")
	g.P(`return fmt.Errorf("rpccgo: cgo native server callback returned unknown error id %d", errID)`)
	g.P("}")
	g.P()
}

func nativeCGOServerRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryRequest"
}

func nativeCGOServerResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeUnaryRequest"
}

func nativeCGOServerResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerErrorIDHelperName(service ServicePlan) string {
	return lowerInitial(service.GoName) + "CGONativeServerErrorFromID"
}

func nativeServerCGONeedsUnsafe(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		for _, field := range method.NativeContract.ResponseFields {
			if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
				return true
			}
		}
	}
	return false
}

type nativeServerCGOErrorNames struct {
	CallbacksNil         string
	UnaryCallbackMissing string
	UnsupportedField     string
	StreamNotImplemented string
}

func nativeServerCGOErrorNamesFor(service ServicePlan) nativeServerCGOErrorNames {
	prefix := lowerInitial(service.GoName)
	return nativeServerCGOErrorNames{
		CallbacksNil:         prefix + "CGONativeServerCallbacksNil",
		UnaryCallbackMissing: prefix + "CGONativeServerUnaryCallbackMissing",
		UnsupportedField:     prefix + "CGONativeServerUnsupportedField",
		StreamNotImplemented: prefix + "CGONativeServerStreamNotImplemented",
	}
}

func validateNativeServerCGOSymbols(service ServicePlan) error {
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
			return fmt.Errorf("native server cgo symbol %s for %s collides with %s", symbol, source, previous)
		}
		if messageSource, exists := messageTypes[symbol]; exists {
			return fmt.Errorf("native server cgo symbol %s for %s collides with protobuf message type from %s", symbol, source, messageSource)
		}
		seen[symbol] = source
		return nil
	}

	errorNames := nativeServerCGOErrorNamesFor(service)
	for symbol, source := range map[string]string{
		service.GoName + "CGONativeServerCallbacks":       errorNames.CallbacksNil,
		lowerInitial(service.GoName) + "CGONativeAdapter": service.FullName + " adapter",
		"Register" + service.GoName + "CGONativeServer":   service.FullName + " registration",
		nativeCGOServerErrorIDHelperName(service):         service.FullName + " error id helper",
		errorNames.CallbacksNil:                           errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                   errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                       errorNames.UnsupportedField,
		errorNames.StreamNotImplemented:                   errorNames.StreamNotImplemented,
	} {
		if err := addGenerated(symbol, source); err != nil {
			return err
		}
	}
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		for _, item := range []struct {
			symbol string
			source string
		}{
			{nativeCGOServerRequestName(service, method), method.FullName + " cgo request"},
			{nativeCGOServerResponseName(service, method), method.FullName + " cgo response"},
			{nativeCGOServerRequestEncoderName(service, method), method.FullName + " request encoder"},
			{nativeCGOServerResponseDecoderName(service, method), method.FullName + " response decoder"},
		} {
			if err := addGenerated(item.symbol, item.source); err != nil {
				return err
			}
		}
		if err := validateNativeClientStructFields(nativeCGOServerRequestName(service, method), method.NativeContract.RequestFields, nativeClientOutputFieldSymbols); err != nil {
			return err
		}
		if err := validateNativeClientStructFields(nativeCGOServerResponseName(service, method), method.NativeContract.ResponseFields, nativeClientInputFieldSymbols); err != nil {
			return err
		}
	}
	return nil
}
