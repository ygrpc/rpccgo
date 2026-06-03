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
		if file, ok := service.Artifact(GeneratedArtifactKindCodec); ok {
			renderCodecFile(plugin, plan, service, file)
		}
	}
	return nil
}

func renderCodecFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) {
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))

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
	g.P("// rpccgo native message codec generated file for ", service.GoName)
	g.P()
	g.P("var ", lowerInitial(service.GoName), `NativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")`)
	g.P()

	for _, method := range service.Methods {
		renderCodecMethodStubs(g, service, method)
	}
}

func codecNeedsRuntime(service ServicePlan) bool {
	for _, method := range service.Methods {
		for _, field := range append(method.Contract.Native.RequestFields, method.Contract.Native.ResponseFields...) {
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

	renderCodecMessageToNativeRequestFunction(g,
		service,
		method,
		requestType,
		method.Contract.Native.RequestFields,
	)
	renderCodecNativeToMessageFunction(g,
		codecNativeRequestToMessageName(service, method),
		requestType,
		nativeGoRequestParams(g, method.Contract.Native.RequestFields),
		method.Contract.Native.RequestFields,
		"request",
		renderCodecNativeRequestValuesToMessage,
	)
	renderCodecMessageToNativeFunction(g,
		codecMessageToNativeResponseName(service, method),
		responseType,
		nativeGoResponseReturns(g, method.Contract.Native.ResponseFields),
		nativeGoZeroReturns(method.Contract.Native.ResponseFields, "err"),
		method.Contract.Native.ResponseFields,
		nativeGoResponseValueNames(method.Contract.Native.ResponseFields),
		renderCodecMessageToNativeValues,
	)
	renderCodecNativeToMessageFunction(g,
		codecNativeResponseToMessageName(service, method),
		responseType,
		nativeGoResponseParams(g, method.Contract.Native.ResponseFields),
		method.Contract.Native.ResponseFields,
		"response",
		renderCodecNativeValuesToMessage,
	)
}

func renderCodecMessageToNativeRequestFunction(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, messageType string, fields []FieldPlan) {
	g.P("func ", codecMessageToNativeRequestName(service, method), "(data []byte) (", codecMessageToNativeRequestReturns(g, fields), ") {")
	g.P("var msg ", strings.TrimPrefix(messageType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P("return ", codecMessageToNativeRequestZeroReturns(fields, "nil", "err"))
	g.P("}")
	g.P("reqOwner := []any{&msg}")
	renderCodecMessageToNativeRequestValues(g, fields)
	g.P("return ", codecMessageToNativeRequestValueNames(fields, "reqOwner", "nil"))
	g.P("}")
	g.P()
}

func codecMessageToNativeRequestReturns(g *protogen.GeneratedFile, fields []FieldPlan) string {
	returns := make([]string, 0, len(fields)+2)
	for _, field := range fields {
		returns = append(returns, nativeGoRequestFieldType(g, field))
	}
	returns = append(returns, "any", "error")
	return strings.Join(returns, ", ")
}

func codecMessageToNativeRequestZeroReturns(fields []FieldPlan, ownerExpr, errExpr string) string {
	values := make([]string, 0, len(fields)+2)
	for _, field := range fields {
		values = append(values, nativeGoRequestZeroValue(field))
	}
	values = append(values, ownerExpr, errExpr)
	return strings.Join(values, ", ")
}

func codecMessageToNativeRequestValueNames(fields []FieldPlan, ownerExpr, errExpr string) string {
	values := make([]string, 0, len(fields)+2)
	for _, field := range fields {
		values = append(values, lowerInitial(field.GoName))
	}
	values = append(values, ownerExpr, errExpr)
	return strings.Join(values, ", ")
}

func renderCodecMessageToNativeRequestValues(g *protogen.GeneratedFile, fields []FieldPlan) {
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		msgField := "msg." + field.GoName
		rawName := lowerInitial(field.GoName) + "Raw"
		g.P("var ", name, " ", nativeGoRequestFieldType(g, field))
		switch field.Kind {
		case FieldKindString:
			g.P("if ", msgField, " != \"\" {")
			g.P(name, " = rpcruntime.NewRpcStringView(unsafe.StringData(", msgField, "), int32(len(", msgField, ")), reqOwner)")
			g.P("} else {")
			g.P(name, " = rpcruntime.EmptyRpcString()")
			g.P("}")
		case FieldKindBytes, FieldKindMessage:
			g.P("if len(", msgField, ") > 0 {")
			g.P(name, " = rpcruntime.NewRpcBytesView(unsafe.SliceData(", msgField, "), int32(len(", msgField, ")), reqOwner)")
			g.P("} else {")
			g.P(name, " = rpcruntime.EmptyRpcBytes()")
			g.P("}")
		case FieldKindBool:
			if field.Repeated {
				g.P(rawName, " := make([]byte, len(", msgField, "))")
				g.P("reqOwner = append(reqOwner, ", rawName, ")")
				g.P("for i := range ", msgField, " {")
				g.P("if ", msgField, "[i] {")
				g.P(rawName, "[i] = 1")
				g.P("}")
				g.P("}")
				g.P("if len(", rawName, ") > 0 {")
				g.P(name, " = rpcruntime.NewRpcBoolRepeatView(unsafe.SliceData(", rawName, "), int32(len(", rawName, ")), reqOwner)")
				g.P("} else {")
				g.P(name, " = rpcruntime.EmptyRpcBoolRepeat()")
				g.P("}")
			} else {
				g.P(name, " = ", msgField)
			}
		case FieldKindEnum:
			if field.Repeated {
				g.P(rawName, " := make([]int32, len(", msgField, "))")
				g.P("reqOwner = append(reqOwner, ", rawName, ")")
				g.P("for i := range ", msgField, " {")
				g.P(rawName, "[i] = int32(", msgField, "[i])")
				g.P("}")
				g.P("if len(", rawName, ") > 0 {")
				g.P(name, " = rpcruntime.NewRpcRepeatView[int32](unsafe.SliceData(", rawName, "), int32(len(", rawName, ")), reqOwner)")
				g.P("} else {")
				g.P(name, " = rpcruntime.EmptyRpcRepeat[int32]()")
				g.P("}")
			} else {
				g.P(name, " = ", msgField)
			}
		default:
			if field.Repeated {
				g.P("if len(", msgField, ") > 0 {")
				g.P(name, " = rpcruntime.NewRpcRepeatView[", nativeGoRequestRepeatElemType(g, field), "](unsafe.SliceData(", msgField, "), int32(len(", msgField, ")), reqOwner)")
				g.P("} else {")
				g.P(name, " = rpcruntime.EmptyRpcRepeat[", nativeGoRequestRepeatElemType(g, field), "]()")
				g.P("}")
			} else {
				g.P(name, " = ", msgField)
			}
		}
	}
}

func codecRequestNeedsOwner(fields []FieldPlan) bool {
	for _, field := range fields {
		if field.Kind == FieldKindString || field.Kind == FieldKindBytes || field.Kind == FieldKindMessage || field.Repeated {
			return true
		}
	}
	return false
}

func renderCodecMessageToNativeValues(g *protogen.GeneratedFile, fields []FieldPlan, msgName, returnNames, _ string) {
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

func renderCodecMessageToNativeFunction(g *protogen.GeneratedFile, name, messageType, nativeReturns, errZero string, fields []FieldPlan, returnNames string, renderValues func(*protogen.GeneratedFile, []FieldPlan, string, string, string)) {
	g.P("func ", name, "(data []byte) (", nativeReturns, ") {")
	g.P("var msg ", strings.TrimPrefix(messageType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P("return ", errZero)
	g.P("}")
	renderValues(g, fields, "msg", returnNames, errZero)
	g.P("}")
	g.P()
}

func renderCodecNativeToMessageFunction(g *protogen.GeneratedFile, name, messageType, nativeArgs string, fields []FieldPlan, label string, renderValues func(*protogen.GeneratedFile, []FieldPlan, string)) {
	g.P("func ", name, "(", strings.TrimPrefix(nativeArgs, ", "), ") ([]byte, error) {")
	g.P("msg := &", strings.TrimPrefix(messageType, "*"), "{}")
	renderValues(g, fields, "msg")
	g.P("data, err := proto.Marshal(msg)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: native `, label, ` protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()
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
