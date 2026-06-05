package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageClientCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
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
			renderMessageUnaryClient(g, plan, service, method, servicePackage)
		case StreamingKindClientStreaming:
			renderMessageClientStreamingClient(g, plan, service, method, servicePackage)
		case StreamingKindServerStreaming:
			renderMessageServerStreamingClient(g, plan, service, method, servicePackage)
		case StreamingKindBidiStreaming:
			renderMessageBidiStreamingClient(g, plan, service, method, servicePackage)
		}
	}
	return nil
}

func renderMessageUnaryClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	funcName := messageUnaryClientFuncName(service, method)
	outputName := service.GoName + "MessageOutput"

	g.P("func ", funcName, "(ctx context.Context, requestPtr uintptr, requestLen int32, output *", outputName, ") int32 {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))`)
	g.P("}")
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("resp, err := ", servicePackage, "Invoke", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := ", messageToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("output.DataPtr = ptr")
	g.P("output.DataLen = length")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageCExportWrappers(g, plan, service, method)

	g.P("func ", messageFromABIName(service, method, "Request"), "(ptr uintptr, length int32) (", messageGoPointerType(g, method.Request), ", error) {")
	renderMessageFromABIBody(g, method.Request, "request")
	g.P("}")
	g.P()

	g.P("func ", messageToABIName(service, method, "Response"), "(message ", messageGoPointerType(g, method.Response), ") (uintptr, int32, error) {")
	renderMessageToABIBody(g, "response")
	g.P("}")
	g.P()
}

func messageUnaryClientFuncName(service ServicePlan, method MethodPlan) string {
	return "Call" + service.GoName + method.GoName + "MessageUnary"
}

func renderMessageClientStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageClientStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", messageClientStreamingSendFuncName(service, method), "(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Send", "ctx, req")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageClientStreamingFinishFuncName(service, method), "(ctx context.Context, handle int32, output *", service.GoName, "MessageOutput) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))`)
	g.P("}")
	g.P("var resp ", messageGoPointerType(g, method.Response))
	g.P("var err error")
	renderMessageClientStreamFacadeResultCall(g, service, method, servicePackage, "resp", "Finish", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageClientWriteOutput(g, service, method)
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageClientStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("var err error")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Cancel", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageClientBytesHelpers(g, plan, service, method)
}

func renderMessageServerStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageServerStreamingStartFuncName(service, method), "(ctx context.Context, requestPtr uintptr, requestLen int32) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("handle, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", messageServerStreamingReadFuncName(service, method), "(ctx context.Context, handle int32, output *", service.GoName, "MessageOutput) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))`)
	g.P("}")
	g.P("var resp ", messageGoPointerType(g, method.Response))
	g.P("var err error")
	renderMessageClientStreamFacadeResultCall(g, service, method, servicePackage, "resp", "Recv", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageClientWriteOutput(g, service, method)
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageServerStreamingFinishFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("var err error")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Finish", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageServerStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("var err error")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Cancel", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageClientBytesHelpers(g, plan, service, method)
}

func renderMessageBidiStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageBidiStreamingStartFuncName(service, method), "(ctx context.Context) (int32, int32) {")
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
	g.P("handle, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return 0, int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return int32(handle), 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingSendFuncName(service, method), "(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(requestPtr, requestLen)")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Send", "ctx, req")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingReadFuncName(service, method), "(ctx context.Context, handle int32, output *", service.GoName, "MessageOutput) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("if output == nil {")
	g.P(`return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))`)
	g.P("}")
	g.P("var resp ", messageGoPointerType(g, method.Response))
	g.P("var err error")
	renderMessageClientStreamFacadeResultCall(g, service, method, servicePackage, "resp", "Recv", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	renderMessageClientWriteOutput(g, service, method)
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingCloseSendFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("var err error")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "CloseSend", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingFinishFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("var err error")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Finish", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	g.P("func ", messageBidiStreamingCancelFuncName(service, method), "(ctx context.Context, handle int32) int32 {")
	renderMessageClientContextPrefix(g)
	g.P("var err error")
	renderMessageClientStreamFacadeCall(g, service, method, servicePackage, "Cancel", "ctx")
	g.P("if err != nil {")
	g.P("return int32(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
	renderMessageClientBytesHelpers(g, plan, service, method)
}

func renderMessageClientBytesHelpers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan) {
	g.P("func ", messageFromABIName(service, method, "Request"), "(ptr uintptr, length int32) (", messageGoPointerType(g, method.Request), ", error) {")
	renderMessageFromABIBody(g, method.Request, "request")
	g.P("}")
	g.P()

	g.P("func ", messageToABIName(service, method, "Response"), "(message ", messageGoPointerType(g, method.Response), ") (uintptr, int32, error) {")
	renderMessageToABIBody(g, "response")
	g.P("}")
	g.P()

	renderMessageCExportWrappers(g, plan, service, method)
}

func renderMessageFromABIBody(g *protogen.GeneratedFile, message MethodIOPlan, label string) {
	g.P("msg := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: message.GoName, GoImportPath: protogen.GoImportPath(message.GoImportPath)}), "{}")
	g.P("if length < 0 {")
	g.P(`return nil, errors.New("rpccgo: message `, label, ` length is negative")`)
	g.P("}")
	g.P("if length == 0 {")
	g.P("return msg, nil")
	g.P("}")
	g.P("if ptr == 0 {")
	g.P(`return nil, errors.New("rpccgo: message `, label, ` pointer is nil")`)
	g.P("}")
	g.P("data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))")
	g.P("if err := protobuf.Unmarshal(data, msg); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: message `, label, ` protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return msg, nil")
}

func renderMessageToABIBody(g *protogen.GeneratedFile, label string) {
	g.P("if message == nil {")
	g.P(`return 0, 0, errors.New("rpccgo: message `, label, ` is nil")`)
	g.P("}")
	g.P("data, err := protobuf.Marshal(message)")
	g.P("if err != nil {")
	g.P(`return 0, 0, fmt.Errorf("rpccgo: message `, label, ` protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("length, err := rpcruntime.LengthToInt32(len(data))")
	g.P("if err != nil {")
	g.P("return 0, 0, err")
	g.P("}")
	g.P("if length == 0 {")
	g.P("return 0, 0, nil")
	g.P("}")
	g.P("ptr, err := rpcruntime.PinBytes(data)")
	g.P("if err != nil {")
	g.P("return 0, 0, err")
	g.P("}")
	g.P("return ptr, length, nil")
}

func renderMessageClientContextPrefix(g *protogen.GeneratedFile) {
	g.P("if ctx == nil {")
	g.P("ctx = context.Background()")
	g.P("}")
}

func renderMessageClientStreamFacadeCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, operation, args string) {
	g.P("err = ", servicePackage, operation, service.GoName, "Message", method.GoName, "(", messageClientStreamOperationArgs(args), ")")
}

func renderMessageClientStreamFacadeResultCall(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, result, operation, args string) {
	g.P(result, ", err = ", servicePackage, operation, service.GoName, "Message", method.GoName, "(", messageClientStreamOperationArgs(args), ")")
}

func messageClientStreamOperationArgs(args string) string {
	return strings.Replace(args, "ctx", "ctx, rpcruntime.StreamHandle(handle)", 1)
}

func renderMessageClientWriteOutput(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("ptr, length, err := ", messageToABIName(service, method, "Response"), "(resp)")
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

func messageServerStreamingFinishFuncName(service ServicePlan, method MethodPlan) string {
	return "Finish" + service.GoName + method.GoName + "MessageServerStream"
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

func messageBidiStreamingFinishFuncName(service ServicePlan, method MethodPlan) string {
	return "Finish" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageBidiStreamingCancelFuncName(service ServicePlan, method MethodPlan) string {
	return "Cancel" + service.GoName + method.GoName + "MessageBidiStream"
}

func messageFromABIName(service ServicePlan, method MethodPlan, suffix string) string {
	return fmt.Sprintf("decode%s%sMessage%s", service.GoName, method.GoName, suffix)
}

func messageToABIName(service ServicePlan, method MethodPlan, suffix string) string {
	return fmt.Sprintf("encode%s%sMessage%s", service.GoName, method.GoName, suffix)
}

func messageGoPointerType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{GoName: message.GoName, GoImportPath: protogen.GoImportPath(message.GoImportPath)})
}

func renderMessageCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan) {
	switch method.Streaming {
	case StreamingKindUnary:
		renderMessageUnaryCExportWrapper(g, plan, service, method)
	case StreamingKindClientStreaming:
		renderMessageClientStreamingCExportWrappers(g, plan, service, method)
	case StreamingKindServerStreaming:
		renderMessageServerStreamingCExportWrappers(g, plan, service, method)
	case StreamingKindBidiStreaming:
		renderMessageBidiStreamingCExportWrappers(g, plan, service, method)
	}
}

func renderMessageUnaryCExportWrapper(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan) {
	exportName := messageCExportFuncName(plan, service, method, "")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(requestPtr C.uintptr_t, requestLen C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	renderMessageCExportOutputValidation(g)
	g.P("var output ", service.GoName, "MessageOutput")
	g.P("errID := ", messageUnaryClientFuncName(service, method), "(context.Background(), uintptr(requestPtr), int32(requestLen), &output)")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(output.DataPtr)")
	g.P("*responseLen = C.int32_t(output.DataLen)")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderMessageClientStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan) {
	startName := messageCExportFuncName(plan, service, method, "start")
	g.P("//export ", startName)
	g.P("func ", startName, "(handle *C.int32_t) C.int32_t {")
	renderMessageCExportHandleValidation(g)
	g.P("handleValue, errID := ", messageClientStreamingStartFuncName(service, method), "(context.Background())")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*handle = C.int32_t(handleValue)")
	g.P("return 0")
	g.P("}")
	g.P()

	sendName := messageCExportFuncName(plan, service, method, "send")
	g.P("//export ", sendName)
	g.P("func ", sendName, "(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageClientStreamingSendFuncName(service, method), "(context.Background(), int32(handle), uintptr(requestPtr), int32(requestLen)))")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	renderMessageCExportOutputValidation(g)
	g.P("var output ", service.GoName, "MessageOutput")
	g.P("errID := ", messageClientStreamingFinishFuncName(service, method), "(context.Background(), int32(handle), &output)")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(output.DataPtr)")
	g.P("*responseLen = C.int32_t(output.DataLen)")
	g.P("return 0")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	g.P("//export ", cancelName)
	g.P("func ", cancelName, "(handle C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageClientStreamingCancelFuncName(service, method), "(context.Background(), int32(handle)))")
	g.P("}")
	g.P()
}

func renderMessageServerStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan) {
	startName := messageCExportFuncName(plan, service, method, "start")
	g.P("//export ", startName)
	g.P("func ", startName, "(requestPtr C.uintptr_t, requestLen C.int32_t, handle *C.int32_t) C.int32_t {")
	renderMessageCExportHandleValidation(g)
	g.P("handleValue, errID := ", messageServerStreamingStartFuncName(service, method), "(context.Background(), uintptr(requestPtr), int32(requestLen))")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*handle = C.int32_t(handleValue)")
	g.P("return 0")
	g.P("}")
	g.P()

	readName := messageCExportFuncName(plan, service, method, "read")
	g.P("//export ", readName)
	g.P("func ", readName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	renderMessageCExportOutputValidation(g)
	g.P("var output ", service.GoName, "MessageOutput")
	g.P("errID := ", messageServerStreamingReadFuncName(service, method), "(context.Background(), int32(handle), &output)")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(output.DataPtr)")
	g.P("*responseLen = C.int32_t(output.DataLen)")
	g.P("return 0")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageServerStreamingFinishFuncName(service, method), "(context.Background(), int32(handle)))")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	g.P("//export ", cancelName)
	g.P("func ", cancelName, "(handle C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageServerStreamingCancelFuncName(service, method), "(context.Background(), int32(handle)))")
	g.P("}")
	g.P()
}

func renderMessageBidiStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan) {
	startName := messageCExportFuncName(plan, service, method, "start")
	g.P("//export ", startName)
	g.P("func ", startName, "(handle *C.int32_t) C.int32_t {")
	renderMessageCExportHandleValidation(g)
	g.P("handleValue, errID := ", messageBidiStreamingStartFuncName(service, method), "(context.Background())")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*handle = C.int32_t(handleValue)")
	g.P("return 0")
	g.P("}")
	g.P()

	sendName := messageCExportFuncName(plan, service, method, "send")
	g.P("//export ", sendName)
	g.P("func ", sendName, "(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageBidiStreamingSendFuncName(service, method), "(context.Background(), int32(handle), uintptr(requestPtr), int32(requestLen)))")
	g.P("}")
	g.P()

	readName := messageCExportFuncName(plan, service, method, "read")
	g.P("//export ", readName)
	g.P("func ", readName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	renderMessageCExportOutputValidation(g)
	g.P("var output ", service.GoName, "MessageOutput")
	g.P("errID := ", messageBidiStreamingReadFuncName(service, method), "(context.Background(), int32(handle), &output)")
	g.P("if errID != 0 {")
	g.P("return C.int32_t(errID)")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(output.DataPtr)")
	g.P("*responseLen = C.int32_t(output.DataLen)")
	g.P("return 0")
	g.P("}")
	g.P()

	closeSendName := messageCExportFuncName(plan, service, method, "close_send")
	g.P("//export ", closeSendName)
	g.P("func ", closeSendName, "(handle C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageBidiStreamingCloseSendFuncName(service, method), "(context.Background(), int32(handle)))")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageBidiStreamingFinishFuncName(service, method), "(context.Background(), int32(handle)))")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	g.P("//export ", cancelName)
	g.P("func ", cancelName, "(handle C.int32_t) C.int32_t {")
	g.P("return C.int32_t(", messageBidiStreamingCancelFuncName(service, method), "(context.Background(), int32(handle)))")
	g.P("}")
	g.P()
}

func renderMessageCExportOutputValidation(g *protogen.GeneratedFile) {
	g.P("if responsePtr != nil {")
	g.P("*responsePtr = 0")
	g.P("}")
	g.P("if responseLen != nil {")
	g.P("*responseLen = 0")
	g.P("}")
	g.P("if responsePtr == nil || responseLen == nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))`)
	g.P("}")
}

func renderMessageCExportHandleValidation(g *protogen.GeneratedFile) {
	g.P("if handle != nil {")
	g.P("*handle = 0")
	g.P("}")
	g.P("if handle == nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client handle pointer is nil")))`)
	g.P("}")
}

func messageCExportFuncName(plan FilePlan, service ServicePlan, method MethodPlan, operation string) string {
	name := "rpccgo_msg_" + plan.GoPackageName + "_" + service.GoName + "_" + method.GoName
	if operation != "" {
		name += "_" + operation
	}
	return name
}
