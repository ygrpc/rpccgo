package generator

import (
	"fmt"
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
	if err := validateNativeClientCGOSymbols(plan, service); err != nil {
		return err
	}
	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := newGeneratedFile(plugin, plan, file, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, lowerInitial(service.GoName)+"ServiceID")
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
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
	if nativeClientNeedsUnsafe(service) {
		g.P(`unsafe "unsafe"`)
	}
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	errorName := lowerInitial(service.GoName) + "NativeClientUnsupportedField"
	streamHandleErrorName := lowerInitial(service.GoName) + "NativeClientStreamHandleInvalid"
	g.P("var ", errorName, ` = errors.New("rpccgo: native unary client field codec is not implemented")`)
	g.P("var ", streamHandleErrorName, ` = errors.New("rpccgo: native client stream handle is invalid")`)
	g.P()

	for _, method := range service.Methods {
		var err error
		switch method.Streaming {
		case StreamingKindUnary:
			err = renderNativeUnaryClient(g, plan, service, method, errorName, servicePackage)
		case StreamingKindClientStreaming:
			err = renderNativeClientStreamingClient(g, plan, service, method, errorName, servicePackage)
		case StreamingKindServerStreaming:
			err = renderNativeServerStreamingClient(g, plan, service, method, errorName, servicePackage)
		case StreamingKindBidiStreaming:
			err = renderNativeBidiStreamingClient(g, plan, service, method, errorName, servicePackage)
		default:
			err = fmt.Errorf("method %s: unsupported native client cgo streaming kind %q", methodPlanName(method), method.Streaming)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func renderNativeUnaryClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) error {
	renderNativeClientRequestDecoder(g, nativeUnaryClientDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
	renderNativeUnaryResponseEncoder(g, service, method, unsupportedError)
	return renderNativeCExportWrappers(g, plan, service, method, servicePackage)
}

func renderNativeClientStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) error {
	renderNativeClientRequestDecoder(g, nativeClientStreamingDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
	renderNativeClientStreamingResponseEncoder(g, service, method, unsupportedError)
	return renderNativeCExportWrappers(g, plan, service, method, servicePackage)
}

func renderNativeServerStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) error {
	renderNativeClientRequestDecoder(g, nativeServerStreamingDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
	renderNativeServerStreamingResponseEncoder(g, service, method, unsupportedError)
	return renderNativeCExportWrappers(g, plan, service, method, servicePackage)
}

func renderNativeBidiStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) error {
	renderNativeClientRequestDecoder(g, nativeBidiStreamingDecoderName(service, method), method.Contract.Native.RequestFields, unsupportedError)
	renderNativeBidiStreamingResponseEncoder(g, service, method, unsupportedError)
	return renderNativeCExportWrappers(g, plan, service, method, servicePackage)
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
	g.P("err = ", servicePackage, runtimeNativeStreamOperationCallName(service, method, operation), "(", nativeClientStreamOperationArgs(args), ")")
}

func renderNativeClientStreamFacadeResultCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, result, operation, args string) {
	g.P(result, ", err = ", servicePackage, runtimeNativeStreamOperationCallName(service, method, operation), "(", nativeClientStreamOperationArgs(args), ")")
}

func renderNativeClientStreamResultCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, result, operation string) {
	if result == "" {
		g.P("err = ", servicePackage, runtimeNativeStreamOperationCallName(service, method, operation), "(ctx, rpcruntime.StreamHandle(handle))")
		return
	}
	renderNativeClientStreamFacadeResultCall(g, service, method, servicePackage, result, operation, "ctx")
}

func nativeClientStreamOperationArgs(args string) string {
	return strings.Replace(args, "ctx", "ctx, rpcruntime.StreamHandle(handle)", 1)
}

func runtimeNativeStreamOperationCallName(service ServicePlan, method MethodPlan, operation string) string {
	return service.GoName + "Native" + method.GoName + operation
}

