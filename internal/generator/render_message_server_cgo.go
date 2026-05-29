package generator

import (
	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := newGeneratedFile(plugin, plan, file, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"CGOMessageServer")

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
	g.P(`protobuf "google.golang.org/protobuf/proto"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(`sync "sync"`)
	g.P(`unsafe "unsafe"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	adapterName := lowerInitial(service.GoName) + "CGOMessageAdapter"
	g.P("var (")
	g.P(lowerInitial(service.GoName), `CGOMessageServerCallbacksNil = errors.New("rpccgo: `, service.GoName, ` cgo message server callbacks are nil")`)
	g.P(lowerInitial(service.GoName), `CGOMessageServerUnaryCallbackMissing = errors.New("rpccgo: `, service.GoName, ` cgo message server unary callback is missing")`)
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu sync.Mutex")
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter = &", adapterName, "{}")
	g.P(")")
	g.P()

	g.P("type ", adapterName, " struct {")
	renderCGOMessageServerAdapterFields(g, service)
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
		renderCGOMessageResponseBytesHelper(g, service, method)
	}

	renderCGOMessageServerRegistration(g, plan, service, adapterName, servicePackage)

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
	for _, method := range service.Methods {
		renderCGOMessageServerTrampolines(g, service, method)
	}
	g.P("*/")
}

func renderCGOMessageServerAdapterFields(g *protogen.GeneratedFile, service ServicePlan) {
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(method.GoName, "Callback C.", messageCGOServerUnaryCallbackName(service, method))
		case StreamingKindClientStreaming:
			g.P(method.GoName, "Start C.", messageCGOServerClientStreamStartCallbackName(service, method))
			g.P(method.GoName, "Send C.", messageCGOServerClientStreamSendCallbackName(service, method))
			g.P(method.GoName, "Finish C.", messageCGOServerClientStreamFinishCallbackName(service, method))
			g.P(method.GoName, "Cancel C.", messageCGOServerClientStreamCancelCallbackName(service, method))
		case StreamingKindServerStreaming:
			g.P(method.GoName, "Start C.", messageCGOServerServerStreamStartCallbackName(service, method))
			g.P(method.GoName, "Recv C.", messageCGOServerServerStreamRecvCallbackName(service, method))
			g.P(method.GoName, "Done C.", messageCGOServerServerStreamDoneCallbackName(service, method))
			g.P(method.GoName, "Cancel C.", messageCGOServerServerStreamCancelCallbackName(service, method))
		case StreamingKindBidiStreaming:
			g.P(method.GoName, "Start C.", messageCGOServerBidiStreamStartCallbackName(service, method))
			g.P(method.GoName, "Send C.", messageCGOServerBidiStreamSendCallbackName(service, method))
			g.P(method.GoName, "Recv C.", messageCGOServerBidiStreamRecvCallbackName(service, method))
			g.P(method.GoName, "CloseSend C.", messageCGOServerBidiStreamCloseSendCallbackName(service, method))
			g.P(method.GoName, "Done C.", messageCGOServerBidiStreamDoneCallbackName(service, method))
			g.P(method.GoName, "Cancel C.", messageCGOServerBidiStreamCancelCallbackName(service, method))
		}
	}
}

func renderCGOMessageServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName string) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, req []byte) ([]byte, error) {")
	g.P("if a == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	g.P("callback := a.", method.GoName, "Callback")
	g.P("if callback == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
	g.P("}")
	renderCGOMessageProtoUnmarshalCheck(g, method.Request, "req", "request", "return nil, fmt.Errorf")
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
	g.P("resp, err := ", messageCGOServerResponseBytesName(service, method), "(responsePtr, responseLen)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	renderCGOMessageProtoUnmarshalCheck(g, method.Response, "resp", "response", "return nil, fmt.Errorf")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerRegistration(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, adapterName, servicePackage string) {
	for _, method := range service.Methods {
		exportName := messageCExportFuncName(plan, service, method, "register")
		goName := exportName
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("//export ", exportName)
			g.P("func ", goName, "(callback C.", messageCGOServerUnaryCallbackName(service, method), ") C.int32_t {")
			g.P("if callback == nil { return C.int32_t(rpcruntime.StoreError(", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing)) }")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Lock()")
			g.P("defer ", lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Unlock()")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Callback = callback")
		case StreamingKindClientStreaming:
			g.P("//export ", exportName)
			g.P("func ", goName, "(start C.", messageCGOServerClientStreamStartCallbackName(service, method), ", send C.", messageCGOServerClientStreamSendCallbackName(service, method), ", finish C.", messageCGOServerClientStreamFinishCallbackName(service, method), ", cancel C.", messageCGOServerClientStreamCancelCallbackName(service, method), ") C.int32_t {")
			g.P("if start == nil || send == nil || finish == nil || cancel == nil { return C.int32_t(rpcruntime.StoreError(", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing)) }")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Lock()")
			g.P("defer ", lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Unlock()")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Start = start")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Send = send")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Finish = finish")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Cancel = cancel")
		case StreamingKindServerStreaming:
			g.P("//export ", exportName)
			g.P("func ", goName, "(start C.", messageCGOServerServerStreamStartCallbackName(service, method), ", recv C.", messageCGOServerServerStreamRecvCallbackName(service, method), ", done C.", messageCGOServerServerStreamDoneCallbackName(service, method), ", cancel C.", messageCGOServerServerStreamCancelCallbackName(service, method), ") C.int32_t {")
			g.P("if start == nil || recv == nil || done == nil || cancel == nil { return C.int32_t(rpcruntime.StoreError(", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing)) }")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Lock()")
			g.P("defer ", lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Unlock()")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Start = start")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Recv = recv")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Done = done")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Cancel = cancel")
		case StreamingKindBidiStreaming:
			g.P("//export ", exportName)
			g.P("func ", goName, "(start C.", messageCGOServerBidiStreamStartCallbackName(service, method), ", send C.", messageCGOServerBidiStreamSendCallbackName(service, method), ", recv C.", messageCGOServerBidiStreamRecvCallbackName(service, method), ", closeSend C.", messageCGOServerBidiStreamCloseSendCallbackName(service, method), ", done C.", messageCGOServerBidiStreamDoneCallbackName(service, method), ", cancel C.", messageCGOServerBidiStreamCancelCallbackName(service, method), ") C.int32_t {")
			g.P("if start == nil || send == nil || recv == nil || closeSend == nil || done == nil || cancel == nil { return C.int32_t(rpcruntime.StoreError(", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing)) }")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Lock()")
			g.P("defer ", lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Unlock()")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Start = start")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Send = send")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Recv = recv")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "CloseSend = closeSend")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Done = done")
			g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter.", method.GoName, "Cancel = cancel")
		}
		g.P("_, err := ", servicePackage, "Register", service.GoName, "CGOMessageServer(", lowerInitial(service.GoName), "CGOMessageServerAdapter)")
		g.P("if err != nil { return C.int32_t(rpcruntime.StoreError(err)) }")
		g.P("return 0")
		g.P("}")
		g.P()
	}
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
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "CGOMessageServer]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
			g.P("}")
		case StreamingKindClientStreaming:
			for _, suffix := range []string{"Start", "Send", "Finish", "Cancel"} {
				g.P("if callbacks.", method.GoName, suffix, " == nil {")
				g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "CGOMessageServer]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
				g.P("}")
			}
		case StreamingKindServerStreaming:
			for _, suffix := range []string{"Start", "Recv", "Done", "Cancel"} {
				g.P("if callbacks.", method.GoName, suffix, " == nil {")
				g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "CGOMessageServer]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
				g.P("}")
			}
		case StreamingKindBidiStreaming:
			for _, suffix := range []string{"Start", "Send", "Recv", "CloseSend", "Done", "Cancel"} {
				g.P("if callbacks.", method.GoName, suffix, " == nil {")
				g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "CGOMessageServer]{}, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
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
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", servicePackage, service.GoName, method.GoName, "MessageStreamSession, error) {")
	g.P("if a == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	g.P("if a.", method.GoName, "Start == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing")
	g.P("}")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerClientStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start, &stream))")
	g.P("if errID != 0 {")
	g.P("return nil, ", messageCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", sessionName, "{send: a.", method.GoName, "Send, finish: a.", method.GoName, "Finish, cancel: a.", method.GoName, "Cancel, stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", servicePackage, service.GoName, method.GoName, "MessageClientStream) ([]byte, error) {")
	g.P("session, err := a.Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("for {")
	g.P("req, err := stream.Recv(ctx)")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return session.Finish(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return nil, err")
	g.P("}")
	g.P("if err := session.Send(ctx, req); err != nil {")
	g.P("_ = session.Cancel(ctx)")
	g.P("return nil, err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	g.P("type ", sessionName, " struct {")
	g.P("send C.", messageCGOServerClientStreamSendCallbackName(service, method))
	g.P("finish C.", messageCGOServerClientStreamFinishCallbackName(service, method))
	g.P("cancel C.", messageCGOServerClientStreamCancelCallbackName(service, method))
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Send(ctx context.Context, req []byte) error {")
	renderCGOMessageProtoUnmarshalCheck(g, method.Request, "req", "request", "return fmt.Errorf")
	renderCGOMessageRequestPtrLen(g, "req", "return err")
	g.P("errID := int32(C.", messageCGOServerClientStreamSendTrampolineName(service, method), "(s.send, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Finish(ctx context.Context) ([]byte, error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerClientStreamFinishTrampolineName(service, method), "(s.finish, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, method, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerClientStreamCancelTrampolineName(service, method), "(s.cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName, servicePackage string) {
	sessionName := lowerInitial(service.GoName) + method.GoName + "CGOMessageServerStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context, req []byte) (", servicePackage, service.GoName, method.GoName, "MessageStreamSession, error) {")
	renderCGOMessageStartGuard(g, service, method)
	renderCGOMessageProtoUnmarshalCheck(g, method.Request, "req", "request", "return nil, fmt.Errorf")
	renderCGOMessageRequestPtrLen(g, "req", "return nil, err")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerServerStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start, C.uintptr_t(requestPtr), C.int32_t(requestLen), &stream))")
	g.P("if errID != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return &", sessionName, "{recv: a.", method.GoName, "Recv, done: a.", method.GoName, "Done, cancel: a.", method.GoName, "Cancel, stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, req []byte, stream ", servicePackage, service.GoName, method.GoName, "MessageServerStream) error {")
	g.P("session, err := a.Start", method.GoName, "(ctx, req)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("for {")
	g.P("resp, err := session.Recv(ctx)")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return session.Done(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("if err := stream.Send(ctx, resp); err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return session.Done(ctx)")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	g.P("type ", sessionName, " struct {")
	g.P("recv C.", messageCGOServerServerStreamRecvCallbackName(service, method))
	g.P("done C.", messageCGOServerServerStreamDoneCallbackName(service, method))
	g.P("cancel C.", messageCGOServerServerStreamCancelCallbackName(service, method))
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Recv(ctx context.Context) ([]byte, error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerServerStreamRecvTrampolineName(service, method), "(s.recv, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, method, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Done(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerServerStreamDoneTrampolineName(service, method), "(s.done, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerServerStreamCancelTrampolineName(service, method), "(s.cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName, servicePackage string) {
	sessionName := lowerInitial(service.GoName) + method.GoName + "CGOMessageBidiStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", servicePackage, service.GoName, method.GoName, "MessageStreamSession, error) {")
	renderCGOMessageStartGuard(g, service, method)
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerBidiStreamStartTrampolineName(service, method), "(a.", method.GoName, "Start, &stream))")
	g.P("if errID != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return &", sessionName, "{send: a.", method.GoName, "Send, recv: a.", method.GoName, "Recv, closeSend: a.", method.GoName, "CloseSend, done: a.", method.GoName, "Done, cancel: a.", method.GoName, "Cancel, stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", servicePackage, service.GoName, method.GoName, "MessageBidiStream) error {")
	g.P("session, err := a.Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("return rpcruntime.RunBidiStream(")
	g.P("func() ([]byte, error) { return stream.Recv(ctx) },")
	g.P("func(req []byte) error { return session.Send(ctx, req) },")
	g.P("func() error { return session.CloseSend(ctx) },")
	g.P("func() ([]byte, error) { return session.Recv(ctx) },")
	g.P("func(resp []byte) error {")
	g.P("err := stream.Send(ctx, resp)")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("if doneErr := session.Done(ctx); doneErr != nil {")
	g.P("return errors.Join(err, doneErr)")
	g.P("}")
	g.P("}")
	g.P("return err")
	g.P("},")
	g.P("func() error { return session.Done(ctx) },")
	g.P("func() error { return session.Cancel(ctx) },")
	g.P(")")
	g.P("}")
	g.P()
	g.P("type ", sessionName, " struct {")
	g.P("send C.", messageCGOServerBidiStreamSendCallbackName(service, method))
	g.P("recv C.", messageCGOServerBidiStreamRecvCallbackName(service, method))
	g.P("closeSend C.", messageCGOServerBidiStreamCloseSendCallbackName(service, method))
	g.P("done C.", messageCGOServerBidiStreamDoneCallbackName(service, method))
	g.P("cancel C.", messageCGOServerBidiStreamCancelCallbackName(service, method))
	g.P("stream int32")
	g.P("lifecycle rpcruntime.StreamLifecycle")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Send(ctx context.Context, req []byte) error {")
	g.P("if err := s.lifecycle.EnsureCanSend(); err != nil { return err }")
	renderCGOMessageProtoUnmarshalCheck(g, method.Request, "req", "request", "return fmt.Errorf")
	renderCGOMessageRequestPtrLen(g, "req", "return err")
	g.P("errID := int32(C.", messageCGOServerBidiStreamSendTrampolineName(service, method), "(s.send, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Recv(ctx context.Context) ([]byte, error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerBidiStreamRecvTrampolineName(service, method), "(s.recv, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, method, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") CloseSend(ctx context.Context) error {")
	g.P("if err := s.lifecycle.EnsureCanSend(); err != nil { return err }")
	g.P("errID := int32(C.", messageCGOServerBidiStreamCloseSendTrampolineName(service, method), "(s.closeSend, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("if err := s.lifecycle.MarkSendClosed(); err != nil { return err }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Done(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamDoneTrampolineName(service, method), "(s.done, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", sessionName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamCancelTrampolineName(service, method), "(s.cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageStartGuard(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("if a == nil { return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil }")
	g.P("if a.", method.GoName, "Start == nil { return nil, ", lowerInitial(service.GoName), "CGOMessageServerUnaryCallbackMissing }")
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

func renderCGOMessageResponseBytesHelper(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", messageCGOServerResponseBytesName(service, method), "(responsePtr C.uintptr_t, responseLen C.int32_t) ([]byte, error) {")
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

func renderCGOMessageResponseReturn(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errIDName string) {
	g.P("if ", errIDName, " != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(", errIDName, ") }")
	g.P("resp, err := ", messageCGOServerResponseBytesName(service, method), "(responsePtr, responseLen)")
	g.P("if err != nil { return nil, err }")
	renderCGOMessageProtoUnmarshalCheck(g, method.Response, "resp", "response", "return nil, fmt.Errorf")
	g.P("return resp, nil")
}

func renderCGOMessageProtoUnmarshalCheck(g *protogen.GeneratedFile, message MethodIOPlan, dataName, label, retPrefix string) {
	g.P("if err := protobuf.Unmarshal(", dataName, ", &", g.QualifiedGoIdent(protogen.GoIdent{GoName: message.GoName, GoImportPath: protogen.GoImportPath(message.GoImportPath)}), "{}); err != nil {")
	g.P(retPrefix, `("rpccgo: message `, label, ` protobuf unmarshal failed: %w", err)`)
	g.P("}")
}

func messageCGOServerResponseBytesName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGOMessageResponseBytes"
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
