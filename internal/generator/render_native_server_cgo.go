package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeServerCGOSymbols(plan, service); err != nil {
		return err
	}

	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}

	g.P("package ", plan.GoPackageName)
	g.P()
	renderCGONativeServerPreamble(g, service)
	g.P(`import "C"`)
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
	renderCGONativeServerAdapter(g, service, runtimeMethods, callbacksName, adapterName, errorNames)
	renderCGONativeServerRegistration(g, service, callbacksName, adapterName, errorNames)
	renderCGONativeServerGoHelper(g, service, runtimeMethods, callbacksName, errorNames)
	renderCGONativeServerErrorStoreExport(g, service)
	return nil
}

func renderCGONativeServerPreamble(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("/*")
	g.P("#include <stdint.h>")
	g.P()
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		renderCGONativeServerCStruct(g, nativeCGOServerRequestName(service, method), method.NativeContract.RequestFields, false)
		renderCGONativeServerCStruct(g, nativeCGOServerResponseName(service, method), method.NativeContract.ResponseFields, true)
		g.P("typedef int32_t (*", nativeCGOServerCallbackName(service, method), ")(", nativeCGOServerRequestName(service, method), "* input, ", nativeCGOServerResponseName(service, method), "* output);")
		g.P()
	}
	g.P("typedef struct ", service.GoName, "CGONativeServerCallbacks {")
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P(nativeCGOServerCallbackName(service, method), " ", method.GoName, ";")
	}
	g.P("} ", service.GoName, "CGONativeServerCallbacks;")
	g.P()
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P("static inline int32_t ", nativeCGOServerTrampolineName(service, method), "(", nativeCGOServerCallbackName(service, method), " callback, ", nativeCGOServerRequestName(service, method), "* input, ", nativeCGOServerResponseName(service, method), "* output) {")
		g.P("	return callback(input, output);")
		g.P("}")
		g.P()
	}
	g.P("*/")
}

func renderCGONativeServerCStruct(g *protogen.GeneratedFile, name string, fields []FieldPlan, output bool) {
	g.P("typedef struct ", name, " {")
	for _, field := range fields {
		renderCGONativeServerCField(g, field, output)
	}
	g.P("} ", name, ";")
	g.P()
}

func renderCGONativeServerCField(g *protogen.GeneratedFile, field FieldPlan, output bool) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("int8_t ", field.GoName, ";")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindEnum:
			g.P("int32_t ", field.GoName, ";")
		case FieldKindSignedInt64:
			g.P("int64_t ", field.GoName, ";")
		case FieldKindFloat:
			g.P("float ", field.GoName, ";")
		case FieldKindDouble:
			g.P("double ", field.GoName, ";")
		case FieldKindString, FieldKindBytes:
			g.P("uintptr_t ", field.GoName, "Ptr;")
			g.P("int32_t ", field.GoName, "Len;")
			if output {
				g.P("int32_t ", field.GoName, "Ownership;")
			}
		default:
			g.P("uintptr_t ", field.GoName, ";")
		}
	default:
		g.P("uintptr_t ", field.GoName, ";")
	}
}

func renderCGONativeServerAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, callbacksName, adapterName string, errorNames nativeServerCGOErrorNames) {
	g.P("type ", adapterName, " struct {")
	g.P("callbacks *C.", callbacksName)
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
		renderCGONativeServerRequestEncoder(g, service, method, errorNames)
		renderCGONativeServerResponseDecoder(g, service, method, errorNames)
		renderCGONativeServerResponseCleanup(g, service, method)
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
	g.P("output := &C.", nativeCGOServerResponseName(service, method), "{}")
	g.P("errID := int32(C.", nativeCGOServerTrampolineName(service, method), "(callback, input, output))")
	g.P("if errID != 0 {")
	g.P(nativeCGOServerResponseCleanupName(service, method), "(output)")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerResponseDecoderName(service, method), "(output)")
	g.P(nativeCGOServerResponseCleanupName(service, method), "(output)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
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

func renderCGONativeServerRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	requestName := nativeCGOServerRequestName(service, method)
	g.P("func ", nativeCGOServerRequestEncoderName(service, method), "(req ", nativeGoMessageType(g, method.Request), ") (*C.", requestName, ", func(), error) {")
	g.P("if req == nil {")
	g.P(`return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")`)
	g.P("}")
	g.P("input := &C.", requestName, "{}")
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
		case FieldKindSignedInt32:
			g.P("input.", field.GoName, " = C.int32_t(req.", field.GoName, ")")
		case FieldKindSignedInt64:
			g.P("input.", field.GoName, " = C.int64_t(req.", field.GoName, ")")
		case FieldKindFloat:
			g.P("input.", field.GoName, " = C.float(req.", field.GoName, ")")
		case FieldKindDouble:
			g.P("input.", field.GoName, " = C.double(req.", field.GoName, ")")
		case FieldKindEnum:
			g.P("input.", field.GoName, " = C.int32_t(req.", field.GoName, ")")
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
			g.P("input.", field.GoName, "Ptr = C.uintptr_t(", field.GoName, "Ptr)")
			g.P("input.", field.GoName, "Len = C.int32_t(", field.GoName, "Len)")
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
			g.P("input.", field.GoName, "Ptr = C.uintptr_t(", field.GoName, "Ptr)")
			g.P("input.", field.GoName, "Len = C.int32_t(", field.GoName, "Len)")
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
	g.P("func ", nativeCGOServerResponseDecoderName(service, method), "(output *C.", responseName, ") (", nativeGoMessageType(g, method.Response), ", error) {")
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
		case FieldKindSignedInt32:
			g.P("resp.", field.GoName, " = int32(output.", field.GoName, ")")
		case FieldKindSignedInt64:
			g.P("resp.", field.GoName, " = int64(output.", field.GoName, ")")
		case FieldKindFloat:
			g.P("resp.", field.GoName, " = float32(output.", field.GoName, ")")
		case FieldKindDouble:
			g.P("resp.", field.GoName, " = float64(output.", field.GoName, ")")
		case FieldKindEnum:
			g.P("resp.", field.GoName, " = ", nativeGoEnumType(g, field), "(int32(output.", field.GoName, "))")
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
	g.P("if _, err := rpcruntime.LengthFromInt32(int32(output.", field.GoName, "Len)); err != nil {")
	g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
	g.P("}")
	g.P(field.GoName, " := rpcruntime.NewRpc", wrapper, "((*byte)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), output.", field.GoName, "Ownership > 0)")
	g.P("resp.", field.GoName, " = ", field.GoName, ".", safeMethod, "()")
}