func renderNativeClientRequestDecoder(g *protogen.GeneratedFile, name string, fields []FieldPlan, unsupportedError string) {
	returns := nativeGoRequestReturns(g, fields)
	g.P("func ", name, "(", nativeClientRequestParams(fields), ") (", returns, ") {")
	if nativeClientRequestCleanupError(fields) != "" {
		g.P("var decoded rpcruntime.NativeReleaseStack")
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
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P(name, " := ", field.GoName, " != 0")
	case NativeABIShapeBoolByteBufferWrapper:
		renderNativeClientRepeatedDecode(g, fields, field, name, "byte", "rpcruntime.RpcBoolRepeat", "rpcruntime.EmptyRpcBoolRepeat()", "rpcruntime.NewRpcBoolRepeatChecked")
	case NativeABIShapeRepeated:
		switch field.Kind {
		case FieldKindSignedInt32:
			renderNativeClientRepeatedDecode(g, fields, field, name, "int32", "rpcruntime.RpcRepeat[int32]", "rpcruntime.EmptyRpcRepeat[int32]()", "rpcruntime.NewRpcRepeatChecked")
		case FieldKindUnsignedInt32:
			renderNativeClientRepeatedDecode(g, fields, field, name, "uint32", "rpcruntime.RpcRepeat[uint32]", "rpcruntime.EmptyRpcRepeat[uint32]()", "rpcruntime.NewRpcRepeatChecked")
		case FieldKindSignedInt64:
			renderNativeClientRepeatedDecode(g, fields, field, name, "int64", "rpcruntime.RpcRepeat[int64]", "rpcruntime.EmptyRpcRepeat[int64]()", "rpcruntime.NewRpcRepeatChecked")
		case FieldKindUnsignedInt64:
			renderNativeClientRepeatedDecode(g, fields, field, name, "uint64", "rpcruntime.RpcRepeat[uint64]", "rpcruntime.EmptyRpcRepeat[uint64]()", "rpcruntime.NewRpcRepeatChecked")
		case FieldKindFloat:
			renderNativeClientRepeatedDecode(g, fields, field, name, "float32", "rpcruntime.RpcRepeat[float32]", "rpcruntime.EmptyRpcRepeat[float32]()", "rpcruntime.NewRpcRepeatChecked")
		case FieldKindDouble:
			renderNativeClientRepeatedDecode(g, fields, field, name, "float64", "rpcruntime.RpcRepeat[float64]", "rpcruntime.EmptyRpcRepeat[float64]()", "rpcruntime.NewRpcRepeatChecked")
		case FieldKindEnum:
			renderNativeClientRepeatedDecode(g, fields, field, name, "int32", "rpcruntime.RpcRepeat[int32]", "rpcruntime.EmptyRpcRepeat[int32]()", "rpcruntime.NewRpcRepeatChecked")
		default:
			g.P("return ", nativeClientDecodeErrorReturn(fields, unsupportedError))
		}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt32, FieldKindUnsignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(name, " := ", field.GoName)
		case FieldKindEnum:
			g.P(name, " := ", nativeGoEnumType(g, field), "(", field.GoName, ")")
		case FieldKindString:
			renderNativeClientStringDecode(g, fields, field, name)
		case FieldKindBytes, FieldKindMessage:
			renderNativeClientBytesDecode(g, fields, field, name)
		default:
			g.P("return ", nativeClientDecodeErrorReturn(fields, unsupportedError))
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

func nativeClientDecodeErrorReturn(fields []FieldPlan, errExpr string) string {
	if nativeClientRequestCleanupError(fields) == "" {
		return nativeClientZeroReturns(fields, errExpr)
	}
	return nativeClientZeroReturns(fields, "errors.Join("+errExpr+", decoded.Release())")
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

func renderNativeClientRepeatedDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, name, elemType, wrapperType, emptyExpr, ctor string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(", field.GoName, "Len); err != nil {")
	g.P(`return `, nativeClientDecodeErrorReturn(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P("var ", name, " *", wrapperType)
	g.P("if ", field.GoName, "Ptr == 0 || ", field.GoName, "Len == 0 {")
	g.P(name, " = ", emptyExpr)
	g.P("} else {")
	g.P("var decodeErr error")
	g.P(name, ", decodeErr = ", ctor, "((*", elemType, ")(unsafe.Pointer(", field.GoName, "Ptr)), ", field.GoName, "Len, ", field.GoName, "Ownership > 0)")
	g.P("if decodeErr != nil {")
	g.P(`return `, nativeClientDecodeErrorReturn(fields, `fmt.Errorf("`+field.FullName+`: %w", decodeErr)`))
	g.P("}")
	g.P("}")
}

func renderNativeClientStringDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, name string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(", field.GoName, "Len); err != nil {")
	g.P(`return `, nativeClientDecodeErrorReturn(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P("var ", name, " *rpcruntime.RpcString")
	g.P("if ", field.GoName, "Ptr == 0 || ", field.GoName, "Len == 0 {")
	g.P(name, " = rpcruntime.EmptyRpcString()")
	g.P("} else {")
	g.P("var decodeErr error")
	g.P(name, ", decodeErr = rpcruntime.NewRpcStringChecked((*byte)(unsafe.Pointer(", field.GoName, "Ptr)), ", field.GoName, "Len, ", field.GoName, "Ownership > 0)")
	g.P("if decodeErr != nil {")
	g.P(`return `, nativeClientDecodeErrorReturn(fields, `fmt.Errorf("`+field.FullName+`: %w", decodeErr)`))
	g.P("}")
	g.P("}")
}

func renderNativeClientBytesDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, name string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(", field.GoName, "Len); err != nil {")
	g.P(`return `, nativeClientDecodeErrorReturn(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P("var ", name, " *rpcruntime.RpcBytes")
	g.P("if ", field.GoName, "Ptr == 0 || ", field.GoName, "Len == 0 {")
	g.P(name, " = rpcruntime.EmptyRpcBytes()")
	g.P("} else {")
	g.P("var decodeErr error")
	g.P(name, ", decodeErr = rpcruntime.NewRpcBytesChecked((*byte)(unsafe.Pointer(", field.GoName, "Ptr)), ", field.GoName, "Len, ", field.GoName, "Ownership > 0)")
	g.P("if decodeErr != nil {")
	g.P(`return `, nativeClientDecodeErrorReturn(fields, `fmt.Errorf("`+field.FullName+`: %w", decodeErr)`))
	g.P("}")
	g.P("}")
}

func renderNativeUnaryResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	renderNativeClientOutputValidator(g, nativeUnaryClientOutputValidatorName(service, method), method.Contract.Native.ResponseFields)
	renderNativeClientResponseEncoder(g, nativeUnaryClientEncoderName(service, method), method.Contract.Native.ResponseFields, unsupportedError)
}

func renderNativeUnaryClientCallBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, requestArgs, responseArgs string) {
	if responseArgs != "" {
		g.P("if err := ", nativeUnaryClientOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	if requestNames == "" {
		g.P("if err := ", nativeUnaryClientDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeUnaryClientDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	if responseNames == "" {
		g.P("err := ", servicePackage, "Invoke", service.GoName, "Native", method.GoName, "(", ctx, nativeGoCallSuffix(requestNames), ")")
	} else {
		g.P(responseNames, ", err := ", servicePackage, "Invoke", service.GoName, "Native", method.GoName, "(", ctx, nativeGoCallSuffix(requestNames), ")")
	}
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeUnaryClientEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeClientStreamingStartBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, outHandle string) {
	g.P("handle, err := ", servicePackage, runtimeNativeStreamOperationCallName(service, method, "Start"), "(", ctx, ")")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*", outHandle, " = C.int32_t(int32(handle))")
	g.P("return 0")
}

func renderNativeClientStreamingSendBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, handle, requestArgs string) {
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	g.P("handle := int32(", handle, ")")
	g.P("var err error")
	if requestNames == "" {
		g.P("if err := ", nativeClientStreamingDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeClientStreamingDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Send", ctx+nativeGoCallSuffix(requestNames))
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeClientStreamingFinishBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, handle, responseArgs string) {
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	g.P("handle := int32(", handle, ")")
	if responseArgs != "" {
		g.P("if err := ", nativeClientStreamingOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("var err error")
	for _, decl := range nativeGoResponseResultVarDecls(g, method.Contract.Native.ResponseFields) {
		g.P(decl)
	}
	renderNativeClientStreamResultCall(g, service, method, servicePackage, responseNames, "Finish")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeClientStreamingEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeServerStreamingStartBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, outHandle, requestArgs string) {
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	g.P("var err error")
	if requestNames == "" {
		g.P("if err := ", nativeServerStreamingDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeServerStreamingDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("handle, err := ", servicePackage, runtimeNativeStreamOperationCallName(service, method, "Start"), "(", ctx, nativeGoCallSuffix(requestNames), ")")
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*", outHandle, " = C.int32_t(int32(handle))")
	g.P("return 0")
}

func renderNativeServerStreamingRecvBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, handle, responseArgs string) {
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	g.P("handle := int32(", handle, ")")
	if responseArgs != "" {
		g.P("if err := ", nativeServerStreamingOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("var err error")
	for _, decl := range nativeGoResponseResultVarDecls(g, method.Contract.Native.ResponseFields) {
		g.P(decl)
	}
	renderNativeClientStreamResultCall(g, service, method, servicePackage, responseNames, "Recv")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeServerStreamingEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeStreamNoResultBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, handle, operation string) {
	g.P("handle := int32(", handle, ")")
	g.P("var err error")
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, operation, ctx)
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeBidiStreamingStartBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, outHandle string) {
	g.P("handle, err := ", servicePackage, runtimeNativeStreamOperationCallName(service, method, "Start"), "(", ctx, ")")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*", outHandle, " = C.int32_t(int32(handle))")
	g.P("return 0")
}

func renderNativeBidiStreamingSendBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, handle, requestArgs string) {
	requestNames := nativeClientRequestValueNames(method.Contract.Native.RequestFields)
	g.P("handle := int32(", handle, ")")
	g.P("var err error")
	if requestNames == "" {
		g.P("if err := ", nativeBidiStreamingDecoderName(service, method), "(", requestArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	} else {
		g.P(requestNames, ", err := ", nativeBidiStreamingDecoderName(service, method), "(", requestArgs, ")")
		g.P("if err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	renderNativeClientStreamFacadeCall(g, service, method, servicePackage, "Send", ctx+nativeGoCallSuffix(requestNames))
	g.P("if cleanupErr := errors.Join(", nativeClientRequestCleanupError(method.Contract.Native.RequestFields), "); cleanupErr != nil {")
	g.P("err = errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeBidiStreamingRecvBody(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, ctx, handle, responseArgs string) {
	responseNames := nativeClientResponseValueNames(method.Contract.Native.ResponseFields)
	g.P("handle := int32(", handle, ")")
	if responseArgs != "" {
		g.P("if err := ", nativeBidiStreamingOutputValidatorName(service, method), "(", responseArgs, "); err != nil {")
		g.P("return C.int32_t(rpcruntime.StoreError(err))")
		g.P("}")
	}
	g.P("var err error")
	for _, decl := range nativeGoResponseResultVarDecls(g, method.Contract.Native.ResponseFields) {
		g.P(decl)
	}
	renderNativeClientStreamResultCall(g, service, method, servicePackage, responseNames, "Recv")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := ", nativeBidiStreamingEncoderName(service, method), "(", nativeClientEncoderCallArgs(responseNames), responseArgs, "); err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
}

func renderNativeCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) error {
	methodABI, err := nativeCOperationABIsByOperation(plan, service, method)
	if err != nil {
		return err
	}
	switch method.Streaming {
	case StreamingKindUnary:
		renderNativeUnaryCExportWrapper(g, service, method, servicePackage, methodABI[NativeCOperationUnary])
	case StreamingKindClientStreaming:
		renderNativeClientStreamingCExportWrappers(g, service, method, servicePackage, methodABI)
	case StreamingKindServerStreaming:
		renderNativeServerStreamingCExportWrappers(g, service, method, servicePackage, methodABI)
	case StreamingKindBidiStreaming:
		renderNativeBidiStreamingCExportWrappers(g, service, method, servicePackage, methodABI)
	}
	return nil
}

func renderNativeUnaryCExportWrapper(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string, unaryABI COperationABI) {
	exportName := unaryABI.Symbol
	renderCGOExportDoc(g, exportName, "invokes the native unary client entrypoint for "+method.FullName+".")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(", nativeCExportParams(unaryABI.Params), ") ", unaryABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, unaryABI.Params)
	renderNativeUnaryClientCallBody(g, service, method, servicePackage, "ctx", nativeCExportGoArgs(service, method), nativeCExportOutputGoArgs(service, method))
	g.P("}")
	g.P()
}

func renderNativeClientStreamingCExportWrappers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string, methodABI map[NativeCOperation]COperationABI) {
	startABI := methodABI[NativeCOperationStart]
	renderCGOExportDoc(g, startABI.Symbol, "starts the native client-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", startABI.Symbol)
	g.P("func ", startABI.Symbol, "(", nativeCExportParams(startABI.Params), ") ", startABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportHandleValidation(g, "stream")
	renderNativeClientStreamingStartBody(g, service, method, servicePackage, "ctx", "stream")
	g.P("}")
	g.P()

	sendABI := methodABI[NativeCOperationSend]
	renderCGOExportDoc(g, sendABI.Symbol, "sends native request values to the client-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", sendABI.Symbol)
	g.P("func ", sendABI.Symbol, "(", nativeCExportParams(sendABI.Params), ") ", sendABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeClientStreamingSendBody(g, service, method, servicePackage, "ctx", "stream", nativeCExportGoArgs(service, method))
	g.P("}")
	g.P()

	finishABI := methodABI[NativeCOperationFinish]
	renderCGOExportDoc(g, finishABI.Symbol, "finishes the native client-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", finishABI.Symbol)
	g.P("func ", finishABI.Symbol, "(", nativeCExportParams(finishABI.Params), ") ", finishABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, finishABI.Params)
	renderNativeClientStreamingFinishBody(g, service, method, servicePackage, "ctx", "stream", nativeCExportOutputGoArgs(service, method))
	g.P("}")
	g.P()

	cancelABI := methodABI[NativeCOperationCancel]
	renderCGOExportDoc(g, cancelABI.Symbol, "cancels the native client-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", cancelABI.Symbol)
	g.P("func ", cancelABI.Symbol, "(", nativeCExportParams(cancelABI.Params), ") ", cancelABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeStreamNoResultBody(g, service, method, servicePackage, "ctx", "stream", "Cancel")
	g.P("}")
	g.P()
}

func renderNativeServerStreamingCExportWrappers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string, methodABI map[NativeCOperation]COperationABI) {
	startABI := methodABI[NativeCOperationStart]
	renderCGOExportDoc(g, startABI.Symbol, "starts the native server-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", startABI.Symbol)
	g.P("func ", startABI.Symbol, "(", nativeCExportParams(startABI.Params), ") ", startABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportHandleValidation(g, "stream")
	renderNativeServerStreamingStartBody(g, service, method, servicePackage, "ctx", "stream", nativeCExportGoArgs(service, method))
	g.P("}")
	g.P()

	recvABI := methodABI[NativeCOperationRecv]
	renderCGOExportDoc(g, recvABI.Symbol, "receives native response values from the server-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", recvABI.Symbol)
	g.P("func ", recvABI.Symbol, "(", nativeCExportParams(recvABI.Params), ") ", recvABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, recvABI.Params)
	renderNativeServerStreamingRecvBody(g, service, method, servicePackage, "ctx", "stream", nativeCExportOutputGoArgs(service, method))
	g.P("}")
	g.P()

	finishABI := methodABI[NativeCOperationFinish]
	renderCGOExportDoc(g, finishABI.Symbol, "finishes the native server-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", finishABI.Symbol)
	g.P("func ", finishABI.Symbol, "(", nativeCExportParams(finishABI.Params), ") ", finishABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeStreamNoResultBody(g, service, method, servicePackage, "ctx", "stream", "Finish")
	g.P("}")
	g.P()

	cancelABI := methodABI[NativeCOperationCancel]
	renderCGOExportDoc(g, cancelABI.Symbol, "cancels the native server-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", cancelABI.Symbol)
	g.P("func ", cancelABI.Symbol, "(", nativeCExportParams(cancelABI.Params), ") ", cancelABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeStreamNoResultBody(g, service, method, servicePackage, "ctx", "stream", "Cancel")
	g.P("}")
	g.P()
}

func renderNativeBidiStreamingCExportWrappers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string, methodABI map[NativeCOperation]COperationABI) {
	startABI := methodABI[NativeCOperationStart]
	renderCGOExportDoc(g, startABI.Symbol, "starts the native bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", startABI.Symbol)
	g.P("func ", startABI.Symbol, "(", nativeCExportParams(startABI.Params), ") ", startABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportHandleValidation(g, "stream")
	renderNativeBidiStreamingStartBody(g, service, method, servicePackage, "ctx", "stream")
	g.P("}")
	g.P()

	sendABI := methodABI[NativeCOperationSend]
	renderCGOExportDoc(g, sendABI.Symbol, "sends native request values to the bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", sendABI.Symbol)
	g.P("func ", sendABI.Symbol, "(", nativeCExportParams(sendABI.Params), ") ", sendABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeBidiStreamingSendBody(g, service, method, servicePackage, "ctx", "stream", nativeCExportGoArgs(service, method))
	g.P("}")
	g.P()

	recvABI := methodABI[NativeCOperationRecv]
	renderCGOExportDoc(g, recvABI.Symbol, "receives native response values from the bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", recvABI.Symbol)
	g.P("func ", recvABI.Symbol, "(", nativeCExportParams(recvABI.Params), ") ", recvABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeCExportOutputValidation(g, method.Contract.Native.ResponseFields, recvABI.Params)
	renderNativeBidiStreamingRecvBody(g, service, method, servicePackage, "ctx", "stream", nativeCExportOutputGoArgs(service, method))
	g.P("}")
	g.P()

	closeSendABI := methodABI[NativeCOperationCloseSend]
	renderCGOExportDoc(g, closeSendABI.Symbol, "closes the native bidi-streaming client send side for "+method.FullName+".")
	g.P("//export ", closeSendABI.Symbol)
	g.P("func ", closeSendABI.Symbol, "(", nativeCExportParams(closeSendABI.Params), ") ", closeSendABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeStreamNoResultBody(g, service, method, servicePackage, "ctx", "stream", "CloseSend")
	g.P("}")
	g.P()

	finishABI := methodABI[NativeCOperationFinish]
	renderCGOExportDoc(g, finishABI.Symbol, "finishes the native bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", finishABI.Symbol)
	g.P("func ", finishABI.Symbol, "(", nativeCExportParams(finishABI.Params), ") ", finishABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeStreamNoResultBody(g, service, method, servicePackage, "ctx", "stream", "Finish")
	g.P("}")
	g.P()

	cancelABI := methodABI[NativeCOperationCancel]
	renderCGOExportDoc(g, cancelABI.Symbol, "cancels the native bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", cancelABI.Symbol)
	g.P("func ", cancelABI.Symbol, "(", nativeCExportParams(cancelABI.Params), ") ", cancelABI.Return.CGoType, " {")
	g.P("ctx := context.Background()")
	renderNativeStreamNoResultBody(g, service, method, servicePackage, "ctx", "stream", "Cancel")
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
	return cgoServiceExportName("native", plan, service, method.GoName, operation)
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
		case FieldKindSignedInt32, FieldKindUnsignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt64, FieldKindFloat, FieldKindDouble, FieldKindEnum:
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

func packageCGOImportPath(pkg PackagePlan) string {
	return path.Join(string(pkg.GoImportPath), cgoDirForPackagePlan(pkg))
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
		if otherService.FullName != service.FullName && otherService.HasArtifact(GeneratedArtifactKindCGONativeClient) {
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
			add(nativeUnaryClientDecoderName(service, method), method.FullName+" unary request decoder")
			add(nativeUnaryClientEncoderName(service, method), method.FullName+" unary response encoder")
			add(nativeUnaryClientOutputValidatorName(service, method), method.FullName+" unary output validator")
		case StreamingKindClientStreaming:
			add(nativeClientStreamingDecoderName(service, method), method.FullName+" client stream request decoder")
			add(nativeClientStreamingEncoderName(service, method), method.FullName+" client stream response encoder")
			add(nativeClientStreamingOutputValidatorName(service, method), method.FullName+" client stream output validator")
		case StreamingKindServerStreaming:
			add(nativeServerStreamingDecoderName(service, method), method.FullName+" server stream request decoder")
			add(nativeServerStreamingEncoderName(service, method), method.FullName+" server stream response encoder")
			add(nativeServerStreamingOutputValidatorName(service, method), method.FullName+" server stream output validator")
		case StreamingKindBidiStreaming:
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
