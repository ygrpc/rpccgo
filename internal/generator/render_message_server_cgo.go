package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
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
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
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
	g.P(lowerInitial(service.GoName), `CGOMessageServerStreamPartiallyRegistered = errors.New("rpccgo: `, service.GoName, ` cgo message server stream callbacks are partially registered")`)
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu sync.Mutex")
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter = &", adapterName, "{}")
	g.P(")")
	g.P()

	g.P("type ", adapterName, " struct {")
	renderCGOMessageServerAdapterFields(g, service)
	g.P("}")
	g.P()
	renderCGOMessageRecvWaiter(g, service)

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGOMessageServerUnaryAdapter(g, service, method, adapterName)
		case StreamingKindClientStreaming:
			renderCGOMessageServerClientStreamAdapter(g, service, method, adapterName)
		case StreamingKindServerStreaming:
			renderCGOMessageServerServerStreamAdapter(g, service, method, adapterName)
		case StreamingKindBidiStreaming:
			renderCGOMessageServerBidiStreamAdapter(g, service, method, adapterName)
		}
	}

	renderCGOMessageServerRegistration(g, plan, service, adapterName, servicePackage)

	renderCGOMessageErrorIDHelper(g, service)
	return nil
}

// renderCGOMessageRecvWaiter emits the coordination types that let Finish or Cancel interrupt a blocking cgo message Recv callback.
func renderCGOMessageRecvWaiter(g *protogen.GeneratedFile, service ServicePlan) {
	prefix := lowerInitial(service.GoName)
	g.P("// ", prefix, "CGOMessageRecvResult carries the result of a blocking cgo message Recv callback.")
	g.P("type ", prefix, "CGOMessageRecvResult[T any] struct {")
	g.P("value T")
	g.P("err error")
	g.P("}")
	g.P()
	g.P("// ", prefix, "AwaitCGOMessageRecv waits for a blocking cgo message Recv callback while allowing Finish or Cancel to interrupt the wait.")
	g.P("func ", prefix, "AwaitCGOMessageRecv[T any](ctx context.Context, finishRequested <-chan struct{}, recv func() (T, error), finish func() error, cancel func() error) (T, error, bool) {")
	g.P("select {")
	g.P("case <-finishRequested:")
	g.P("var zero T")
	g.P("return zero, finish(), true")
	g.P("case <-ctx.Done():")
	g.P("var zero T")
	g.P("return zero, errors.Join(ctx.Err(), cancel()), true")
	g.P("default:")
	g.P("}")
	g.P("results := make(chan ", prefix, "CGOMessageRecvResult[T], 1)")
	g.P("go func() {")
	g.P("value, err := recv()")
	g.P("results <- ", prefix, "CGOMessageRecvResult[T]{value: value, err: err}")
	g.P("}()")
	g.P("var zero T")
	g.P("select {")
	g.P("case result := <-results:")
	g.P("return result.value, result.err, false")
	g.P("case <-finishRequested:")
	g.P("return zero, finish(), true")
	g.P("case <-ctx.Done():")
	g.P("return zero, errors.Join(ctx.Err(), cancel()), true")
	g.P("}")
	g.P("}")
	g.P()
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
			g.P("typedef int32_t (*", messageCGOServerServerStreamFinishCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", messageCGOServerServerStreamCancelCallbackName(service, method), ")(int32_t stream);")
		case StreamingKindBidiStreaming:
			g.P("typedef int32_t (*", messageCGOServerBidiStreamStartCallbackName(service, method), ")(int32_t* stream);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamSendCallbackName(service, method), ")(int32_t stream, uintptr_t request_ptr, int32_t request_len);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamRecvCallbackName(service, method), ")(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamCloseSendCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", messageCGOServerBidiStreamFinishCallbackName(service, method), ")(int32_t stream);")
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
			g.P(cgoMessageServerCallbackFieldName(method, "Start"), " C.", messageCGOServerClientStreamStartCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Send"), " C.", messageCGOServerClientStreamSendCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Finish"), " C.", messageCGOServerClientStreamFinishCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Cancel"), " C.", messageCGOServerClientStreamCancelCallbackName(service, method))
		case StreamingKindServerStreaming:
			g.P(cgoMessageServerCallbackFieldName(method, "Start"), " C.", messageCGOServerServerStreamStartCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Recv"), " C.", messageCGOServerServerStreamRecvCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Finish"), " C.", messageCGOServerServerStreamFinishCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Cancel"), " C.", messageCGOServerServerStreamCancelCallbackName(service, method))
		case StreamingKindBidiStreaming:
			g.P(cgoMessageServerCallbackFieldName(method, "Start"), " C.", messageCGOServerBidiStreamStartCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Send"), " C.", messageCGOServerBidiStreamSendCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Recv"), " C.", messageCGOServerBidiStreamRecvCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "CloseSend"), " C.", messageCGOServerBidiStreamCloseSendCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Finish"), " C.", messageCGOServerBidiStreamFinishCallbackName(service, method))
			g.P(cgoMessageServerCallbackFieldName(method, "Cancel"), " C.", messageCGOServerBidiStreamCancelCallbackName(service, method))
		}
	}
}

func renderCGOMessageServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName string) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, req ", messageGoPointerType(g, method.Request), ") (", messageGoPointerType(g, method.Response), ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	g.P("callback := a.", method.GoName, "Callback")
	g.P("if callback == nil {")
	g.P("return nil, ", cgoMessageServerMethodUnimplementedError(service, method))
	g.P("}")
	renderCGOMessageMarshalRequest(g, "req", "reqBytes", "return nil, err")
	renderCGOMessageRequestPtrLen(g, "reqBytes", "return nil, err")
	g.P("var responsePtr C.uintptr_t")
	g.P("var responseLen C.int32_t")
	g.P("errID := int32(C.", messageCGOServerUnaryTrampolineName(service, method), "(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))")
	g.P("if errID != 0 {")
	g.P("return nil, ", messageCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	g.P(`if err := rpcruntime.DecodeMessage(uintptr(responsePtr), int32(responseLen), resp); err != nil { return nil, fmt.Errorf("rpccgo: message server response decode failed: %w", err) }`)
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerRegistration(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, adapterName, servicePackage string) {
	exportName := messageCServiceRegisterExportFuncName(plan, service)
	var params []string
	for _, method := range service.Methods {
		params = append(params, cgoMessageServerRegisterParams(service, method)...)
	}
	renderCGOExportDoc(g, exportName, "registers cgo message callbacks as the current server for "+service.FullName+".")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(", strings.Join(params, ", "), ") C.int32_t {")
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Lock()")
	g.P("defer ", lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Unlock()")
	g.P("next := ", lowerInitial(service.GoName), "CGOMessageServerAdapterForRegister()")
	g.P("var registerErr error")
	renderCGOMessageServerServiceRegistrationAssignments(g, service)
	g.P("if err := ", servicePackage, "Register", service.GoName, "CGOMessageServer(next); err != nil { return C.int32_t(rpcruntime.StoreError(err)) }")
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter = next")
	g.P("if registerErr != nil { return C.int32_t(rpcruntime.StoreError(registerErr)) }")
	g.P("return 0")
	g.P("}")
	g.P()
	for _, method := range service.Methods {
		renderCGOMessageServerMethodRegistration(g, plan, service, method, adapterName, servicePackage)
	}
	g.P("func ", lowerInitial(service.GoName), "CGOMessageServerAdapterForRegister() *", adapterName, " {")
	g.P("registered, err := ", servicePackage, "Load", service.GoName, "RegisteredServer()")
	g.P("if err == nil && registered.Kind == rpcruntime.ServerKindCGOMessage {")
	g.P("if current, ok := registered.Server.(*", adapterName, "); ok {")
	g.P("next := *current")
	g.P("return &next")
	g.P("}")
	g.P("}")
	g.P("return &", adapterName, "{}")
	g.P("}")
	g.P()
}

func renderCGOMessageServerServiceRegistrationAssignments(g *protogen.GeneratedFile, service ServicePlan) {
	for _, method := range service.Methods {
		renderCGOMessageServerServiceMethodAssignment(g, service, method, "next")
	}
}

func renderCGOMessageServerServiceMethodAssignment(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, target string) {
	prefix := lowerInitial(method.GoName)
	suffixes := cgoMessageServerRegisterSuffixes(method)
	if method.Streaming == StreamingKindUnary {
		g.P("if ", prefix, "Callback != nil {")
		g.P(target, ".", method.GoName, "Callback = ", prefix, "Callback")
		g.P("}")
		return
	}
	allNil := make([]string, 0, len(suffixes))
	allPresent := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		param := prefix + suffix
		allNil = append(allNil, param+" == nil")
		allPresent = append(allPresent, param+" != nil")
	}
	g.P("if ", strings.Join(allNil, " && "), " {")
	g.P("// Preserve existing callbacks for methods omitted from a service-level update.")
	g.P("} else if ", strings.Join(allPresent, " && "), " {")
	for _, suffix := range suffixes {
		g.P(target, ".", cgoMessageServerCallbackFieldName(method, suffix), " = ", prefix, suffix)
	}
	g.P("} else {")
	for _, suffix := range suffixes {
		g.P(target, ".", cgoMessageServerCallbackFieldName(method, suffix), " = nil")
	}
	g.P(`registerErr = errors.Join(registerErr, fmt.Errorf("%w: %s", `, lowerInitial(service.GoName), `CGOMessageServerStreamPartiallyRegistered, "`, method.FullName, `"))`)
	g.P("}")
}

func renderCGOMessageServerMethodRegistration(g *protogen.GeneratedFile, plan FilePlan, service ServicePlan, method MethodPlan, adapterName, servicePackage string) {
	exportName := messageCServiceMethodRegisterExportFuncName(plan, service, method)
	renderCGOExportDoc(g, exportName, "registers cgo message callbacks for "+method.FullName+".")
	g.P("//export ", exportName)
	g.P("func ", exportName, "(", strings.Join(cgoMessageServerRegisterParams(service, method), ", "), ") C.int32_t {")
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Lock()")
	g.P("defer ", lowerInitial(service.GoName), "CGOMessageServerAdapterMu.Unlock()")
	g.P("next := ", lowerInitial(service.GoName), "CGOMessageServerAdapterForRegister()")
	g.P("var registerErr error")
	renderCGOMessageServerMethodAssignment(g, service, method, "next")
	g.P("if err := ", servicePackage, "Register", service.GoName, "CGOMessageServer(next); err != nil { return C.int32_t(rpcruntime.StoreError(err)) }")
	g.P(lowerInitial(service.GoName), "CGOMessageServerAdapter = next")
	g.P("if registerErr != nil { return C.int32_t(rpcruntime.StoreError(registerErr)) }")
	g.P("return 0")
	g.P("}")
	g.P()
}

func renderCGOMessageServerMethodAssignment(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, target string) {
	prefix := lowerInitial(method.GoName)
	suffixes := cgoMessageServerRegisterSuffixes(method)
	if method.Streaming == StreamingKindUnary {
		g.P(target, ".", method.GoName, "Callback = ", prefix, "Callback")
		return
	}
	allNil := make([]string, 0, len(suffixes))
	allPresent := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		param := prefix + suffix
		allNil = append(allNil, param+" == nil")
		allPresent = append(allPresent, param+" != nil")
	}
	g.P("if ", strings.Join(allNil, " && "), " {")
	for _, suffix := range suffixes {
		g.P(target, ".", cgoMessageServerCallbackFieldName(method, suffix), " = nil")
	}
	g.P("} else if ", strings.Join(allPresent, " && "), " {")
	for _, suffix := range suffixes {
		g.P(target, ".", cgoMessageServerCallbackFieldName(method, suffix), " = ", prefix, suffix)
	}
	g.P("} else {")
	for _, suffix := range suffixes {
		g.P(target, ".", cgoMessageServerCallbackFieldName(method, suffix), " = nil")
	}
	g.P(`registerErr = errors.Join(registerErr, fmt.Errorf("%w: %s", `, lowerInitial(service.GoName), `CGOMessageServerStreamPartiallyRegistered, "`, method.FullName, `"))`)
	g.P("}")
}

func cgoMessageServerCallbackFieldName(method MethodPlan, suffix string) string {
	if suffix == "Callback" {
		return method.GoName + suffix
	}
	return lowerInitial(method.GoName) + suffix
}

func cgoMessageServerRegisterParams(service ServicePlan, method MethodPlan) []string {
	prefix := lowerInitial(method.GoName)
	switch method.Streaming {
	case StreamingKindUnary:
		return []string{prefix + "Callback C." + messageCGOServerUnaryCallbackName(service, method)}
	case StreamingKindClientStreaming:
		return []string{
			prefix + "Start C." + messageCGOServerClientStreamStartCallbackName(service, method),
			prefix + "Send C." + messageCGOServerClientStreamSendCallbackName(service, method),
			prefix + "Finish C." + messageCGOServerClientStreamFinishCallbackName(service, method),
			prefix + "Cancel C." + messageCGOServerClientStreamCancelCallbackName(service, method),
		}
	case StreamingKindServerStreaming:
		return []string{
			prefix + "Start C." + messageCGOServerServerStreamStartCallbackName(service, method),
			prefix + "Recv C." + messageCGOServerServerStreamRecvCallbackName(service, method),
			prefix + "Finish C." + messageCGOServerServerStreamFinishCallbackName(service, method),
			prefix + "Cancel C." + messageCGOServerServerStreamCancelCallbackName(service, method),
		}
	case StreamingKindBidiStreaming:
		return []string{
			prefix + "Start C." + messageCGOServerBidiStreamStartCallbackName(service, method),
			prefix + "Send C." + messageCGOServerBidiStreamSendCallbackName(service, method),
			prefix + "Recv C." + messageCGOServerBidiStreamRecvCallbackName(service, method),
			prefix + "CloseSend C." + messageCGOServerBidiStreamCloseSendCallbackName(service, method),
			prefix + "Finish C." + messageCGOServerBidiStreamFinishCallbackName(service, method),
			prefix + "Cancel C." + messageCGOServerBidiStreamCancelCallbackName(service, method),
		}
	default:
		return nil
	}
}

func cgoMessageServerRegisterSuffixes(method MethodPlan) []string {
	switch method.Streaming {
	case StreamingKindUnary:
		return []string{"Callback"}
	case StreamingKindClientStreaming:
		return []string{"Start", "Send", "Finish", "Cancel"}
	case StreamingKindServerStreaming:
		return []string{"Start", "Recv", "Finish", "Cancel"}
	case StreamingKindBidiStreaming:
		return []string{"Start", "Send", "Recv", "CloseSend", "Finish", "Cancel"}
	default:
		return nil
	}
}

func messageCServiceRegisterExportFuncName(plan FilePlan, service ServicePlan) string {
	return cgoServiceExportName("msg", plan, service, "register")
}

func messageCServiceMethodRegisterExportFuncName(plan FilePlan, service ServicePlan, method MethodPlan) string {
	return cgoServiceExportName("msg", plan, service, "register", method.GoName)
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
		g.P("static inline int32_t ", messageCGOServerServerStreamFinishTrampolineName(service, method), "(", messageCGOServerServerStreamFinishCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerServerStreamCancelTrampolineName(service, method), "(", messageCGOServerServerStreamCancelCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
	case StreamingKindBidiStreaming:
		g.P("static inline int32_t ", messageCGOServerBidiStreamStartTrampolineName(service, method), "(", messageCGOServerBidiStreamStartCallbackName(service, method), " callback, int32_t* stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamSendTrampolineName(service, method), "(", messageCGOServerBidiStreamSendCallbackName(service, method), " callback, int32_t stream, uintptr_t request_ptr, int32_t request_len) { return callback(stream, request_ptr, request_len); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamRecvTrampolineName(service, method), "(", messageCGOServerBidiStreamRecvCallbackName(service, method), " callback, int32_t stream, uintptr_t* response_ptr, int32_t* response_len) { return callback(stream, response_ptr, response_len); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamCloseSendTrampolineName(service, method), "(", messageCGOServerBidiStreamCloseSendCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamFinishTrampolineName(service, method), "(", messageCGOServerBidiStreamFinishCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
		g.P("static inline int32_t ", messageCGOServerBidiStreamCancelTrampolineName(service, method), "(", messageCGOServerBidiStreamCancelCallbackName(service, method), " callback, int32_t stream) { return callback(stream); }")
	}
	g.P()
}

func renderCGOMessageServerClientStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName string) {
	clientName := lowerInitial(service.GoName) + method.GoName + "CGOMessageClientStreamingClient"
	g.P("func (a *", adapterName, ") ", method.GoName, "Start(ctx context.Context) (", cgoMessageClientStreamingClientType(g, method), ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil")
	g.P("}")
	g.P("if a.", cgoMessageServerCallbackFieldName(method, "Start"), " == nil || a.", cgoMessageServerCallbackFieldName(method, "Send"), " == nil || a.", cgoMessageServerCallbackFieldName(method, "Finish"), " == nil || a.", cgoMessageServerCallbackFieldName(method, "Cancel"), " == nil {")
	g.P("return nil, ", cgoMessageServerMethodUnimplementedError(service, method))
	g.P("}")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerClientStreamStartTrampolineName(service, method), "(a.", cgoMessageServerCallbackFieldName(method, "Start"), ", &stream))")
	g.P("if errID != 0 {")
	g.P("return nil, ", messageCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", clientName, "{send: a.", cgoMessageServerCallbackFieldName(method, "Send"), ", finish: a.", cgoMessageServerCallbackFieldName(method, "Finish"), ", cancel: a.", cgoMessageServerCallbackFieldName(method, "Cancel"), ", stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", cgoMessageClientStreamType(g, method), ") (", messageGoPointerType(g, method.Response), ", error) {")
	g.P("session, err := a.", method.GoName, "Start(ctx)")
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
	g.P("type ", clientName, " struct {")
	g.P("send C.", messageCGOServerClientStreamSendCallbackName(service, method))
	g.P("finish C.", messageCGOServerClientStreamFinishCallbackName(service, method))
	g.P("cancel C.", messageCGOServerClientStreamCancelCallbackName(service, method))
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Send(ctx context.Context, req ", messageGoPointerType(g, method.Request), ") error {")
	renderCGOMessageMarshalRequest(g, "req", "reqBytes", "return err")
	renderCGOMessageRequestPtrLen(g, "reqBytes", "return err")
	g.P("errID := int32(C.", messageCGOServerClientStreamSendTrampolineName(service, method), "(s.send, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Finish(ctx context.Context) (", messageGoPointerType(g, method.Response), ", error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerClientStreamFinishTrampolineName(service, method), "(s.finish, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, method, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerClientStreamCancelTrampolineName(service, method), "(s.cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName string) {
	clientName := lowerInitial(service.GoName) + method.GoName + "CGOMessageServerStreamingClient"
	g.P("func (a *", adapterName, ") ", method.GoName, "Start(ctx context.Context, req ", messageGoPointerType(g, method.Request), ") (", cgoMessageServerStreamingClientType(g, method), ", error) {")
	renderCGOMessageStartGuard(g, service, method)
	renderCGOMessageMarshalRequest(g, "req", "reqBytes", "return nil, err")
	renderCGOMessageRequestPtrLen(g, "reqBytes", "return nil, err")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerServerStreamStartTrampolineName(service, method), "(a.", cgoMessageServerCallbackFieldName(method, "Start"), ", C.uintptr_t(requestPtr), C.int32_t(requestLen), &stream))")
	g.P("if errID != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return &", clientName, "{recv: a.", cgoMessageServerCallbackFieldName(method, "Recv"), ", finish: a.", cgoMessageServerCallbackFieldName(method, "Finish"), ", cancel: a.", cgoMessageServerCallbackFieldName(method, "Cancel"), ", stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, req ", messageGoPointerType(g, method.Request), ", stream ", cgoMessageServerStreamType(g, method), ") error {")
	g.P("session, err := a.", method.GoName, "Start(ctx, req)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("finishSession := func() error {")
	g.P("finisher, ok := any(session).(interface{ Finish(context.Context) error })")
	g.P("if !ok { return nil }")
	g.P("return finisher.Finish(ctx)")
	g.P("}")
	g.P("for {")
	g.P("resp, err, stopped := ", lowerInitial(service.GoName), "AwaitCGOMessageRecv(ctx, stream.FinishRequested(), func() (", messageGoPointerType(g, method.Response), ", error) { return session.Recv(ctx) }, finishSession, func() error { return session.Cancel(ctx) })")
	g.P("if stopped {")
	g.P("return err")
	g.P("}")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return finishSession()")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("if err := stream.Send(ctx, resp); err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return finishSession()")
	g.P("}")
	g.P("_ = session.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
	g.P("type ", clientName, " struct {")
	g.P("recv C.", messageCGOServerServerStreamRecvCallbackName(service, method))
	g.P("finish C.", messageCGOServerServerStreamFinishCallbackName(service, method))
	g.P("cancel C.", messageCGOServerServerStreamCancelCallbackName(service, method))
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Recv(ctx context.Context) (", messageGoPointerType(g, method.Response), ", error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerServerStreamRecvTrampolineName(service, method), "(s.recv, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, method, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Finish(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerServerStreamFinishTrampolineName(service, method), "(s.finish, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerServerStreamCancelTrampolineName(service, method), "(s.cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, adapterName string) {
	clientName := lowerInitial(service.GoName) + method.GoName + "CGOMessageBidiStreamingClient"
	g.P("func (a *", adapterName, ") ", method.GoName, "Start(ctx context.Context) (", cgoMessageBidiStreamingClientType(g, method), ", error) {")
	renderCGOMessageStartGuard(g, service, method)
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", messageCGOServerBidiStreamStartTrampolineName(service, method), "(a.", cgoMessageServerCallbackFieldName(method, "Start"), ", &stream))")
	g.P("if errID != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return &", clientName, "{send: a.", cgoMessageServerCallbackFieldName(method, "Send"), ", recv: a.", cgoMessageServerCallbackFieldName(method, "Recv"), ", closeSend: a.", cgoMessageServerCallbackFieldName(method, "CloseSend"), ", finish: a.", cgoMessageServerCallbackFieldName(method, "Finish"), ", cancel: a.", cgoMessageServerCallbackFieldName(method, "Cancel"), ", stream: int32(stream)}, nil")
	g.P("}")
	g.P()
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, stream ", cgoMessageBidiStreamType(g, method), ") error {")
	g.P("session, err := a.", method.GoName, "Start(ctx)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("bridgeCtx, cancelBridge := context.WithCancel(ctx)")
	g.P("defer cancelBridge()")
	g.P("cancelSession := sync.OnceValue(func() error { return session.Cancel(bridgeCtx) })")
	g.P("errs := make(chan error, 2)")
	g.P("go func() {")
	g.P("for {")
	g.P("req, err := stream.Recv(bridgeCtx)")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.CloseSend(bridgeCtx)")
	g.P("return")
	g.P("}")
	g.P("if err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if err := session.Send(bridgeCtx, req); err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("go func() {")
	g.P("for {")
	g.P("resp, err, stopped := ", lowerInitial(service.GoName), "AwaitCGOMessageRecv(bridgeCtx, stream.FinishRequested(), func() (", messageGoPointerType(g, method.Response), ", error) { return session.Recv(bridgeCtx) }, func() error { return session.Finish(bridgeCtx) }, cancelSession)")
	g.P("if stopped {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.Finish(bridgeCtx)")
	g.P("return")
	g.P("}")
	g.P("if err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if err := stream.Send(bridgeCtx, resp); err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("if finishErr := session.Finish(bridgeCtx); finishErr != nil {")
	g.P("errs <- errors.Join(err, finishErr)")
	g.P("return")
	g.P("}")
	g.P("errs <- nil")
	g.P("return")
	g.P("}")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("var resultErr error")
	g.P("for range 2 {")
	g.P("if err := <-errs; err != nil {")
	g.P("if resultErr == nil {")
	g.P("_ = cancelSession()")
	g.P("cancelBridge()")
	g.P("}")
	g.P("resultErr = errors.Join(resultErr, err)")
	g.P("}")
	g.P("}")
	g.P("return resultErr")
	g.P("}")
	g.P()
	g.P("type ", clientName, " struct {")
	g.P("send C.", messageCGOServerBidiStreamSendCallbackName(service, method))
	g.P("recv C.", messageCGOServerBidiStreamRecvCallbackName(service, method))
	g.P("closeSend C.", messageCGOServerBidiStreamCloseSendCallbackName(service, method))
	g.P("finish C.", messageCGOServerBidiStreamFinishCallbackName(service, method))
	g.P("cancel C.", messageCGOServerBidiStreamCancelCallbackName(service, method))
	g.P("stream int32")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Send(ctx context.Context, req ", messageGoPointerType(g, method.Request), ") error {")
	renderCGOMessageMarshalRequest(g, "req", "reqBytes", "return err")
	renderCGOMessageRequestPtrLen(g, "reqBytes", "return err")
	g.P("errID := int32(C.", messageCGOServerBidiStreamSendTrampolineName(service, method), "(s.send, C.int32_t(s.stream), C.uintptr_t(requestPtr), C.int32_t(requestLen)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Recv(ctx context.Context) (", messageGoPointerType(g, method.Response), ", error) {")
	renderCGOMessageResponseVars(g)
	g.P("errID := int32(C.", messageCGOServerBidiStreamRecvTrampolineName(service, method), "(s.recv, C.int32_t(s.stream), &responsePtr, &responseLen))")
	renderCGOMessageResponseReturn(g, service, method, "errID")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") CloseSend(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamCloseSendTrampolineName(service, method), "(s.closeSend, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Finish(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamFinishTrampolineName(service, method), "(s.finish, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
	g.P("func (s *", clientName, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", messageCGOServerBidiStreamCancelTrampolineName(service, method), "(s.cancel, C.int32_t(s.stream)))")
	g.P("if errID != 0 { return ", messageCGOServerErrorIDHelperName(service), "(errID) }")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGOMessageStartGuard(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("if a == nil { return nil, ", lowerInitial(service.GoName), "CGOMessageServerCallbacksNil }")
	suffixes := cgoMessageServerRegisterSuffixes(method)
	conditions := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		conditions = append(conditions, "a."+cgoMessageServerCallbackFieldName(method, suffix)+" == nil")
	}
	g.P("if ", strings.Join(conditions, " || "), " { return nil, ", cgoMessageServerMethodUnimplementedError(service, method), " }")
}

func cgoMessageServerMethodUnimplementedError(service ServicePlan, method MethodPlan) string {
	return `errors.New("rpccgo: ` + service.GoName + `.` + method.GoName + ` cgo message server method is not implemented")`
}

func renderCGOMessageRequestPtrLen(g *protogen.GeneratedFile, dataName, errReturn string) {
	g.P("var requestPtr uintptr")
	g.P("if len(", dataName, ") != 0 { requestPtr = uintptr(unsafe.Pointer(&", dataName, "[0])) }")
	g.P("requestLen, err := rpcruntime.LengthToInt32(len(", dataName, "))")
	g.P("if err != nil { ", errReturn, " }")
}

func renderCGOMessageMarshalRequest(g *protogen.GeneratedFile, valueName, dataName, errReturn string) {
	g.P("if ", valueName, " == nil {")
	g.P(`err := errors.New("rpccgo: message request is nil")`)
	g.P(errReturn)
	g.P("}")
	g.P(dataName, ", err := protobuf.Marshal(", valueName, ")")
	g.P("if err != nil {")
	g.P(`err = fmt.Errorf("rpccgo: message request protobuf marshal failed: %w", err)`)
	g.P(errReturn)
	g.P("}")
}

func renderCGOMessageResponseVars(g *protogen.GeneratedFile) {
	g.P("var responsePtr C.uintptr_t")
	g.P("var responseLen C.int32_t")
}

func renderCGOMessageResponseReturn(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errIDName string) {
	g.P("if ", errIDName, " != 0 { return nil, ", messageCGOServerErrorIDHelperName(service), "(", errIDName, ") }")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	g.P(`if err := rpcruntime.DecodeMessage(uintptr(responsePtr), int32(responseLen), resp); err != nil { return nil, fmt.Errorf("rpccgo: message server response decode failed: %w", err) }`)
	g.P("return resp, nil")
}

func cgoMessageClientStreamType(g *protogen.GeneratedFile, method MethodPlan) string {
	return "rpcruntime.ClientStreamingServer[" + messageGoPointerType(g, method.Request) + "]"
}

func cgoMessageServerStreamType(g *protogen.GeneratedFile, method MethodPlan) string {
	return "rpcruntime.ServerStreamingServer[" + messageGoPointerType(g, method.Response) + "]"
}

func cgoMessageBidiStreamType(g *protogen.GeneratedFile, method MethodPlan) string {
	return "rpcruntime.BidiStreamingServer[" + messageGoPointerType(g, method.Request) + ", " + messageGoPointerType(g, method.Response) + "]"
}

func cgoMessageClientStreamingClientType(g *protogen.GeneratedFile, method MethodPlan) string {
	return "rpcruntime.ClientStreamingClient[" + messageGoPointerType(g, method.Request) + ", " + messageGoPointerType(g, method.Response) + "]"
}

func cgoMessageServerStreamingClientType(g *protogen.GeneratedFile, method MethodPlan) string {
	return "rpcruntime.ServerStreamingClient[" + messageGoPointerType(g, method.Response) + "]"
}

func cgoMessageBidiStreamingClientType(g *protogen.GeneratedFile, method MethodPlan) string {
	return "rpcruntime.BidiStreamingClient[" + messageGoPointerType(g, method.Request) + ", " + messageGoPointerType(g, method.Response) + "]"
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

func messageCGOServerServerStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageServerStreamFinishCallback"
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

func messageCGOServerServerStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageServerStreamFinish"
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

func messageCGOServerBidiStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGOMessageBidiStreamFinishCallback"
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

func messageCGOServerBidiStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamFinish"
}

func messageCGOServerBidiStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGOMessageBidiStreamCancel"
}
