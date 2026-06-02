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
	g.P(`goruntime "runtime"`)
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
		codecMessageToNativeRequestName(service, method),
		requestType,
		nativeGoRequestParams(g, method.Contract.Native.RequestFields),
		method.Contract.Native.RequestFields,
		nativeGoResponseValueNames(method.Contract.Native.RequestFields),
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

func renderCodecMessageToNativeRequestFunction(g *protogen.GeneratedFile, name, messageType, nativeArgs string, fields []FieldPlan, argNames string) {
	g.P("func ", name, "(data []byte, fn func(", strings.TrimPrefix(nativeArgs, ", "), ") error) error {")
	g.P("var msg ", strings.TrimPrefix(messageType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P("return err")
	g.P("}")
	renderCodecMessageToNativeRequestValues(g, fields, "msg", argNames, "")
	g.P("err := fn(", argNames, ")")
	g.P("goruntime.KeepAlive(&msg)")
	for _, owner := range codecMessageToNativeRequestRawOwners(fields) {
		g.P("goruntime.KeepAlive(", owner, ")")
	}
	g.P("return err")
	g.P("}")
	g.P()
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

func renderCodecMessageToNativeRequestValues(g *protogen.GeneratedFile, fields []FieldPlan, msgName, _, _ string) {
	if codecRequestNeedsOwner(fields) {
		g.P("msgOwner := &", msgName)
	}
	for _, field := range fields {
		name := lowerInitial(field.GoName)
		switch field.Kind {
		case FieldKindString:
			g.P("var ", name, " *rpcruntime.RpcString")
			g.P("if ", msgName, ".", field.GoName, " != \"\" {")
			g.P(name, " = rpcruntime.NewRpcStringView(unsafe.StringData(", msgName, ".", field.GoName, "), int32(len(", msgName, ".", field.GoName, ")), msgOwner)")
			g.P("} else {")
			g.P(name, " = rpcruntime.EmptyRpcString()")
			g.P("}")
		case FieldKindBytes, FieldKindMessage:
			g.P("var ", name, " *rpcruntime.RpcBytes")
			g.P("if len(", msgName, ".", field.GoName, ") > 0 {")
			g.P(name, " = rpcruntime.NewRpcBytesView(unsafe.SliceData(", msgName, ".", field.GoName, "), int32(len(", msgName, ".", field.GoName, ")), msgOwner)")
			g.P("} else {")
			g.P(name, " = rpcruntime.EmptyRpcBytes()")
			g.P("}")
		case FieldKindBool:
			if field.Repeated {
				g.P(name, "Raw := make([]byte, len(", msgName, ".", field.GoName, "))")
				g.P("for i := range ", msgName, ".", field.GoName, " {")
				g.P("if ", msgName, ".", field.GoName, "[i] {")
				g.P(name, "Raw[i] = 1")
				g.P("}")
				g.P("}")
				g.P("var ", name, " *rpcruntime.RpcBoolRepeat")
				g.P("if len(", name, "Raw) > 0 {")
				g.P(name, " = rpcruntime.NewRpcBoolRepeatView(unsafe.SliceData(", name, "Raw), int32(len(", name, "Raw)), ", name, "Raw)")
				g.P("} else {")
				g.P(name, " = rpcruntime.EmptyRpcBoolRepeat()")
				g.P("}")
			} else {
				g.P(name, " := ", msgName, ".", field.GoName)
			}
		case FieldKindEnum:
			if field.Repeated {
				g.P(name, "Raw := make([]int32, len(", msgName, ".", field.GoName, "))")
				g.P("for i := range ", msgName, ".", field.GoName, " {")
				g.P(name, "Raw[i] = int32(", msgName, ".", field.GoName, "[i])")
				g.P("}")
				g.P("var ", name, " *rpcruntime.RpcRepeat[int32]")
				g.P("if len(", name, "Raw) > 0 {")
				g.P(name, " = rpcruntime.NewRpcRepeatView[int32](unsafe.SliceData(", name, "Raw), int32(len(", name, "Raw)), ", name, "Raw)")
				g.P("} else {")
				g.P(name, " = rpcruntime.EmptyRpcRepeat[int32]()")
				g.P("}")
			} else {
				g.P(name, " := ", msgName, ".", field.GoName)
			}
		default:
			if field.Repeated {
				g.P("var ", name, " *rpcruntime.RpcRepeat[", nativeGoScalarType(g, field), "]")
				g.P("if len(", msgName, ".", field.GoName, ") > 0 {")
				g.P(name, " = rpcruntime.NewRpcRepeatView[", nativeGoScalarType(g, field), "](unsafe.SliceData(", msgName, ".", field.GoName, "), int32(len(", msgName, ".", field.GoName, ")), msgOwner)")
				g.P("} else {")
				g.P(name, " = rpcruntime.EmptyRpcRepeat[", nativeGoScalarType(g, field), "]()")
				g.P("}")
			} else {
				g.P(name, " := ", msgName, ".", field.GoName)
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

func codecMessageToNativeRequestRawOwners(fields []FieldPlan) []string {
	var owners []string
	for _, field := range fields {
		if field.Repeated && (field.Kind == FieldKindBool || field.Kind == FieldKindEnum) {
			owners = append(owners, lowerInitial(field.GoName)+"Raw")
		}
	}
	return owners
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

func codecMessageToNativeRequestName(service ServicePlan, method MethodPlan) string {
	return "with" + service.GoName + method.GoName + "MessageToNativeRequest"
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
