package generator

import (
	"fmt"
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeClientCGOSymbols(plan, service); err != nil {
		return err
	}
	nativeCABIPlan, err := BuildNativeCABIPlan(service)
	if err != nil {
		return err
	}

	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := plugin.NewGeneratedFile(file.Filename, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"CGONativeClientBridge")
	g.P("package main")
	g.P()
	g.P("/*")
	g.P("#include <stdint.h>")
	g.P("*/")
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if nativeClientNeedsFmt(service) {
		g.P(`fmt "fmt"`)
	}
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	if nativeClientNeedsUnsafe(service) {
		g.P(`unsafe "unsafe"`)
	}
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	errorName := lowerInitial(service.GoName) + "NativeClientUnsupportedField"
	streamHandleErrorName := lowerInitial(service.GoName) + "NativeClientStreamHandleInvalid"
	g.P("var ", errorName, ` = errors.New("rpccgo: native unary client field bridge is not implemented")`)
	g.P("var ", streamHandleErrorName, ` = errors.New("rpccgo: native client stream handle is invalid")`)
	g.P()

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderNativeUnaryClient(g, nativeCABIPlan, service, method, errorName, servicePackage)
		case StreamingKindClientStreaming:
			renderNativeClientStreamingClient(g, nativeCABIPlan, service, method, errorName, servicePackage)
		case StreamingKindServerStreaming:
			renderNativeServerStreamingClient(g, nativeCABIPlan, service, method, errorName, servicePackage)
		case StreamingKindBidiStreaming:
			renderNativeBidiStreamingClient(g, nativeCABIPlan, service, method, errorName, servicePackage)
		}
	}
	return nil
}

