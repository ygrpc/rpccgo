package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeServerCGOSymbols(plan, service); err != nil {
		return err
	}
	nativeCABIPlan, err := BuildNativeCABIPlan(plan, service)
	if err != nil {
		return err
	}

	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := newGeneratedFile(plugin, plan, file, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"NativeServer")
	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	runtimeMethods = qualifyRuntimeAdapterMethods(runtimeMethods, servicePackage)

	g.P("package main")
	g.P()
	renderCGONativeServerPreamble(g, service, nativeCABIPlan)
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	g.P(`io "io"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(`sync "sync"`)
	g.P(`unsafe "unsafe"`)
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
	g.P(lowerInitial(service.GoName), "CGONativeServerAdapterMu sync.Mutex")
	g.P(lowerInitial(service.GoName), "CGONativeServerAdapter = &", lowerInitial(service.GoName), "CGONativeAdapter{}")
	g.P(")")
	g.P()

	adapterName := lowerInitial(service.GoName) + "CGONativeAdapter"
	renderCGONativeServerAdapter(g, service, nativeCABIPlan, runtimeMethods, adapterName, errorNames, servicePackage)
	renderCGONativeServerRegistration(g, service, nativeCABIPlan, errorNames, servicePackage)
	renderCGONativeServerErrorStoreExport(g, service)
	return nil
}

func qualifyRuntimeAdapterMethods(methods []runtimeAdapterMethod, servicePackage string) []runtimeAdapterMethod {
	qualified := make([]runtimeAdapterMethod, len(methods))
	copy(qualified, methods)
	for i := range qualified {
		if !qualified[i].Streaming {
			continue
		}
		rawSessionName := qualified[i].SessionName
		qualified[i].SessionName = servicePackage + rawSessionName
		qualified[i].AdapterResult = strings.ReplaceAll(qualified[i].AdapterResult, rawSessionName, qualified[i].SessionName)
	}
	return qualified
}

func renderCGONativeServerPreamble(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan) {
	methodABI := nativeCABIPlanByMethod(abiPlan)

	g.P("/*")
	g.P("#include <stdint.h>")
	g.P()
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			unaryABI := nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationUnary)
			g.P("typedef ", unaryABI.Return.CType, " (*", unaryABI.TypeName, ")(", nativeCABIParamList(unaryABI.Params), ");")
			g.P()
		case StreamingKindClientStreaming:
			renderCGONativeServerCallbackTypedefs(g, methodABI[method.FullName], NativeCOperationStart, NativeCOperationSend, NativeCOperationFinish, NativeCOperationCancel)
			g.P()
		case StreamingKindServerStreaming:
			renderCGONativeServerCallbackTypedefs(g, methodABI[method.FullName], NativeCOperationStart, NativeCOperationRecv, NativeCOperationFinish, NativeCOperationCancel)
			g.P()
		case StreamingKindBidiStreaming:
			renderCGONativeServerCallbackTypedefs(g, methodABI[method.FullName], NativeCOperationStart, NativeCOperationSend, NativeCOperationRecv, NativeCOperationCloseSend, NativeCOperationFinish, NativeCOperationCancel)
			g.P()
		}
	}
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			unaryABI := nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationUnary)
			g.P("static inline ", unaryABI.Return.CType, " ", nativeCGOServerTrampolineName(service, method), "(", unaryABI.TypeName, " callback", nativeCGOServerTypedParamSuffix(nativeCABIParamListValues(unaryABI.Params)), ") {")
			g.P("\treturn callback(", nativeCABIArgNames(unaryABI.Params), ");")
			g.P("}")
			g.P()
		case StreamingKindClientStreaming:
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerClientStreamStartTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationStart))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerClientStreamSendTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationSend))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerClientStreamFinishTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationFinish))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerClientStreamCancelTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationCancel))
		case StreamingKindServerStreaming:
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerServerStreamStartTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationStart))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerServerStreamRecvTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationRecv))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerServerStreamFinishTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationFinish))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerServerStreamCancelTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationCancel))
		case StreamingKindBidiStreaming:
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerBidiStreamStartTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationStart))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerBidiStreamSendTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationSend))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerBidiStreamRecvTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationRecv))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationCloseSend))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerBidiStreamFinishTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationFinish))
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerBidiStreamCancelTrampolineName(service, method), nativeCABIPlanOperation(methodABI[method.FullName], NativeCOperationCancel))
		}
	}
	g.P("*/")
}

func renderCGONativeServerCallbackTypedefs(g *protogen.GeneratedFile, method MethodNativeCABIPlan, operations ...NativeCOperation) {
	for _, operation := range operations {
		abi := nativeCABIPlanOperation(method, operation)
		g.P("typedef ", abi.Return.CType, " (*", abi.TypeName, ")(", nativeCABIParamList(abi.Params), ");")
	}
}

func renderCGONativeServerCallbackTrampoline(g *protogen.GeneratedFile, name string, abi COperationABI) {
	g.P("static inline ", abi.Return.CType, " ", name, "(", abi.TypeName, " callback", nativeCGOServerTypedParamSuffix(nativeCABIParamListValues(abi.Params)), ") {")
	g.P("\treturn callback(", nativeCABIArgNames(abi.Params), ");")
	g.P("}")
	g.P()
}

func nativeCABIPlanByMethod(plan NativeCABIPlan) map[string]MethodNativeCABIPlan {
	byMethod := make(map[string]MethodNativeCABIPlan, len(plan.Methods))
	for _, method := range plan.Methods {
		byMethod[method.MethodFullName] = method
	}
	return byMethod
}

func nativeCABIPlanOperation(method MethodNativeCABIPlan, operation NativeCOperation) COperationABI {
	for _, current := range method.Operations {
		if current.Operation == operation {
			return current
		}
	}
	return COperationABI{}
}

func nativeCABIParamList(params []CABISlot) string {
	return strings.Join(nativeCABIParamListValues(params), ", ")
}

func nativeCABIParamListValues(params []CABISlot) []string {
	values := make([]string, 0, len(params))
	for _, param := range params {
		values = append(values, nativeCABIParamDecl(param))
	}
	return values
}

func nativeCABIParamDecl(param CABISlot) string {
	if ctype, ok := strings.CutSuffix(param.CType, "*"); ok {
		return ctype + " *" + param.Name
	}
	return param.CType + " " + param.Name
}

func nativeCABIArgNames(params []CABISlot) string {
	args := make([]string, 0, len(params))
	for _, param := range params {
		args = append(args, param.Name)
	}
	return strings.Join(args, ", ")
}

func nativeCABIRegisterParamList(params []CABISlot) string {
	values := make([]string, 0, len(params))
	for _, param := range params {
		values = append(values, param.Name+" "+param.CGoType)
	}
	return strings.Join(values, ", ")
}

func nativeCGOServerTypedParamSuffix(params []string) string {
	if len(params) == 0 {
		return ""
	}
	return ", " + strings.Join(params, ", ")
}

func nativeCGOServerArgSuffix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func nativeCGOServerArgList(first []string, rest ...string) string {
	args := append([]string{}, first...)
	args = append(args, rest...)
	return strings.Join(args, ", ")
}

func nativeCGOServerCParamName(name string, output bool) string {
	if output {
		return "out" + name
	}
	return name
}

func nativeCGOServerCArgName(name string, output bool) string {
	if output {
		return "out" + name
	}
	return name
}

func nativeCGOServerGoInputCallArgs(fields []FieldPlan) []string {
	args := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		args = append(args, nativeCGOServerGoFieldArgs(field, false)...)
	}
	return args
}

func nativeCGOServerGoOutputCallArgs(fields []FieldPlan) []string {
	args := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		for _, arg := range nativeCGOServerGoOutputValueFieldArgs(field) {
			args = append(args, "&"+arg)
		}
	}
	return args
}

func nativeCGOServerGoOutputValueArgs(fields []FieldPlan) []string {
	args := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		args = append(args, nativeCGOServerGoOutputValueFieldArgs(field)...)
	}
	return args
}

func nativeCGOServerGoOutputValueFieldArgs(field FieldPlan) []string {
	fieldName := "out" + field.GoName
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return []string{fieldName + "Value"}
	case NativeABIShapeRepeated, NativeABIShapeBoolByteBufferWrapper:
		return []string{fieldName + "Ptr", fieldName + "Len", fieldName + "Ownership"}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			return []string{fieldName + "Ptr", fieldName + "Len", fieldName + "Ownership"}
		default:
			return []string{fieldName + "Value"}
		}
	default:
		return []string{fieldName + "Value"}
	}
}

func nativeCGOServerGoFieldArgs(field FieldPlan, output bool) []string {
	name := lowerInitial(field.GoName)
	prefix := ""
	if output {
		prefix = "&"
	}
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return []string{prefix + name + "Value"}
	case NativeABIShapeRepeated, NativeABIShapeBoolByteBufferWrapper:
		return []string{prefix + name + "Ptr", prefix + name + "Len", prefix + name + "Ownership"}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			return []string{prefix + name + "Ptr", prefix + name + "Len", prefix + name + "Ownership"}
		default:
			return []string{prefix + name + "Value"}
		}
	default:
		return []string{prefix + name + "Value"}
	}
}

func nativeCGOServerGoCallSuffix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func nativeCGOServerGoUnaryCallArgs(requestFields, responseFields []FieldPlan) string {
	args := nativeCGOServerGoInputCallArgs(requestFields)
	args = append(args, nativeCGOServerGoOutputCallArgs(responseFields)...)
	return strings.Join(args, ", ")
}

func nativeCGOServerGoABIArgs(params []CABISlot, handleArg string) []string {
	args := make([]string, 0, len(params))
	for _, param := range params {
		args = append(args, nativeCGOServerGoABIArg(param, handleArg))
	}
	return args
}

func nativeCGOServerGoABIArg(param CABISlot, handleArg string) string {
	if param.Role == CABISlotRoleHandle {
		return handleArg
	}
	if param.Source == nil {
		return param.Name
	}
	fieldName := param.Source.GoName
	output := strings.HasPrefix(param.Name, "out"+fieldName)
	base := fieldName
	localBase := lowerInitial(fieldName)
	if output {
		base = "out" + fieldName
		localBase = base
	}
	suffix := strings.TrimPrefix(param.Name, base)
	if suffix == "" {
		suffix = "Value"
	}
	arg := localBase + suffix
	if output {
		return "&" + arg
	}
	return arg
}

func nativeCGOServerGoABICallSuffix(params []CABISlot, handleArg string) string {
	return nativeCGOServerGoCallSuffix(nativeCGOServerGoABIArgs(params, handleArg))
}

func nativeCGOServerGoABIArgList(params []CABISlot, handleArg string) string {
	return strings.Join(nativeCGOServerGoABIArgs(params, handleArg), ", ")
}

func nativeCGOServerOperationABI(abiPlan NativeCABIPlan, method MethodPlan, operation NativeCOperation) COperationABI {
	methodABI := nativeCABIPlanByMethod(abiPlan)
	return nativeCABIPlanOperation(methodABI[method.FullName], operation)
}

func nativeCGOServerGoFieldTypes(fields []FieldPlan) []string {
	types := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		types = append(types, nativeCGOServerGoFieldType(field)...)
	}
	return types
}

func nativeCGOServerGoFieldType(field FieldPlan) []string {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return []string{"C.int8_t"}
	case NativeABIShapeRepeated, NativeABIShapeBoolByteBufferWrapper:
		return []string{"C.uintptr_t", "C.int32_t", "C.int32_t"}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindEnum:
			return []string{"C.int32_t"}
		case FieldKindUnsignedInt32:
			return []string{"C.uint32_t"}
		case FieldKindSignedInt64:
			return []string{"C.int64_t"}
		case FieldKindUnsignedInt64:
			return []string{"C.uint64_t"}
		case FieldKindFloat:
			return []string{"C.float"}
		case FieldKindDouble:
			return []string{"C.double"}
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			return []string{"C.uintptr_t", "C.int32_t", "C.int32_t"}
		default:
			return []string{"C.uintptr_t"}
		}
	default:
		return []string{"C.uintptr_t"}
	}
}

func nativeCGOServerRequestEncoderReturns(fields []FieldPlan) string {
	returns := nativeCGOServerGoFieldTypes(fields)
	returns = append(returns, "func()", "error")
	return "(" + strings.Join(returns, ", ") + ")"
}

func nativeCGOServerRequestEncoderResultArgs(fields []FieldPlan) []string {
	args := nativeCGOServerGoInputCallArgs(fields)
	args = append(args, "cleanup", "nil")
	return args
}

func nativeCGOServerRequestEncoderErrorReturn(fields []FieldPlan, errExpr string) string {
	returns := make([]string, 0, len(fields)*3+2)
	for range nativeCGOServerGoFieldTypes(fields) {
		returns = append(returns, "0")
	}
	returns = append(returns, "func() {}", errExpr)
	return strings.Join(returns, ", ")
}

func nativeCGOServerRequestEncoderAssignArgs(fields []FieldPlan) string {
	args := nativeCGOServerGoInputCallArgs(fields)
	args = append(args, "cleanup", "err")
	return strings.Join(args, ", ")
}

func renderCGONativeServerResponseLocals(g *protogen.GeneratedFile, fields []FieldPlan) {
	for _, field := range fields {
		types := nativeCGOServerGoFieldType(field)
		for i, name := range nativeCGOServerGoOutputValueFieldArgs(field) {
			g.P("var ", name, " ", types[i])
		}
	}
}

func nativeCGOServerFlatValueParams(fields []FieldPlan) string {
	args := nativeCGOServerGoInputCallArgs(fields)
	types := nativeCGOServerGoFieldTypes(fields)
	params := make([]string, 0, len(args))
	for i, arg := range args {
		params = append(params, arg+" "+types[i])
	}
	return strings.Join(params, ", ")
}

func nativeCGOServerFlatPointerParams(fields []FieldPlan) string {
	args := nativeCGOServerGoOutputValueArgs(fields)
	types := nativeCGOServerGoFieldTypes(fields)
	params := make([]string, 0, len(args))
	for i, arg := range args {
		params = append(params, arg+" *"+types[i])
	}
	return strings.Join(params, ", ")
}

func nativeCGOServerFlatOutputValueArgs(fields []FieldPlan) string {
	return strings.Join(nativeCGOServerGoOutputValueArgs(fields), ", ")
}

func nativeCGOServerFlatOutputPointerArgs(fields []FieldPlan) string {
	return strings.Join(nativeCGOServerGoOutputCallArgs(fields), ", ")
}

func nativeCGOServerPrefixedParams(prefix string, params string) string {
	if params == "" {
		return ""
	}
	return prefix + params
}

func nativeCGOServerCInputArgNames(fields []FieldPlan) []string {
	args := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		args = append(args, nativeCGOServerCFieldArgNames(field, false)...)
	}
	return args
}

func nativeCGOServerCOutputArgNames(fields []FieldPlan) []string {
	args := make([]string, 0, len(fields)*3)
	for _, field := range fields {
		args = append(args, nativeCGOServerCFieldArgNames(field, true)...)
	}
	return args
}

func nativeCGOServerCallbackArgNames(requestFields, responseFields []FieldPlan) string {
	return nativeCGOServerArgList(nativeCGOServerCInputArgNames(requestFields), nativeCGOServerCOutputArgNames(responseFields)...)
}

func nativeCGOServerCFieldArgNames(field FieldPlan, output bool) []string {
	prefix := ""
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return []string{prefix + nativeCGOServerCArgName(field.GoName, output)}
	case NativeABIShapeRepeated, NativeABIShapeBoolByteBufferWrapper:
		return []string{prefix + nativeCGOServerCArgName(field.GoName+"Ptr", output), prefix + nativeCGOServerCArgName(field.GoName+"Len", output), prefix + nativeCGOServerCArgName(field.GoName+"Ownership", output)}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			return []string{prefix + nativeCGOServerCArgName(field.GoName+"Ptr", output), prefix + nativeCGOServerCArgName(field.GoName+"Len", output), prefix + nativeCGOServerCArgName(field.GoName+"Ownership", output)}
		default:
			return []string{prefix + nativeCGOServerCArgName(field.GoName, output)}
		}
	default:
		return []string{prefix + nativeCGOServerCArgName(field.GoName, output)}
	}
}

func renderCGONativeServerAdapter(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, methods []runtimeAdapterMethod, adapterName string, errorNames nativeServerCGOErrorNames, servicePackage string) {
	g.P("type ", adapterName, " struct {")
	renderCGONativeServerAdapterFields(g, service, abiPlan)
	g.P("}")
	g.P()

	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
		if !ok {
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod, errorNames)
			continue
		}
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGONativeServerUnaryAdapter(g, service, abiPlan, adapterName, method, errorNames)
		case StreamingKindClientStreaming:
			renderCGONativeServerClientStreamAdapter(g, service, abiPlan, adapterName, method, errorNames, servicePackage)
		case StreamingKindServerStreaming:
			renderCGONativeServerServerStreamAdapter(g, service, abiPlan, adapterName, method, errorNames, servicePackage)
		case StreamingKindBidiStreaming:
			renderCGONativeServerBidiStreamAdapter(g, service, abiPlan, adapterName, method, errorNames, servicePackage)
		default:
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod, errorNames)
		}
	}
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindClientStreaming:
			renderCGONativeServerClientStreamServerMethod(g, service, adapterName, method, servicePackage)
		case StreamingKindServerStreaming:
			renderCGONativeServerServerStreamServerMethod(g, service, adapterName, method, servicePackage)
		case StreamingKindBidiStreaming:
			renderCGONativeServerBidiStreamServerMethod(g, service, adapterName, method, servicePackage)
		}
	}

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGONativeServerRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerResponseCleanup(g, service, method)
		case StreamingKindClientStreaming:
			renderCGONativeServerClientStreamRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerClientStreamResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerClientStreamResponseCleanup(g, service, method)
		case StreamingKindServerStreaming:
			renderCGONativeServerServerStreamRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerServerStreamResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerServerStreamResponseCleanup(g, service, method)
		case StreamingKindBidiStreaming:
			renderCGONativeServerBidiStreamRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerBidiStreamResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerBidiStreamResponseCleanup(g, service, method)
		}
	}
	renderCGONativeErrorIDHelper(g, service)
}

func renderCGONativeServerAdapterFields(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan) {
	methodABI := nativeCABIPlanByMethod(abiPlan)
	callbackTypeName := func(method MethodPlan, operation NativeCOperation) string {
		return nativeCABIPlanOperation(methodABI[method.FullName], operation).TypeName
	}
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(method.GoName, "Callback C.", callbackTypeName(method, NativeCOperationUnary))
		case StreamingKindClientStreaming:
			g.P(method.GoName, "Start C.", callbackTypeName(method, NativeCOperationStart))
			g.P(method.GoName, "Send C.", callbackTypeName(method, NativeCOperationSend))
			g.P(method.GoName, "Finish C.", callbackTypeName(method, NativeCOperationFinish))
			g.P(method.GoName, "Cancel C.", callbackTypeName(method, NativeCOperationCancel))
		case StreamingKindServerStreaming:
			g.P(method.GoName, "Start C.", callbackTypeName(method, NativeCOperationStart))
			g.P(method.GoName, "Recv C.", callbackTypeName(method, NativeCOperationRecv))
			g.P(method.GoName, "Finish C.", callbackTypeName(method, NativeCOperationFinish))
			g.P(method.GoName, "Cancel C.", callbackTypeName(method, NativeCOperationCancel))
		case StreamingKindBidiStreaming:
			g.P(method.GoName, "Start C.", callbackTypeName(method, NativeCOperationStart))
			g.P(method.GoName, "Send C.", callbackTypeName(method, NativeCOperationSend))
			g.P(method.GoName, "Recv C.", callbackTypeName(method, NativeCOperationRecv))
			g.P(method.GoName, "CloseSend C.", callbackTypeName(method, NativeCOperationCloseSend))
			g.P(method.GoName, "Finish C.", callbackTypeName(method, NativeCOperationFinish))
			g.P(method.GoName, "Cancel C.", callbackTypeName(method, NativeCOperationCancel))
		}
	}
}

func renderCGONativeServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context", nativeGoRequestParams(g, method.Contract.Native.RequestFields), ") (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	g.P("if a == nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, errorNames.CallbacksNil))
	g.P("}")
	g.P("callback := a.", method.GoName, "Callback")
	g.P("if callback == nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, errorNames.UnaryCallbackMissing))
	g.P("}")
	g.P(nativeCGOServerRequestEncoderAssignArgs(method.Contract.Native.RequestFields), " := ", nativeCGOServerRequestEncoderName(service, method), "(", nativeCGOServerRequestEncoderCallArgs(method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	g.P("defer cleanup()")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	unaryABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationUnary)
	g.P("errID := int32(C.", nativeCGOServerTrampolineName(service, method), "(callback, ", nativeCGOServerGoABIArgList(unaryABI.Params, ""), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(callbackErr, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "callbackErr"))
	g.P("}")
	responseNames := nativeGoResponseResultNames(method.Contract.Native.ResponseFields)
	if responseNames == "" {
		g.P("err = ", nativeCGOServerResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	} else {
		g.P(responseNames, ", err := ", nativeCGOServerResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	}
	g.P("cleanupErr := ", nativeCGOServerResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(err, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "cleanupErr"))
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	if responseNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", responseNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil || a.", method.GoName, "Send == nil || a.", method.GoName, "Finish == nil || a.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", errorNames.StreamNotImplemented)
	g.P("}")
	g.P("var stream C.int32_t")
	startABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationStart)
	g.P("errID := int32(C.", nativeCGOServerClientStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start", nativeCGOServerGoABICallSuffix(startABI.Params, "&stream"), "))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeClientStreamSession{send: a.", method.GoName, "Send, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: stream}, nil")
	g.P("}")
	g.P()

	sendABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationSend)
	finishABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationFinish)
	cancelABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCancel)
	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeClientStreamSession struct {")
	g.P("send C.", sendABI.TypeName)
	g.P("finish C.", finishABI.TypeName)
	g.P("cancel C.", cancelABI.TypeName)
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerClientStreamSend(g, service, abiPlan, method)
	renderCGONativeServerClientStreamFinish(g, service, abiPlan, method)
	renderCGONativeServerClientStreamCancel(g, service, abiPlan, method)
}

func renderCGONativeServerClientStreamSend(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Send(ctx context.Context", nativeGoRequestParams(g, method.Contract.Native.RequestFields), ") error {")
	g.P(nativeCGOServerRequestEncoderAssignArgs(method.Contract.Native.RequestFields), " := ", nativeCGOServerClientStreamRequestEncoderName(service, method), "(", nativeCGOServerRequestEncoderCallArgs(method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer cleanup()")
	sendABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationSend)
	g.P("errID := int32(C.", nativeCGOServerClientStreamSendTrampolineName(service, method), "(s.send", nativeCGOServerGoABICallSuffix(sendABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamFinish(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Finish(ctx context.Context) (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	finishABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationFinish)
	g.P("errID := int32(C.", nativeCGOServerClientStreamFinishTrampolineName(service, method), "(s.finish", nativeCGOServerGoABICallSuffix(finishABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(callbackErr, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "callbackErr"))
	g.P("}")
	responseNames := nativeGoResponseResultNames(method.Contract.Native.ResponseFields)
	if responseNames == "" {
		g.P("err := ", nativeCGOServerClientStreamResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	} else {
		g.P(responseNames, ", err := ", nativeCGOServerClientStreamResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	}
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(err, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "cleanupErr"))
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	if responseNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", responseNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamCancel(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	cancelABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCancel)
	g.P("errID := int32(C.", nativeCGOServerClientStreamCancelTrampolineName(service, method), "(s.cancel", nativeCGOServerGoABICallSuffix(cancelABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamServerMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, servicePackage string) {
	streamName := servicePackage + service.GoName + method.GoName + "NativeClientStream"
	requestNames := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", streamName, ") (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	g.P("session, err := a.Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	g.P("for {")
	renderNativeStreamRecvAssign(g, method.Contract.Native.RequestFields, "stream.Recv(ctx)", false)
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return session.Finish(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	if requestNames == "" {
		g.P("if err := session.Send(ctx); err != nil {")
	} else {
		g.P("if err := session.Send(ctx, ", requestNames, "); err != nil {")
	}
	g.P("_ = session.Cancel(ctx)")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context", nativeGoRequestParams(g, method.Contract.Native.RequestFields), ") (", sessionName, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil || a.", method.GoName, "Recv == nil || a.", method.GoName, "Finish == nil || a.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", errorNames.StreamNotImplemented)
	g.P("}")
	g.P(nativeCGOServerRequestEncoderAssignArgs(method.Contract.Native.RequestFields), " := ", nativeCGOServerServerStreamRequestEncoderName(service, method), "(", nativeCGOServerRequestEncoderCallArgs(method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("var stream C.int32_t")
	startABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationStart)
	g.P("errID := int32(C.", nativeCGOServerServerStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start", nativeCGOServerGoABICallSuffix(startABI.Params, "&stream"), "))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeServerStreamSession{recv: a.", method.GoName, "Recv, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: stream}, nil")
	g.P("}")
	g.P()

	recvABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationRecv)
	finishABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationFinish)
	cancelABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCancel)
	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeServerStreamSession struct {")
	g.P("recv C.", recvABI.TypeName)
	g.P("finish C.", finishABI.TypeName)
	g.P("cancel C.", cancelABI.TypeName)
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerServerStreamRecv(g, service, abiPlan, method)
	renderCGONativeServerServerStreamFinish(g, service, abiPlan, method)
	renderCGONativeServerServerStreamCancel(g, service, abiPlan, method)
}

func renderCGONativeServerServerStreamRecv(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	recvABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationRecv)
	g.P("errID := int32(C.", nativeCGOServerServerStreamRecvTrampolineName(service, method), "(s.recv", nativeCGOServerGoABICallSuffix(recvABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(callbackErr, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "callbackErr"))
	g.P("}")
	responseNames := nativeGoResponseResultNames(method.Contract.Native.ResponseFields)
	if responseNames == "" {
		g.P("err := ", nativeCGOServerServerStreamResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	} else {
		g.P(responseNames, ", err := ", nativeCGOServerServerStreamResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	}
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(err, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "cleanupErr"))
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	if responseNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", responseNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamFinish(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Finish(ctx context.Context) error {")
	finishABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationFinish)
	g.P("errID := int32(C.", nativeCGOServerServerStreamFinishTrampolineName(service, method), "(s.finish", nativeCGOServerGoABICallSuffix(finishABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamCancel(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	cancelABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCancel)
	g.P("errID := int32(C.", nativeCGOServerServerStreamCancelTrampolineName(service, method), "(s.cancel", nativeCGOServerGoABICallSuffix(cancelABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamServerMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, servicePackage string) {
	streamName := servicePackage + service.GoName + method.GoName + "NativeServerStream"
	requestParams := nativeGoRequestParams(g, method.Contract.Native.RequestFields)
	requestArgs := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	responseNames := nativeGoResponseResultNames(method.Contract.Native.ResponseFields)
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context", requestParams, ", stream ", streamName, ") error {")
	if requestArgs == "" {
		g.P("session, err := a.Start", method.GoName, "(ctx)")
	} else {
		g.P("session, err := a.Start", method.GoName, "(ctx, ", requestArgs, ")")
	}
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("for {")
	renderNativeStreamRecvAssign(g, method.Contract.Native.ResponseFields, "session.Recv(ctx)", true)
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return session.Finish(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	if responseNames == "" {
		g.P("if err := stream.Send(ctx); err != nil {")
	} else {
		g.P("if err := stream.Send(ctx, ", responseNames, "); err != nil {")
	}
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return session.Finish(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil || a.", method.GoName, "Send == nil || a.", method.GoName, "Recv == nil || a.", method.GoName, "CloseSend == nil || a.", method.GoName, "Finish == nil || a.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", errorNames.StreamNotImplemented)
	g.P("}")
	g.P("var stream C.int32_t")
	startABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationStart)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start", nativeCGOServerGoABICallSuffix(startABI.Params, "&stream"), "))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeBidiStreamSession{send: a.", method.GoName, "Send, recv: a.", method.GoName, "Recv, closeSend: a.", method.GoName, "CloseSend, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: stream}, nil")
	g.P("}")
	g.P()

	sendABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationSend)
	recvABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationRecv)
	closeSendABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCloseSend)
	finishABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationFinish)
	cancelABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCancel)
	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeBidiStreamSession struct {")
	g.P("send C.", sendABI.TypeName)
	g.P("recv C.", recvABI.TypeName)
	g.P("closeSend C.", closeSendABI.TypeName)
	g.P("finish C.", finishABI.TypeName)
	g.P("cancel C.", cancelABI.TypeName)
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerBidiStreamSend(g, service, abiPlan, method)
	renderCGONativeServerBidiStreamRecv(g, service, abiPlan, method)
	renderCGONativeServerBidiStreamCloseSend(g, service, abiPlan, method)
	renderCGONativeServerBidiStreamFinish(g, service, abiPlan, method)
	renderCGONativeServerBidiStreamCancel(g, service, abiPlan, method)
}

func renderCGONativeServerBidiStreamSend(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Send(ctx context.Context", nativeGoRequestParams(g, method.Contract.Native.RequestFields), ") error {")
	g.P(nativeCGOServerRequestEncoderAssignArgs(method.Contract.Native.RequestFields), " := ", nativeCGOServerBidiStreamRequestEncoderName(service, method), "(", nativeCGOServerRequestEncoderCallArgs(method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer cleanup()")
	sendABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationSend)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamSendTrampolineName(service, method), "(s.send", nativeCGOServerGoABICallSuffix(sendABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamRecv(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	recvABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationRecv)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamRecvTrampolineName(service, method), "(s.recv", nativeCGOServerGoABICallSuffix(recvABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(callbackErr, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "callbackErr"))
	g.P("}")
	responseNames := nativeGoResponseResultNames(method.Contract.Native.ResponseFields)
	if responseNames == "" {
		g.P("err := ", nativeCGOServerBidiStreamResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	} else {
		g.P(responseNames, ", err := ", nativeCGOServerBidiStreamResponseDecoderName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	}
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "errors.Join(err, cleanupErr)"))
	g.P("}")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "cleanupErr"))
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	if responseNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", responseNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamCloseSend(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	closeSendABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCloseSend)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), "(s.closeSend", nativeCGOServerGoABICallSuffix(closeSendABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamFinish(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Finish(ctx context.Context) error {")
	finishABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationFinish)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamFinishTrampolineName(service, method), "(s.finish", nativeCGOServerGoABICallSuffix(finishABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamCancel(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	cancelABI := nativeCGOServerOperationABI(abiPlan, method, NativeCOperationCancel)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamCancelTrampolineName(service, method), "(s.cancel", nativeCGOServerGoABICallSuffix(cancelABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamServerMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, servicePackage string) {
	streamName := servicePackage + service.GoName + method.GoName + "NativeBidiStream"
	requestNames := nativeGoRequestArgNames(method.Contract.Native.RequestFields)
	responseNames := nativeGoResponseResultNames(method.Contract.Native.ResponseFields)
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", streamName, ") error {")
	g.P("session, err := a.Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("sendDone := make(chan error, 1)")
	g.P("go func() {")
	g.P("for {")
	renderNativeStreamRecvAssign(g, method.Contract.Native.RequestFields, "stream.Recv(ctx)", false)
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("sendDone <- session.CloseSend(ctx)")
	g.P("return")
	g.P("}")
	g.P("sendDone <- err")
	g.P("return")
	g.P("}")
	if requestNames == "" {
		g.P("if err := session.Send(ctx); err != nil {")
	} else {
		g.P("if err := session.Send(ctx, ", requestNames, "); err != nil {")
	}
	g.P("sendDone <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("for {")
	renderNativeStreamRecvAssign(g, method.Contract.Native.ResponseFields, "session.Recv(ctx)", true)
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("if sendErr := <-sendDone; sendErr != nil {")
	g.P("_ = session.Cancel(ctx)")
	g.P("return sendErr")
	g.P("}")
	g.P("return session.Finish(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	if responseNames == "" {
		g.P("if err := stream.Send(ctx); err != nil {")
	} else {
		g.P("if err := stream.Send(ctx, ", responseNames, "); err != nil {")
	}
	g.P("if errors.Is(err, io.EOF) {")
	g.P("if sendErr := <-sendDone; sendErr != nil {")
	g.P("_ = session.Cancel(ctx)")
	g.P("return sendErr")
	g.P("}")
	g.P("return session.Finish(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderNativeStreamRecvAssign(g *protogen.GeneratedFile, fields []FieldPlan, call string, resultNames bool) {
	names := nativeGoRequestArgNames(fields)
	if resultNames {
		names = nativeGoResponseResultNames(fields)
	}
	if names == "" {
		g.P("err := ", call)
		return
	}
	g.P(names, ", err := ", call)
}

func renderCGONativeServerStreamingFallback(g *protogen.GeneratedFile, adapterName string, method runtimeAdapterMethod, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult, " {")
	if method.Streaming {
		g.P("return nil, ", errorNames.StreamNotImplemented)
	} else if method.AdapterResult == " error" {
		g.P("return ", errorNames.StreamNotImplemented)
	} else {
		g.P("return nil, ", errorNames.UnaryCallbackMissing)
	}
	g.P("}")
	g.P()
}

func nativeCGOServerRequestEncoderArgs(g *protogen.GeneratedFile, fields []FieldPlan) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, lowerInitial(field.GoName)+" "+nativeGoRequestFieldType(g, field))
	}
	return strings.Join(parts, ", ")
}

func nativeCGOServerRequestEncoderCallArgs(fields []FieldPlan) string {
	return nativeGoRequestArgNames(fields)
}

func renderCGONativeServerRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerRequestEncoderName(service, method), method.Contract.Native.RequestFields, errorNames)
}

func renderCGONativeServerFlatRequestEncoder(g *protogen.GeneratedFile, name string, fields []FieldPlan, errorNames nativeServerCGOErrorNames) {
	g.P("func ", name, "(", nativeCGOServerRequestEncoderArgs(g, fields), ") ", nativeCGOServerRequestEncoderReturns(fields), " {")
	for _, field := range fields {
		types := nativeCGOServerGoFieldType(field)
		for i, arg := range nativeCGOServerGoFieldArgs(field, false) {
			g.P("var ", arg, " ", types[i])
		}
	}
	g.P("var pinned []uintptr")
	g.P("cleanup := func() {")
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
	g.P("}")
	for _, field := range fields {
		renderCGONativeServerRequestFieldEncode(g, fields, field, errorNames)
	}
	g.P("return ", strings.Join(nativeCGOServerRequestEncoderResultArgs(fields), ", "))
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestFieldEncode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	name := lowerInitial(field.GoName)
	errorReturn := nativeCGOServerRequestEncoderErrorReturn(fields, "err")
	unsupportedReturn := nativeCGOServerRequestEncoderErrorReturn(fields, errorNames.UnsupportedField)
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("if ", name, " {")
		g.P(name, "Value = 1")
		g.P("}")
	case NativeABIShapeBoolByteBufferWrapper:
		g.P(name, "Values := ", name, ".SafeSlice()")
		g.P(name, "LenValue, err := rpcruntime.LengthToInt32(len(", name, "Values))")
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return ", errorReturn)
		g.P("}")
		g.P(name, "Bytes := make([]byte, len(", name, "Values))")
		g.P("for i := range ", name, "Values {")
		g.P("if ", name, "Values[i] {")
		g.P(name, "Bytes[i] = 1")
		g.P("}")
		g.P("}")
		g.P(name, "PtrValue, err := rpcruntime.PinBytes(", name, "Bytes)")
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return ", errorReturn)
		g.P("}")
		g.P("if ", name, "PtrValue != 0 {")
		g.P("pinned = append(pinned, ", name, "PtrValue)")
		g.P("}")
		g.P(name, "Ptr = C.uintptr_t(", name, "PtrValue)")
		g.P(name, "Len = C.int32_t(", name, "LenValue)")
	case NativeABIShapeRepeated:
		g.P(name, "Values := ", name, ".SafeSlice()")
		g.P(name, "LenValue, err := rpcruntime.LengthToInt32(len(", name, "Values))")
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return ", errorReturn)
		g.P("}")
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt32, FieldKindUnsignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(name, "PtrValue, err := rpcruntime.PinSlice(", name, "Values)")
		case FieldKindEnum:
			g.P(name, "RawValues := make([]int32, len(", name, "Values))")
			g.P("for i := range ", name, "Values {")
			g.P(name, "RawValues[i] = int32(", name, "Values[i])")
			g.P("}")
			g.P(name, "PtrValue, err := rpcruntime.PinSlice(", name, "RawValues)")
		default:
			g.P("cleanup()")
			g.P("return ", unsupportedReturn)
		}
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return ", errorReturn)
		g.P("}")
		g.P("if ", name, "PtrValue != 0 {")
		g.P("pinned = append(pinned, ", name, "PtrValue)")
		g.P("}")
		g.P(name, "Ptr = C.uintptr_t(", name, "PtrValue)")
		g.P(name, "Len = C.int32_t(", name, "LenValue)")
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32:
			g.P(name, "Value = C.int32_t(", name, ")")
		case FieldKindSignedInt64:
			g.P(name, "Value = C.int64_t(", name, ")")
		case FieldKindUnsignedInt32:
			g.P(name, "Value = C.uint32_t(", name, ")")
		case FieldKindUnsignedInt64:
			g.P(name, "Value = C.uint64_t(", name, ")")
		case FieldKindFloat:
			g.P(name, "Value = C.float(", name, ")")
		case FieldKindDouble:
			g.P(name, "Value = C.double(", name, ")")
		case FieldKindEnum:
			g.P(name, "Value = C.int32_t(", name, ")")
		case FieldKindString:
			g.P(name, "LenValue, err := rpcruntime.LengthToInt32(len(", name, ".SafeString()))")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return ", errorReturn)
			g.P("}")
			g.P("_, ", name, "PtrValue, err := rpcruntime.PinString(", name, ".SafeString())")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return ", errorReturn)
			g.P("}")
			g.P("if ", name, "PtrValue != 0 {")
			g.P("pinned = append(pinned, ", name, "PtrValue)")
			g.P("}")
			g.P(name, "Ptr = C.uintptr_t(", name, "PtrValue)")
			g.P(name, "Len = C.int32_t(", name, "LenValue)")
		case FieldKindBytes, FieldKindMessage:
			g.P(name, "Bytes := ", name, ".SafeBytes()")
			g.P(name, "LenValue, err := rpcruntime.LengthToInt32(len(", name, "Bytes))")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return ", errorReturn)
			g.P("}")
			g.P(name, "PtrValue, err := rpcruntime.PinBytes(", name, "Bytes)")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return ", errorReturn)
			g.P("}")
			g.P("if ", name, "PtrValue != 0 {")
			g.P("pinned = append(pinned, ", name, "PtrValue)")
			g.P("}")
			g.P(name, "Ptr = C.uintptr_t(", name, "PtrValue)")
			g.P(name, "Len = C.int32_t(", name, "LenValue)")
		default:
			g.P("cleanup()")
			g.P("return ", unsupportedReturn)
		}
	default:
		g.P("cleanup()")
		g.P("return ", unsupportedReturn)
	}
}

func renderCGONativeServerResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
}

func renderCGONativeServerFlatResponseDecoder(g *protogen.GeneratedFile, name string, fields []FieldPlan, errorNames nativeServerCGOErrorNames) {
	g.P("func ", name, "(", nativeCGOServerFlatValueParams(fields), ") (", nativeGoResponseReturns(g, fields), ") {")
	for _, field := range fields {
		renderCGONativeServerResponseFieldDecode(g, fields, field, errorNames)
	}
	responseNames := nativeGoResponseResultNames(fields)
	if responseNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", responseNames, ", nil")
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerResponseFieldDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	errReturn := func(errExpr string) string { return nativeGoZeroReturns(fields, errExpr) }
	fieldName := lowerInitial(field.GoName)
	name := fieldName + "Result"
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P(name, " := ", fieldName, "Value != 0")
	case NativeABIShapeBoolByteBufferWrapper:
		renderCGONativeServerResponseRepeatDecode(g, field, name, "byte", "rpcruntime.NewRpcBoolRepeatChecked", errReturn)
		g.P(name, " := ", name, "Wrapper.SafeSlice()")
	case NativeABIShapeRepeated:
		switch field.Kind {
		case FieldKindSignedInt32:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "int32", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindUnsignedInt32:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "uint32", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindSignedInt64:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "int64", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindUnsignedInt64:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "uint64", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindFloat:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "float32", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindDouble:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "float64", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindEnum:
			renderCGONativeServerResponseRepeatDecode(g, field, name, "int32", "rpcruntime.NewRpcRepeatChecked", errReturn)
			g.P(name, "Raw := ", name, "Wrapper.SafeSlice()")
			g.P(name, " := make([]", nativeGoEnumType(g, field), ", len(", name, "Raw))")
			g.P("for i := range ", name, "Raw {")
			g.P(name, "[i] = ", nativeGoEnumType(g, field), "(", name, "Raw[i])")
			g.P("}")
		default:
			g.P("return ", errReturn(errorNames.UnsupportedField))
		}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32:
			g.P(name, " := int32(", fieldName, "Value)")
		case FieldKindSignedInt64:
			g.P(name, " := int64(", fieldName, "Value)")
		case FieldKindUnsignedInt32:
			g.P(name, " := uint32(", fieldName, "Value)")
		case FieldKindUnsignedInt64:
			g.P(name, " := uint64(", fieldName, "Value)")
		case FieldKindFloat:
			g.P(name, " := float32(", fieldName, "Value)")
		case FieldKindDouble:
			g.P(name, " := float64(", fieldName, "Value)")
		case FieldKindEnum:
			g.P(name, " := ", nativeGoEnumType(g, field), "(int32(", fieldName, "Value))")
		case FieldKindString:
			renderCGONativeServerResponseTextDecode(g, field, name, "String", "SafeString", errReturn)
		case FieldKindBytes, FieldKindMessage:
			renderCGONativeServerResponseTextDecode(g, field, name, "Bytes", "SafeBytes", errReturn)
		default:
			g.P("return ", errReturn(errorNames.UnsupportedField))
		}
	default:
		g.P("return ", errReturn(errorNames.UnsupportedField))
	}
}

func renderCGONativeServerResponseRepeatDecode(g *protogen.GeneratedFile, field FieldPlan, name, elemType, ctor string, errReturn func(string) string) {
	fieldName := lowerInitial(field.GoName)
	g.P("if _, err := rpcruntime.LengthFromInt32(int32(", fieldName, "Len)); err != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P(name, "Wrapper, err := ", ctor, "((*", elemType, ")(unsafe.Pointer(uintptr(", fieldName, "Ptr))), int32(", fieldName, "Len), false)")
	g.P("if err != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
}

func renderCGONativeServerResponseTextDecode(g *protogen.GeneratedFile, field FieldPlan, name, wrapper, safeMethod string, errReturn func(string) string) {
	fieldName := lowerInitial(field.GoName)
	g.P("if _, err := rpcruntime.LengthFromInt32(int32(", fieldName, "Len)); err != nil {")
	g.P(`return `, errReturn(`fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P(fieldName, "Wrapper := rpcruntime.NewRpc", wrapper, "((*byte)(unsafe.Pointer(uintptr(", fieldName, "Ptr))), int32(", fieldName, "Len), false)")
	g.P(name, " := ", fieldName, "Wrapper.", safeMethod, "()")
}

