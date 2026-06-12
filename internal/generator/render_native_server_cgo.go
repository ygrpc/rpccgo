package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
	if err := validateNativeServerCGOSymbols(plan, service); err != nil {
		return err
	}
	nativeABI, err := nativeCServiceABIs(plan, service)
	if err != nil {
		return err
	}

	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := newGeneratedFile(plugin, plan, file, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"NativeServer")
	runtimeMethods, err := buildRuntimeMethodProjectionsWithMessageTypes(g, service, false)
	if err != nil {
		return err
	}
	runtimeMethods = qualifyRuntimeMethodProjections(runtimeMethods, servicePackage)

	g.P("package main")
	g.P()
	renderCGONativeServerPreamble(g, service, nativeABI)
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	g.P(`io "io"`)
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
	g.P(`sync "sync"`)
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
	g.P(errorNames.UnsupportedField, ` = errors.New("rpccgo: cgo native server field codec is not implemented")`)
	g.P(errorNames.StreamPartiallyRegistered, ` = errors.New("rpccgo: cgo native server stream callbacks are partially registered")`)
	g.P(lowerInitial(service.GoName), "CGONativeServerAdapterMu sync.Mutex")
	g.P(lowerInitial(service.GoName), "CGONativeServerAdapter = &", lowerInitial(service.GoName), "CGONativeAdapter{}")
	g.P(")")
	g.P()

	adapterName := lowerInitial(service.GoName) + "CGONativeAdapter"
	renderCGONativeServerAdapter(g, service, nativeABI, runtimeMethods, adapterName, errorNames, servicePackage)
	renderCGONativeServerRegistration(g, service, nativeABI, errorNames, servicePackage)
	return nil
}

func qualifyRuntimeMethodProjections(methods []runtimeMethodProjection, servicePackage string) []runtimeMethodProjection {
	qualified := make([]runtimeMethodProjection, len(methods))
	copy(qualified, methods)
	for i := range qualified {
		if !qualified[i].Stream.Streaming {
			continue
		}
		rawRequestName := qualified[i].Symbols.NativeStreamRequestType
		rawResponseName := qualified[i].Symbols.NativeStreamResponseType
		qualified[i].Symbols.NativeStreamRequestType = servicePackage + rawRequestName
		qualified[i].Symbols.NativeStreamResponseType = servicePackage + rawResponseName
		qualified[i].Native.EntryResult = strings.ReplaceAll(qualified[i].Native.EntryResult, rawRequestName, qualified[i].Symbols.NativeStreamRequestType)
		qualified[i].Native.EntryResult = strings.ReplaceAll(qualified[i].Native.EntryResult, rawResponseName, qualified[i].Symbols.NativeStreamResponseType)
	}
	return qualified
}

func renderCGONativeServerPreamble(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI) {
	g.P("/*")
	g.P("#include <stdint.h>")
	g.P()
	for _, method := range service.Methods {
		operations, _ := NativeCOperationsForMethod(method)
		for _, operation := range operations {
			current := abi.Methods[method.FullName][operation]
			g.P("typedef ", current.Return.CType, " (*", current.TypeName, ")(", nativeCABIParamList(current.Params), ");")
		}
		g.P()
	}
	for _, method := range service.Methods {
		operations, _ := NativeCOperationsForMethod(method)
		for _, operation := range operations {
			current := abi.Methods[method.FullName][operation]
			if operation == NativeCOperationUnary {
				unaryABI := current
				g.P("static inline ", unaryABI.Return.CType, " ", nativeCGOServerTrampolineName(service, method), "(", unaryABI.TypeName, " callback", nativeCGOServerTypedParamSuffix(nativeCABIParamListValues(unaryABI.Params)), ") {")
				g.P("\treturn callback(", nativeCABIArgNames(unaryABI.Params), ");")
				g.P("}")
				g.P()
				continue
			}
			renderCGONativeServerCallbackTrampoline(g, nativeCGOServerCallbackTrampolineName(service, method, operation), current)
		}
	}
	g.P("*/")
}

