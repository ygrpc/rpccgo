package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := plugin.NewGeneratedFile(file.Filename, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"CGOMessageClientBridge")

	g.P("package main")
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	g.P(`protobuf "google.golang.org/protobuf/proto"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(`unsafe "unsafe"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	g.P("type ", service.GoName, "MessageOutput struct {")
	g.P("DataPtr uintptr")
	g.P("DataLen int32")
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderMessageUnaryClient(g, service, method, servicePackage)
		case StreamingKindClientStreaming:
			renderMessageClientStreamingClient(g, service, method, servicePackage)
		case StreamingKindServerStreaming:
			renderMessageServerStreamingClient(g, service, method, servicePackage)
		case StreamingKindBidiStreaming:
			renderMessageBidiStreamingClient(g, service, method, servicePackage)
		}
	}
	return nil
}

func renderMessageUnaryClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	funcName := messageUnaryClientFuncName(service, method)
	outputName := service.GoName + "MessageOutput"

	g.P("func ", funcName, "(ctx context.Context, requestPtr uintptr, requestLen int32, output *", outputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))`)
	g.P("}")
	g.P("req, err := ", messageBytesFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := protobuf.Unmarshal(req, &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}); err != nil {")
	g.P(`return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)))`)
	g.P("}")
	g.P("resp, err := ", servicePackage, "New", service.GoName, "CGOMessageClientBridge().", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("if err := protobuf.Unmarshal(resp, &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}); err != nil {")
	g.P(`return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)))`)
	g.P("}")
	g.P("ptr, length, err := ", messageBytesToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("output.DataPtr = ptr")
	g.P("output.DataLen = length")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBytesFromABIName(service, method, "Request"), "(ptr uintptr, length int32) ([]byte, error) {")
	renderMessageBytesFromABIBody(g, "request")
	g.P("}")
	g.P()

	g.P("func ", messageBytesToABIName(service, method, "Response"), "(data []byte) (uintptr, int32, error) {")
	renderMessageBytesToABIBody(g)
	g.P("}")
	g.P()
}

func messageUnaryClientFuncName(service ServicePlan, method MethodPlan) string {
	return "Call" + service.GoName + method.GoName + "MessageUnary"
}

func renderMessageClientStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageClientStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGOMessageClientBridge().Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", messageClientStreamingSendFuncName(service, method), "(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {")
	renderMessageClientLoadSessionPrefix(g, service, method, servicePackage)
	g.P("req, err := ", messageBytesFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageProtoUnmarshalCheck(g, method.Request, "req", "request")
	g.P("if err := session.Send(ctx, req); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageClientStreamingFinishFuncName(service, method), "(ctx context.Context, handle int32, output *", service.GoName, "MessageOutput) int32 {")
	renderMessageClientTakeSessionPrefix(g, service, method, servicePackage, true)
	g.P("resp, err := session.Finish(ctx)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageProtoUnmarshalCheck(g, method.Response, "resp", "response")
	renderMessageClientWriteOutput(g, service, method)
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageClientStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientTakeSessionPrefix(g, service, method, servicePackage, false)
	g.P("if err := session.Cancel(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageClientBytesHelpers(g, service, method)
}

func renderMessageServerStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageServerStreamingStartFuncName(service, method), "(ctx context.Context, requestPtr uintptr, requestLen int32) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("req, err := ", messageBytesFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageProtoUnmarshalStartCheck(g, method.Request, "req", "request")
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGOMessageClientBridge().Start", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", messageServerStreamingReadFuncName(service, method), "(ctx context.Context, handle int32, output *", service.GoName, "MessageOutput) int32 {")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))`)
	g.P("}")
	renderMessageClientLoadSessionPrefix(g, service, method, servicePackage)
	g.P("resp, err := session.Recv(ctx)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageProtoUnmarshalCheck(g, method.Response, "resp", "response")
	renderMessageClientWriteOutput(g, service, method)
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageServerStreamingDoneFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientTakeSessionPrefix(g, service, method, servicePackage, false)
	g.P("if err := session.Done(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageServerStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientTakeSessionPrefix(g, service, method, servicePackage, false)
	g.P("if err := session.Cancel(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageClientBytesHelpers(g, service, method)
}

func renderMessageBidiStreamingClient(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageBidiStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", servicePackage, "New", service.GoName, "CGOMessageClientBridge().Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingSendFuncName(service, method), "(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {")
	renderMessageClientLoadSessionPrefix(g, service, method, servicePackage)
	g.P("req, err := ", messageBytesFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageProtoUnmarshalCheck(g, method.Request, "req", "request")
	g.P("if err := session.Send(ctx, req); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingReadFuncName(service, method), "(ctx context.Context, handle int32, output *", service.GoName, "MessageOutput) int32 {")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))`)
	g.P("}")
	renderMessageClientLoadSessionPrefix(g, service, method, servicePackage)
	g.P("resp, err := session.Recv(ctx)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageProtoUnmarshalCheck(g, method.Response, "resp", "response")
	renderMessageClientWriteOutput(g, service, method)
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingCloseSendFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientLoadSessionPrefix(g, service, method, servicePackage)
	g.P("if err := session.CloseSend(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingDoneFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientTakeSessionPrefix(g, service, method, servicePackage, false)
	g.P("if err := session.Done(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientTakeSessionPrefix(g, service, method, servicePackage, false)
	g.P("if err := session.Cancel(ctx); err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageClientBytesHelpers(g, service, method)
}

func renderMessageClientBytesHelpers(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", messageBytesFromABIName(service, method, "Request"), "(ptr uintptr, length int32) ([]byte, error) {")
	renderMessageBytesFromABIBody(g, "request")
	g.P("}")
	g.P()

	g.P("func ", messageBytesToABIName(service, method, "Response"), "(data []byte) (uintptr, int32, error) {")
	renderMessageBytesToABIBody(g)
	g.P("}")
	g.P()
}

func renderMessageBytesFromABIBody(g *protogen.GeneratedFile, label string) {
	g.P("if length < 0 {")
	g.P(`return nil, errors.New("rpccgo: message `, label, ` length is negative")`)
	g.P("}")
	g.P("if length == 0 {")
	g.P("return nil, nil")
	g.P("}")
	g.P("if ptr == 0 {")
	g.P(`return nil, errors.New("rpccgo: message `, label, ` pointer is nil")`)
	g.P("}")
	g.P("return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))...), nil")
}

func renderMessageBytesToABIBody(g *protogen.GeneratedFile) {
	g.P("length, err := rpcruntime.LengthToInt32(len(data))")
	g.P("if err != nil {")
	g.P("return 0, 0, err")
	g.P("}")
	g.P("ptr, err := rpcruntime.PinBytes(data)")
	g.P("if err != nil {")
	g.P("return 0, 0, err")
	g.P("}")
	g.P("return ptr, length, nil")
}

func renderMessageClientLoadSessionPrefix(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGOMessageClientBridge().Load", method.GoName, "MessageStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message client stream handle is invalid")))`)
	g.P("}")
}

func renderMessageClientTakeSessionPrefix(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string, needsOutput bool) {
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	if needsOutput {
		g.P("if output == nil {")
		g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))`)
		g.P("}")
	}
	g.P("session, ok := ", servicePackage, "New", service.GoName, "CGOMessageClientBridge().Take", method.GoName, "MessageStream(rpcruntime.StreamHandle(handle))")
	g.P("if !ok {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message client stream handle is invalid")))`)
	g.P("}")
}

func renderMessageProtoUnmarshalCheck(g *protogen.GeneratedFile, message MethodIOPlan, dataName, label string) {
	g.P("if err := protobuf.Unmarshal(", dataName, ", &", g.QualifiedGoIdent(protogen.GoIdent{GoName: message.GoName, GoImportPath: protogen.GoImportPath(message.GoImportPath)}), "{}); err != nil {")
	g.P(`return int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message `, label, ` protobuf unmarshal failed: %w", err)))`)
	g.P("}")
}

func renderMessageProtoUnmarshalStartCheck(g *protogen.GeneratedFile, message MethodIOPlan, dataName, label string) {
	g.P("if err := protobuf.Unmarshal(", dataName, ", &", g.QualifiedGoIdent(protogen.GoIdent{GoName: message.GoName, GoImportPath: protogen.GoImportPath(message.GoImportPath)}), "{}); err != nil {")
	g.P(`return 0, int32(rpcruntime.StoreError(fmt.Errorf("rpccgo: message `, label, ` protobuf unmarshal failed: %w", err)))`)
	g.P("}")
}

func renderMessageClientWriteOutput(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("ptr, length, err := ", messageBytesToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("output.DataPtr = ptr")
	g.P("output.DataLen = length")
}

func messageClientStreamingStartFuncName(service ServicePlan, method MethodPlan) string {
	return "Start" + service.GoName + method.GoName + "MessageClientStream"
}

func messageClientStreamingSendFuncName(service ServicePlan, method MethodPlan) string {
	return "Send" + service.GoName + method.GoName + "MessageClientStream"
}

func messageClientStreamingFinishFuncName(service ServicePlan, method MethodPlan) string {
	return "Finish" + service.GoName + method.GoName + "MessageClientStream"
}

func messageClientStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "MessageClientStream"
}

func messageServerStreamingStartFuncName(service ServicePlan, method MethodPlan) string {
	return "Start" + service.GoName + method.GoName + "MessageServerStream"
}

func messageServerStreamingReadFuncName(service ServicePlan, method MethodPlan) string {
	return "Read" + service.GoName + method.GoName + "MessageServerStream"
}

func messageServerStreamingDoneFuncName(service ServicePlan, method MethodPlan) string {
	return "Done" + service.GoName + method.GoName + "MessageServerStream"
}

func messageServerStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "MessageServerStream"
}

func messageBidiStreamingStartFuncName(service ServicePlan, method MethodPlan) string {
	return "Start" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBidiStreamingSendFuncName(service ServicePlan, method MethodPlan) string {
	return "Send" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBidiStreamingReadFuncName(service ServicePlan, method MethodPlan) string {
	return "Read" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBidiStreamingCloseSendFuncName(service ServicePlan, method MethodPlan) string {
	return "CloseSend" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBidiStreamingDoneFuncName(service ServicePlan, method MethodPlan) string {
	return "Done" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBidiStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBytesFromABIName(service ServicePlan, method MethodPlan, suffix string) string {
	return fmt.Sprintf("decode%s%sMessage%sBytes", service.GoName, method.GoName, suffix)
}

func messageBytesToABIName(service ServicePlan, method MethodPlan, suffix string) string {
	return fmt.Sprintf("encode%s%sMessage%sBytes", service.GoName, method.GoName, suffix)
}
