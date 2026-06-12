package generator

import "google.golang.org/protobuf/compiler/protogen"

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
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
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
}

func renderMessageClientStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	renderMessageCExportWrappers(g, plan, service, method, servicePackage)
}

func renderMessageServerStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	renderMessageCExportWrappers(g, plan, service, method, servicePackage)
}

func renderMessageBidiStreamingClient(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	renderMessageCExportWrappers(g, plan, service, method, servicePackage)
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
	renderCGOExportDoc(g, exportName, "invokes the message unary client entrypoint for "+method.FullName+".")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(requestPtr C.uintptr_t, requestLen C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	g.P("if err := rpcruntime.DecodeMessage(uintptr(requestPtr), int32(requestLen), req); err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request decode failed: %w", err)))`)
	g.P("}")
	g.P("resp, err := ", servicePackage, "Invoke", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := rpcruntime.EncodeMessage(resp)")
	g.P("if err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response encode failed: %w", err)))`)
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderMessageClientStreamingCExportWrappers(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, servicePackage string) {
	startName := messageCExportFuncName(plan, service, method, "start")
	renderCGOExportDoc(g, startName, "starts the message client-streaming client entrypoint for "+method.FullName+".")
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
	renderCGOExportDoc(g, sendName, "sends a message request to the client-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", sendName)
	g.P("func ", sendName, "(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	g.P("if err := rpcruntime.DecodeMessage(uintptr(requestPtr), int32(requestLen), req); err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request decode failed: %w", err)))`)
	g.P("}")
	g.P("if err := ", servicePackage, "Send", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue), req); err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	renderCGOExportDoc(g, finishName, "finishes the message client-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", finishName)
	g.P("func ", finishName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("handleValue := int32(handle)")
	g.P("resp, err := ", servicePackage, "Finish", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := rpcruntime.EncodeMessage(resp)")
	g.P("if err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response encode failed: %w", err)))`)
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()

	cancelName := messageCExportFuncName(plan, service, method, "cancel")
	renderCGOExportDoc(g, cancelName, "cancels the message client-streaming client entrypoint for "+method.FullName+".")
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
	renderCGOExportDoc(g, startName, "starts the message server-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", startName)
	g.P("func ", startName, "(requestPtr C.uintptr_t, requestLen C.int32_t, handle *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportHandleValidation(g)
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	g.P("if err := rpcruntime.DecodeMessage(uintptr(requestPtr), int32(requestLen), req); err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request decode failed: %w", err)))`)
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
	renderCGOExportDoc(g, readName, "reads a message response from the server-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", readName)
	g.P("func ", readName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("handleValue := int32(handle)")
	g.P("resp, err := ", servicePackage, "Recv", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := rpcruntime.EncodeMessage(resp)")
	g.P("if err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response encode failed: %w", err)))`)
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()

	finishName := messageCExportFuncName(plan, service, method, "finish")
	renderCGOExportDoc(g, finishName, "finishes the message server-streaming client entrypoint for "+method.FullName+".")
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
	renderCGOExportDoc(g, cancelName, "cancels the message server-streaming client entrypoint for "+method.FullName+".")
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
	renderCGOExportDoc(g, startName, "starts the message bidi-streaming client entrypoint for "+method.FullName+".")
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
	renderCGOExportDoc(g, sendName, "sends a message request to the bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", sendName)
	g.P("func ", sendName, "(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	g.P("handleValue := int32(handle)")
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	g.P("if err := rpcruntime.DecodeMessage(uintptr(requestPtr), int32(requestLen), req); err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message request decode failed: %w", err)))`)
	g.P("}")
	g.P("if err := ", servicePackage, "Send", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue), req); err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("return 0")
	g.P("}")
	g.P()

	readName := messageCExportFuncName(plan, service, method, "read")
	renderCGOExportDoc(g, readName, "reads a message response from the bidi-streaming client entrypoint for "+method.FullName+".")
	g.P("//export ", readName)
	g.P("func ", readName, "(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {")
	g.P("ctx := context.Background()")
	renderMessageCExportOutputValidation(g)
	g.P("handleValue := int32(handle)")
	g.P("resp, err := ", servicePackage, "Recv", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(handleValue))")
	g.P("if err != nil {")
	g.P("return C.int32_t(rpcruntime.StoreError(err))")
	g.P("}")
	g.P("ptr, length, err := rpcruntime.EncodeMessage(resp)")
	g.P("if err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response encode failed: %w", err)))`)
	g.P("}")
	g.P("*responsePtr = C.uintptr_t(ptr)")
	g.P("*responseLen = C.int32_t(length)")
	g.P("return 0")
	g.P("}")
	g.P()

	closeSendName := messageCExportFuncName(plan, service, method, "close_send")
	renderCGOExportDoc(g, closeSendName, "closes the message bidi-streaming client send side for "+method.FullName+".")
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
	renderCGOExportDoc(g, finishName, "finishes the message bidi-streaming client entrypoint for "+method.FullName+".")
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
	renderCGOExportDoc(g, cancelName, "cancels the message bidi-streaming client entrypoint for "+method.FullName+".")
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