func renderCGONativeServerCallbackTrampoline(g *protogen.GeneratedFile, name string, abi COperationABI) {
	g.P("static inline ", abi.Return.CType, " ", name, "(", abi.TypeName, " callback", nativeCGOServerTypedParamSuffix(nativeCABIParamListValues(abi.Params)), ") {")
	g.P("\treturn callback(", nativeCABIArgNames(abi.Params), ");")
	g.P("}")
	g.P()
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

func nativeCGOServerGoABIArgsWithRequest(params []CABISlot, handleArg, requestArg string) []string {
	args := make([]string, 0, len(params))
	for _, param := range params {
		args = append(args, nativeCGOServerGoABIArgWithRequest(param, handleArg, requestArg))
	}
	return args
}

func nativeCGOServerGoABIArg(param CABISlot, handleArg string) string {
	return nativeCGOServerGoABIArgWithRequest(param, handleArg, "")
}

func nativeCGOServerGoABIArgWithRequest(param CABISlot, handleArg, requestArg string) string {
	if param.Role == CABISlotRoleHandle {
		return handleArg
	}
	if param.FieldGoName == "" {
		return param.Name
	}
	fieldName := param.FieldGoName
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
	if requestArg != "" {
		return requestArg + "." + arg
	}
	return arg
}

func nativeCGOServerGoABICallSuffix(params []CABISlot, handleArg string) string {
	return nativeCGOServerGoCallSuffix(nativeCGOServerGoABIArgs(params, handleArg))
}

func nativeCGOServerGoABICallSuffixWithRequest(params []CABISlot, handleArg, requestArg string) string {
	return nativeCGOServerGoCallSuffix(nativeCGOServerGoABIArgsWithRequest(params, handleArg, requestArg))
}

func nativeCGOServerGoABIArgList(params []CABISlot, handleArg string) string {
	return strings.Join(nativeCGOServerGoABIArgs(params, handleArg), ", ")
}

func nativeCGOServerGoABIArgListWithRequest(params []CABISlot, handleArg, requestArg string) string {
	return strings.Join(nativeCGOServerGoABIArgsWithRequest(params, handleArg, requestArg), ", ")
}

func nativeCGOServerOperationABI(abi nativeCServiceABI, method MethodPlan, operation NativeCOperation) COperationABI {
	return abi.Methods[method.FullName][operation]
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

func nativeCGOServerRequestEncoderReturns(requestName string) string {
	return "(*" + requestName + ", error)"
}

func nativeCGOServerRequestEncoderResultArgs(requestName string, fields []FieldPlan) []string {
	args := make([]string, 0, len(fields)*3+1)
	for _, field := range fields {
		for _, arg := range nativeCGOServerGoFieldArgs(field, false) {
			args = append(args, arg+": "+arg)
		}
	}
	args = append(args, "pinned: pinned")
	return []string{"&" + requestName + "{" + strings.Join(args, ", ") + "}", "nil"}
}

func nativeCGOServerRequestEncoderErrorReturn(errExpr string) string {
	return "nil, " + errExpr
}

func renderCGONativeServerRequestEncoderReleasePinned(g *protogen.GeneratedFile) {
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
}

func nativeCGOServerRequestEncoderResultName(operationName string) string {
	return lowerInitial(strings.TrimPrefix(operationName, "encode"))
}

func nativeCGOServerRequestEncoderAssignResult(operationName string) string {
	return nativeCGOServerRequestEncoderResultName(operationName) + ", err"
}

func nativeCGOServerRequestEncoderReleaseCall(operationName string) string {
	return nativeCGOServerRequestEncoderResultName(operationName) + ".Release()"
}

func nativeCGOServerRequestEncoderRequestArg(operationName string) string {
	return nativeCGOServerRequestEncoderResultName(operationName)
}

func nativeCGOServerRequestEncoderCallSuffix(params []CABISlot, handleArg, operationName string) string {
	return nativeCGOServerGoABICallSuffixWithRequest(params, handleArg, nativeCGOServerRequestEncoderRequestArg(operationName))
}

func nativeCGOServerRequestEncoderArgList(params []CABISlot, handleArg, operationName string) string {
	return nativeCGOServerGoABIArgListWithRequest(params, handleArg, nativeCGOServerRequestEncoderRequestArg(operationName))
}

func renderCGONativeServerRequestEncoderResult(g *protogen.GeneratedFile, requestName string, fields []FieldPlan) {
	g.P("return ", strings.Join(nativeCGOServerRequestEncoderResultArgs(requestName, fields), ", "))
}

func renderCGONativeServerRequestType(g *protogen.GeneratedFile, requestName string, fields []FieldPlan) {
	g.P("type ", requestName, " struct {")
	for _, field := range fields {
		types := nativeCGOServerGoFieldType(field)
		for i, arg := range nativeCGOServerGoFieldArgs(field, false) {
			g.P(arg, " ", types[i])
		}
	}
	g.P("pinned []uintptr")
	g.P("}")
	g.P()
	g.P("func (r *", requestName, ") Release() {")
	g.P("if r == nil {")
	g.P("return")
	g.P("}")
	g.P("for i := len(r.pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(r.pinned[i])")
	g.P("}")
	g.P("r.pinned = nil")
	g.P("}")
	g.P()
}

func nativeCGOServerRequestEncoderCallArgs(fields []FieldPlan) string {
	return nativeGoRequestArgNames(fields)
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

func renderCGONativeServerAdapter(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, methods []runtimeMethodProjection, adapterName string, errorNames nativeServerCGOErrorNames, servicePackage string) {
	g.P("type ", adapterName, " struct {")
	renderCGONativeServerAdapterFields(g, service, abi)
	g.P("}")
	g.P()
	renderCGONativeRecvWaiter(g, service)

	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.Identity.GoName]
		if !ok {
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod)
			continue
		}
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGONativeServerUnaryAdapter(g, service, abi, adapterName, method, errorNames)
		case StreamingKindClientStreaming:
			renderCGONativeServerClientStreamAdapter(g, service, abi, adapterName, method, errorNames, servicePackage)
		case StreamingKindServerStreaming:
			renderCGONativeServerServerStreamAdapter(g, service, abi, adapterName, method, errorNames, servicePackage)
		case StreamingKindBidiStreaming:
			renderCGONativeServerBidiStreamAdapter(g, service, abi, adapterName, method, errorNames, servicePackage)
		default:
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod)
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
			renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerRequestEncoderName(service, method), nativeCGOServerRequestName(service, method), method.Contract.Native.RequestFields, errorNames)
			renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
			renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
		case StreamingKindClientStreaming:
			renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerClientStreamRequestEncoderName(service, method), nativeCGOServerClientStreamRequestName(service, method), method.Contract.Native.RequestFields, errorNames)
			renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerClientStreamResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
			renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerClientStreamResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
		case StreamingKindServerStreaming:
			renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerServerStreamRequestEncoderName(service, method), nativeCGOServerServerStreamRequestName(service, method), method.Contract.Native.RequestFields, errorNames)
			renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerServerStreamResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
			renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerServerStreamResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
		case StreamingKindBidiStreaming:
			renderCGONativeServerFlatRequestEncoder(g, nativeCGOServerBidiStreamRequestEncoderName(service, method), nativeCGOServerBidiStreamRequestName(service, method), method.Contract.Native.RequestFields, errorNames)
			renderCGONativeServerFlatResponseDecoder(g, nativeCGOServerBidiStreamResponseDecoderName(service, method), method.Contract.Native.ResponseFields, errorNames)
			renderCGONativeServerFlatResponseCleanup(g, nativeCGOServerBidiStreamResponseCleanupName(service, method), method.Contract.Native.ResponseFields)
		}
	}
	renderCGONativeErrorIDHelper(g, service)
}