func renderCGONativeServerRegistration(g *protogen.GeneratedFile, service ServicePlan, abiPlan NativeCABIPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	adapterName := lowerInitial(service.GoName) + "CGONativeServerAdapter"
	registerABI := abiPlan.Register
	g.P("//export ", registerABI.Symbol)
	g.P("func ", registerABI.Symbol, "(", nativeCABIRegisterParamList(registerABI.Params), ") ", registerABI.Return.CGoType, " {")
	g.P("next := &", lowerInitial(service.GoName), "CGONativeAdapter{}")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			callbackName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationUnary)
			g.P("if ", callbackName, " == nil { return C.int32_t(rpcruntime.StoreError(", errorNames.UnaryCallbackMissing, ")) }")
			g.P("next.", method.GoName, "Callback = ", callbackName)
		case StreamingKindClientStreaming:
			startName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationStart)
			sendName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationSend)
			finishName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationFinish)
			cancelName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationCancel)
			g.P("if ", startName, " == nil || ", sendName, " == nil || ", finishName, " == nil || ", cancelName, " == nil { return C.int32_t(rpcruntime.StoreError(", errorNames.StreamNotImplemented, ")) }")
			g.P("next.", method.GoName, "Start = ", startName)
			g.P("next.", method.GoName, "Send = ", sendName)
			g.P("next.", method.GoName, "Finish = ", finishName)
			g.P("next.", method.GoName, "Cancel = ", cancelName)
		case StreamingKindServerStreaming:
			startName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationStart)
			recvName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationRecv)
			finishName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationFinish)
			cancelName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationCancel)
			g.P("if ", startName, " == nil || ", recvName, " == nil || ", finishName, " == nil || ", cancelName, " == nil { return C.int32_t(rpcruntime.StoreError(", errorNames.StreamNotImplemented, ")) }")
			g.P("next.", method.GoName, "Start = ", startName)
			g.P("next.", method.GoName, "Recv = ", recvName)
			g.P("next.", method.GoName, "Finish = ", finishName)
			g.P("next.", method.GoName, "Cancel = ", cancelName)
		case StreamingKindBidiStreaming:
			startName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationStart)
			sendName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationSend)
			recvName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationRecv)
			closeSendName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationCloseSend)
			finishName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationFinish)
			cancelName := nativeCGOServerRegisterCallbackParamName(method, NativeCOperationCancel)
			g.P("if ", startName, " == nil || ", sendName, " == nil || ", recvName, " == nil || ", closeSendName, " == nil || ", finishName, " == nil || ", cancelName, " == nil { return C.int32_t(rpcruntime.StoreError(", errorNames.StreamNotImplemented, ")) }")
			g.P("next.", method.GoName, "Start = ", startName)
			g.P("next.", method.GoName, "Send = ", sendName)
			g.P("next.", method.GoName, "Recv = ", recvName)
			g.P("next.", method.GoName, "CloseSend = ", closeSendName)
			g.P("next.", method.GoName, "Finish = ", finishName)
			g.P("next.", method.GoName, "Cancel = ", cancelName)
		}
	}
	g.P("if err := ", servicePackage, "Register", service.GoName, "CGONativeServer(next); err != nil { return C.int32_t(rpcruntime.StoreError(err)) }")
	g.P(adapterName, "Mu.Lock()")
	g.P(adapterName, " = next")
	g.P(adapterName, "Mu.Unlock()")
	g.P("return 0")
	g.P("}")
	g.P()
}

