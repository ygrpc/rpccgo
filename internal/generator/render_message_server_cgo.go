package generator

import (
	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := plugin.NewGeneratedFile(file.Filename, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"MessageAdapter")

	g.P("package main")
	g.P()
	renderCGOMessageServerPreamble(g, service)
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	g.P(`io "io"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(`unsafe "unsafe"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	callbacksName := service.GoName + "CGOMessageServerCallbacks"
	adapterName := lowerInitial(service.GoName) + "CGOMessageAdapter"
	g.P("var (")
	g.P(lowerInitial(service.GoName), `CGOMessageServerCallbacksNil = errors.New("rpccgo: `, service.GoName, ` cgo message server callbacks are nil")`)
	g.P(lowerInitial(service.GoName), `CGOMessageServerUnaryCallbackMissing = errors.New("rpccgo: `, service.GoName, ` cgo message server unary callback is missing")`)
	g.P(")")
	g.P()

	g.P("type ", adapterName, " struct {")
	g.P("callbacks C.", callbacksName)
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGOMessageServerUnaryAdapter(g, service, method, adapterName)
		case StreamingKindClientStreaming:
			renderCGOMessageServerClientStreamAdapter(g, service, method, adapterName, servicePackage)
		case StreamingKindServerStreaming:
			renderCGOMessageServerServerStreamAdapter(g, service, method, adapterName, servicePackage)
		case StreamingKindBidiStreaming:
			renderCGOMessageServerBidiStreamAdapter(g, service, method, adapterName, servicePackage)
		}
	}

	g.P("func Register", service.GoName, "CGOMessageServer(callbacks *C.", callbacksName, ") (rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "MessageAdapter], error) {")
	g.P("if callbacks == nil {")
	g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "MessageAdapter]{}, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	renderCGOMessageServerCallbackValidation(g, service, servicePackage)
	g.P("callbacksCopy := *callbacks")
	g.P("return ", servicePackage, "Register", service.GoName, "CGOMessageActiveServer(rpcruntime.ServerKindCGOMessage, &", adapterName, "{callbacks: callbacksCopy})")
	g.P("}")
	g.P()

	renderCGOMessageErrorIDHelper(g, service)
	renderCGOMessageStreamEOFHelper(g, service)
	return nil
}