// renderCGONativeRecvWaiter emits the coordination types that let Finish or Cancel interrupt a blocking cgo native Recv callback.
func renderCGONativeRecvWaiter(g *protogen.GeneratedFile, service ServicePlan) {
	prefix := lowerInitial(service.GoName)
	g.P("// ", prefix, "CGONativeRecvResult carries the result of a blocking cgo native Recv callback.")
	g.P("type ", prefix, "CGONativeRecvResult[T any] struct {")
	g.P("value T")
	g.P("err error")
	g.P("}")
	g.P()
	g.P("// ", prefix, "AwaitCGONativeRecv waits for a blocking cgo native Recv callback while allowing Finish or Cancel to interrupt the wait.")
	g.P("func ", prefix, "AwaitCGONativeRecv[T any](ctx context.Context, finishRequested <-chan struct{}, recv func() (T, error), finish func() error, cancel func() error) (T, error, bool) {")
	g.P("select {")
	g.P("case <-finishRequested:")
	g.P("var zero T")
	g.P("return zero, finish(), true")
	g.P("case <-ctx.Done():")
	g.P("var zero T")
	g.P("return zero, errors.Join(ctx.Err(), cancel()), true")
	g.P("default:")
	g.P("}")
	g.P("results := make(chan ", prefix, "CGONativeRecvResult[T], 1)")
	g.P("go func() {")
	g.P("value, err := recv()")
	g.P("results <- ", prefix, "CGONativeRecvResult[T]{value: value, err: err}")
	g.P("}()")
	g.P("var zero T")
	g.P("select {")
	g.P("case result := <-results:")
	g.P("return result.value, result.err, false")
	g.P("case <-finishRequested:")
	g.P("return zero, finish(), true")
	g.P("case <-ctx.Done():")
	g.P("return zero, errors.Join(ctx.Err(), cancel()), true")
	g.P("}")
	g.P("}")
	g.P()
}