func renderCGONativeServerRegistration(g *protogen.GeneratedFile, service ServicePlan, callbacksName, adapterName string, errorNames nativeServerCGOErrorNames) {
	g.P("func Register", service.GoName, "CGONativeServer(callbacks *C.", callbacksName, ") (rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter], error) {")
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

func renderCGONativeServerResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", nativeCGOServerResponseCleanupName(service, method), "(output *C.", nativeCGOServerResponseName(service, method), ") {")
	g.P("if output == nil {")
	g.P("return")
	g.P("}")
	for _, field := range method.NativeContract.ResponseFields {
		if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("_ = rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\")")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerGoHelper(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, callbacksName string, errorNames nativeServerCGOErrorNames) {
	helperName := service.GoName + "GoCGONativeServerCallbacks"
	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}
	g.P("type ", helperName, " struct {")
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P(method.GoName, " func(ctx context.Context, input *C.", nativeCGOServerRequestName(service, method), ", output *C.", nativeCGOServerResponseName(service, method), ") int32")
	}
	g.P("}")
	g.P()
	g.P("func Register", service.GoName, "GoCGONativeServerForTesting(callbacks *", helperName, ") (rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter], error) {")
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
	g.P("return register", service.GoName, "ActiveServer(rpcruntime.ServerKindCGONative, &", lowerInitial(service.GoName), "GoCGONativeAdapter{callbacks: callbacks})")
	g.P("}")
	g.P()
	g.P("type ", lowerInitial(service.GoName), "GoCGONativeAdapter struct {")
	g.P("callbacks *", helperName)
	g.P("}")
	g.P()
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P("func (a *", lowerInitial(service.GoName), "GoCGONativeAdapter) ", method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", nativeGoMessageType(g, method.Response), ", error) {")
		g.P("input, cleanup, err := ", nativeCGOServerRequestEncoderName(service, method), "(req)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("defer cleanup()")
		g.P("output := &C.", nativeCGOServerResponseName(service, method), "{}")
		g.P("errID := a.callbacks.", method.GoName, "(ctx, input, output)")
		g.P("if errID != 0 {")
		g.P(nativeCGOServerResponseCleanupName(service, method), "(output)")
		g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
		g.P("}")
		g.P("resp, err := ", nativeCGOServerResponseDecoderName(service, method), "(output)")
		g.P(nativeCGOServerResponseCleanupName(service, method), "(output)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return resp, nil")
		g.P("}")
		g.P()
	}
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
		if ok && method.Streaming == StreamingKindUnary {
			continue
		}
		renderCGONativeServerStreamingFallback(g, lowerInitial(service.GoName)+"GoCGONativeAdapter", runtimeMethod, errorNames)
	}
}

func renderCGONativeServerErrorStoreExport(g *protogen.GeneratedFile, service ServicePlan) {
	exportName := "Store" + service.GoName + "CGONativeServerErrorTextForExport"
	g.P("//export ", exportName)
	g.P("func ", exportName, "(text *C.char, textLen C.int32_t) C.int32_t {")
	g.P("length, err := rpcruntime.LengthFromInt32(int32(textLen))")
	g.P("if err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: cgo native server error text: %w", err)))`)
	g.P("}")
	g.P("if text == nil && length != 0 {")
	g.P(`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: cgo native server error text pointer is nil")))`)
	g.P("}")
	g.P("var data []byte")
	g.P("if length != 0 {")
	g.P("data = unsafe.Slice((*byte)(unsafe.Pointer(text)), length)")
	g.P("}")
	g.P("return C.int32_t(rpcruntime.StoreError(errors.New(string(data))))")
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

func nativeCGOServerResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryCallback"
}

func nativeCGOServerTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeUnaryCallback"
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