func renderCGOMessageServerPreamble(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("/*")
	g.P("#include <stdint.h>")
	g.P()
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("typedef int32_t (*", messageCGOServerUnaryCallbackName(service, method), ")(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);")
		case StreamingKindClientStreaming:
			g.P("typedef int32_t (*", messageCGOServerClientStreamStartCallbackName(service, method), ")(int32_t* stream);")
			g.P("typedef int32_t (*", messageCGOServerClientStreamSendCallbackName(service, method), ")(int32_t stream, uintptr_t request_ptr, int32_t request_len);")
			g.P("typedef int32_t (*", messageCGOServerClientStreamFinishCallbackName(service, method), ")(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);")
			g.P("typedef int32_t (*", messageCGOServerClientStreamCancelCallbackName(service, method), ")(int32_t stream);")
		case StreamingKindServerStreaming:
			g.P("typedef int32_t (*", messageCGOServerServerStreamStartCallbackName(service, method), ")(uintptr_t request_ptr, int32_t request_len, int32_t* stream);")
			g.P("typedef int32_t (*", messageCGOServerServerStreamRecvCallbackName(service, method), ")(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);")
			g.P("typedef int32_t (*", messageCGOServerServerStreamDoneCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", messageCGOServerServerStreamCancelCallbackName(service, method), ")(int32_t stream);")
		case StreamingKindBidiStreaming:
			g.P("typedef int32_t (*", messageCGOServerBidiStreamStartCallbackName(service, method), ")(int32_t* stream);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamSendCallbackName(service, method), ")(int32_t stream, uintptr_t request_ptr, int32_t request_len);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamRecvCallbackName(service, method), ")(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamCloseSendCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamDoneCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamCancelCallbackName(service, method), ")(int32_t stream);")
		}
		g.P()
	}
	g.P("typedef struct ", service.GoName, "CGOMessageServerCallbacks {")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(messageCGOServerUnaryCallbackName(service, method), " ", method.GoName, ";")
		case StreamingKindClientStreaming:
			g.P(messageCGOServerClientStreamStartCallbackName(service, method), " ", method.GoName, "Start;")
			g.P(messageCGOServerClientStreamSendCallbackName(service, method), " ", method.GoName, "Send;")
			g.P(messageCGOServerClientStreamFinishCallbackName(service, method), " ", method.GoName, "Finish;")
			g.P(messageCGOServerClientStreamCancelCallbackName(service, method), " ", method.GoName, "Cancel;")
		case StreamingKindServerStreaming:
			g.P(messageCGOServerServerStreamStartCallbackName(service, method), " ", method.GoName, "Start;")
			g.P(messageCGOServerServerStreamRecvCallbackName(service, method), " ", method.GoName, "Recv;")
			g.P(messageCGOServerServerStreamDoneCallbackName(service, method), " ", method.GoName, "Done;")
			g.P(messageCGOServerServerStreamCancelCallbackName(service, method), " ", method.GoName, "Cancel;")
		case StreamingKindBidiStreaming:
			g.P(messageCGOServerBidiStreamStartCallbackName(service, method), " ", method.GoName, "Start;")
			g.P(messageCGOServerBidiStreamSendCallbackName(service, method), " ", method.GoName, "Send;")
			g.P(messageCGOServerBidiStreamRecvCallbackName(service, method), " ", method.GoName, "Recv;")
			g.P(messageCGOServerBidiStreamCloseSendCallbackName(service, method), " ", method.GoName, "CloseSend;")
			g.P(messageCGOServerBidiStreamDoneCallbackName(service, method), " ", method.GoName, "Done;")
			g.P(messageCGOServerBidiStreamCancelCallbackName(service, method), " ", method.GoName, "Cancel;")
		}
	}
	g.P("} ", service.GoName, "CGOMessageServerCallbacks;")
	g.P()
	for _, method := range service.Methods {
		renderCGOMessageServerTrampolines(g, service, method)
	}
	g.P("*/")
}

func renderCGOMessageServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName string) {
	g.P("func (a *", adapterName, ") ", method.GoName, "Message(ctx context.Context, req []byte) ([]byte, error) {")
	g.P("if a == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	g.P("callback := a.callbacks.", method.GoName)
	g.P("if callback == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
	g.P("}")
	g.P("var requestPtr uintptr")
	g.P("if len(req) != 0 {")
	g.P("requestPtr = uintptr(unsafe.Pointer(&req[0]))")
	g.P("}")
	g.P("requestLen, err := rpcruntime.LengthToInt32(len(req))")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("var responsePtr C.uintptr_t")
	g.P("var responseLen C.int32_t")
	g.P("errID := int32(C.", messageCGOServerUnaryTrampolineName(service, method), "(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))")
	g.P("if errID != 0 {")
	g.P("return nil, ", messageCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("if responseLen < 0 {")
	g.P(`return nil, errors.New("rpccgo: message server response length is negative")`)
	g.P("}")
	g.P("if responseLen == 0 {")
	g.P("return nil, nil")
	g.P("}")
	g.P("if responsePtr == 0 {")
	g.P(`return nil, errors.New("rpccgo: message server response pointer is nil")`)
	g.P("}")
	g.P("return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(uintptr(responsePtr))), int(responseLen))...), nil")
	g.P("}")
	g.P()
}

func renderCGOMessageStreamEOFHelper(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("func ", service.GoName, "CGOMessageStreamEOFErrorID() int32 {")
	g.P("return int32(rpcruntime.StoreError(io.EOF))")
	g.P("}")
	g.P()
}

func renderCGOMessageErrorIDHelper(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("func ", messageCGOServerErrorIDHelperName(service), "(errID int32) error {")
	g.P("if errID == 0 {")
	g.P("return nil")
	g.P("}")
	g.P("text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))")
	g.P("if ok {")
	g.P("if ptr != 0 {")
	g.P("defer rpcruntime.Release(ptr)")
	g.P("}")
	g.P("if string(text) == io.EOF.Error() {")
	g.P("return io.EOF")
	g.P("}")
	g.P("return errors.New(string(text))")
	g.P("}")
	g.P(`return fmt.Errorf("rpccgo: cgo message server callback returned unknown error id %d", errID)`)
	g.P("}")
	g.P()
}

func renderCGOMessageServerCallbackValidation(g *protogen.GeneratedFile, service ServicePlan, servicePackage string) {
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("if callbacks.", method.GoName, " == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "MessageAdapter]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
			g.P("}")
		case StreamingKindClientStreaming:
			for _, suffix := range []string{"Start", "Send", "Finish", "Cancel"} {
				g.P("if callbacks.", method.GoName, suffix, " == nil {")
				g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "MessageAdapter]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
				g.P("}")
			}
		case StreamingKindServerStreaming:
			for _, suffix := range []string{"Start", "Recv", "Done", "Cancel"} {
				g.P("if callbacks.", method.GoName, suffix, " == nil {")
				g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "MessageAdapter]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
				g.P("}")
			}
		case StreamingKindBidiStreaming:
			for _, suffix := range []string{"Start", "Send", "Recv", "CloseSend", "Done", "Cancel"} {
				g.P("if callbacks.", method.GoName, suffix, " == nil {")
				g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "MessageAdapter]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
				g.P("}")
			}
		}
	}
}

func renderCGOMessageServerTrampolines(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	switch method.Streaming {
	case StreamingKindUnary:
		g.P("static inline int32_t ", messageCGOServerUnaryTrampolineName(service, method), "(", messageCGOServerUnaryCallbackName(service, method), " callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {")
		g.P("	return callback(request_ptr, request_len, response_ptr, response_len);")
		g.P("}")
	case StreamingKindClientStreaming:
		g.P("static inline int32_t ", messageCGOServerClientStreamStartTrampolineName(service, method), "(", messageCGOServerClientStreamStartCallbackName(service, method), " callback, int32_t* stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerClientStreamSendTrampolineName(service, method), "(", messageCGOServerClientStreamSendCallbackName(service, method), " callback, int32_t stream, uintptr_t request_ptr, int32_t request_len) { return callback(stream, request_ptr, request_len); }")
		g.P("static inline int32_t ", messageCGOServerClientStreamFinishTrampolineName(service, method), "(", messageCGOServerClientStreamFinishCallbackName(service, method), " callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }")
		g.P("static inline int32_t ", messageCGOServerClientStreamCancelTrampolineName(service, method), "(", messageCGOServerClientStreamCancelCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
	case StreamingKindServerStreaming:
		g.P("static inline int32_t ", messageCGOServerServerStreamStartTrampolineName(service, method), "(", messageCGOServerServerStreamStartCallbackName(service, method), " callback, uintptr_t request_ptr, int32_t request_len, int32_t* stream) { return callback(request_ptr, request_len, stream); }")
		g.P("static inline int32_t ", messageCGOServerServerStreamRecvTrampolineName(service, method), "(", messageCGOServerServerStreamRecvCallbackName(service, method), " callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }")
		g.P("static inline int32_t ", messageCGOServerServerStreamDoneTrampolineName(service, method), "(", messageCGOServerServerStreamDoneCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerServerStreamCancelTrampolineName(service, method), "(", messageCGOServerServerStreamCancelCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
	case StreamingKindBidiStreaming:
		g.P("static inline int32_t ", messageCGOServerBidiStreamStartTrampolineName(service, method), "(", messageCGOServerBidiStreamStartCallbackName(service, method), " callback, int32_t* stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamSendTrampolineName(service, method), "(", messageCGOServerBidiStreamSendCallbackName(service, method), " callback, int32_t stream, uintptr_t request_ptr, int32_t request_len) { return callback(stream, request_ptr, request_len); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamRecvTrampolineName(service, method), "(", messageCGOServerBidiStreamRecvCallbackName(service, method), " callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamCloseSendTrampolineName(service, method), "(", messageCGOServerBidiStreamCloseSendCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamDoneTrampolineName(service, method), "(", messageCGOServerBidiStreamDoneCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamCancelTrampolineName(service, method), "(", messageCGOServerBidiStreamCancelCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
	}
	g.P()
}

func renderCGOMessageServerClientStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName, servicePackage string) {
	sessionName := lowerInitial(service.GoName) + method.GoName + "CGOMessageClientStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "Message(ctx context.Context) (", servicePackage, service.GoName, method.GoName, "MessageStreamSession, error) {")
	g.P("if a == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	g.P("if a.callbacks.", method.GoName, "Start == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
	g.P("}")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerClientStreamStartTrampolineName(service, method), "(a.callbacks.", method.GoName, "Start, &stream))")
	g.P("if errID != 0 {")
	g.P("return nil, ", messageCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", sessionName, "{callbacks: a.callbacks, stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("type ", sessionName, " struct {")
	g.P("callbacks C.", service.GoName, "CGOMessageServerCallbacks")
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Send(ctx context.Context, req []byte) error {")
	renderCGOMessageRequestPtrLen(g, "req", "return err")
	g.P("errID := int32(C.", messageCGOServerClientStreamSendTrampolineName(service, method), "(s.callbacks.", method.GoName, "Send, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Finish(ctx context.Context) ([]byte, error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerClientStreamFinishTrampolineName(service, method), "(s.callbacks.", method.GoName, "Finish, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerClientStreamCancelTrampolineName(service, method), "(s.callbacks.", method.GoName, "Cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName, servicePackage string) {
	sessionName := lowerInitial(service.GoName) + method.GoName + "CGOMessageServerStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "Message(ctx context.Context, req []byte) (", servicePackage, service.GoName, method.GoName, "MessageStreamSession, error) {")
	renderCGOMessageStartGuard(g, service, method)
	renderCGOMessageRequestPtrLen(g, "req", "return nil, err")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerServerStreamStartTrampolineName(service, method), "(a.callbacks.", method.GoName, "Start, C.uintptr_t(requestPtr), C.int32_t(requestLen), &stream))")
	g.P("if errID != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return &", sessionName, "{callbacks: a.callbacks, stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("type ", sessionName, " struct {")
	g.P("callbacks C.", service.GoName, "CGOMessageServerCallbacks")
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Recv(ctx context.Context) ([]byte, error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerServerStreamRecvTrampolineName(service, method), "(s.callbacks.", method.GoName, "Recv, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Done(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerServerStreamDoneTrampolineName(service, method), "(s.callbacks.", method.GoName, "Done, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerServerStreamCancelTrampolineName(service, method), "(s.callbacks.", method.GoName, "Cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName, servicePackage string) {
	sessionName := lowerInitial(service.GoName) + method.GoName + "CGOMessageBidiStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "Message(ctx context.Context) (", servicePackage, service.GoName, method.GoName, "MessageStreamSession, error) {")
	renderCGOMessageStartGuard(g, service, method)
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerBidiStreamStartTrampolineName(service, method), "(a.callbacks.", method.GoName, "Start, &stream))")
	g.P("if errID != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return &", sessionName, "{callbacks: a.callbacks, stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("type ", sessionName, " struct {")
	g.P("callbacks C.", service.GoName, "CGOMessageServerCallbacks")
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Send(ctx context.Context, req []byte) error {")
	renderCGOMessageRequestPtrLen(g, "req", "return err")
	g.P("errID := int32(C.", messageCGOServerBidiStreamSendTrampolineName(service, method), "(s.callbacks.", method.GoName, "Send, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Recv(ctx context.Context) ([]byte, error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerBidiStreamRecvTrampolineName(service, method), "(s.callbacks.", method.GoName, "Recv, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") CloseSend(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamCloseSendTrampolineName(service, method), "(s.callbacks.", method.GoName, "CloseSend, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Done(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamDoneTrampolineName(service, method), "(s.callbacks.", method.GoName, "Done, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamCancelTrampolineName(service, method), "(s.callbacks.", method.GoName, "Cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageStartGuard(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("if a == nil { return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil }")
	g.P("if a.callbacks.", method.GoName, "Start == nil { return nil, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing }")
}

func renderCGOMessageRequestPtrLen(g *protogen.GeneratedFile, dataName, errReturn string) {
	g.P("var requestPtr uintptr")
	g.P("if len(", dataName, ") != 0 { requestPtr = uintptr(unsafe.Pointer(&", dataName, "[0])) }")
	g.P("requestLen, err := rpcruntime.LengthToInt32(len(", dataName, "))")
	g.P("if err != nil { ", errReturn, " }")
}

func renderCGOMessageResponseVars(g *protogen.GeneratedFile) {
	g.P("var responsePtr C.uintptr_t")
	g.P("var responseLen C.int32_t")
}

func renderCGOMessageResponseReturn(g *protogen.GeneratedFile, service ServicePlan, errIDName string) {
	g.P("if ", errIDName, " != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(", errIDName, ") }")
	g.P("if responseLen < 0 { return nil, errors.New(\"rpccgo: message server response length is negative\") }")
	g.P("if responseLen == 0 { return nil, nil }")
	g.P("if responsePtr == 0 { return nil, errors.New(\"rpccgo: message server response pointer is nil\") }")
	g.P("return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(uintptr(responsePtr))), int(responseLen))...), nil")
}

func messageCGOServerUnaryCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageUnaryCallback"
}

func messageCGOServerUnaryTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageUnary"
}

func messageCGOServerErrorIDHelperName(service ServicePlan) string {
	return lowerInitial(service.GoName) + "CGOMessageServerError"
}

func messageCGOServerClientStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageClientStreamStartCallback"
}

func messageCGOServerClientStreamSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageClientStreamSendCallback"
}

func messageCGOServerClientStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageClientStreamFinishCallback"
}

func messageCGOServerClientStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageClientStreamCancelCallback"
}

func messageCGOServerClientStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageClientStreamStart"
}

func messageCGOServerClientStreamSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageClientStreamSend"
}

func messageCGOServerClientStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageClientStreamFinish"
}

func messageCGOServerClientStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageClientStreamCancel"
}

func messageCGOServerServerStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageServerStreamStartCallback"
}

func messageCGOServerServerStreamRecvCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageServerStreamRecvCallback"
}

func messageCGOServerServerStreamDoneCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageServerStreamDoneCallback"
}

func messageCGOServerServerStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageServerStreamCancelCallback"
}

func messageCGOServerServerStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageServerStreamStart"
}

func messageCGOServerServerStreamRecvTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageServerStreamRecv"
}

func messageCGOServerServerStreamDoneTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageServerStreamDone"
}

func messageCGOServerServerStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageServerStreamCancel"
}

func messageCGOServerBidiStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamStartCallback"
}

func messageCGOServerBidiStreamSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamSendCallback"
}

func messageCGOServerBidiStreamRecvCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamRecvCallback"
}

func messageCGOServerBidiStreamCloseSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamCloseSendCallback"
}

func messageCGOServerBidiStreamDoneCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamDoneCallback"
}

func messageCGOServerBidiStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamCancelCallback"
}

func messageCGOServerBidiStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamStart"
}

func messageCGOServerBidiStreamSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamSend"
}

func messageCGOServerBidiStreamRecvTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamRecv"
}

func messageCGOServerBidiStreamCloseSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamCloseSend"
}

func messageCGOServerBidiStreamDoneTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamDone"
}

func messageCGOServerBidiStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamCancel"
}
