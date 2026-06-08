package generator

import (
	"fmt"

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
	renderMessageCExportWrappers(g, plan, service, method, servicePackage)

	g.P("func ", messageFromABIName(service, method, "Request"), "(ptr uintptr, length int32) (", messageGoPointerType(g, method.Request), ", error) {")
	renderMessageFromABIBody(g, method.Request, "request")
	g.P("}")
	g.P()

	g.P("func ", messageToABIName(service, method, "Response"), "(message ", messageGoPointerType(g, method.Response), ") (uintptr, int32, error) {")
	renderMessageToABIBody(g, "response")
	g.P("}")
	g.P()
}

func renderMessageClientStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	renderMessageClientBytesHelpers(g, plan, service, method, servicePackage)
}

func renderMessageServerStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	renderMessageClientBytesHelpers(g, plan, service, method, servicePackage)
}

func renderMessageBidiStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	renderMessageClientBytesHelpers(g, plan, service, method, servicePackage)
}

func renderMessageClientBytesHelpers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	g.P("func ", messageFromABIName(service, method, "Request"), "(ptr uintptr, length int32) (", messageGoPointerType(g, method.Request), ", error) {")
	renderMessageFromABIBody(g, method.Request, "request")
	g.P("}")
	g.P()

	g.P("func ", messageToABIName(service, method, "Response"), "(message ", messageGoPointerType(g, method.Response), ") (uintptr, int32, error) {")
	renderMessageToABIBody(g, "response")
	g.P("}")
	g.P()

	renderMessageCExportWrappers(g, plan, service, method, servicePackage)
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

func messageFromABIName(service ServicePlan, method MethodPlan, suffix string) string {
	return fmt.Sprintf("decode%s%sMessage%s", service.GoName, method.GoName, suffix)
}

func messageToABIName(service ServicePlan, method MethodPlan, suffix string) string {
	return fmt.Sprintf("encode%s%sMessage%s", service.GoName, method.GoName, suffix)
}

func messageGoPointerType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{GoName: message.GoName, GoImportPath: protogen.GoImportPath(message.GoImportPath)})
}

func renderMessageCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	switch method.Streaming {
	case StreamingKindUnary:
		renderMessageUnaryCExportWrapper(g, plan, service, method, servicePackage)
	case StreamingKindClientStreaming:
		renderMessageClientStreamingCExportWrappers(g, plan, service, method, servicePackage)
	case StreamingKindServerStreaming:
		renderMessageServerStreamingCExportWrappers(g, plan, service, method, servicePackage)
	case StreamingKindBidiStreaming:
		renderMessageBidiStreamingCExportWrappers(g, plan, service, method, servicePackage)
	}
}

func renderMessageUnaryCExportWrapper(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	exportName := messageCExportFuncName(plan, service, method, "")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(requestPtr C.uintptr_t, requestLen C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(uintptr(requestPtr), int32(requestLen))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("resp, err := ", servicePackage, "Invoke", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := ", messageToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderMessageClientStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	startName := messageCExportFuncName(plan, service, method, "start")
	g.P("//export ", startName)
	g.P("func ", startName, "(handle *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportHandleValidation(g)
	g.P("handleValue, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*handle = C.int32_t(int32(handleValue))")
	g.P("return 0")
	g.P("}")
	g.P()

	sendName := messageCExportFuncName(plan, service, method, "send")
	g.P("//export ", sendName)
	g.P("func ", sendName, "(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(uintptr(requestPtr), int32(requestLen))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("err = ", servicePackage, "Send", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue), req)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("handleValue := int32(handle)")
	g.P("resp, err := ", servicePackage, "Finish", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := ", messageToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	g.P("//export ", cancelName)
	g.P("func ", cancelName, "(handle C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("err := ", servicePackage, "Cancel", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderMessageServerStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	startName := messageCExportFuncName(plan, service, method, "start")
	g.P("//export ", startName)
	g.P("func ", startName, "(requestPtr C.uintptr_t, requestLen C.int32_t, handle *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportHandleValidation(g)
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(uintptr(requestPtr), int32(requestLen))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("handleValue, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*handle = C.int32_t(int32(handleValue))")
	g.P("return 0")
	g.P("}")
	g.P()

	readName := messageCExportFuncName(plan, service, method, "read")
	g.P("//export ", readName)
	g.P("func ", readName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("handleValue := int32(handle)")
	g.P("resp, err := ", servicePackage, "Recv", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := ", messageToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("err := ", servicePackage, "Finish", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	g.P("//export ", cancelName)
	g.P("func ", cancelName, "(handle C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("err := ", servicePackage, "Cancel", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderMessageBidiStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	startName := messageCExportFuncName(plan, service, method, "start")
	g.P("//export ", startName)
	g.P("func ", startName, "(handle *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportHandleValidation(g)
	g.P("handleValue, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*handle = C.int32_t(int32(handleValue))")
	g.P("return 0")
	g.P("}")
	g.P()

	sendName := messageCExportFuncName(plan, service, method, "send")
	g.P("//export ", sendName)
	g.P("func ", sendName, "(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("req, err := ", messageFromABIName(service, method, "Request"), "(uintptr(requestPtr), int32(requestLen))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("err = ", servicePackage, "Send", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue), req)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	readName := messageCExportFuncName(plan, service, method, "read")
	g.P("//export ", readName)
	g.P("func ", readName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("handleValue := int32(handle)")
	g.P("resp, err := ", servicePackage, "Recv", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := ", messageToABIName(service, method, "Response"), "(resp)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()

	closeSendName := messageCExportFuncName(plan, service, method, "close_send")
	g.P("//export ", closeSendName)
	g.P("func ", closeSendName, "(handle C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("err := ", servicePackage, "CloseSend", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("err := ", servicePackage, "Finish", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	g.P("//export ", cancelName)
	g.P("func ", cancelName, "(handle C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("err := ", servicePackage, "Cancel", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
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