func nativeCGOServerRegisterCallbackParamName(method MethodPlan, operation NativeCOperation) string {
	return lowerInitial(method.GoName) + upperInitial(nativeCABIRegisterParamName(operation))
}

func renderCGONativeServerResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
}

func renderCGONativeServerFlatResponseCleanup(g *protogen.GeneratedFile, name string, fields []FieldPlan) {
	g.P("func ", name, "(", nativeCGOServerFlatValueParams(fields), ") error {")
	g.P("var cleanupErr error")
	for _, field := range fields {
		fieldName := lowerInitial(field.GoName)
		if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
			g.P("if ", fieldName, "Ownership > 0 && ", fieldName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(", fieldName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("}")
		}
		if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
			g.P("if ", fieldName, "Ownership > 0 && ", fieldName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(", fieldName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("}")
		}
	}
	g.P("return cleanupErr")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerClientStreamRequestEncoderName(service, method), method.Contract.Native.RequestFields, errorNames)
}

func renderCGONativeServerClientStreamResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerClientStreamResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
}

func renderCGONativeServerClientStreamResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerClientStreamResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
}

func renderCGONativeServerServerStreamRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerServerStreamRequestEncoderName(service, method), method.Contract.Native.RequestFields, errorNames)
}

func renderCGONativeServerServerStreamResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerServerStreamResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
}

func renderCGONativeServerServerStreamResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerServerStreamResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
}

func renderCGONativeServerBidiStreamRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerBidiStreamRequestEncoderName(service, method), method.Contract.Native.RequestFields, errorNames)
}

func renderCGONativeServerBidiStreamResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerBidiStreamResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
}

func renderCGONativeServerBidiStreamResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerBidiStreamResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
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
	g.P("if string(text) == io.EOF.Error() {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return errors.New(string(text))")
	g.P("}")
	g.P(`return fmt.Errorf("rpccgo: cgo native server callback returned unknown error id %d", errID)`)
	g.P("}")
	g.P()
}

func nativeCGOServerGoRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryRequest"
}

func nativeCGOServerGoResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerGoClientStreamRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamRequest"
}

func nativeCGOServerGoClientStreamResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerGoServerStreamRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamRequest"
}

func nativeCGOServerGoServerStreamResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerGoBidiStreamRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamRequest"
}

func nativeCGOServerGoBidiStreamResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerRequestName(service ServicePlan, method MethodPlan) string {
	return nativeCGOServerGoRequestCarrierName(service, method)
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

func nativeCGOServerClientStreamRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamRequest"
}

func nativeCGOServerClientStreamResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerClientStreamRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeClientStreamRequest"
}

func nativeCGOServerClientStreamResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerClientStreamResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerClientStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamStartCallback"
}

func nativeCGOServerClientStreamSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamSendCallback"
}

func nativeCGOServerClientStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamFinishCallback"
}

func nativeCGOServerClientStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamCancelCallback"
}

func nativeCGOServerClientStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamStartCallback"
}

func nativeCGOServerClientStreamSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamSendCallback"
}

func nativeCGOServerClientStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamFinishCallback"
}

func nativeCGOServerClientStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamCancelCallback"
}

func nativeCGOServerServerStreamRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamRequest"
}

func nativeCGOServerServerStreamResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerServerStreamRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeServerStreamRequest"
}

func nativeCGOServerServerStreamResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerServerStreamResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerServerStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamStartCallback"
}

func nativeCGOServerServerStreamRecvCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamRecvCallback"
}

func nativeCGOServerServerStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamFinishCallback"
}

func nativeCGOServerServerStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamCancelCallback"
}

func nativeCGOServerServerStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamStartCallback"
}

func nativeCGOServerServerStreamRecvTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamRecvCallback"
}

func nativeCGOServerServerStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamFinishCallback"
}

func nativeCGOServerServerStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamCancelCallback"
}

func nativeCGOServerBidiStreamRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamRequest"
}

func nativeCGOServerBidiStreamResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerBidiStreamRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeBidiStreamRequest"
}

func nativeCGOServerBidiStreamResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerBidiStreamResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerBidiStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamStartCallback"
}

func nativeCGOServerBidiStreamSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamSendCallback"
}

func nativeCGOServerBidiStreamRecvCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamRecvCallback"
}

func nativeCGOServerBidiStreamCloseSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamCloseSendCallback"
}

func nativeCGOServerBidiStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamFinishCallback"
}

func nativeCGOServerBidiStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamCancelCallback"
}

func nativeCGOServerBidiStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamStartCallback"
}

func nativeCGOServerBidiStreamSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamSendCallback"
}

func nativeCGOServerBidiStreamRecvTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamRecvCallback"
}

func nativeCGOServerBidiStreamCloseSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamCloseSendCallback"
}

func nativeCGOServerBidiStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamFinishCallback"
}

func nativeCGOServerBidiStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamCancelCallback"
}

func nativeServerCGONeedsUnsafe() bool {
	return true
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
		lowerInitial(service.GoName) + "CGONativeAdapter":              service.FullName + " adapter",
		"Register" + service.GoName + "CGONativeServer":                service.FullName + " registration",
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
		if method.Streaming != StreamingKindUnary && method.Streaming != StreamingKindClientStreaming && method.Streaming != StreamingKindServerStreaming && method.Streaming != StreamingKindBidiStreaming {
			continue
		}
		requestName := nativeCGOServerRequestName(service, method)
		responseName := nativeCGOServerResponseName(service, method)
		switch method.Streaming {
		case StreamingKindClientStreaming:
			requestName = nativeCGOServerClientStreamRequestName(service, method)
			responseName = nativeCGOServerClientStreamResponseName(service, method)
		case StreamingKindServerStreaming:
			requestName = nativeCGOServerServerStreamRequestName(service, method)
			responseName = nativeCGOServerServerStreamResponseName(service, method)
		case StreamingKindBidiStreaming:
			requestName = nativeCGOServerBidiStreamRequestName(service, method)
			responseName = nativeCGOServerBidiStreamResponseName(service, method)
		}
		for _, item := range []struct {
			symbol string
			source string
		}{
			{requestName, method.FullName + " cgo request"},
			{responseName, method.FullName + " cgo response"},
		} {
			if err := addGenerated(item.symbol, item.source); err != nil {
				return err
			}
		}
		switch method.Streaming {
		case StreamingKindUnary:
			for _, item := range []struct {
				symbol string
				source string
			}{
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
		case StreamingKindClientStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerClientStreamStartCallbackName(service, method), method.FullName + " cgo stream start callback"},
				{nativeCGOServerClientStreamSendCallbackName(service, method), method.FullName + " cgo stream send callback"},
				{nativeCGOServerClientStreamFinishCallbackName(service, method), method.FullName + " cgo stream finish callback"},
				{nativeCGOServerClientStreamCancelCallbackName(service, method), method.FullName + " cgo stream cancel callback"},
				{nativeCGOServerClientStreamStartTrampolineName(service, method), method.FullName + " cgo stream start trampoline"},
				{nativeCGOServerClientStreamSendTrampolineName(service, method), method.FullName + " cgo stream send trampoline"},
				{nativeCGOServerClientStreamFinishTrampolineName(service, method), method.FullName + " cgo stream finish trampoline"},
				{nativeCGOServerClientStreamCancelTrampolineName(service, method), method.FullName + " cgo stream cancel trampoline"},
				{nativeCGOServerClientStreamRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerClientStreamResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerClientStreamResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		case StreamingKindServerStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerServerStreamStartCallbackName(service, method), method.FullName + " cgo stream start callback"},
				{nativeCGOServerServerStreamRecvCallbackName(service, method), method.FullName + " cgo stream recv callback"},
				{nativeCGOServerServerStreamFinishCallbackName(service, method), method.FullName + " cgo stream finish callback"},
				{nativeCGOServerServerStreamCancelCallbackName(service, method), method.FullName + " cgo stream cancel callback"},
				{nativeCGOServerServerStreamStartTrampolineName(service, method), method.FullName + " cgo stream start trampoline"},
				{nativeCGOServerServerStreamRecvTrampolineName(service, method), method.FullName + " cgo stream recv trampoline"},
				{nativeCGOServerServerStreamFinishTrampolineName(service, method), method.FullName + " cgo stream finish trampoline"},
				{nativeCGOServerServerStreamCancelTrampolineName(service, method), method.FullName + " cgo stream cancel trampoline"},
				{nativeCGOServerServerStreamRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerServerStreamResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerServerStreamResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		case StreamingKindBidiStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerBidiStreamStartCallbackName(service, method), method.FullName + " cgo stream start callback"},
				{nativeCGOServerBidiStreamSendCallbackName(service, method), method.FullName + " cgo stream send callback"},
				{nativeCGOServerBidiStreamRecvCallbackName(service, method), method.FullName + " cgo stream recv callback"},
				{nativeCGOServerBidiStreamCloseSendCallbackName(service, method), method.FullName + " cgo stream close send callback"},
				{nativeCGOServerBidiStreamFinishCallbackName(service, method), method.FullName + " cgo stream finish callback"},
				{nativeCGOServerBidiStreamCancelCallbackName(service, method), method.FullName + " cgo stream cancel callback"},
				{nativeCGOServerBidiStreamStartTrampolineName(service, method), method.FullName + " cgo stream start trampoline"},
				{nativeCGOServerBidiStreamSendTrampolineName(service, method), method.FullName + " cgo stream send trampoline"},
				{nativeCGOServerBidiStreamRecvTrampolineName(service, method), method.FullName + " cgo stream recv trampoline"},
				{nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), method.FullName + " cgo stream close send trampoline"},
				{nativeCGOServerBidiStreamFinishTrampolineName(service, method), method.FullName + " cgo stream finish trampoline"},
				{nativeCGOServerBidiStreamCancelTrampolineName(service, method), method.FullName + " cgo stream cancel trampoline"},
				{nativeCGOServerBidiStreamRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerBidiStreamResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerBidiStreamResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		}
		if err := validateNativeClientStructFields(requestName, method.Contract.Native.RequestFields, nativeClientOutputFieldSymbols); err != nil {
			return err
		}
		if err := validateNativeClientStructFields(responseName, method.Contract.Native.ResponseFields, nativeClientInputFieldSymbols); err != nil {
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
		lowerInitial(service.GoName) + "CGONativeAdapter":              service.FullName + " adapter",
		"Register" + service.GoName + "CGONativeServer":                service.FullName + " registration",
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
		switch method.Streaming {
		case StreamingKindUnary:
			add(nativeCGOServerRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerCallbackName(service, method), method.FullName+" cgo callback")
			add(nativeCGOServerTrampolineName(service, method), method.FullName+" cgo trampoline")
			add(nativeCGOServerRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerResponseCleanupName(service, method), method.FullName+" response cleanup")
		case StreamingKindClientStreaming:
			add(nativeCGOServerClientStreamRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerClientStreamResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerClientStreamStartCallbackName(service, method), method.FullName+" cgo stream start callback")
			add(nativeCGOServerClientStreamSendCallbackName(service, method), method.FullName+" cgo stream send callback")
			add(nativeCGOServerClientStreamFinishCallbackName(service, method), method.FullName+" cgo stream finish callback")
			add(nativeCGOServerClientStreamCancelCallbackName(service, method), method.FullName+" cgo stream cancel callback")
			add(nativeCGOServerClientStreamStartTrampolineName(service, method), method.FullName+" cgo stream start trampoline")
			add(nativeCGOServerClientStreamSendTrampolineName(service, method), method.FullName+" cgo stream send trampoline")
			add(nativeCGOServerClientStreamFinishTrampolineName(service, method), method.FullName+" cgo stream finish trampoline")
			add(nativeCGOServerClientStreamCancelTrampolineName(service, method), method.FullName+" cgo stream cancel trampoline")
			add(nativeCGOServerClientStreamRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerClientStreamResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerClientStreamResponseCleanupName(service, method), method.FullName+" response cleanup")
		case StreamingKindServerStreaming:
			add(nativeCGOServerServerStreamRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerServerStreamResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerServerStreamStartCallbackName(service, method), method.FullName+" cgo stream start callback")
			add(nativeCGOServerServerStreamRecvCallbackName(service, method), method.FullName+" cgo stream recv callback")
			add(nativeCGOServerServerStreamFinishCallbackName(service, method), method.FullName+" cgo stream finish callback")
			add(nativeCGOServerServerStreamCancelCallbackName(service, method), method.FullName+" cgo stream cancel callback")
			add(nativeCGOServerServerStreamStartTrampolineName(service, method), method.FullName+" cgo stream start trampoline")
			add(nativeCGOServerServerStreamRecvTrampolineName(service, method), method.FullName+" cgo stream recv trampoline")
			add(nativeCGOServerServerStreamFinishTrampolineName(service, method), method.FullName+" cgo stream finish trampoline")
			add(nativeCGOServerServerStreamCancelTrampolineName(service, method), method.FullName+" cgo stream cancel trampoline")
			add(nativeCGOServerServerStreamRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerServerStreamResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerServerStreamResponseCleanupName(service, method), method.FullName+" response cleanup")
		case StreamingKindBidiStreaming:
			add(nativeCGOServerBidiStreamRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerBidiStreamResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerBidiStreamStartCallbackName(service, method), method.FullName+" cgo stream start callback")
			add(nativeCGOServerBidiStreamSendCallbackName(service, method), method.FullName+" cgo stream send callback")
			add(nativeCGOServerBidiStreamRecvCallbackName(service, method), method.FullName+" cgo stream recv callback")
			add(nativeCGOServerBidiStreamCloseSendCallbackName(service, method), method.FullName+" cgo stream close send callback")
			add(nativeCGOServerBidiStreamFinishCallbackName(service, method), method.FullName+" cgo stream finish callback")
			add(nativeCGOServerBidiStreamCancelCallbackName(service, method), method.FullName+" cgo stream cancel callback")
			add(nativeCGOServerBidiStreamStartTrampolineName(service, method), method.FullName+" cgo stream start trampoline")
			add(nativeCGOServerBidiStreamSendTrampolineName(service, method), method.FullName+" cgo stream send trampoline")
			add(nativeCGOServerBidiStreamRecvTrampolineName(service, method), method.FullName+" cgo stream recv trampoline")
			add(nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), method.FullName+" cgo stream close send trampoline")
			add(nativeCGOServerBidiStreamFinishTrampolineName(service, method), method.FullName+" cgo stream finish trampoline")
			add(nativeCGOServerBidiStreamCancelTrampolineName(service, method), method.FullName+" cgo stream cancel trampoline")
			add(nativeCGOServerBidiStreamRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerBidiStreamResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerBidiStreamResponseCleanupName(service, method), method.FullName+" response cleanup")
		}
	}
}