func validateNativeServerCGOSymbols(plan FilePlan, service ServicePlan) error {
	seen := make(map[string]string)
	protobufSymbols := make(map[string]TopLevelSymbolPlan)
	for _, symbol := range plan.TopLevelSymbols {
		if symbol.GoName == "" {
			continue
		}
		protobufSymbols[symbol.GoName] = symbol
	}
	for _, method := range service.Methods {
		if method.Request.GoName != "" && method.Request.GoImportPath == plan.GoImportPath {
			protobufSymbols[method.Request.GoName] = TopLevelSymbolPlan{
				GoName:   method.Request.GoName,
				FullName: method.Request.FullName,
				Kind:     TopLevelSymbolKindMessage,
			}
		}
		if method.Response.GoName != "" && method.Response.GoImportPath == plan.GoImportPath {
			protobufSymbols[method.Response.GoName] = TopLevelSymbolPlan{
				GoName:   method.Response.GoName,
				FullName: method.Response.FullName,
				Kind:     TopLevelSymbolKindMessage,
			}
		}
	}
	for _, otherService := range plan.Services {
		if otherService.FullName != service.FullName && otherService.NativeFileFamily.CGONativeServer.Enabled {
			addNativeServerCGOGeneratedSymbols(seen, otherService)
		}
	}
	addGenerated := func(symbol, source string) error {
		if symbol == "" {
			return nil
		}
		if previous, exists := seen[symbol]; exists {
			if previous != source {
				return fmt.Errorf("native server cgo symbol %s for %s collides with %s", symbol, source, previous)
			}
			return nil
		}
		if protobufSymbol, exists := protobufSymbols[symbol]; exists {
			return fmt.Errorf("native server cgo symbol %s for %s collides with protobuf %s %s", symbol, source, protobufSymbol.Kind, protobufSymbol.FullName)
		}
		seen[symbol] = source
		return nil
	}

	errorNames := nativeServerCGOErrorNamesFor(service)
	for symbol, source := range map[string]string{
		service.GoName + "CGONativeServerCallbacks":                    errorNames.CallbacksNil,
		service.GoName + "GoCGONativeServerCallbacks":                  service.FullName + " go helper callbacks",
		lowerInitial(service.GoName) + "CGONativeAdapter":              service.FullName + " adapter",
		lowerInitial(service.GoName) + "GoCGONativeAdapter":            service.FullName + " go helper adapter",
		"Register" + service.GoName + "CGONativeServer":                service.FullName + " registration",
		"Register" + service.GoName + "GoCGONativeServerForTesting":    service.FullName + " go helper registration",
		"Store" + service.GoName + "CGONativeServerErrorTextForExport": service.FullName + " error text export",
		nativeCGOServerErrorIDHelperName(service):                      service.FullName + " error id helper",
		errorNames.CallbacksNil:                                        errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                                errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                                    errorNames.UnsupportedField,
		errorNames.StreamNotImplemented:                                errorNames.StreamNotImplemented,
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
			{nativeCGOServerCallbackName(service, method), method.FullName + " cgo callback"},
			{nativeCGOServerTrampolineName(service, method), method.FullName + " cgo trampoline"},
			{nativeCGOServerRequestEncoderName(service, method), method.FullName + " request encoder"},
			{nativeCGOServerResponseDecoderName(service, method), method.FullName + " response decoder"},
			{nativeCGOServerResponseCleanupName(service, method), method.FullName + " response cleanup"},
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

func addNativeServerCGOGeneratedSymbols(seen map[string]string, service ServicePlan) {
	add := func(symbol, source string) {
		if symbol == "" {
			return
		}
		if _, exists := seen[symbol]; !exists {
			seen[symbol] = source
		}
	}
	errorNames := nativeServerCGOErrorNamesFor(service)
	for symbol, source := range map[string]string{
		service.GoName + "CGONativeServerCallbacks":                    errorNames.CallbacksNil,
		service.GoName + "GoCGONativeServerCallbacks":                  service.FullName + " go helper callbacks",
		lowerInitial(service.GoName) + "CGONativeAdapter":              service.FullName + " adapter",
		lowerInitial(service.GoName) + "GoCGONativeAdapter":            service.FullName + " go helper adapter",
		"Register" + service.GoName + "CGONativeServer":                service.FullName + " registration",
		"Register" + service.GoName + "GoCGONativeServerForTesting":    service.FullName + " go helper registration",
		"Store" + service.GoName + "CGONativeServerErrorTextForExport": service.FullName + " error text export",
		nativeCGOServerErrorIDHelperName(service):                      service.FullName + " error id helper",
		errorNames.CallbacksNil:                                        errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                                errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                                    errorNames.UnsupportedField,
		errorNames.StreamNotImplemented:                                errorNames.StreamNotImplemented,
	} {
		add(symbol, source)
	}
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		add(nativeCGOServerRequestName(service, method), method.FullName+" cgo request")
		add(nativeCGOServerResponseName(service, method), method.FullName+" cgo response")
		add(nativeCGOServerCallbackName(service, method), method.FullName+" cgo callback")
		add(nativeCGOServerTrampolineName(service, method), method.FullName+" cgo trampoline")
		add(nativeCGOServerRequestEncoderName(service, method), method.FullName+" request encoder")
		add(nativeCGOServerResponseDecoderName(service, method), method.FullName+" response decoder")
		add(nativeCGOServerResponseCleanupName(service, method), method.FullName+" response cleanup")
	}
}
