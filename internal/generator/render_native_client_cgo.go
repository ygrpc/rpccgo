package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeClientCGOSymbols(plan, service); err != nil {
		return err
	}

	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
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
			renderNativeUnaryClient(g, service, method, errorName)
		case StreamingKindClientStreaming:
			renderNativeClientStreamingClient(g, service, method, errorName, streamHandleErrorName)
		}
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
	g.P("req, err := ", nativeUnaryClientDecoderName(service, method), "(input)")
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
	g.P("if err := ", nativeUnaryClientEncoderName(service, method), "(resp, output); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeUnaryRequestDecoder(g, service, method, inputName, unsupportedError)
	renderNativeUnaryResponseEncoder(g, service, method, outputName, unsupportedError)
}

func renderNativeClientStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError, invalidHandleError string) {
	inputName := nativeClientStreamingInputName(service, method)
	outputName := nativeClientStreamingOutputName(service, method)

	g.P("type ", inputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.RequestFields, false)
	g.P("}")
	g.P()

	g.P("type ", outputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.ResponseFields, true)
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", lowerInitial(service.GoName), "Dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter]) (any, error) {")
	g.P("return snapshot.Adapter.Start", method.GoName, "(ctx)")
	g.P("})")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingSendFuncName(service, method), "(ctx context.Context, handle int32, input *", inputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if input == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native client stream input is nil")))`)
	g.P("}")
	g.P("session, ok := load", service.GoName, method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("req, err := ", nativeClientStreamingDecoderName(service, method), "(input)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := session.Send(ctx, req); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingFinishFuncName(service, method), "(ctx context.Context, handle int32, output *", outputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native client stream output is nil")))`)
	g.P("}")
	g.P("session, ok := take", service.GoName, method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("resp, err := session.Finish(ctx)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if resp == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native client stream server returned nil response")))`)
	g.P("}")
	g.P("if err := ", nativeClientStreamingEncoderName(service, method), "(resp, output); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeClientStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := take", service.GoName, method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("if err := session.Cancel(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeClientStreamingRequestDecoder(g, service, method, inputName, unsupportedError)
	renderNativeClientStreamingResponseEncoder(g, service, method, outputName, unsupportedError)
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
	g.P("func ", nativeUnaryClientDecoderName(service, method), "(input *", inputName, ") (", nativeGoMessageType(g, method.Request), ", error) {")
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.RequestFields {
		renderNativeRequestFieldDecode(g, field, unsupportedError)
	}
	g.P("return req, nil")
	g.P("}")
	g.P()
}

func renderNativeClientStreamingRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, inputName, unsupportedError string) {
	g.P("func ", nativeClientStreamingDecoderName(service, method), "(input *", inputName, ") (", nativeGoMessageType(g, method.Request), ", error) {")
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
		case FieldKindEnum:
			g.P("req.", field.GoName, " = ", nativeGoEnumType(g, field), "(input.", field.GoName, ")")
		case FieldKindString:
			g.P("if _, err := rpcruntime.LengthFromInt32(input.", field.GoName, "Len); err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P(field.GoName, " := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.", field.GoName, "Ptr)), input.", field.GoName, "Len, input.", field.GoName, "Ownership > 0)")
			g.P("req.", field.GoName, " = ", field.GoName, ".SafeString()")
			g.P("if err := ", field.GoName, ".Release(); err != nil {")
			g.P("return nil, err")
			g.P("}")
		case FieldKindBytes:
			g.P("if _, err := rpcruntime.LengthFromInt32(input.", field.GoName, "Len); err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P(field.GoName, " := rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(input.", field.GoName, "Ptr)), input.", field.GoName, "Len, input.", field.GoName, "Ownership > 0)")
			g.P("req.", field.GoName, " = ", field.GoName, ".SafeBytes()")
			g.P("if err := ", field.GoName, ".Release(); err != nil {")
			g.P("return nil, err")
			g.P("}")
		default:
			g.P("return nil, ", unsupportedError)
		}
	default:
		g.P("return nil, ", unsupportedError)
	}
}

func renderNativeUnaryResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, outputName, unsupportedError string) {
	g.P("func ", nativeUnaryClientEncoderName(service, method), "(resp ", nativeGoMessageType(g, method.Response), ", output *", outputName, ") error {")
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldValidate(g, field, unsupportedError)
	}
	var pinned []FieldPlan
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldStage(g, field, pinned)
		if nativeClientFieldPinsOutput(field) {
			pinned = append(pinned, field)
		}
	}
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldCommit(g, field)
	}
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderNativeClientStreamingResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, outputName, unsupportedError string) {
	g.P("func ", nativeClientStreamingEncoderName(service, method), "(resp ", nativeGoMessageType(g, method.Response), ", output *", outputName, ") error {")
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldValidate(g, field, unsupportedError)
	}
	var pinned []FieldPlan
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldStage(g, field, pinned)
		if nativeClientFieldPinsOutput(field) {
			pinned = append(pinned, field)
		}
	}
	for _, field := range method.NativeContract.ResponseFields {
		renderNativeResponseFieldCommit(g, field)
	}
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderNativeResponseFieldValidate(g *protogen.GeneratedFile, field FieldPlan, unsupportedError string) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble, FieldKindEnum:
			return
		case FieldKindString, FieldKindBytes:
			g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(resp.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			return
		}
	}
	g.P("return ", unsupportedError)
}

