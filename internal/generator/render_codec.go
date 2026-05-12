package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func RenderCodecFiles(plugin *protogen.Plugin, plan FilePlan) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}

	for _, service := range plan.Services {
		file := BuildCodecFilePlan(plan, service)
		if !file.Enabled {
			continue
		}
		renderCodecFile(plugin, plan, service, file)
	}
	return nil
}

func renderCodecFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	if codecNeedsRuntime(service) {
		g.P(`rpcruntime "rpccgo/rpcruntime"`)
	}
	if codecNeedsUnsafe(service) {
		g.P(`unsafe "unsafe"`)
	}
	g.P(`proto "google.golang.org/protobuf/proto"`)
	g.P(")")
	g.P()
	g.P("// rpccgo native message codec stage file for ", service.GoName)
	g.P()
	g.P("var ", lowerInitial(service.GoName), `NativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")`)
	g.P()

	for _, method := range service.Methods {
		renderCodecMethodStubs(g, service, method)
	}
}

func codecNeedsRuntime(service ServicePlan) bool {
	for _, method := range service.Methods {
		for _, field := range append(method.NativeContract.RequestFields, method.NativeContract.ResponseFields...) {
			if field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage || field.Repeated {
				return true
			}
		}
	}
	return false
}

func codecNeedsUnsafe(service ServicePlan) bool {
	return codecNeedsRuntime(service)
}