func renderCGONativeServerAdapterFields(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI) {
	callbackTypeName := func(method MethodPlan, operation NativeCOperation) string {
		return nativeCGOServerOperationABI(abi, method, operation).TypeName
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

func renderCGONativeServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context", nativeGoRequestParams(g, method.Contract.Native.RequestFields), ") (", nativeGoResponseReturns(g, method.Contract.Native.ResponseFields), ") {")
	g.P("if a == nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, errorNames.CallbacksNil))
	g.P("}")
	g.P("callback := a.", method.GoName, "Callback")
	g.P("if callback == nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, cgoNativeServerMethodUnimplementedError(service, method)))
	g.P("}")
	encoderName := nativeCGOServerRequestEncoderName(service, method)
	g.P(nativeCGOServerRequestEncoderAssignResult(encoderName), " := ", encoderName, "(", nativeCGOServerRequestEncoderCallArgs(method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	g.P("defer ", nativeCGOServerRequestEncoderReleaseCall(encoderName))
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	unaryABI := nativeCGOServerOperationABI(abi, method, NativeCOperationUnary)
	g.P("errID := int32(C.", nativeCGOServerTrampolineName(service, method), "(callback, ", nativeCGOServerRequestEncoderArgList(unaryABI.Params, "", encoderName), "))")
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

func renderCGONativeServerClientStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	clientType := nativeRuntimeStreamingClientInterface(method, servicePackage)
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", clientType, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil || a.", method.GoName, "Send == nil || a.", method.GoName, "Finish == nil || a.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", cgoNativeServerMethodUnimplementedError(service, method))
	g.P("}")
	g.P("var stream C.int32_t")
	startABI := nativeCGOServerOperationABI(abi, method, NativeCOperationStart)
	g.P("errID := int32(C.", nativeCGOServerClientStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start", nativeCGOServerGoABICallSuffix(startABI.Params, "&stream"), "))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeClientStreamingClient{send: a.", method.GoName, "Send, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: stream}, nil")
	g.P("}")
	g.P()

	sendABI := nativeCGOServerOperationABI(abi, method, NativeCOperationSend)
	finishABI := nativeCGOServerOperationABI(abi, method, NativeCOperationFinish)
	cancelABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCancel)
	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeClientStreamingClient struct {")
	g.P("send C.", sendABI.TypeName)
	g.P("finish C.", finishABI.TypeName)
	g.P("cancel C.", cancelABI.TypeName)
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerClientStreamSend(g, service, abi, method, servicePackage)
	renderCGONativeServerClientStreamFinish(g, service, abi, method, servicePackage)
	renderCGONativeServerClientStreamCancel(g, service, abi, method)
}

func renderCGONativeServerClientStreamSend(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan, servicePackage string) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamingClient"
	requestType := servicePackage + method.RenderPlan.Symbols.NativeStreamRequestType
	g.P("func (s *", receiver, ") Send(ctx context.Context, reqData ", requestType, ") error {")
	encoderName := nativeCGOServerClientStreamRequestEncoderName(service, method)
	g.P(nativeCGOServerRequestEncoderAssignResult(encoderName), " := ", encoderName, "(", nativeExportedEnvelopeFieldArgs("reqData", method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer ", nativeCGOServerRequestEncoderReleaseCall(encoderName))
	sendABI := nativeCGOServerOperationABI(abi, method, NativeCOperationSend)
	g.P("errID := int32(C.", nativeCGOServerClientStreamSendTrampolineName(service, method), "(s.send", nativeCGOServerRequestEncoderCallSuffix(sendABI.Params, "s.stream", encoderName), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamFinish(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan, servicePackage string) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamingClient"
	responseType := servicePackage + method.RenderPlan.Symbols.NativeStreamResponseType
	g.P("func (s *", receiver, ") Finish(ctx context.Context) (", responseType, ", error) {")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	finishABI := nativeCGOServerOperationABI(abi, method, NativeCOperationFinish)
	g.P("errID := int32(C.", nativeCGOServerClientStreamFinishTrampolineName(service, method), "(s.finish", nativeCGOServerGoABICallSuffix(finishABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", responseType, "{}, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return ", responseType, "{}, callbackErr")
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
	g.P("return ", responseType, "{}, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return ", responseType, "{}, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", responseType, "{}, err")
	g.P("}")
	g.P("return ", nativeResponseEnvelopeLiteralFromResults(method, servicePackage), ", nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamCancel(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamingClient"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	cancelABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCancel)
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
	if len(method.Contract.Native.ResponseFields) == 0 {
		g.P("_, err := session.Finish(ctx)")
	} else {
		g.P("resp, err := session.Finish(ctx)")
	}
	g.P("if err != nil { return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"), " }")
	if len(method.Contract.Native.ResponseFields) == 0 {
		g.P("return nil")
	} else {
		g.P("return ", nativeExportedEnvelopeFieldArgs("resp", method.Contract.Native.ResponseFields), ", nil")
	}
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	if requestNames == "" {
		g.P("if err := session.Send(ctx, ", servicePackage, method.RenderPlan.Symbols.NativeStreamRequestType, "{}); err != nil {")
	} else {
		g.P("if err := session.Send(ctx, ", servicePackage, method.RenderPlan.Symbols.NativeStreamRequestType, "{", nativeExportedEnvelopeLiteralFromLocals(method.Contract.Native.RequestFields), "}); err != nil {")
	}
	g.P("_ = session.Cancel(ctx)")
	g.P("return ", nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"))
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	clientType := nativeRuntimeStreamingClientInterface(method, servicePackage)
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context", nativeGoRequestParams(g, method.Contract.Native.RequestFields), ") (", clientType, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil || a.", method.GoName, "Recv == nil || a.", method.GoName, "Finish == nil || a.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", cgoNativeServerMethodUnimplementedError(service, method))
	g.P("}")
	encoderName := nativeCGOServerServerStreamRequestEncoderName(service, method)
	g.P(nativeCGOServerRequestEncoderAssignResult(encoderName), " := ", encoderName, "(", nativeCGOServerRequestEncoderCallArgs(method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("defer ", nativeCGOServerRequestEncoderReleaseCall(encoderName))
	g.P("var stream C.int32_t")
	startABI := nativeCGOServerOperationABI(abi, method, NativeCOperationStart)
	g.P("errID := int32(C.", nativeCGOServerServerStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start", nativeCGOServerRequestEncoderCallSuffix(startABI.Params, "&stream", encoderName), "))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeServerStreamingClient{recv: a.", method.GoName, "Recv, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: stream}, nil")
	g.P("}")
	g.P()

	recvABI := nativeCGOServerOperationABI(abi, method, NativeCOperationRecv)
	finishABI := nativeCGOServerOperationABI(abi, method, NativeCOperationFinish)
	cancelABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCancel)
	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeServerStreamingClient struct {")
	g.P("recv C.", recvABI.TypeName)
	g.P("finish C.", finishABI.TypeName)
	g.P("cancel C.", cancelABI.TypeName)
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerServerStreamRecv(g, service, abi, method, servicePackage)
	renderCGONativeServerServerStreamFinish(g, service, abi, method)
	renderCGONativeServerServerStreamCancel(g, service, abi, method)
}

func renderCGONativeServerServerStreamRecv(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan, servicePackage string) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamingClient"
	responseType := servicePackage + method.RenderPlan.Symbols.NativeStreamResponseType
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", responseType, ", error) {")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	recvABI := nativeCGOServerOperationABI(abi, method, NativeCOperationRecv)
	g.P("errID := int32(C.", nativeCGOServerServerStreamRecvTrampolineName(service, method), "(s.recv", nativeCGOServerGoABICallSuffix(recvABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", responseType, "{}, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return ", responseType, "{}, callbackErr")
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
	g.P("return ", responseType, "{}, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return ", responseType, "{}, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", responseType, "{}, err")
	g.P("}")
	g.P("return ", nativeResponseEnvelopeLiteralFromResults(method, servicePackage), ", nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamFinish(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamingClient"
	g.P("func (s *", receiver, ") Finish(ctx context.Context) error {")
	finishABI := nativeCGOServerOperationABI(abi, method, NativeCOperationFinish)
	g.P("errID := int32(C.", nativeCGOServerServerStreamFinishTrampolineName(service, method), "(s.finish", nativeCGOServerGoABICallSuffix(finishABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamCancel(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamingClient"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	cancelABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCancel)
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
	if len(method.Contract.Native.ResponseFields) == 0 {
		g.P("_, err, stopped := ", lowerInitial(service.GoName), "AwaitCGONativeRecv(ctx, stream.FinishRequested(), func() (", servicePackage, method.RenderPlan.Symbols.NativeStreamResponseType, ", error) { return session.Recv(ctx) }, func() error { return session.Finish(ctx) }, func() error { return session.Cancel(ctx) })")
	} else {
		g.P("resp, err, stopped := ", lowerInitial(service.GoName), "AwaitCGONativeRecv(ctx, stream.FinishRequested(), func() (", servicePackage, method.RenderPlan.Symbols.NativeStreamResponseType, ", error) { return session.Recv(ctx) }, func() error { return session.Finish(ctx) }, func() error { return session.Cancel(ctx) })")
	}
	g.P("if stopped {")
	g.P("return err")
	g.P("}")
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
		g.P("if err := stream.Send(ctx, ", nativeExportedEnvelopeFieldArgs("resp", method.Contract.Native.ResponseFields), "); err != nil {")
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

func renderCGONativeServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	clientType := nativeRuntimeStreamingClientInterface(method, servicePackage)
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", clientType, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil || a.", method.GoName, "Send == nil || a.", method.GoName, "Recv == nil || a.", method.GoName, "CloseSend == nil || a.", method.GoName, "Finish == nil || a.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", cgoNativeServerMethodUnimplementedError(service, method))
	g.P("}")
	g.P("var stream C.int32_t")
	startABI := nativeCGOServerOperationABI(abi, method, NativeCOperationStart)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start", nativeCGOServerGoABICallSuffix(startABI.Params, "&stream"), "))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeBidiStreamingClient{send: a.", method.GoName, "Send, recv: a.", method.GoName, "Recv, closeSend: a.", method.GoName, "CloseSend, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: stream}, nil")
	g.P("}")
	g.P()

	sendABI := nativeCGOServerOperationABI(abi, method, NativeCOperationSend)
	recvABI := nativeCGOServerOperationABI(abi, method, NativeCOperationRecv)
	closeSendABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCloseSend)
	finishABI := nativeCGOServerOperationABI(abi, method, NativeCOperationFinish)
	cancelABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCancel)
	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeBidiStreamingClient struct {")
	g.P("send C.", sendABI.TypeName)
	g.P("recv C.", recvABI.TypeName)
	g.P("closeSend C.", closeSendABI.TypeName)
	g.P("finish C.", finishABI.TypeName)
	g.P("cancel C.", cancelABI.TypeName)
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerBidiStreamSend(g, service, abi, method, servicePackage)
	renderCGONativeServerBidiStreamRecv(g, service, abi, method, servicePackage)
	renderCGONativeServerBidiStreamCloseSend(g, service, abi, method)
	renderCGONativeServerBidiStreamFinish(g, service, abi, method)
	renderCGONativeServerBidiStreamCancel(g, service, abi, method)
}

func renderCGONativeServerBidiStreamSend(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan, servicePackage string) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamingClient"
	requestType := servicePackage + method.RenderPlan.Symbols.NativeStreamRequestType
	g.P("func (s *", receiver, ") Send(ctx context.Context, reqData ", requestType, ") error {")
	encoderName := nativeCGOServerBidiStreamRequestEncoderName(service, method)
	g.P(nativeCGOServerRequestEncoderAssignResult(encoderName), " := ", encoderName, "(", nativeExportedEnvelopeFieldArgs("reqData", method.Contract.Native.RequestFields), ")")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer ", nativeCGOServerRequestEncoderReleaseCall(encoderName))
	sendABI := nativeCGOServerOperationABI(abi, method, NativeCOperationSend)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamSendTrampolineName(service, method), "(s.send", nativeCGOServerRequestEncoderCallSuffix(sendABI.Params, "s.stream", encoderName), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamRecv(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan, servicePackage string) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamingClient"
	responseType := servicePackage + method.RenderPlan.Symbols.NativeStreamResponseType
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", responseType, ", error) {")
	renderCGONativeServerResponseLocals(g, method.Contract.Native.ResponseFields)
	recvABI := nativeCGOServerOperationABI(abi, method, NativeCOperationRecv)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamRecvTrampolineName(service, method), "(s.recv", nativeCGOServerGoABICallSuffix(recvABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(", nativeCGOServerFlatOutputValueArgs(method.Contract.Native.ResponseFields), ")")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return ", responseType, "{}, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return ", responseType, "{}, callbackErr")
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
	g.P("return ", responseType, "{}, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return ", responseType, "{}, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return ", responseType, "{}, err")
	g.P("}")
	g.P("return ", nativeResponseEnvelopeLiteralFromResults(method, servicePackage), ", nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamCloseSend(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamingClient"
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	closeSendABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCloseSend)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), "(s.closeSend", nativeCGOServerGoABICallSuffix(closeSendABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamFinish(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamingClient"
	g.P("func (s *", receiver, ") Finish(ctx context.Context) error {")
	finishABI := nativeCGOServerOperationABI(abi, method, NativeCOperationFinish)
	g.P("errID := int32(C.", nativeCGOServerBidiStreamFinishTrampolineName(service, method), "(s.finish", nativeCGOServerGoABICallSuffix(finishABI.Params, "s.stream"), "))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamCancel(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamingClient"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	cancelABI := nativeCGOServerOperationABI(abi, method, NativeCOperationCancel)
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
		g.P("if err := session.Send(ctx, ", servicePackage, method.RenderPlan.Symbols.NativeStreamRequestType, "{}); err != nil {")
	} else {
		g.P("if err := session.Send(ctx, ", servicePackage, method.RenderPlan.Symbols.NativeStreamRequestType, "{", nativeExportedEnvelopeLiteralFromLocals(method.Contract.Native.RequestFields), "}); err != nil {")
	}
	g.P("sendDone <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("for {")
	if len(method.Contract.Native.ResponseFields) == 0 {
		g.P("_, err, stopped := ", lowerInitial(service.GoName), "AwaitCGONativeRecv(ctx, stream.FinishRequested(), func() (", servicePackage, method.RenderPlan.Symbols.NativeStreamResponseType, ", error) { return session.Recv(ctx) }, func() error { return session.Finish(ctx) }, func() error { return session.Cancel(ctx) })")
	} else {
		g.P("resp, err, stopped := ", lowerInitial(service.GoName), "AwaitCGONativeRecv(ctx, stream.FinishRequested(), func() (", servicePackage, method.RenderPlan.Symbols.NativeStreamResponseType, ", error) { return session.Recv(ctx) }, func() error { return session.Finish(ctx) }, func() error { return session.Cancel(ctx) })")
	}
	g.P("if stopped {")
	g.P("return errors.Join(err, <-sendDone)")
	g.P("}")
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
		g.P("if err := stream.Send(ctx, ", nativeExportedEnvelopeFieldArgs("resp", method.Contract.Native.ResponseFields), "); err != nil {")
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

func renderCGONativeServerStreamingFallback(g *protogen.GeneratedFile, adapterName string, method runtimeMethodProjection) {
	g.P("func (a *", adapterName, ") ", method.Symbols.NativeEntryMethod, "(ctx context.Context", method.Native.EntryArgs, ")", method.Native.EntryResult, " {")
	if method.Stream.Streaming {
		g.P("return nil, ", `errors.New("rpccgo: `, method.Identity.SourceFullName, ` native server method is not implemented")`)
	} else if method.Native.EntryResult == " error" {
		g.P("return ", `errors.New("rpccgo: `, method.Identity.SourceFullName, ` native server method is not implemented")`)
	} else {
		g.P("return nil, ", `errors.New("rpccgo: `, method.Identity.SourceFullName, ` native server method is not implemented")`)
	}
	g.P("}")
	g.P()
}

func cgoNativeServerMethodUnimplementedError(service ServicePlan, method MethodPlan) string {
	return `errors.New("rpccgo: ` + service.GoName + `.` + method.GoName + ` native server method is not implemented")`
}

func renderCGONativeServerFlatRequestEncoder(g *protogen.GeneratedFile, name, requestName string, fields []FieldPlan, errorNames nativeServerCGOErrorNames) {
	renderCGONativeServerRequestType(g, requestName, fields)
	g.P("func ", name, "(", nativeCGOServerRequestEncoderArgs(g, fields), ") ", nativeCGOServerRequestEncoderReturns(requestName), " {")
	for _, field := range fields {
		types := nativeCGOServerGoFieldType(field)
		for i, arg := range nativeCGOServerGoFieldArgs(field, false) {
			g.P("var ", arg, " ", types[i])
		}
	}
	g.P("var pinned []uintptr")
	for _, field := range fields {
		renderCGONativeServerRequestFieldEncode(g, field, errorNames)
	}
	renderCGONativeServerRequestEncoderResult(g, requestName, fields)
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestFieldEncode(g *protogen.GeneratedFile, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	name := lowerInitial(field.GoName)
	errorReturn := nativeCGOServerRequestEncoderErrorReturn("err")
	unsupportedReturn := nativeCGOServerRequestEncoderErrorReturn(errorNames.UnsupportedField)
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("if ", name, " {")
		g.P(name, "Value = 1")
		g.P("}")
	case NativeABIShapeBoolByteBufferWrapper:
		g.P(name, "Values := ", name, ".SafeSlice()")
		g.P(name, "LenValue, err := rpcruntime.LengthToInt32(len(", name, "Values))")
		g.P("if err != nil {")
		renderCGONativeServerRequestEncoderReleasePinned(g)
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
		renderCGONativeServerRequestEncoderReleasePinned(g)
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
		renderCGONativeServerRequestEncoderReleasePinned(g)
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
			renderCGONativeServerRequestEncoderReleasePinned(g)
			g.P("return ", unsupportedReturn)
		}
		g.P("if err != nil {")
		renderCGONativeServerRequestEncoderReleasePinned(g)
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
			renderCGONativeServerRequestEncoderReleasePinned(g)
			g.P("return ", errorReturn)
			g.P("}")
			g.P("_, ", name, "PtrValue, err := rpcruntime.PinString(", name, ".SafeString())")
			g.P("if err != nil {")
			renderCGONativeServerRequestEncoderReleasePinned(g)
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
			renderCGONativeServerRequestEncoderReleasePinned(g)
			g.P("return ", errorReturn)
			g.P("}")
			g.P(name, "PtrValue, err := rpcruntime.PinBytes(", name, "Bytes)")
			g.P("if err != nil {")
			renderCGONativeServerRequestEncoderReleasePinned(g)
			g.P("return ", errorReturn)
			g.P("}")
			g.P("if ", name, "PtrValue != 0 {")
			g.P("pinned = append(pinned, ", name, "PtrValue)")
			g.P("}")
			g.P(name, "Ptr = C.uintptr_t(", name, "PtrValue)")
			g.P(name, "Len = C.int32_t(", name, "LenValue)")
		default:
			renderCGONativeServerRequestEncoderReleasePinned(g)
			g.P("return ", unsupportedReturn)
		}
	default:
		renderCGONativeServerRequestEncoderReleasePinned(g)
		g.P("return ", unsupportedReturn)
	}
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
	fieldName := lowerInitial(field.GoName)
	name := fieldName + "Result"
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P(name, " := ", fieldName, "Value != 0")
	case NativeABIShapeBoolByteBufferWrapper:
		renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "byte", "rpcruntime.NewRpcBoolRepeatChecked")
		g.P(name, " := ", name, "Wrapper.SafeSlice()")
	case NativeABIShapeRepeated:
		switch field.Kind {
		case FieldKindSignedInt32:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "int32", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindUnsignedInt32:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "uint32", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindSignedInt64:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "int64", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindUnsignedInt64:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "uint64", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindFloat:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "float32", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindDouble:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "float64", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, " := ", name, "Wrapper.SafeSlice()")
		case FieldKindEnum:
			renderCGONativeServerResponseRepeatDecode(g, fields, field, name, "int32", "rpcruntime.NewRpcRepeatChecked")
			g.P(name, "Raw := ", name, "Wrapper.SafeSlice()")
			g.P(name, " := make([]", nativeGoEnumType(g, field), ", len(", name, "Raw))")
			g.P("for i := range ", name, "Raw {")
			g.P(name, "[i] = ", nativeGoEnumType(g, field), "(", name, "Raw[i])")
			g.P("}")
		default:
			g.P("return ", nativeGoZeroReturns(fields, errorNames.UnsupportedField))
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
			renderCGONativeServerResponseTextDecode(g, fields, field, name, "String", "SafeString")
		case FieldKindBytes, FieldKindMessage:
			renderCGONativeServerResponseTextDecode(g, fields, field, name, "Bytes", "SafeBytes")
		default:
			g.P("return ", nativeGoZeroReturns(fields, errorNames.UnsupportedField))
		}
	default:
		g.P("return ", nativeGoZeroReturns(fields, errorNames.UnsupportedField))
	}
}

func renderCGONativeServerResponseRepeatDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, name, elemType, ctor string) {
	fieldName := lowerInitial(field.GoName)
	g.P("if _, err := rpcruntime.LengthFromInt32(int32(", fieldName, "Len)); err != nil {")
	g.P(`return `, nativeGoZeroReturns(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P(name, "Wrapper, err := ", ctor, "((*", elemType, ")(unsafe.Pointer(uintptr(", fieldName, "Ptr))), int32(", fieldName, "Len), false)")
	g.P("if err != nil {")
	g.P(`return `, nativeGoZeroReturns(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
}

func renderCGONativeServerResponseTextDecode(g *protogen.GeneratedFile, fields []FieldPlan, field FieldPlan, name, wrapper, safeMethod string) {
	fieldName := lowerInitial(field.GoName)
	g.P("if _, err := rpcruntime.LengthFromInt32(int32(", fieldName, "Len)); err != nil {")
	g.P(`return `, nativeGoZeroReturns(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P(fieldName, "Wrapper, err := rpcruntime.NewRpc", wrapper, "Checked((*byte)(unsafe.Pointer(uintptr(", fieldName, "Ptr))), int32(", fieldName, "Len), false)")
	g.P("if err != nil {")
	g.P(`return `, nativeGoZeroReturns(fields, `fmt.Errorf("`+field.FullName+`: %w", err)`))
	g.P("}")
	g.P(name, " := ", fieldName, "Wrapper.", safeMethod, "()")
}

func renderCGONativeServerRegistration(g *protogen.GeneratedFile, service ServicePlan, abi nativeCServiceABI, errorNames nativeServerCGOErrorNames, servicePackage string) {
	adapterVarName := lowerInitial(service.GoName) + "CGONativeServerAdapter"
	adapterTypeName := lowerInitial(service.GoName) + "CGONativeAdapter"
	registerABI := abi.Register
	renderCGOExportDoc(g, registerABI.Symbol, "registers cgo native callbacks as the current server for "+service.FullName+".")
	g.P("//export ", registerABI.Symbol)
	g.P("func ", registerABI.Symbol, "(", nativeCABIRegisterParamList(registerABI.Params), ") ", registerABI.Return.CGoType, " {")
	g.P(adapterVarName, "Mu.Lock()")
	g.P("defer ", adapterVarName, "Mu.Unlock()")
	g.P("next := ", adapterVarName, "ForRegister()")
	g.P("var registerErr error")
	for _, method := range service.Methods {
		renderCGONativeServerServiceMethodAssignment(g, service, method, "next", errorNames)
	}
	g.P("if err := ", servicePackage, "Register", service.GoName, "CGONativeServer(next); err != nil { return C.int32_t(rpcruntime.StoreError(err)) }")
	g.P(adapterVarName, " = next")
	g.P("if registerErr != nil { return C.int32_t(rpcruntime.StoreError(registerErr)) }")
	g.P("return 0")
	g.P("}")
	g.P()
	for _, method := range service.Methods {
		renderCGONativeServerMethodRegistration(g, service, method, registerABI, adapterVarName, errorNames, servicePackage)
	}
	g.P("func ", adapterVarName, "ForRegister() *", adapterTypeName, " {")
	g.P("registered, err := ", servicePackage, "Load", service.GoName, "RegisteredServer()")
	g.P("if err == nil && registered.Kind == rpcruntime.ServerKindCGONative {")
	g.P("if current, ok := registered.Server.(*", adapterTypeName, "); ok {")
	g.P("next := *current")
	g.P("return &next")
	g.P("}")
	g.P("}")
	g.P("return &", adapterTypeName, "{}")
	g.P("}")
	g.P()
}

func renderCGONativeServerMethodRegistration(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, registerABI COperationABI, adapterVarName string, errorNames nativeServerCGOErrorNames, servicePackage string) {
	exportName := registerABI.Symbol + "_" + method.GoName
	renderCGOExportDoc(g, exportName, "registers cgo native callbacks for "+method.FullName+".")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(", nativeCGOServerMethodRegisterParamList(service, method), ") ", registerABI.Return.CGoType, " {")
	g.P(adapterVarName, "Mu.Lock()")
	g.P("defer ", adapterVarName, "Mu.Unlock()")
	g.P("next := ", adapterVarName, "ForRegister()")
	g.P("var registerErr error")
	renderCGONativeServerMethodAssignment(g, service, method, "next", errorNames)
	g.P("if err := ", servicePackage, "Register", service.GoName, "CGONativeServer(next); err != nil { return C.int32_t(rpcruntime.StoreError(err)) }")
	g.P(adapterVarName, " = next")
	g.P("if registerErr != nil { return C.int32_t(rpcruntime.StoreError(registerErr)) }")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderCGONativeServerServiceMethodAssignment(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, target string, errorNames nativeServerCGOErrorNames) {
	operations, _ := NativeCOperationsForMethod(method)
	callbackNames := make([]string, 0, len(operations))
	fieldNames := make([]string, 0, len(operations))
	for _, operation := range operations {
		callbackNames = append(callbackNames, nativeCGOServerRegisterCallbackParamName(method, operation))
		fieldName := upperInitial(nativeCABIRegisterParamName(operation))
		if operation == NativeCOperationUnary {
			fieldName = "Callback"
		}
		fieldNames = append(fieldNames, fieldName)
	}
	if method.Streaming == StreamingKindUnary {
		g.P("if ", callbackNames[0], " != nil {")
		g.P(target, ".", method.GoName, fieldNames[0], " = ", callbackNames[0])
		g.P("}")
		return
	}
	allNil := make([]string, 0, len(callbackNames))
	allPresent := make([]string, 0, len(callbackNames))
	for _, callbackName := range callbackNames {
		allNil = append(allNil, callbackName+" == nil")
		allPresent = append(allPresent, callbackName+" != nil")
	}
	g.P("if ", strings.Join(allNil, " && "), " {")
	g.P("// Preserve existing callbacks for methods omitted from a service-level update.")
	g.P("} else if ", strings.Join(allPresent, " && "), " {")
	for i, fieldName := range fieldNames {
		g.P(target, ".", method.GoName, fieldName, " = ", callbackNames[i])
	}
	g.P("} else {")
	for _, fieldName := range fieldNames {
		g.P(target, ".", method.GoName, fieldName, " = nil")
	}
	g.P(`registerErr = errors.Join(registerErr, fmt.Errorf("%w: %s", `, errorNames.StreamPartiallyRegistered, `, "`, method.FullName, `"))`)
	g.P("}")
}

func renderCGONativeServerMethodAssignment(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, target string, errorNames nativeServerCGOErrorNames) {
	operations, _ := NativeCOperationsForMethod(method)
	callbackNames := make([]string, 0, len(operations))
	fieldNames := make([]string, 0, len(operations))
	for _, operation := range operations {
		callbackNames = append(callbackNames, nativeCGOServerRegisterCallbackParamName(method, operation))
		fieldName := upperInitial(nativeCABIRegisterParamName(operation))
		if operation == NativeCOperationUnary {
			fieldName = "Callback"
		}
		fieldNames = append(fieldNames, fieldName)
	}
	if method.Streaming == StreamingKindUnary {
		g.P(target, ".", method.GoName, fieldNames[0], " = ", callbackNames[0])
		return
	}
	allNil := make([]string, 0, len(callbackNames))
	allPresent := make([]string, 0, len(callbackNames))
	for _, callbackName := range callbackNames {
		allNil = append(allNil, callbackName+" == nil")
		allPresent = append(allPresent, callbackName+" != nil")
	}
	g.P("if ", strings.Join(allNil, " && "), " {")
	for _, fieldName := range fieldNames {
		g.P(target, ".", method.GoName, fieldName, " = nil")
	}
	g.P("} else if ", strings.Join(allPresent, " && "), " {")
	for i, fieldName := range fieldNames {
		g.P(target, ".", method.GoName, fieldName, " = ", callbackNames[i])
	}
	g.P("} else {")
	for _, fieldName := range fieldNames {
		g.P(target, ".", method.GoName, fieldName, " = nil")
	}
	g.P(`registerErr = errors.Join(registerErr, fmt.Errorf("%w: %s", `, errorNames.StreamPartiallyRegistered, `, "`, method.FullName, `"))`)
	g.P("}")
}

func nativeCGOServerMethodRegisterParamList(service ServicePlan, method MethodPlan) string {
	operations, _ := NativeCOperationsForMethod(method)
	params := make([]string, 0, len(operations))
	for _, operation := range operations {
		params = append(params, nativeCGOServerRegisterCallbackParamName(method, operation)+" C."+nativeCGOServerCallbackTypeName(service, method, operation))
	}
	return strings.Join(params, ", ")
}

func nativeCGOServerRegisterCallbackParamName(method MethodPlan, operation NativeCOperation) string {
	return lowerInitial(method.GoName) + upperInitial(nativeCABIRegisterParamName(operation))
}

func nativeCGOServerCallbackTypeName(service ServicePlan, method MethodPlan, operation NativeCOperation) string {
	switch method.Streaming {
	case StreamingKindUnary:
		return nativeCGOServerCallbackName(service, method)
	case StreamingKindClientStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerClientStreamStartCallbackName(service, method)
		case NativeCOperationSend:
			return nativeCGOServerClientStreamSendCallbackName(service, method)
		case NativeCOperationFinish:
			return nativeCGOServerClientStreamFinishCallbackName(service, method)
		case NativeCOperationCancel:
			return nativeCGOServerClientStreamCancelCallbackName(service, method)
		}
	case StreamingKindServerStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerServerStreamStartCallbackName(service, method)
		case NativeCOperationRecv:
			return nativeCGOServerServerStreamRecvCallbackName(service, method)
		case NativeCOperationFinish:
			return nativeCGOServerServerStreamFinishCallbackName(service, method)
		case NativeCOperationCancel:
			return nativeCGOServerServerStreamCancelCallbackName(service, method)
		}
	case StreamingKindBidiStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerBidiStreamStartCallbackName(service, method)
		case NativeCOperationSend:
			return nativeCGOServerBidiStreamSendCallbackName(service, method)
		case NativeCOperationRecv:
			return nativeCGOServerBidiStreamRecvCallbackName(service, method)
		case NativeCOperationCloseSend:
			return nativeCGOServerBidiStreamCloseSendCallbackName(service, method)
		case NativeCOperationFinish:
			return nativeCGOServerBidiStreamFinishCallbackName(service, method)
		case NativeCOperationCancel:
			return nativeCGOServerBidiStreamCancelCallbackName(service, method)
		}
	}
	return ""
}

func nativeCGOServerCallbackTrampolineName(service ServicePlan, method MethodPlan, operation NativeCOperation) string {
	switch method.Streaming {
	case StreamingKindClientStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerClientStreamStartTrampolineName(service, method)
		case NativeCOperationSend:
			return nativeCGOServerClientStreamSendTrampolineName(service, method)
		case NativeCOperationFinish:
			return nativeCGOServerClientStreamFinishTrampolineName(service, method)
		case NativeCOperationCancel:
			return nativeCGOServerClientStreamCancelTrampolineName(service, method)
		}
	case StreamingKindServerStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerServerStreamStartTrampolineName(service, method)
		case NativeCOperationRecv:
			return nativeCGOServerServerStreamRecvTrampolineName(service, method)
		case NativeCOperationFinish:
			return nativeCGOServerServerStreamFinishTrampolineName(service, method)
		case NativeCOperationCancel:
			return nativeCGOServerServerStreamCancelTrampolineName(service, method)
		}
	case StreamingKindBidiStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerBidiStreamStartTrampolineName(service, method)
		case NativeCOperationSend:
			return nativeCGOServerBidiStreamSendTrampolineName(service, method)
		case NativeCOperationRecv:
			return nativeCGOServerBidiStreamRecvTrampolineName(service, method)
		case NativeCOperationCloseSend:
			return nativeCGOServerBidiStreamCloseSendTrampolineName(service, method)
		case NativeCOperationFinish:
			return nativeCGOServerBidiStreamFinishTrampolineName(service, method)
		case NativeCOperationCancel:
			return nativeCGOServerBidiStreamCancelTrampolineName(service, method)
		}
	}
	return ""
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
	return lowerInitial(service.GoName) + method.GoName + "CGONativeUnaryRequest"
}

func nativeCGOServerGoResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerGoClientStreamRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamRequest"
}

func nativeCGOServerGoClientStreamResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerGoServerStreamRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamRequest"
}

func nativeCGOServerGoServerStreamResponseCarrierName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerGoBidiStreamRequestCarrierName(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamRequest"
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
	return nativeCGOServerGoClientStreamRequestCarrierName(service, method)
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
	return nativeCGOServerGoServerStreamRequestCarrierName(service, method)
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
	return nativeCGOServerGoBidiStreamRequestCarrierName(service, method)
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

func nativeServerCGONeedsUnsafe(service ServicePlan) bool {
	for _, method := range service.Methods {
		for _, field := range method.Contract.Native.RequestFields {
			if nativeServerCGOFieldUsesUnsafe(field) {
				return true
			}
		}
		for _, field := range method.Contract.Native.ResponseFields {
			if nativeServerCGOFieldUsesUnsafe(field) {
				return true
			}
		}
	}
	return false
}

func nativeServerCGOFieldUsesUnsafe(field FieldPlan) bool {
	if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
		return true
	}
	return (field.Native.Shape == NativeABIShapeScalar || field.Native.Shape == NativeABIShapeMessageBytes) &&
		(field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage)
}

type nativeServerCGOErrorNames struct {
	CallbacksNil              string
	UnaryCallbackMissing      string
	UnsupportedField          string
	StreamPartiallyRegistered string
}

func nativeServerCGOErrorNamesFor(service ServicePlan) nativeServerCGOErrorNames {
	prefix := lowerInitial(service.GoName)
	return nativeServerCGOErrorNames{
		CallbacksNil:              prefix + "CGONativeServerCallbacksNil",
		UnaryCallbackMissing:      prefix + "CGONativeServerUnaryCallbackMissing",
		UnsupportedField:          prefix + "CGONativeServerUnsupportedField",
		StreamPartiallyRegistered: prefix + "CGONativeServerStreamPartiallyRegistered",
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
		if otherService.FullName != service.FullName && otherService.HasArtifact(GeneratedArtifactKindCGONativeServer) {
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
		lowerInitial(service.GoName) + "CGONativeAdapter": service.FullName + " entry",
		"Register" + service.GoName + "CGONativeServer":   service.FullName + " registration",
		nativeCGOServerErrorIDHelperName(service):         service.FullName + " error id helper",
		errorNames.CallbacksNil:                           errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                   errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                       errorNames.UnsupportedField,
		errorNames.StreamPartiallyRegistered:              errorNames.StreamPartiallyRegistered,
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
		lowerInitial(service.GoName) + "CGONativeAdapter": service.FullName + " entry",
		"Register" + service.GoName + "CGONativeServer":   service.FullName + " registration",
		nativeCGOServerErrorIDHelperName(service):         service.FullName + " error id helper",
		errorNames.CallbacksNil:                           errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                   errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                       errorNames.UnsupportedField,
		errorNames.StreamPartiallyRegistered:              errorNames.StreamPartiallyRegistered,
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