func renderNativeUnaryClient(g *protogen.GeneratedFile, abiPlan NativeCABIPlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) {
	funcName := nativeUnaryClientFuncName(service, method)
	requestParams := nativeClientRequestParams(method.Contract.Native.RequestFields)
	requestArgs := nativeClientRequestCallArgs(method.Contract.Native.RequestFields)
	responseParams := nativeClientResponseOutputParams(method.Contract.Native.ResponseFields)
	responseArgs := nativeClientResponseOutputCallArgs(method.Contract.Native.ResponseFields)

	g.P("func ", funcName, "(ctx context.Context", nativeClientAppendParams("", requestParams, responseParams), ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	if responseParams != "" {
		g.P("if err := ", nativeUnaryClientOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	if requestNames == "" {
		g.P("if err := ", nativeUnaryClientDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeUnaryClientDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	if responseNames == "" {
		g.P("err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().", method.GoName, "(ctx", nativeGoCallSuffix(requestNames), ")")
	} else {
		g.P(responseNames, ", err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().", method.GoName, "(ctx", nativeGoCallSuffix(requestNames), ")")
	}
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeUnaryClientEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeUnaryRequestDecoder(g, service, method, unsupportedError)
	renderNativeUnaryResponseEncoder(g, service, method, unsupportedError)
	renderNativeCExportWrappers(g, abiPlan, service, method)
}

func renderNativeClientStreamingClient(g *protogen.GeneratedFile, abiPlan NativeCABIPlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) {
	requestParams := nativeClientRequestParams(method.Contract.Native.RequestFields)
	requestArgs := nativeClientRequestCallArgs(method.Contract.Native.RequestFields)
	responseParams := nativeClientResponseOutputParams(method.Contract.Native.ResponseFields)
	responseArgs := nativeClientResponseOutputCallArgs(method.Contract.Native.ResponseFields)

	g.P("func ", nativeClientStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingSendFuncName(service, method), "(ctx context.Context, handle int32", nativeClientAppendParams("", requestParams), ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	g.P("var err error")
	if requestNames == "" {
		g.P("if err := ", nativeClientStreamingDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeClientStreamingDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Send", "ctx"+nativeGoCallSuffix(requestNames))
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingFinishFuncName(service, method), "(ctx context.Context, handle int32", nativeClientAppendParams("", responseParams), ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	if responseParams != "" {
		g.P("if err := ", nativeClientStreamingOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("var err error")
	for _, decl := range nativeGoResponseResultVarDecls(g, method.Contract.Native.ResponseFields) {
		g.P(decl)
	}
	renderNativeClientStreamResultCall(g, service, method, servicePackage, responseNames, "Finish")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeClientStreamingEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Cancel", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeClientStreamingRequestDecoder(g, service, method, unsupportedError)
	renderNativeClientStreamingResponseEncoder(g, service, method, unsupportedError)
	renderNativeCExportWrappers(g, abiPlan, service, method)
}

func renderNativeServerStreamingClient(g *protogen.GeneratedFile, abiPlan NativeCABIPlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) {
	requestParams := nativeClientRequestParams(method.Contract.Native.RequestFields)
	requestArgs := nativeClientRequestCallArgs(method.Contract.Native.RequestFields)
	responseParams := nativeClientResponseOutputParams(method.Contract.Native.ResponseFields)
	responseArgs := nativeClientResponseOutputCallArgs(method.Contract.Native.ResponseFields)

	g.P("func ", nativeServerStreamingStartFuncName(service, method), "(ctx context.Context", nativeClientAppendParams("", requestParams), ") (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	g.P("var err error")
	if requestNames == "" {
		g.P("if err := ", nativeServerStreamingDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return 0, int32(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeServerStreamingDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return 0, int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Start", method.GoName, "(ctx", nativeGoCallSuffix(requestNames), ")")
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingReadFuncName(service, method), "(ctx context.Context, handle int32", nativeClientAppendParams("", responseParams), ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	if responseParams != "" {
		g.P("if err := ", nativeServerStreamingOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("var err error")
	for _, decl := range nativeGoResponseResultVarDecls(g, method.Contract.Native.ResponseFields) {
		g.P(decl)
	}
	renderNativeClientStreamResultCall(g, service, method, servicePackage, responseNames, "Recv")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeServerStreamingEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingDoneFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Done", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Cancel", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeServerStreamingRequestDecoder(g, service, method, unsupportedError)
	renderNativeServerStreamingResponseEncoder(g, service, method, unsupportedError)
	renderNativeCExportWrappers(g, abiPlan, service, method)
}

func renderNativeBidiStreamingClient(g *protogen.GeneratedFile, abiPlan NativeCABIPlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) {
	requestParams := nativeClientRequestParams(method.Contract.Native.RequestFields)
	requestArgs := nativeClientRequestCallArgs(method.Contract.Native.RequestFields)
	responseParams := nativeClientResponseOutputParams(method.Contract.Native.ResponseFields)
	responseArgs := nativeClientResponseOutputCallArgs(method.Contract.Native.ResponseFields)

	g.P("func ", nativeBidiStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingSendFuncName(service, method), "(ctx context.Context, handle int32", nativeClientAppendParams("", requestParams), ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	g.P("var err error")
	if requestNames == "" {
		g.P("if err := ", nativeBidiStreamingDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeBidiStreamingDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Send", "ctx"+nativeGoCallSuffix(requestNames))
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingReadFuncName(service, method), "(ctx context.Context, handle int32", nativeClientAppendParams("", responseParams), ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	if responseParams != "" {
		g.P("if err := ", nativeBidiStreamingOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return int32(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("var err error")
	for _, decl := range nativeGoResponseResultVarDecls(g, method.Contract.Native.ResponseFields) {
		g.P(decl)
	}
	renderNativeClientStreamResultCall(g, service, method, servicePackage, responseNames, "Recv")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeBidiStreamingEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingCloseSendFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "CloseSend", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingDoneFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Done", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Cancel", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeBidiStreamingRequestDecoder(g, service, method, unsupportedError)
	renderNativeBidiStreamingResponseEncoder(g, service, method, unsupportedError)
	renderNativeCExportWrappers(g, abiPlan, service, method)
}

func nativeClientRequestParams(fields []FieldPlan) string {
	parts := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		for _, param := range nativeClientInputFieldSymbols(field) {
			parts = append(parts, param+" "+nativeClientRequestParamType(field, param))
		}
	}
	return strings.Join(parts, ", ")
}

func nativeClientResponseOutputParams(fields []FieldPlan) string {
	parts := make([]string, 0, len(fields)*2)
	for _, field := range fields {
		for _, param := range nativeClientOutputFieldSymbols(field) {
			parts = append(parts, param+" *"+nativeClientOutputParamType(field, param))
		}
	}
	return strings.Join(parts, ", ")
}

func nativeClientRequestParamType(field FieldPlan, param string) string {
	if strings.HasSuffix(param, "Ptr") {
		return "uintptr"
	}
	if strings.HasSuffix(param, "Len") || strings.HasSuffix(param, "Ownership") {
		return "int32"
	}
	return nativeClientScalarParamType(field)
}

func nativeClientOutputParamType(field FieldPlan, param string) string {
	if strings.HasSuffix(param, "Ptr") {
		return "uintptr"
	}
	if strings.HasSuffix(param, "Len") {
		return "int32"
	}
	return nativeClientScalarParamType(field)
}

func nativeClientScalarParamType(field FieldPlan) string {
	switch field.Kind {
	case FieldKindBool:
		return "int8"
	case FieldKindSignedInt32, FieldKindEnum:
		return "int32"
	case FieldKindUnsignedInt32:
		return "uint32"
	case FieldKindSignedInt64:
		return "int64"
	case FieldKindUnsignedInt64:
		return "uint64"
	case FieldKindFloat:
		return "float32"
	case FieldKindDouble:
		return "float64"
	default:
		return "uintptr"
	}
}

func nativeClientRequestCallArgs(fields []FieldPlan) string {
	return strings.Join(nativeClientFlatSymbols(fields, nativeClientInputFieldSymbols), ", ")
}

func nativeClientResponseOutputCallArgs(fields []FieldPlan) string {
	return strings.Join(nativeClientFlatSymbols(fields, nativeClientOutputFieldSymbols), ", ")
}

func nativeClientFlatSymbols(fields []FieldPlan, symbols func(FieldPlan) []string) []string {
	parts := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		parts = append(parts, symbols(field)...)
	}
	return parts
}

func nativeClientAppendParams(base string, params ...string) string {
	parts := make([]string, 0, 1+len(params))
	if base != "" {
		parts = append(parts, base)
	}
	for _, param := range params {
		if param != "" {
			parts = append(parts, param)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return ", " + strings.Join(parts, ", ")
}

func nativeClientOutputPtrLocal(field FieldPlan) string {
	return lowerInitial(field.GoName) + "PtrValue"
}

func nativeClientOutputLenLocal(field FieldPlan) string {
	return lowerInitial(field.GoName) + "LenValue"
}

func nativeClientOutputValueSymbol(field FieldPlan) string {
	return "out" + field.GoName
}

func nativeClientOutputPtrSymbol(field FieldPlan) string {
	return "out" + field.GoName + "Ptr"
}

func nativeClientOutputLenSymbol(field FieldPlan) string {
	return "out" + field.GoName + "Len"
}

func renderNativeClientStreamFacadeCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, operation, args string) {
	g.P("err = ", servicePackage, "New", service.GoName, method.GoName, "NativeStream(rpcruntime.StreamHandle(handle)).", operation, "(", args, ")")
}

func renderNativeClientStreamFacadeResultCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, result, operation, args string) {
	g.P(result, ", err = ", servicePackage, "New", service.GoName, method.GoName, "NativeStream(rpcruntime.StreamHandle(handle)).", operation, "(", args, ")")
}

func renderNativeClientStreamResultCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, result, operation string) {
	if result == "" {
		g.P("err = ", servicePackage, "New", service.GoName, method.GoName, "NativeStream(rpcruntime.StreamHandle(handle)).", operation, "(ctx)")
		return
	}
	renderNativeClientStreamFacadeResultCall(g, service, method, servicePackage, result, operation, "ctx")
}

func renderNativeUnaryRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientRequestDecoder(g, nativeUnaryClientDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
}

func renderNativeClientStreamingRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientRequestDecoder(g, nativeClientStreamingDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
}

func renderNativeServerStreamingRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientRequestDecoder(g, nativeServerStreamingDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
}

func renderNativeBidiStreamingRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientRequestDecoder(g, nativeBidiStreamingDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
}

func renderNativeClientRequestDecoder(g *protogen.GeneratedFile, name string, fields []FieldPlan, unsupportedError string) {
	returns := nativeGoRequestReturns(g, fields)
	g.P("func ", name, "(", nativeClientRequestParams(fields), ") (", returns, ") {")
	if nativeClientRequestCleanupError(fields) != "" {
		g.P("var decoded []interface{ Release() error }")
		g.P("cleanupDecoded := func() error {")
		g.P("var errs []error")
		g.P("for i := len(decoded) - 1; i >= 0; i-- {")
		g.P("errs = append(errs, decoded[i].Release())")
		g.P("}")
		g.P("return errors.Join(errs...)")
		g.P("}")
	}

	for _, field := range fields {
		renderNativeRequestFieldDecode(g, fields, field, unsupportedError)
	}
	argNames := nativeClientRequestValueNames(fields)
	if argNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", argNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderNativeRequestFieldDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, unsupportedError string) {
	name := nativeClientValueName(field)
	errReturn := func(errExpr string) string {
		if nativeClientRequestCleanupError(fields) == "" {
			return nativeClientZeroReturns(fields, errExpr)
		}
		return nativeClientZeroReturns(fields, "errors.Join("+errExpr+", cleanupDecoded())")
	}
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P(name, " := ", field.GoName, " != 0")
	case NativeABIShapeBoolByteBufferWrapper:
		renderNativeClientRepeatedDecode(g, fields, field, name, "byte", "rpcruntime.RpcBoolRepeat", "rpcruntime.EmptyRpcBoolRepeat()", "rpcruntime.NewRpcBoolRepeatChecked", errReturn)
	case NativeABIShapeRepeated:
		switch field.Kind {
		case FieldKindSignedInt32:
			renderNativeClientRepeatedDecode(g, fields, field, name, "int32", "rpcruntime.RpcRepeat[int32]", "rpcruntime.EmptyRpcRepeat[int32]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		case FieldKindUnsignedInt32:
			renderNativeClientRepeatedDecode(g, fields, field, name, "uint32", "rpcruntime.RpcRepeat[uint32]", "rpcruntime.EmptyRpcRepeat[uint32]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		case FieldKindSignedInt64:
			renderNativeClientRepeatedDecode(g, fields, field, name, "int64", "rpcruntime.RpcRepeat[int64]", "rpcruntime.EmptyRpcRepeat[int64]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		case FieldKindUnsignedInt64:
			renderNativeClientRepeatedDecode(g, fields, field, name, "uint64", "rpcruntime.RpcRepeat[uint64]", "rpcruntime.EmptyRpcRepeat[uint64]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		case FieldKindFloat:
			renderNativeClientRepeatedDecode(g, fields, field, name, "float32", "rpcruntime.RpcRepeat[float32]", "rpcruntime.EmptyRpcRepeat[float32]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		case FieldKindDouble:
			renderNativeClientRepeatedDecode(g, fields, field, name, "float64", "rpcruntime.RpcRepeat[float64]", "rpcruntime.EmptyRpcRepeat[float64]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		case FieldKindEnum:
			renderNativeClientRepeatedDecode(g, fields, field, name, "int32", "rpcruntime.RpcRepeat[int32]", "rpcruntime.EmptyRpcRepeat[int32]()", "rpcruntime.NewRpcRepeatChecked", errReturn)
		default:
			g.P("return ", errReturn(unsupportedError))
		}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt32, FieldKindUnsignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(name, " := ", field.GoName)
		case FieldKindEnum:
			g.P(name, " := ", nativeGoEnumType(g, field), "(", field.GoName, ")")
		case FieldKindString:
			renderNativeClientStringDecode(g, fields, field, name, errReturn)
		case FieldKindBytes, FieldKindMessage:
			renderNativeClientBytesDecode(g, fields, field, name, errReturn)
		default:
			g.P("return ", errReturn(unsupportedError))
		}
	}
	if nativeClientFieldNeedsRequestRelease(field) {
		g.P("decoded = append(decoded, ", name, ")")
	}
}

func nativeGoRequestReturns(g *protogen.GeneratedFile, fields []FieldPlan) string {
	returns := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		returns = append(returns, nativeGoRequestFieldType(g, field))
	}
	returns = append(returns, "error")
	return strings.Join(returns, ", ")
}

func nativeClientValueName(field FieldPlan) string {
	return lowerInitial(field.GoName) + "Value"
}

func nativeClientResponseValueName(field FieldPlan) string {
	return lowerInitial(field.GoName) + "Result"
}

func nativeClientRequestValueNames(fields []FieldPlan) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, nativeClientValueName(field))
	}
	return strings.Join(names, ", ")
}

func nativeClientResponseValueNames(fields []FieldPlan) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, nativeClientResponseValueName(field))
	}
	return strings.Join(names, ", ")
}

func nativeClientZeroReturns(fields []FieldPlan, errExpr string) string {
	values := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		values = append(values, nativeGoRequestZeroValue(field))
	}
	values = append(values, errExpr)
	return strings.Join(values, ", ")
}

func nativeClientRequestCleanupError(fields []FieldPlan) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if !nativeClientFieldNeedsRequestRelease(field) {
			continue
		}
		parts = append(parts, nativeClientValueName(field)+".Release()")
	}
	return strings.Join(parts, ", ")
}

func nativeClientFieldNeedsRequestRelease(field FieldPlan) bool {
	switch field.Native.Shape {
	case NativeABIShapeBoolByteBufferWrapper, NativeABIShapeRepeated:
		return true
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		return field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage
	default:
		return false
	}
}

func nativeGoRequestZeroValue(field FieldPlan) string {
	if field.Repeated || field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage {
		return "nil"
	}
	return nativeGoZeroValue(field)
}

func renderNativeClientRepeatedDecode(g *protogen.GeneratedFile, _ []FieldPlan, field FieldPlan, name, elemType, wrapperType, emptyExpr, ctor string, errReturn func(string) string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(", field.GoName, "Len); err != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P("var ", name, " *", wrapperType)
	g.P("if ", field.GoName, "Ptr == 0 || ", field.GoName, "Len == 0 {")
	g.P(name, " = ", emptyExpr)
	g.P("} else {")
	g.P("var decodeErr error")
	g.P(name, ", decodeErr = ", ctor, "((*", elemType, ")(unsafe.Pointer(", field.GoName, "Ptr)), ", field.GoName, "Len, ", field.GoName, "Ownership > 0)")
	g.P("if decodeErr != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", decodeErr)`))
	g.P("}")
	g.P("}")
}

func renderNativeClientStringDecode(g *protogen.GeneratedFile, _ []FieldPlan, field FieldPlan, name string, errReturn func(string) string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(", field.GoName, "Len); err != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P("var ", name, " *rpcruntime.RpcString")
	g.P("if ", field.GoName, "Ptr == 0 || ", field.GoName, "Len == 0 {")
	g.P(name, " = rpcruntime.EmptyRpcString()")
	g.P("} else {")
	g.P("var decodeErr error")
	g.P(name, ", decodeErr = rpcruntime.NewRpcStringChecked((*byte)(unsafe.Pointer(", field.GoName, "Ptr)), ", field.GoName, "Len, ", field.GoName, "Ownership > 0)")
	g.P("if decodeErr != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", decodeErr)`))
	g.P("}")
	g.P("}")
}

func renderNativeClientBytesDecode(g *protogen.GeneratedFile, _ []FieldPlan, field FieldPlan, name string, errReturn func(string) string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(", field.GoName, "Len); err != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P("var ", name, " *rpcruntime.RpcBytes")
	g.P("if ", field.GoName, "Ptr == 0 || ", field.GoName, "Len == 0 {")
	g.P(name, " = rpcruntime.EmptyRpcBytes()")
	g.P("} else {")
	g.P("var decodeErr error")
	g.P(name, ", decodeErr = rpcruntime.NewRpcBytesChecked((*byte)(unsafe.Pointer(", field.GoName, "Ptr)), ", field.GoName, "Len, ", field.GoName, "Ownership > 0)")
	g.P("if decodeErr != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", decodeErr)`))
	g.P("}")
	g.P("}")
}

func renderNativeUnaryResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientOutputValidator(g, nativeUnaryClientOutputValidatorName(service, method), method.Contract.Native.ResponseFields)
	renderNativeClientResponseEncoder(g, nativeUnaryClientEncoderName(service, method), method.Contract.Native.ResponseFields, unsupportedError)
}

func renderNativeCExportWrappers(g *protogen.GeneratedFile, abiPlan NativeCABIPlan, service ServicePlan, method MethodPlan) {
	methodABI := nativeCABIPlanByMethod(abiPlan)[method.FullName]
	switch method.Streaming {
	case StreamingKindUnary:
		renderNativeUnaryCExportWrapper(g, service, method, nativeCABIPlanOperation(methodABI, NativeCOperationUnary))
	case StreamingKindClientStreaming:
		renderNativeClientStreamingCExportWrappers(g, service, method, methodABI)
	case StreamingKindServerStreaming:
		renderNativeServerStreamingCExportWrappers(g, service, method, methodABI)
	case StreamingKindBidiStreaming:
		renderNativeBidiStreamingCExportWrappers(g, service, method, methodABI)
	}
}

func renderNativeUnaryCExportWrapper(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unaryABI COperationABI) {
	exportName := unaryABI.Symbol
	g.P("//export ", exportName)
	g.P("func ", exportName, "(", nativeCExportParams(unaryABI.Params), ") ", unaryABI.Return.CGoType, " {")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, unaryABI.Params)
	g.P("return C.int32_t(", nativeUnaryClientFuncName(service, method), "(context.Background()", nativeCExportCallSuffix(nativeCExportGoArgs(service, method), nativeCExportOutputGoArgs(service, method)), "))")
	g.P("}")
	g.P()
}

func renderNativeClientStreamingCExportWrappers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, methodABI MethodNativeCABIPlan) {
	startABI := nativeCABIPlanOperation(methodABI, NativeCOperationStart)
	g.P("//export ", startABI.Symbol)
	g.P("func ", startABI.Symbol, "(", nativeCExportParams(startABI.Params), ") ", startABI.Return.CGoType, " {")
	renderNativeCExportHandleValidation(g, "stream")
	g.P("streamValue, errID := ", nativeClientStreamingStartFuncName(service, method), "(context.Background())")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*stream = C.int32_t(streamValue)")
	g.P("return 0")
	g.P("}")
	g.P()

	sendABI := nativeCABIPlanOperation(methodABI, NativeCOperationSend)
	g.P("//export ", sendABI.Symbol)
	g.P("func ", sendABI.Symbol, "(", nativeCExportParams(sendABI.Params), ") ", sendABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeClientStreamingSendFuncName(service, method), "(context.Background(), int32(stream)", nativeCExportCallSuffix(nativeCExportGoArgs(service, method)), "))")
	g.P("}")
	g.P()

	finishABI := nativeCABIPlanOperation(methodABI, NativeCOperationFinish)
	g.P("//export ", finishABI.Symbol)
	g.P("func ", finishABI.Symbol, "(", nativeCExportParams(finishABI.Params), ") ", finishABI.Return.CGoType, " {")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, finishABI.Params)
	g.P("return C.int32_t(", nativeClientStreamingFinishFuncName(service, method), "(context.Background(), int32(stream)", nativeCExportOutputArgs(method.Contract.Native.ResponseFields), "))")
	g.P("}")
	g.P()

	cancelABI := nativeCABIPlanOperation(methodABI, NativeCOperationCancel)
	g.P("//export ", cancelABI.Symbol)
	g.P("func ", cancelABI.Symbol, "(", nativeCExportParams(cancelABI.Params), ") ", cancelABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeClientStreamingCancelFuncName(service, method), "(context.Background(), int32(stream)))")
	g.P("}")
	g.P()
}

func renderNativeServerStreamingCExportWrappers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, methodABI MethodNativeCABIPlan) {
	startABI := nativeCABIPlanOperation(methodABI, NativeCOperationStart)
	g.P("//export ", startABI.Symbol)
	g.P("func ", startABI.Symbol, "(", nativeCExportParams(startABI.Params), ") ", startABI.Return.CGoType, " {")
	renderNativeCExportHandleValidation(g, "stream")
	g.P("streamValue, errID := ", nativeServerStreamingStartFuncName(service, method), "(context.Background()", nativeCExportCallSuffix(nativeCExportGoArgs(service, method)), ")")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*stream = C.int32_t(streamValue)")
	g.P("return 0")
	g.P("}")
	g.P()

	readABI := nativeCABIPlanOperation(methodABI, NativeCOperationRecv)
	g.P("//export ", readABI.Symbol)
	g.P("func ", readABI.Symbol, "(", nativeCExportParams(readABI.Params), ") ", readABI.Return.CGoType, " {")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, readABI.Params)
	g.P("return C.int32_t(", nativeServerStreamingReadFuncName(service, method), "(context.Background(), int32(stream)", nativeCExportOutputArgs(method.Contract.Native.ResponseFields), "))")
	g.P("}")
	g.P()

	doneABI := nativeCABIPlanOperation(methodABI, NativeCOperationDone)
	g.P("//export ", doneABI.Symbol)
	g.P("func ", doneABI.Symbol, "(", nativeCExportParams(doneABI.Params), ") ", doneABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeServerStreamingDoneFuncName(service, method), "(context.Background(), int32(stream)))")
	g.P("}")
	g.P()

	cancelABI := nativeCABIPlanOperation(methodABI, NativeCOperationCancel)
	g.P("//export ", cancelABI.Symbol)
	g.P("func ", cancelABI.Symbol, "(", nativeCExportParams(cancelABI.Params), ") ", cancelABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeServerStreamingCancelFuncName(service, method), "(context.Background(), int32(stream)))")
	g.P("}")
	g.P()
}

func renderNativeBidiStreamingCExportWrappers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, methodABI MethodNativeCABIPlan) {
	startABI := nativeCABIPlanOperation(methodABI, NativeCOperationStart)
	g.P("//export ", startABI.Symbol)
	g.P("func ", startABI.Symbol, "(", nativeCExportParams(startABI.Params), ") ", startABI.Return.CGoType, " {")
	renderNativeCExportHandleValidation(g, "stream")
	g.P("streamValue, errID := ", nativeBidiStreamingStartFuncName(service, method), "(context.Background())")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*stream = C.int32_t(streamValue)")
	g.P("return 0")
	g.P("}")
	g.P()

	sendABI := nativeCABIPlanOperation(methodABI, NativeCOperationSend)
	g.P("//export ", sendABI.Symbol)
	g.P("func ", sendABI.Symbol, "(", nativeCExportParams(sendABI.Params), ") ", sendABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeBidiStreamingSendFuncName(service, method), "(context.Background(), int32(stream)", nativeCExportCallSuffix(nativeCExportGoArgs(service, method)), "))")
	g.P("}")
	g.P()

	readABI := nativeCABIPlanOperation(methodABI, NativeCOperationRecv)
	g.P("//export ", readABI.Symbol)
	g.P("func ", readABI.Symbol, "(", nativeCExportParams(readABI.Params), ") ", readABI.Return.CGoType, " {")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, readABI.Params)
	g.P("return C.int32_t(", nativeBidiStreamingReadFuncName(service, method), "(context.Background(), int32(stream)", nativeCExportOutputArgs(method.Contract.Native.ResponseFields), "))")
	g.P("}")
	g.P()

	closeSendABI := nativeCABIPlanOperation(methodABI, NativeCOperationCloseSend)
	g.P("//export ", closeSendABI.Symbol)
	g.P("func ", closeSendABI.Symbol, "(", nativeCExportParams(closeSendABI.Params), ") ", closeSendABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeBidiStreamingCloseSendFuncName(service, method), "(context.Background(), int32(stream)))")
	g.P("}")
	g.P()

	doneABI := nativeCABIPlanOperation(methodABI, NativeCOperationDone)
	g.P("//export ", doneABI.Symbol)
	g.P("func ", doneABI.Symbol, "(", nativeCExportParams(doneABI.Params), ") ", doneABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeBidiStreamingDoneFuncName(service, method), "(context.Background(), int32(stream)))")
	g.P("}")
	g.P()

	cancelABI := nativeCABIPlanOperation(methodABI, NativeCOperationCancel)
	g.P("//export ", cancelABI.Symbol)
	g.P("func ", cancelABI.Symbol, "(", nativeCExportParams(cancelABI.Params), ") ", cancelABI.Return.CGoType, " {")
	g.P("return C.int32_t(", nativeBidiStreamingCancelFuncName(service, method), "(context.Background(), int32(stream)))")
	g.P("}")
	g.P()
}

func renderNativeCExportOutputValidation(g *protogen.GeneratedFile, fields []FieldPlan, slots []CABISlot) {
	for _, slot := range nativeCExportOutputSlots(slots) {
		g.P("if ", slot.Name, " != nil {")
		g.P("*", slot.Name, " = 0")
		g.P("}")
	}
	for _, field := range fields {
		for _, symbol := range nativeClientOutputFieldSymbols(field) {
			g.P("if ", symbol, " == nil {")
			g.P(`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client output pointer is nil")))`)
			g.P("}")
		}
	}
}

func renderNativeCExportOutputCommit(g *protogen.GeneratedFile, _ ServicePlan, method MethodPlan) {
	for _, field := range method.Contract.Native.ResponseFields {
		renderNativeResponseFieldCommit(g, field)
	}
}

func renderNativeCExportHandleValidation(g *protogen.GeneratedFile, name string) {
	g.P("if ", name, " != nil {")
	g.P("*", name, " = 0")
	g.P("}")
	g.P("if ", name, " == nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: native client handle pointer is nil")))`)
	g.P("}")
}

func nativeCExportFuncName(plan FilePlan, service ServicePlan, method MethodPlan, operation string) string {
	name := "rpccgo_native_" + plan.GoPackageName + "_" + service.GoName + "_" + method.GoName
	if operation != "" {
		name += "_" + operation
	}
	return name
}

func nativeGoUnaryOutputTypeName(service ServicePlan, method MethodPlan) string {
	return nativeUnaryClientOutputName(service, method)
}

func nativeGoResponseOutputTypeName(service ServicePlan, method MethodPlan) string {
	return nativeClientStreamingOutputName(service, method)
}

func nativeCExportParams(slots []CABISlot) string {
	parts := make([]string, 0, len(slots))
	for _, slot := range slots {
		parts = append(parts, slot.Name+" "+slot.CGoType)
	}
	return strings.Join(parts, ", ")
}

func nativeCExportOutputSlots(slots []CABISlot) []CABISlot {
	out := make([]CABISlot, 0, len(slots))
	for _, slot := range slots {
		if strings.HasPrefix(slot.CGoType, "*") {
			out = append(out, slot)
		}
	}
	return out
}

func nativeCExportParamJoin(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, ", ")
}

func nativeCExportCallSuffix(parts ...string) string {
	joined := nativeCExportParamJoin(parts...)
	if joined == "" {
		return ""
	}
	return ", " + joined
}

func nativeCExportGoArgs(_ ServicePlan, method MethodPlan) string {
	args := make([]string, 0)
	for _, field := range method.Contract.Native.RequestFields {
		for _, symbol := range nativeClientInputFieldSymbols(field) {
			args = append(args, nativeCExportGoArg(symbol, field))
		}
	}
	return strings.Join(args, ", ")
}

func nativeCExportOutputGoArgs(_ ServicePlan, method MethodPlan) string {
	args := make([]string, 0)
	for _, field := range method.Contract.Native.ResponseFields {
		for _, symbol := range nativeClientOutputFieldSymbols(field) {
			args = append(args, nativeCExportOutputGoArg(symbol, field))
		}
	}
	return strings.Join(args, ", ")
}

func nativeCExportOutputArgs(fields []FieldPlan) string {
	args := make([]string, 0)
	for _, field := range fields {
		for _, symbol := range nativeClientOutputFieldSymbols(field) {
			args = append(args, nativeCExportOutputGoArg(symbol, field))
		}
	}
	return nativeCExportCallSuffix(args...)
}

func nativeCExportGoArg(symbol string, field FieldPlan) string {
	if strings.HasSuffix(symbol, "Ptr") {
		return "uintptr(" + symbol + ")"
	}
	if strings.HasSuffix(symbol, "Len") || strings.HasSuffix(symbol, "Ownership") {
		return "int32(" + symbol + ")"
	}
	if field.Kind == FieldKindBool {
		return "int8(" + symbol + ")"
	}
	return nativeCExportScalarGoArg(symbol, field)
}


func nativeCExportScalarGoArg(symbol string, field FieldPlan) string {
	switch field.Kind {
	case FieldKindBool:
		return "int8(" + symbol + ")"
	case FieldKindSignedInt32, FieldKindEnum:
		return "int32(" + symbol + ")"
	case FieldKindUnsignedInt32:
		return "uint32(" + symbol + ")"
	case FieldKindSignedInt64:
		return "int64(" + symbol + ")"
	case FieldKindUnsignedInt64:
		return "uint64(" + symbol + ")"
	case FieldKindFloat:
		return "float32(" + symbol + ")"
	case FieldKindDouble:
		return "float64(" + symbol + ")"
	default:
		return symbol
	}
}

func nativeCExportOutputGoArg(symbol string, field FieldPlan) string {
	return "(*" + nativeClientOutputParamType(field, symbol) + ")(unsafe.Pointer(" + symbol + "))"
}

func renderNativeClientStreamingResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientOutputValidator(g, nativeClientStreamingOutputValidatorName(service, method), method.Contract.Native.ResponseFields)
	renderNativeClientResponseEncoder(g, nativeClientStreamingEncoderName(service, method), method.Contract.Native.ResponseFields, unsupportedError)
}

func renderNativeServerStreamingResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientOutputValidator(g, nativeServerStreamingOutputValidatorName(service, method), method.Contract.Native.ResponseFields)
	renderNativeClientResponseEncoder(g, nativeServerStreamingEncoderName(service, method), method.Contract.Native.ResponseFields, unsupportedError)
}

func renderNativeBidiStreamingResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientOutputValidator(g, nativeBidiStreamingOutputValidatorName(service, method), method.Contract.Native.ResponseFields)
	renderNativeClientResponseEncoder(g, nativeBidiStreamingEncoderName(service, method), method.Contract.Native.ResponseFields, unsupportedError)
}

func renderNativeResponseFieldValidate(g *protogen.GeneratedFile, field FieldPlan, unsupportedError string) {
	name := nativeClientResponseValueName(field)
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return
	case NativeABIShapeBoolByteBufferWrapper:
		g.P(nativeClientOutputLenLocal(field), ", err := rpcruntime.LengthToInt32(len(", name, "))")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		return
	case NativeABIShapeRepeated:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble, FieldKindEnum:
			g.P(nativeClientOutputLenLocal(field), ", err := rpcruntime.LengthToInt32(len(", name, "))")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			return
		}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble, FieldKindEnum:
			return
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			g.P(nativeClientOutputLenLocal(field), ", err := rpcruntime.LengthToInt32(len(", name, "))")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			return
		}
	}
	g.P("return ", unsupportedError)
}

func renderNativeResponseFieldStage(g *protogen.GeneratedFile, field FieldPlan, pinned []FieldPlan) {
	name := nativeClientResponseValueName(field)
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("var ", name, "Value int8")
		g.P("if ", name, " {")
		g.P(name, "Value = 1")
		g.P("}")
	case NativeABIShapeBoolByteBufferWrapper:
		g.P(name, "Bytes := make([]byte, len(", name, "))")
		g.P("for i := range ", name, " {")
		g.P("if ", name, "[i] {")
		g.P(name, "Bytes[i] = 1")
		g.P("}")
		g.P("}")
		g.P(nativeClientOutputPtrLocal(field), ", err := rpcruntime.PinBytes(", name, "Bytes)")
		g.P("if err != nil {")
		renderReleasePinnedOutputFields(g, pinned)
		g.P("return err")
		g.P("}")
		g.P("_ = ", nativeClientOutputPtrLocal(field))
	case NativeABIShapeRepeated:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt32, FieldKindUnsignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(nativeClientOutputPtrLocal(field), ", err := rpcruntime.PinSlice(", name, ")")
			g.P("if err != nil {")
			renderReleasePinnedOutputFields(g, pinned)
			g.P("return err")
			g.P("}")
			g.P("_ = ", nativeClientOutputPtrLocal(field))
		case FieldKindEnum:
			g.P(name, "Values := make([]int32, len(", name, "))")
			g.P("for i := range ", name, " {")
			g.P(name, "Values[i] = int32(", name, "[i])")
			g.P("}")
			g.P(nativeClientOutputPtrLocal(field), ", err := rpcruntime.PinSlice(", name, "Values)")
			g.P("if err != nil {")
			renderReleasePinnedOutputFields(g, pinned)
			g.P("return err")
			g.P("}")
			g.P("_ = ", nativeClientOutputPtrLocal(field))
		}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt32, FieldKindUnsignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(name, "Value := ", name)
		case FieldKindEnum:
			g.P(name, "Value := int32(", name, ")")
		case FieldKindString:
			g.P("data, ", nativeClientOutputPtrLocal(field), ", err := rpcruntime.PinString(", name, ")")
			g.P("_ = data")
			g.P("if err != nil {")
			renderReleasePinnedOutputFields(g, pinned)
			g.P("return err")
			g.P("}")
			g.P("_ = ", nativeClientOutputPtrLocal(field))
		case FieldKindBytes, FieldKindMessage:
			g.P(nativeClientOutputPtrLocal(field), ", err := rpcruntime.PinBytes(", name, ")")
			g.P("if err != nil {")
			renderReleasePinnedOutputFields(g, pinned)
			g.P("return err")
			g.P("}")
			g.P("_ = ", nativeClientOutputPtrLocal(field))
		}
	}
}

func renderReleasePinnedOutputFields(g *protogen.GeneratedFile, fields []FieldPlan) {
	for _, field := range fields {
		g.P("rpcruntime.Release(", nativeClientOutputPtrLocal(field), ")")
	}
}

func renderNativeClientOutputValidator(g *protogen.GeneratedFile, name string, fields []FieldPlan) {
	params := nativeClientResponseOutputParams(fields)
	if params == "" {
		return
	}
	g.P("func ", name, "(", params, ") error {")
	for _, symbol := range nativeClientFlatSymbols(fields, nativeClientOutputFieldSymbols) {
		g.P("if ", symbol, " == nil {")
		g.P(`return errors.New("rpccgo: native client output pointer is nil")`)
		g.P("}")
	}
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderNativeClientResponseEncoder(g *protogen.GeneratedFile, name string, fields []FieldPlan, unsupportedError string) {
	g.P("func ", name, "(", nativeGoRequestArgsForResponse(g, fields), nativeClientResponseOutputParams(fields), ") error {")
	if nativeClientResponseOutputParams(fields) != "" {
		g.P("if err := ", strings.Replace(name, "encode", "validate", 1), "(", nativeClientResponseOutputCallArgs(fields), "); err != nil {")
		g.P("return err")
		g.P("}")
	}
	for _, field := range fields {
		renderNativeResponseFieldValidate(g, field, unsupportedError)
	}
	var pinned []FieldPlan
	for _, field := range fields {
		renderNativeResponseFieldStage(g, field, pinned)
		if nativeClientFieldPinsOutput(field) {
			pinned = append(pinned, field)
		}
	}
	for _, field := range fields {
		renderNativeResponseFieldCommit(g, field)
	}
	g.P("return nil")
	g.P("}")
	g.P()
}

func nativeGoRequestArgsForResponse(g *protogen.GeneratedFile, fields []FieldPlan) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, nativeClientResponseValueName(field)+" "+nativeGoResponseFieldType(g, field))
	}
	return strings.Join(parts, ", ") + ", "
}

func nativeClientEncoderCallArgs(args string) string {
	if args == "" {
		return ""
	}
	return args + ", "
}

func renderNativeResponseFieldCommit(g *protogen.GeneratedFile, field FieldPlan) {
	name := nativeClientResponseValueName(field)
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("*", nativeClientOutputValueSymbol(field), " = ", name, "Value")
	case NativeABIShapeBoolByteBufferWrapper, NativeABIShapeRepeated:
		g.P("*", nativeClientOutputPtrSymbol(field), " = ", nativeClientOutputPtrLocal(field))
		g.P("*", nativeClientOutputLenSymbol(field), " = ", nativeClientOutputLenLocal(field))
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble, FieldKindEnum:
			g.P("*", nativeClientOutputValueSymbol(field), " = ", name, "Value")
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			g.P("*", nativeClientOutputPtrSymbol(field), " = ", nativeClientOutputPtrLocal(field))
			g.P("*", nativeClientOutputLenSymbol(field), " = ", nativeClientOutputLenLocal(field))
		}
	}
}

func nativeGoEnumType(g *protogen.GeneratedFile, field FieldPlan) string {
	return g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       field.EnumType.GoName,
		GoImportPath: protogen.GoImportPath(field.EnumType.GoImportPath),
	})
}

func cgoGoImportPath(plan FilePlan) string {
	return path.Join(string(plan.GoImportPath), cgoDirForFilePlan(plan))
}

func cgoServicePackageQualifier(g *protogen.GeneratedFile, goImportPath string, symbol string) string {
	qualified := g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       symbol,
		GoImportPath: protogen.GoImportPath(goImportPath),
	})
	return qualified[:len(qualified)-len(symbol)]
}

func nativeUnaryClientInputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeUnaryInput"
}

func nativeClientStreamingInputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeClientStreamInput"
}

func nativeClientStreamingOutputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeClientStreamOutput"
}

func nativeClientStreamingStartFuncName(service ServicePlan, method MethodPlan) string {
	return "Start" + service.GoName + method.GoName + "NativeClientStream"
}

func nativeClientStreamingSendFuncName(service ServicePlan, method MethodPlan) string {
	return "Send" + service.GoName + method.GoName + "NativeClientStream"
}

func nativeClientStreamingFinishFuncName(service ServicePlan, method MethodPlan) string {
	return "Finish" + service.GoName + method.GoName + "NativeClientStream"
}

func nativeClientStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "NativeClientStream"
}

func nativeClientStreamingDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "NativeClientStreamRequest"
}

func nativeClientStreamingEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "NativeClientStreamResponse"
}

func nativeClientStreamingOutputValidatorName(service ServicePlan, method MethodPlan) string {
	return "validate" + service.GoName + method.GoName + "NativeClientStreamResponse"
}

func nativeServerStreamingInputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeServerStreamInput"
}

func nativeServerStreamingOutputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeServerStreamOutput"
}

func nativeServerStreamingStartFuncName(service ServicePlan, method MethodPlan) string {
	return "Start" + service.GoName + method.GoName + "NativeServerStream"
}

func nativeServerStreamingReadFuncName(service ServicePlan, method MethodPlan) string {
	return "Read" + service.GoName + method.GoName + "NativeServerStream"
}

func nativeServerStreamingDoneFuncName(service ServicePlan, method MethodPlan) string {
	return "Done" + service.GoName + method.GoName + "NativeServerStream"
}

func nativeServerStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "NativeServerStream"
}

func nativeServerStreamingDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "NativeServerStreamRequest"
}

func nativeServerStreamingEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "NativeServerStreamResponse"
}

func nativeServerStreamingOutputValidatorName(service ServicePlan, method MethodPlan) string {
	return "validate" + service.GoName + method.GoName + "NativeServerStreamResponse"
}

func nativeBidiStreamingInputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeBidiStreamInput"
}

func nativeBidiStreamingOutputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeBidiStreamOutput"
}

func nativeBidiStreamingStartFuncName(service ServicePlan, method MethodPlan) string {
	return "Start" + service.GoName + method.GoName + "NativeBidiStream"
}

func nativeBidiStreamingSendFuncName(service ServicePlan, method MethodPlan) string {
	return "Send" + service.GoName + method.GoName + "NativeBidiStream"
}

func nativeBidiStreamingReadFuncName(service ServicePlan, method MethodPlan) string {
	return "Read" + service.GoName + method.GoName + "NativeBidiStream"
}

func nativeBidiStreamingCloseSendFuncName(service ServicePlan, method MethodPlan) string {
	return "CloseSend" + service.GoName + method.GoName + "NativeBidiStream"
}

func nativeBidiStreamingDoneFuncName(service ServicePlan, method MethodPlan) string {
	return "Done" + service.GoName + method.GoName + "NativeBidiStream"
}

func nativeBidiStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "NativeBidiStream"
}

func nativeBidiStreamingDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "NativeBidiStreamRequest"
}

func nativeBidiStreamingEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "NativeBidiStreamResponse"
}

func nativeBidiStreamingOutputValidatorName(service ServicePlan, method MethodPlan) string {
	return "validate" + service.GoName + method.GoName + "NativeBidiStreamResponse"
}

func nativeUnaryClientOutputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeUnaryOutput"
}

func nativeUnaryClientFuncName(service ServicePlan, method MethodPlan) string {
	return "Call" + service.GoName + method.GoName + "NativeUnary"
}

func nativeUnaryClientDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "NativeUnaryRequest"
}

func nativeUnaryClientEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "NativeUnaryResponse"
}

func nativeUnaryClientOutputValidatorName(service ServicePlan, method MethodPlan) string {
	return "validate" + service.GoName + method.GoName + "NativeUnaryResponse"
}

func nativeClientNeedsFmt(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary && method.Streaming != StreamingKindClientStreaming && method.Streaming != StreamingKindServerStreaming && method.Streaming != StreamingKindBidiStreaming {
			continue
		}
		for _, field := range method.Contract.Native.RequestFields {
			if (field.Native.Shape == NativeABIShapeScalar || field.Native.Shape == NativeABIShapeMessageBytes) && (field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage) {
				return true
			}
			if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
				return true
			}
		}
	}
	return false
}

func nativeClientNeedsUnsafe(service ServicePlan) bool {
	for _, method := range service.Methods {
		for _, field := range method.Contract.Native.RequestFields {
			if (field.Native.Shape == NativeABIShapeScalar || field.Native.Shape == NativeABIShapeMessageBytes) && (field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage) {
				return true
			}
			if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
				return true
			}
		}
		if len(method.Contract.Native.ResponseFields) > 0 {
			return true
		}
	}
	return false
}

func nativeClientFieldPinsOutput(field FieldPlan) bool {
	if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
		return true
	}
	return (field.Native.Shape == NativeABIShapeScalar || field.Native.Shape == NativeABIShapeMessageBytes) && (field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage)
}

func nativeClientInputFieldSymbols(field FieldPlan) []string {
	if (field.Native.Shape == NativeABIShapeScalar || field.Native.Shape == NativeABIShapeMessageBytes) && (field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage) {
		return []string{field.GoName + "Ptr", field.GoName + "Len", field.GoName + "Ownership"}
	}
	if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
		return []string{field.GoName + "Ptr", field.GoName + "Len", field.GoName + "Ownership"}
	}
	return []string{field.GoName}
}

func nativeClientOutputFieldSymbols(field FieldPlan) []string {
	if nativeClientFieldPinsOutput(field) {
		return []string{nativeClientOutputPtrSymbol(field), nativeClientOutputLenSymbol(field)}
	}
	return []string{nativeClientOutputValueSymbol(field)}
}

func validateNativeClientCGOSymbols(plan FilePlan, service ServicePlan) error {
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
		if otherService.FullName != service.FullName && otherService.NativeFileFamily.CGONativeClient.Enabled {
			addNativeClientGeneratedSymbols(seen, otherService)
		}
	}

	addGenerated := func(symbol, source string) error {
		if symbol == "" {
			return nil
		}
		if previous, exists := seen[symbol]; exists {
			if previous != source {
				return fmt.Errorf("native client cgo symbol %s for %s collides with %s", symbol, source, previous)
			}
			return nil
		}
		if protobufSymbol, exists := protobufSymbols[symbol]; exists {
			return fmt.Errorf("native client cgo symbol %s for %s collides with protobuf %s %s", symbol, source, protobufSymbol.Kind, protobufSymbol.FullName)
		}
		seen[symbol] = source
		return nil
	}

	if err := addGenerated(lowerInitial(service.GoName)+"NativeClientUnsupportedField", service.FullName+" unsupported field error"); err != nil {
		return err
	}
	if err := addGenerated(lowerInitial(service.GoName)+"NativeClientStreamHandleInvalid", service.FullName+" stream handle error"); err != nil {
		return err
	}
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			if err := addGenerated(nativeUnaryClientFuncName(service, method), method.FullName+" unary client call"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientDecoderName(service, method), method.FullName+" unary request decoder"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientEncoderName(service, method), method.FullName+" unary response encoder"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientOutputValidatorName(service, method), method.FullName+" unary output validator"); err != nil {
				return err
			}
			if err := validateNativeClientFlatFields(method.FullName+" unary request", method.Contract.Native.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientFlatFields(method.FullName+" unary response", method.Contract.Native.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
				return err
			}
		case StreamingKindClientStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeClientStreamingStartFuncName(service, method), method.FullName + " client stream start"},
				{nativeClientStreamingSendFuncName(service, method), method.FullName + " client stream send"},
				{nativeClientStreamingFinishFuncName(service, method), method.FullName + " client stream finish"},
				{nativeClientStreamingCancelFuncName(service, method), method.FullName + " client stream cancel"},
				{nativeClientStreamingDecoderName(service, method), method.FullName + " client stream request decoder"},
				{nativeClientStreamingEncoderName(service, method), method.FullName + " client stream response encoder"},
				{nativeClientStreamingOutputValidatorName(service, method), method.FullName + " client stream output validator"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
			if err := validateNativeClientFlatFields(method.FullName+" client stream request", method.Contract.Native.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientFlatFields(method.FullName+" client stream response", method.Contract.Native.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
				return err
			}
		case StreamingKindServerStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeServerStreamingStartFuncName(service, method), method.FullName + " server stream start"},
				{nativeServerStreamingReadFuncName(service, method), method.FullName + " server stream read"},
				{nativeServerStreamingDoneFuncName(service, method), method.FullName + " server stream done"},
				{nativeServerStreamingCancelFuncName(service, method), method.FullName + " server stream cancel"},
				{nativeServerStreamingDecoderName(service, method), method.FullName + " server stream request decoder"},
				{nativeServerStreamingEncoderName(service, method), method.FullName + " server stream response encoder"},
				{nativeServerStreamingOutputValidatorName(service, method), method.FullName + " server stream output validator"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
			if err := validateNativeClientFlatFields(method.FullName+" server stream request", method.Contract.Native.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientFlatFields(method.FullName+" server stream response", method.Contract.Native.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
				return err
			}
		case StreamingKindBidiStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeBidiStreamingStartFuncName(service, method), method.FullName + " bidi stream start"},
				{nativeBidiStreamingSendFuncName(service, method), method.FullName + " bidi stream send"},
				{nativeBidiStreamingReadFuncName(service, method), method.FullName + " bidi stream read"},
				{nativeBidiStreamingCloseSendFuncName(service, method), method.FullName + " bidi stream close send"},
				{nativeBidiStreamingDoneFuncName(service, method), method.FullName + " bidi stream done"},
				{nativeBidiStreamingCancelFuncName(service, method), method.FullName + " bidi stream cancel"},
				{nativeBidiStreamingDecoderName(service, method), method.FullName + " bidi stream request decoder"},
				{nativeBidiStreamingEncoderName(service, method), method.FullName + " bidi stream response encoder"},
				{nativeBidiStreamingOutputValidatorName(service, method), method.FullName + " bidi stream output validator"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
			if err := validateNativeClientFlatFields(method.FullName+" bidi stream request", method.Contract.Native.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientFlatFields(method.FullName+" bidi stream response", method.Contract.Native.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
				return err
			}
		}
	}
	return nil
}

func addNativeClientGeneratedSymbols(seen map[string]string, service ServicePlan) {
	add := func(symbol, source string) {
		if symbol == "" {
			return
		}
		if _, exists := seen[symbol]; !exists {
			seen[symbol] = source
		}
	}

	add(lowerInitial(service.GoName)+"NativeClientUnsupportedField", service.FullName+" unsupported field error")
	add(lowerInitial(service.GoName)+"NativeClientStreamHandleInvalid", service.FullName+" stream handle error")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			add(nativeUnaryClientFuncName(service, method), method.FullName+" unary client call")
			add(nativeUnaryClientDecoderName(service, method), method.FullName+" unary request decoder")
			add(nativeUnaryClientEncoderName(service, method), method.FullName+" unary response encoder")
			add(nativeUnaryClientOutputValidatorName(service, method), method.FullName+" unary output validator")
		case StreamingKindClientStreaming:
			add(nativeClientStreamingStartFuncName(service, method), method.FullName+" client stream start")
			add(nativeClientStreamingSendFuncName(service, method), method.FullName+" client stream send")
			add(nativeClientStreamingFinishFuncName(service, method), method.FullName+" client stream finish")
			add(nativeClientStreamingCancelFuncName(service, method), method.FullName+" client stream cancel")
			add(nativeClientStreamingDecoderName(service, method), method.FullName+" client stream request decoder")
			add(nativeClientStreamingEncoderName(service, method), method.FullName+" client stream response encoder")
			add(nativeClientStreamingOutputValidatorName(service, method), method.FullName+" client stream output validator")
		case StreamingKindServerStreaming:
			add(nativeServerStreamingStartFuncName(service, method), method.FullName+" server stream start")
			add(nativeServerStreamingReadFuncName(service, method), method.FullName+" server stream read")
			add(nativeServerStreamingDoneFuncName(service, method), method.FullName+" server stream done")
			add(nativeServerStreamingCancelFuncName(service, method), method.FullName+" server stream cancel")
			add(nativeServerStreamingDecoderName(service, method), method.FullName+" server stream request decoder")
			add(nativeServerStreamingEncoderName(service, method), method.FullName+" server stream response encoder")
			add(nativeServerStreamingOutputValidatorName(service, method), method.FullName+" server stream output validator")
		case StreamingKindBidiStreaming:
			add(nativeBidiStreamingStartFuncName(service, method), method.FullName+" bidi stream start")
			add(nativeBidiStreamingSendFuncName(service, method), method.FullName+" bidi stream send")
			add(nativeBidiStreamingReadFuncName(service, method), method.FullName+" bidi stream read")
			add(nativeBidiStreamingCloseSendFuncName(service, method), method.FullName+" bidi stream close send")
			add(nativeBidiStreamingDoneFuncName(service, method), method.FullName+" bidi stream done")
			add(nativeBidiStreamingCancelFuncName(service, method), method.FullName+" bidi stream cancel")
			add(nativeBidiStreamingDecoderName(service, method), method.FullName+" bidi stream request decoder")
			add(nativeBidiStreamingEncoderName(service, method), method.FullName+" bidi stream response encoder")
			add(nativeBidiStreamingOutputValidatorName(service, method), method.FullName+" bidi stream output validator")
		}
	}
}

func validateNativeClientFlatFields(source string, fields []FieldPlan, symbols func(FieldPlan) []string) error {
	seen := make(map[string]string)
	for _, field := range fields {
		for _, symbol := range symbols(field) {
			if previous, exists := seen[symbol]; exists {
				return fmt.Errorf("native client cgo flat field %s for %s collides with %s in %s", symbol, field.FullName, previous, source)
			}
			seen[symbol] = field.FullName
		}
	}
	return nil
}

func validateNativeClientStructFields(structName string, fields []FieldPlan, symbols func(FieldPlan) []string) error {
	seen := make(map[string]string)
	for _, field := range fields {
		for _, symbol := range symbols(field) {
			if previous, exists := seen[symbol]; exists {
				return fmt.Errorf("native client cgo struct field %s.%s for %s collides with %s", structName, symbol, field.FullName, previous)
			}
			seen[symbol] = field.FullName
		}
	}
	return nil
}