func renderCodecMethodStubs(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	requestType := nativeRuntimeMessageType(g, method.Request)
	responseType := nativeRuntimeMessageType(g, method.Response)
	requestNativeReturns := nativeGoRequestReturns(g, method.NativeContract.RequestFields)
	requestNativeArgs := nativeGoRequestParams(g, method.NativeContract.RequestFields)
	requestNativeNames := nativeGoRequestArgNames(method.NativeContract.RequestFields)
	requestNativeErrZero := nativeClientZeroReturns(method.NativeContract.RequestFields, "err")
	responseNativeReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
	responseNativeArgs := nativeGoResponseParams(g, method.NativeContract.ResponseFields)
	responseNativeNames := nativeGoResponseValueNames(method.NativeContract.ResponseFields)
	responseNativeErrZero := nativeGoZeroReturns(method.NativeContract.ResponseFields, "err")

	g.P("func ", codecMessageToNativeRequestName(service, method), "(data []byte) (", requestNativeReturns, ") {")
	g.P("var msg ", strings.TrimPrefix(requestType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P(`return `, requestNativeErrZero)
	g.P("}")
	renderCodecMessageToNativeRequestValues(g, method.NativeContract.RequestFields, "msg", requestNativeNames)
	g.P("}")
	g.P()

	g.P("func ", codecNativeRequestToMessageName(service, method), "(", strings.TrimPrefix(requestNativeArgs, ", "), ") ([]byte, error) {")
	g.P("msg := &", strings.TrimPrefix(requestType, "*"), "{}")
	renderCodecNativeRequestValuesToMessage(g, method.NativeContract.RequestFields, "msg")
	g.P("data, err := proto.Marshal(msg)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()

	g.P("func ", codecMessageToNativeResponseName(service, method), "(data []byte) (", responseNativeReturns, ") {")
	g.P("var msg ", strings.TrimPrefix(responseType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P(`return `, responseNativeErrZero)
	g.P("}")
	renderCodecMessageToNativeValues(g, method.NativeContract.ResponseFields, "msg", responseNativeNames)
	g.P("}")
	g.P()

	g.P("func ", codecNativeResponseToMessageName(service, method), "(", strings.TrimPrefix(responseNativeArgs, ", "), ") ([]byte, error) {")
	g.P("msg := &", strings.TrimPrefix(responseType, "*"), "{}")
	renderCodecNativeValuesToMessage(g, method.NativeContract.ResponseFields, "msg")
	g.P("data, err := proto.Marshal(msg)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()
}

func renderCodecMessageToNativeRequestValues(g *protogen.GeneratedFile, fields []FieldPlan, msgName, returnNames string) {
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		switch field.Kind {
		case FieldKindString:
			g.P(name, " := rpcruntime.NewRpcString(nil, 0, false)")
			g.P("if msg.", field.GoName, " != \"\" {")
			g.P("data, ptr, err := rpcruntime.PinString(msg.", field.GoName, ")")
			g.P("_ = data")
			g.P("if err != nil {")
			g.P("return ", nativeGoRequestZeroReturns(fields, "err"))
			g.P("}")
			g.P("defer rpcruntime.Release(ptr)")
			g.P(name, " = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.", field.GoName, ")), false)")
			g.P("}")
		case FieldKindBytes, FieldKindMessage:
			g.P(name, " := rpcruntime.NewRpcBytes(nil, 0, false)")
			g.P("if len(msg.", field.GoName, ") > 0 {")
			g.P("ptr, err := rpcruntime.PinBytes(msg.", field.GoName, ")")
			g.P("if err != nil {")
			g.P("return ", nativeGoRequestZeroReturns(fields, "err"))
			g.P("}")
			g.P("defer rpcruntime.Release(ptr)")
			g.P(name, " = rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(ptr)), int32(len(msg.", field.GoName, ")), false)")
			g.P("}")
		case FieldKindBool:
			if field.Repeated {
				g.P(name, "Raw := make([]byte, len(msg.", field.GoName, "))")
				g.P("for i := range msg.", field.GoName, " {")
				g.P("if msg.", field.GoName, "[i] {")
				g.P(name, "Raw[i] = 1")
				g.P("}")
				g.P("}")
				g.P(name, " := rpcruntime.NewRpcBoolRepeat(nil, 0, false)")
				g.P("if len(", name, "Raw) > 0 {")
				g.P("ptr, err := rpcruntime.PinBytes(", name, "Raw)")
				g.P("if err != nil {")
				g.P("return ", nativeGoRequestZeroReturns(fields, "err"))
				g.P("}")
				g.P("defer rpcruntime.Release(ptr)")
				g.P(name, " = rpcruntime.NewRpcBoolRepeat((*byte)(unsafe.Pointer(ptr)), int32(len(", name, "Raw)), false)")
				g.P("}")
			} else {
				g.P(name, " := msg.", field.GoName)
			}
		case FieldKindEnum:
			if field.Repeated {
				g.P(name, "Raw := make([]int32, len(msg.", field.GoName, "))")
				g.P("for i := range msg.", field.GoName, " {")
				g.P(name, "Raw[i] = int32(msg.", field.GoName, "[i])")
				g.P("}")
				g.P(name, " := rpcruntime.NewRpcRepeat[int32](nil, 0, false)")
				g.P("if len(", name, "Raw) > 0 {")
				g.P("ptr, err := rpcruntime.PinSlice(", name, "Raw)")
				g.P("if err != nil {")
				g.P("return ", nativeGoRequestZeroReturns(fields, "err"))
				g.P("}")
				g.P("defer rpcruntime.Release(ptr)")
				g.P(name, " = rpcruntime.NewRpcRepeat[int32]((*int32)(unsafe.Pointer(ptr)), int32(len(", name, "Raw)), false)")
				g.P("}")
			} else {
				g.P(name, " := msg.", field.GoName)
			}
		default:
			if field.Repeated {
				g.P(name, " := rpcruntime.NewRpcRepeat[", nativeGoScalarType(g, field), "](nil, 0, false)")
				g.P("if len(msg.", field.GoName, ") > 0 {")
				g.P("ptr, err := rpcruntime.PinSlice(msg.", field.GoName, ")")
				g.P("if err != nil {")
				g.P("return ", nativeGoRequestZeroReturns(fields, "err"))
				g.P("}")
				g.P("defer rpcruntime.Release(ptr)")
				g.P(name, " = rpcruntime.NewRpcRepeat[", nativeGoScalarType(g, field), "]((*", nativeGoScalarType(g, field), ")(unsafe.Pointer(ptr)), int32(len(msg.", field.GoName, ")), false)")
				g.P("}")
			} else {
				g.P(name, " := msg.", field.GoName)
			}
		}
	}
	if returnNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", returnNames, ", nil")
	}
}

func renderCodecMessageToNativeValues(g *protogen.GeneratedFile, fields []FieldPlan, msgName, returnNames string) {
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		switch field.Kind {
		case FieldKindString:
			g.P(name, " := msg.", field.GoName)
		case FieldKindBytes, FieldKindMessage:
			g.P(name, " := msg.", field.GoName)
		case FieldKindBool:
			g.P(name, " := msg.", field.GoName)
		case FieldKindEnum:
			if field.Repeated {
				g.P(name, "Raw := msg.", field.GoName)
				g.P(name, " := make([]", nativeGoEnumType(g, field), ", len(", name, "Raw))")
				g.P("copy(", name, ", ", name, "Raw)")
			} else {
				g.P(name, " := msg.", field.GoName)
			}
		default:
			g.P(name, " := msg.", field.GoName)
		}
	}
	if returnNames == "" {
		g.P("return nil")
	} else {
		g.P("return ", returnNames, ", nil")
	}
}

func renderCodecNativeValuesToMessage(g *protogen.GeneratedFile, fields []FieldPlan, msgName string) {
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		switch field.Kind {
		case FieldKindString:
			g.P(msgName, ".", field.GoName, " = ", name)
		case FieldKindBytes, FieldKindMessage:
			g.P(msgName, ".", field.GoName, " = ", name)
		case FieldKindBool:
			g.P(msgName, ".", field.GoName, " = ", name)
		case FieldKindEnum:
			if field.Repeated {
				g.P(msgName, ".", field.GoName, " = make([]", nativeGoEnumType(g, field), ", len(", name, "))")
				g.P("copy(", msgName, ".", field.GoName, ", ", name, ")")
			} else {
				g.P(msgName, ".", field.GoName, " = ", name)
			}
		default:
			g.P(msgName, ".", field.GoName, " = ", name)
		}
	}
}

func renderCodecNativeRequestValuesToMessage(g *protogen.GeneratedFile, fields []FieldPlan, msgName string) {
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		switch field.Kind {
		case FieldKindString:
			g.P(msgName, ".", field.GoName, " = ", name, ".SafeString()")
		case FieldKindBytes, FieldKindMessage:
			g.P(msgName, ".", field.GoName, " = ", name, ".SafeBytes()")
		case FieldKindBool:
			if field.Repeated {
				g.P(msgName, ".", field.GoName, " = ", name, ".SafeSlice()")
			} else {
				g.P(msgName, ".", field.GoName, " = ", name)
			}
		case FieldKindEnum:
			if field.Repeated {
				g.P(name, "Raw := ", name, ".SafeSlice()")
				g.P(msgName, ".", field.GoName, " = make([]", nativeGoEnumType(g, field), ", len(", name, "Raw))")
				g.P("for i := range ", name, "Raw {")
				g.P(msgName, ".", field.GoName, "[i] = ", nativeGoEnumType(g, field), "(", name, "Raw[i])")
				g.P("}")
			} else {
				g.P(msgName, ".", field.GoName, " = ", name)
			}
		default:
			if field.Repeated {
				g.P(msgName, ".", field.GoName, " = ", name, ".SafeSlice()")
			} else {
				g.P(msgName, ".", field.GoName, " = ", name)
			}
		}
	}
}

func codecMessageToNativeRequestName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "MessageToNativeRequest"
}

func codecNativeRequestToMessageName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "NativeToMessageRequest"
}

func codecMessageToNativeResponseName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "MessageToNativeResponse"
}

func codecNativeResponseToMessageName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "NativeToMessageResponse"
}