func renderNativeResponseFieldStage(g *protogen.GeneratedFile, field FieldPlan, pinned []FieldPlan) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("var ", field.GoName, "Value int8")
		g.P("if resp.", field.GoName, " {")
		g.P(field.GoName, "Value = 1")
		g.P("}")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(field.GoName, "Value := resp.", field.GoName)
		case FieldKindEnum:
			g.P(field.GoName, "Value := int32(resp.", field.GoName, ")")
		case FieldKindString:
			g.P("data, ", field.GoName, "Ptr, err := rpcruntime.PinString(resp.", field.GoName, ")")
			g.P("_ = data")
			g.P("if err != nil {")
			renderReleasePinnedOutputFields(g, pinned)
			g.P("return err")
			g.P("}")
			g.P("_ = ", field.GoName, "Ptr")
		case FieldKindBytes:
			g.P(field.GoName, "Ptr, err := rpcruntime.PinBytes(resp.", field.GoName, ")")
			g.P("if err != nil {")
			renderReleasePinnedOutputFields(g, pinned)
			g.P("return err")
			g.P("}")
			g.P("_ = ", field.GoName, "Ptr")
		}
	}
}

func renderReleasePinnedOutputFields(g *protogen.GeneratedFile, fields []FieldPlan) {
	for _, field := range fields {
		g.P("rpcruntime.Release(", field.GoName, "Ptr)")
	}
}

func renderNativeResponseFieldCommit(g *protogen.GeneratedFile, field FieldPlan) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("output.", field.GoName, " = ", field.GoName, "Value")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble, FieldKindEnum:
			g.P("output.", field.GoName, " = ", field.GoName, "Value")
		case FieldKindString, FieldKindBytes:
			g.P("output.", field.GoName, "Ptr = ", field.GoName, "Ptr")
			g.P("output.", field.GoName, "Len = ", field.GoName, "Len")
		}
	}
}

func nativeGoEnumType(g *protogen.GeneratedFile, field FieldPlan) string {
	return g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       field.EnumType.GoName,
		GoImportPath: protogen.GoImportPath(field.EnumType.GoImportPath),
	})
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

func nativeClientNeedsFmt(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary && method.Streaming != StreamingKindClientStreaming {
			continue
		}
		for _, field := range method.NativeContract.RequestFields {
			if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
				return true
			}
		}
	}
	return false
}

func nativeClientNeedsUnsafe(service ServicePlan) bool {
	return nativeClientNeedsFmt(service)
}

func nativeClientFieldPinsOutput(field FieldPlan) bool {
	return field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes)
}

func nativeClientInputFieldSymbols(field FieldPlan) []string {
	if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
		return []string{field.GoName + "Ptr", field.GoName + "Len", field.GoName + "Ownership"}
	}
	return []string{field.GoName}
}

func nativeClientOutputFieldSymbols(field FieldPlan) []string {
	if nativeClientFieldPinsOutput(field) {
		return []string{field.GoName + "Ptr", field.GoName + "Len"}
	}
	return []string{field.GoName}
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
			if err := addGenerated(nativeUnaryClientInputName(service, method), method.FullName+" unary input"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientOutputName(service, method), method.FullName+" unary output"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientFuncName(service, method), method.FullName+" unary client call"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientDecoderName(service, method), method.FullName+" unary request decoder"); err != nil {
				return err
			}
			if err := addGenerated(nativeUnaryClientEncoderName(service, method), method.FullName+" unary response encoder"); err != nil {
				return err
			}
			if err := validateNativeClientStructFields(nativeUnaryClientInputName(service, method), method.NativeContract.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientStructFields(nativeUnaryClientOutputName(service, method), method.NativeContract.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
				return err
			}
		case StreamingKindClientStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeClientStreamingInputName(service, method), method.FullName + " client stream input"},
				{nativeClientStreamingOutputName(service, method), method.FullName + " client stream output"},
				{nativeClientStreamingStartFuncName(service, method), method.FullName + " client stream start"},
				{nativeClientStreamingSendFuncName(service, method), method.FullName + " client stream send"},
				{nativeClientStreamingFinishFuncName(service, method), method.FullName + " client stream finish"},
				{nativeClientStreamingCancelFuncName(service, method), method.FullName + " client stream cancel"},
				{nativeClientStreamingDecoderName(service, method), method.FullName + " client stream request decoder"},
				{nativeClientStreamingEncoderName(service, method), method.FullName + " client stream response encoder"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
			if err := validateNativeClientStructFields(nativeClientStreamingInputName(service, method), method.NativeContract.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientStructFields(nativeClientStreamingOutputName(service, method), method.NativeContract.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
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
			add(nativeUnaryClientInputName(service, method), method.FullName+" unary input")
			add(nativeUnaryClientOutputName(service, method), method.FullName+" unary output")
			add(nativeUnaryClientFuncName(service, method), method.FullName+" unary client call")
			add(nativeUnaryClientDecoderName(service, method), method.FullName+" unary request decoder")
			add(nativeUnaryClientEncoderName(service, method), method.FullName+" unary response encoder")
		case StreamingKindClientStreaming:
			add(nativeClientStreamingInputName(service, method), method.FullName+" client stream input")
			add(nativeClientStreamingOutputName(service, method), method.FullName+" client stream output")
			add(nativeClientStreamingStartFuncName(service, method), method.FullName+" client stream start")
			add(nativeClientStreamingSendFuncName(service, method), method.FullName+" client stream send")
			add(nativeClientStreamingFinishFuncName(service, method), method.FullName+" client stream finish")
			add(nativeClientStreamingCancelFuncName(service, method), method.FullName+" client stream cancel")
			add(nativeClientStreamingDecoderName(service, method), method.FullName+" client stream request decoder")
			add(nativeClientStreamingEncoderName(service, method), method.FullName+" client stream response encoder")
		}
	}
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
