package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeClientCGOSymbols(service); err != nil {
		return err
	}

	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(`unsafe "unsafe"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	errorName := lowerInitial(service.GoName) + "NativeClientUnsupportedField"
	g.P("var ", errorName, ` = errors.New("rpccgo: native unary client field bridge is not implemented")`)
	g.P()

	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		renderNativeUnaryClient(g, service, method, errorName)
	}
	return nil
}

func renderNativeUnaryClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError string) {
	inputName := nativeUnaryClientInputName(service, method)
	outputName := nativeUnaryClientOutputName(service, method)
	funcName := nativeUnaryClientFuncName(service, method)

	g.P("type ", inputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.RequestFields, false)
	g.P("}")
	g.P()

	g.P("type ", outputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.ResponseFields, true)
	g.P("}")
	g.P()

	g.P("func ", funcName, "(ctx context.Context, input *", inputName, ", output *", outputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if input == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary client input is nil")))`)
	g.P("}")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary client output is nil")))`)
	g.P("}")
	g.P("req, err := decode", service.GoName, method.GoName, "NativeUnaryRequest(input)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("var resp ", nativeGoMessageType(g, method.Response))
	g.P("err = ", lowerInitial(service.GoName), "Dispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter]) error {")
	g.P("var callErr error")
	g.P("resp, callErr = snapshot.Adapter.", method.GoName, "(ctx, req)")
	g.P("return callErr")
	g.P("})")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if resp == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native unary server returned nil response")))`)
	g.P("}")
	g.P("if err := encode", service.GoName, method.GoName, "NativeUnaryResponse(resp, output); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeUnaryRequestDecoder(g, service, method, inputName, unsupportedError)
	renderNativeUnaryResponseEncoder(g, service, method, outputName, unsupportedError)
}

func renderNativeClientFields(g *protogen.GeneratedFile, fields []FieldPlan, output bool) {
	for _, field := range fields {
		switch field.Native.Shape {
		case NativeABIShapeBoolByte:
			g.P(field.GoName, " int8")
		case NativeABIShapeScalar:
			renderNativeScalarField(g, field, output)
		default:
			g.P(field.GoName, " uintptr")
		}
	}
}

func renderNativeScalarField(g *protogen.GeneratedFile, field FieldPlan, output bool) {
	switch field.Kind {
	case FieldKindSignedInt32, FieldKindEnum:
		g.P(field.GoName, " int32")
	case FieldKindSignedInt64:
		g.P(field.GoName, " int64")
	case FieldKindFloat:
		g.P(field.GoName, " float32")
	case FieldKindDouble:
		g.P(field.GoName, " float64")
	case FieldKindString, FieldKindBytes:
		if output {
			g.P(field.GoName, "Ptr uintptr")
			g.P(field.GoName, "Len int32")
			return
		}
		g.P(field.GoName, "Ptr uintptr")
		g.P(field.GoName, "Len int32")
		g.P(field.GoName, "Ownership int32")
	default:
		g.P(field.GoName, " uintptr")
	}
}

func renderNativeUnaryRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, inputName, unsupportedError string) {
	g.P("func decode", service.GoName, method.GoName, "NativeUnaryRequest(input *", inputName, ") (", nativeGoMessageType(g, method.Request), ", error) {")
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.RequestFields {
		renderNativeRequestFieldDecode(g, field, unsupportedError)
	}
	g.P("return req, nil")
	g.P("}")
	g.P()
}

func renderNativeRequestFieldDecode(g *protogen.GeneratedFile, field FieldPlan, unsupportedError string) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("req.", field.GoName, " = input.", field.GoName, " != 0")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble:
			g.P("req.", field.GoName, " = input.", field.GoName)
		case FieldKindString:
			g.P("req.", field.GoName, " = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.", field.GoName, "Ptr)), input.", field.GoName, "Len, input.", field.GoName, "Ownership > 0).SafeString()")
		case FieldKindBytes:
			g.P("req.", field.GoName, " = rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(input.", field.GoName, "Ptr)), input.", field.GoName, "Len, input.", field.GoName, "Ownership > 0).SafeBytes()")
		default:
			g.P("return nil, ", unsupportedError)
		}
	default:
		g.P("return nil, ", unsupportedError)
	}
}

func renderNativeUnaryResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, outputName, unsupportedError string) {
	g.P("func encode", service.GoName, method.GoName, "NativeUnaryResponse(resp ", nativeGoMessageType(g, method.Response), ", output *", outputName, ") error {")
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldEncode(g, field, unsupportedError)
	}
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderNativeResponseFieldEncode(g *protogen.GeneratedFile, field FieldPlan, unsupportedError string) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("if resp.", field.GoName, " {")
		g.P("output.", field.GoName, " = 1")
		g.P("} else {")
		g.P("output.", field.GoName, " = 0")
		g.P("}")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble:
			g.P("output.", field.GoName, " = resp.", field.GoName)
		case FieldKindString:
			g.P("data, ptr, err := rpcruntime.PinString(resp.", field.GoName, ")")
			g.P("_ = data")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("length, err := rpcruntime.LengthToInt32(len(resp.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = ptr")
			g.P("output.", field.GoName, "Len = length")
		case FieldKindBytes:
			g.P("ptr, err := rpcruntime.PinBytes(resp.", field.GoName, ")")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("length, err := rpcruntime.LengthToInt32(len(resp.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = ptr")
			g.P("output.", field.GoName, "Len = length")
		default:
			g.P("return ", unsupportedError)
		}
	default:
		g.P("return ", unsupportedError)
	}
}

func nativeUnaryClientInputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeUnaryInput"
}

func nativeUnaryClientOutputName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "NativeUnaryOutput"
}

func nativeUnaryClientFuncName(service ServicePlan, method MethodPlan) string {
	return "Call" + service.GoName + method.GoName + "NativeUnary"
}

func validateNativeClientCGOSymbols(service ServicePlan) error {
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
			return fmt.Errorf("native client cgo symbol %s for %s collides with %s", symbol, source, previous)
		}
		if messageSource, exists := messageTypes[symbol]; exists {
			return fmt.Errorf("native client cgo symbol %s for %s collides with protobuf message type from %s", symbol, source, messageSource)
		}
		seen[symbol] = source
		return nil
	}

	if err := addGenerated(lowerInitial(service.GoName)+"NativeClientUnsupportedField", service.FullName+" unsupported field error"); err != nil {
		return err
	}
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		if err := addGenerated(nativeUnaryClientInputName(service, method), method.FullName+" unary input"); err != nil {
			return err
		}
		if err := addGenerated(nativeUnaryClientOutputName(service, method), method.FullName+" unary output"); err != nil {
			return err
		}
		if err := addGenerated(nativeUnaryClientFuncName(service, method), method.FullName+" unary client call"); err != nil {
			return err
		}
	}
	return nil
}
