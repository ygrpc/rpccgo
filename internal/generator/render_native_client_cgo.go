package generator

import (
	"fmt"
	"path"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeClientCGOSymbols(plan, service); err != nil {
		return err
	}

	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := plugin.NewGeneratedFile(file.Filename, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"CGONativeClientBridge")
	g.P("package main")
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
			renderNativeUnaryClient(g, service, method, errorName, servicePackage)
		case StreamingKindClientStreaming:
			renderNativeClientStreamingClient(g, service, method, errorName, streamHandleErrorName, servicePackage)
		case StreamingKindServerStreaming:
			renderNativeServerStreamingClient(g, service, method, errorName, streamHandleErrorName, servicePackage)
		case StreamingKindBidiStreaming:
			renderNativeBidiStreamingClient(g, service, method, errorName, streamHandleErrorName, servicePackage)
		}
	}
	return nil
}

func renderNativeUnaryClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError, servicePackage string) {
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
	g.P("resp, err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().", method.GoName, "(ctx, req)")
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

func renderNativeClientStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError, invalidHandleError, servicePackage string) {
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
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Start", method.GoName, "(ctx)")
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
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Load", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
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
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Take", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
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
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Take", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
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

func renderNativeServerStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError, invalidHandleError, servicePackage string) {
	inputName := nativeServerStreamingInputName(service, method)
	outputName := nativeServerStreamingOutputName(service, method)

	g.P("type ", inputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.RequestFields, false)
	g.P("}")
	g.P()

	g.P("type ", outputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.ResponseFields, true)
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingStartFuncName(service, method), "(ctx context.Context, input *", inputName, ") (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if input == nil {")
	g.P(`return 0, int32(rpcruntime.StoreError(errors.New("rpccgo: native server stream input is nil")))`)
	g.P("}")
	g.P("req, err := ", nativeServerStreamingDecoderName(service, method), "(input)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Start", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingReadFuncName(service, method), "(ctx context.Context, handle int32, output *", outputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native server stream output is nil")))`)
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Load", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("resp, err := session.Recv(ctx)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if resp == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native server stream server returned nil response")))`)
	g.P("}")
	g.P("if err := ", nativeServerStreamingEncoderName(service, method), "(resp, output); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingDoneFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Take", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("if done, ok := session.(interface{ Done(context.Context) error }); ok {")
	g.P("if err := done.Done(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeServerStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Take", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("if err := session.Cancel(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeServerStreamingRequestDecoder(g, service, method, inputName, unsupportedError)
	renderNativeServerStreamingResponseEncoder(g, service, method, outputName, unsupportedError)
}

func renderNativeBidiStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, unsupportedError, invalidHandleError, servicePackage string) {
	inputName := nativeBidiStreamingInputName(service, method)
	outputName := nativeBidiStreamingOutputName(service, method)

	g.P("type ", inputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.RequestFields, false)
	g.P("}")
	g.P()

	g.P("type ", outputName, " struct {")
	renderNativeClientFields(g, method.NativeContract.ResponseFields, true)
	g.P("}")
	g.P()

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

	g.P("func ", nativeBidiStreamingSendFuncName(service, method), "(ctx context.Context, handle int32, input *", inputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if input == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native bidi stream input is nil")))`)
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Load", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("req, err := ", nativeBidiStreamingDecoderName(service, method), "(input)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := session.Send(ctx, req); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingReadFuncName(service, method), "(ctx context.Context, handle int32, output *", outputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native bidi stream output is nil")))`)
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Load", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("resp, err := session.Recv(ctx)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if resp == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: native bidi stream server returned nil response")))`)
	g.P("}")
	g.P("if err := ", nativeBidiStreamingEncoderName(service, method), "(resp, output); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingCloseSendFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Load", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("if err := session.CloseSend(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingDoneFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Take", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("if done, ok := session.(interface{ Done(context.Context) error }); ok {")
	g.P("if err := done.Done(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", nativeBidiStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGONativeClientBridge().Take", method.GoName, "NativeStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P("return int32(rpcruntime.StoreError(", invalidHandleError, "))")
	g.P("}")
	g.P("if err := session.Cancel(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	renderNativeBidiStreamingRequestDecoder(g, service, method, inputName, unsupportedError)
	renderNativeBidiStreamingResponseEncoder(g, service, method, outputName, unsupportedError)
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

func renderNativeServerStreamingRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, inputName, unsupportedError string) {
	g.P("func ", nativeServerStreamingDecoderName(service, method), "(input *", inputName, ") (", nativeGoMessageType(g, method.Request), ", error) {")
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.RequestFields {
		renderNativeRequestFieldDecode(g, field, unsupportedError)
	}
	g.P("return req, nil")
	g.P("}")
	g.P()
}

func renderNativeBidiStreamingRequestDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, inputName, unsupportedError string) {
	g.P("func ", nativeBidiStreamingDecoderName(service, method), "(input *", inputName, ") (", nativeGoMessageType(g, method.Request), ", error) {")
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

func renderNativeServerStreamingResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, outputName, unsupportedError string) {
	g.P("func ", nativeServerStreamingEncoderName(service, method), "(resp ", nativeGoMessageType(g, method.Response), ", output *", outputName, ") error {")
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

func renderNativeBidiStreamingResponseEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, outputName, unsupportedError string) {
	g.P("func ", nativeBidiStreamingEncoderName(service, method), "(resp ", nativeGoMessageType(g, method.Response), ", output *", outputName, ") error {")
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
		if method.Streaming != StreamingKindUnary && method.Streaming != StreamingKindClientStreaming && method.Streaming != StreamingKindServerStreaming && method.Streaming != StreamingKindBidiStreaming {
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
		case StreamingKindServerStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeServerStreamingInputName(service, method), method.FullName + " server stream input"},
				{nativeServerStreamingOutputName(service, method), method.FullName + " server stream output"},
				{nativeServerStreamingStartFuncName(service, method), method.FullName + " server stream start"},
				{nativeServerStreamingReadFuncName(service, method), method.FullName + " server stream read"},
				{nativeServerStreamingDoneFuncName(service, method), method.FullName + " server stream done"},
				{nativeServerStreamingCancelFuncName(service, method), method.FullName + " server stream cancel"},
				{nativeServerStreamingDecoderName(service, method), method.FullName + " server stream request decoder"},
				{nativeServerStreamingEncoderName(service, method), method.FullName + " server stream response encoder"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
			if err := validateNativeClientStructFields(nativeServerStreamingInputName(service, method), method.NativeContract.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientStructFields(nativeServerStreamingOutputName(service, method), method.NativeContract.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
				return err
			}
		case StreamingKindBidiStreaming:
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeBidiStreamingInputName(service, method), method.FullName + " bidi stream input"},
				{nativeBidiStreamingOutputName(service, method), method.FullName + " bidi stream output"},
				{nativeBidiStreamingStartFuncName(service, method), method.FullName + " bidi stream start"},
				{nativeBidiStreamingSendFuncName(service, method), method.FullName + " bidi stream send"},
				{nativeBidiStreamingReadFuncName(service, method), method.FullName + " bidi stream read"},
				{nativeBidiStreamingCloseSendFuncName(service, method), method.FullName + " bidi stream close send"},
				{nativeBidiStreamingDoneFuncName(service, method), method.FullName + " bidi stream done"},
				{nativeBidiStreamingCancelFuncName(service, method), method.FullName + " bidi stream cancel"},
				{nativeBidiStreamingDecoderName(service, method), method.FullName + " bidi stream request decoder"},
				{nativeBidiStreamingEncoderName(service, method), method.FullName + " bidi stream response encoder"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
			if err := validateNativeClientStructFields(nativeBidiStreamingInputName(service, method), method.NativeContract.RequestFields, nativeClientInputFieldSymbols); err != nil {
				return err
			}
			if err := validateNativeClientStructFields(nativeBidiStreamingOutputName(service, method), method.NativeContract.ResponseFields, nativeClientOutputFieldSymbols); err != nil {
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
		case StreamingKindServerStreaming:
			add(nativeServerStreamingInputName(service, method), method.FullName+" server stream input")
			add(nativeServerStreamingOutputName(service, method), method.FullName+" server stream output")
			add(nativeServerStreamingStartFuncName(service, method), method.FullName+" server stream start")
			add(nativeServerStreamingReadFuncName(service, method), method.FullName+" server stream read")
			add(nativeServerStreamingDoneFuncName(service, method), method.FullName+" server stream done")
			add(nativeServerStreamingCancelFuncName(service, method), method.FullName+" server stream cancel")
			add(nativeServerStreamingDecoderName(service, method), method.FullName+" server stream request decoder")
			add(nativeServerStreamingEncoderName(service, method), method.FullName+" server stream response encoder")
		case StreamingKindBidiStreaming:
			add(nativeBidiStreamingInputName(service, method), method.FullName+" bidi stream input")
			add(nativeBidiStreamingOutputName(service, method), method.FullName+" bidi stream output")
			add(nativeBidiStreamingStartFuncName(service, method), method.FullName+" bidi stream start")
			add(nativeBidiStreamingSendFuncName(service, method), method.FullName+" bidi stream send")
			add(nativeBidiStreamingReadFuncName(service, method), method.FullName+" bidi stream read")
			add(nativeBidiStreamingCloseSendFuncName(service, method), method.FullName+" bidi stream close send")
			add(nativeBidiStreamingDoneFuncName(service, method), method.FullName+" bidi stream done")
			add(nativeBidiStreamingCancelFuncName(service, method), method.FullName+" bidi stream cancel")
			add(nativeBidiStreamingDecoderName(service, method), method.FullName+" bidi stream request decoder")
			add(nativeBidiStreamingEncoderName(service, method), method.FullName+" bidi stream response encoder")
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
