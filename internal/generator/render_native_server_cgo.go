package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderNativeServerCGOFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	if err := validateNativeServerCGOSymbols(plan, service); err != nil {
		return err
	}

	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := plugin.NewGeneratedFile(file.Filename, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, service.GoName+"NativeAdapter")
	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	runtimeMethods = qualifyRuntimeAdapterMethods(runtimeMethods, servicePackage)

	g.P("package main")
	g.P()
	renderCGONativeServerPreamble(g, service)
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(`unsafe "unsafe"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	errorNames := nativeServerCGOErrorNamesFor(service)
	g.P("var (")
	g.P(errorNames.CallbacksNil, ` = errors.New("rpccgo: `, service.GoName, ` cgo native server callbacks are nil")`)
	g.P(errorNames.UnaryCallbackMissing, ` = errors.New("rpccgo: `, service.GoName, ` cgo native server unary callback is missing")`)
	g.P(errorNames.UnsupportedField, ` = errors.New("rpccgo: cgo native server field bridge is not implemented")`)
	g.P(errorNames.StreamNotImplemented, ` = errors.New("rpccgo: cgo native server streaming is not implemented")`)
	g.P(")")
	g.P()

	callbacksName := service.GoName + "CGONativeServerCallbacks"
	adapterName := lowerInitial(service.GoName) + "CGONativeAdapter"
	renderCGONativeServerAdapter(g, service, runtimeMethods, callbacksName, adapterName, errorNames, servicePackage)
	renderCGONativeServerRegistration(g, service, callbacksName, adapterName, errorNames, servicePackage)
	renderCGONativeServerGoHelper(g, service, runtimeMethods, callbacksName, errorNames, servicePackage)
	renderCGONativeServerErrorStoreExport(g, service)
	return nil
}

func qualifyRuntimeAdapterMethods(methods []runtimeAdapterMethod, servicePackage string) []runtimeAdapterMethod {
	qualified := make([]runtimeAdapterMethod, len(methods))
	copy(qualified, methods)
	for i := range qualified {
		if !qualified[i].Streaming {
			continue
		}
		rawSessionName := qualified[i].SessionName
		qualified[i].SessionName = servicePackage + rawSessionName
		qualified[i].AdapterResult = strings.ReplaceAll(qualified[i].AdapterResult, rawSessionName, qualified[i].SessionName)
	}
	return qualified
}

func renderCGONativeServerPreamble(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("/*")
	g.P("#include <stdint.h>")
	g.P()
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGONativeServerCStruct(g, nativeCGOServerRequestName(service, method), method.NativeContract.RequestFields, false)
			renderCGONativeServerCStruct(g, nativeCGOServerResponseName(service, method), method.NativeContract.ResponseFields, true)
			g.P("typedef int32_t (*", nativeCGOServerCallbackName(service, method), ")(", nativeCGOServerRequestName(service, method), "* input, ", nativeCGOServerResponseName(service, method), "* output);")
			g.P()
		case StreamingKindClientStreaming:
			renderCGONativeServerCStruct(g, nativeCGOServerClientStreamRequestName(service, method), method.NativeContract.RequestFields, false)
			renderCGONativeServerCStruct(g, nativeCGOServerClientStreamResponseName(service, method), method.NativeContract.ResponseFields, true)
			g.P("typedef int32_t (*", nativeCGOServerClientStreamStartCallbackName(service, method), ")(int32_t* stream);")
			g.P("typedef int32_t (*", nativeCGOServerClientStreamSendCallbackName(service, method), ")(int32_t stream, ", nativeCGOServerClientStreamRequestName(service, method), "* input);")
			g.P("typedef int32_t (*", nativeCGOServerClientStreamFinishCallbackName(service, method), ")(int32_t stream, ", nativeCGOServerClientStreamResponseName(service, method), "* output);")
			g.P("typedef int32_t (*", nativeCGOServerClientStreamCancelCallbackName(service, method), ")(int32_t stream);")
			g.P()
		case StreamingKindServerStreaming:
			renderCGONativeServerCStruct(g, nativeCGOServerServerStreamRequestName(service, method), method.NativeContract.RequestFields, false)
			renderCGONativeServerCStruct(g, nativeCGOServerServerStreamResponseName(service, method), method.NativeContract.ResponseFields, true)
			g.P("typedef int32_t (*", nativeCGOServerServerStreamStartCallbackName(service, method), ")(", nativeCGOServerServerStreamRequestName(service, method), "* input, int32_t* stream);")
			g.P("typedef int32_t (*", nativeCGOServerServerStreamRecvCallbackName(service, method), ")(int32_t stream, ", nativeCGOServerServerStreamResponseName(service, method), "* output);")
			g.P("typedef int32_t (*", nativeCGOServerServerStreamDoneCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", nativeCGOServerServerStreamCancelCallbackName(service, method), ")(int32_t stream);")
			g.P()
		case StreamingKindBidiStreaming:
			renderCGONativeServerCStruct(g, nativeCGOServerBidiStreamRequestName(service, method), method.NativeContract.RequestFields, false)
			renderCGONativeServerCStruct(g, nativeCGOServerBidiStreamResponseName(service, method), method.NativeContract.ResponseFields, true)
			g.P("typedef int32_t (*", nativeCGOServerBidiStreamStartCallbackName(service, method), ")(int32_t* stream);")
			g.P("typedef int32_t (*", nativeCGOServerBidiStreamSendCallbackName(service, method), ")(int32_t stream, ", nativeCGOServerBidiStreamRequestName(service, method), "* input);")
			g.P("typedef int32_t (*", nativeCGOServerBidiStreamRecvCallbackName(service, method), ")(int32_t stream, ", nativeCGOServerBidiStreamResponseName(service, method), "* output);")
			g.P("typedef int32_t (*", nativeCGOServerBidiStreamCloseSendCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", nativeCGOServerBidiStreamDoneCallbackName(service, method), ")(int32_t stream);")
			g.P("typedef int32_t (*", nativeCGOServerBidiStreamCancelCallbackName(service, method), ")(int32_t stream);")
			g.P()
		}
	}
	g.P("typedef struct ", service.GoName, "CGONativeServerCallbacks {")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(nativeCGOServerCallbackName(service, method), " ", method.GoName, ";")
		case StreamingKindClientStreaming:
			g.P(nativeCGOServerClientStreamStartCallbackName(service, method), " ", method.GoName, "Start;")
			g.P(nativeCGOServerClientStreamSendCallbackName(service, method), " ", method.GoName, "Send;")
			g.P(nativeCGOServerClientStreamFinishCallbackName(service, method), " ", method.GoName, "Finish;")
			g.P(nativeCGOServerClientStreamCancelCallbackName(service, method), " ", method.GoName, "Cancel;")
		case StreamingKindServerStreaming:
			g.P(nativeCGOServerServerStreamStartCallbackName(service, method), " ", method.GoName, "Start;")
			g.P(nativeCGOServerServerStreamRecvCallbackName(service, method), " ", method.GoName, "Recv;")
			g.P(nativeCGOServerServerStreamDoneCallbackName(service, method), " ", method.GoName, "Done;")
			g.P(nativeCGOServerServerStreamCancelCallbackName(service, method), " ", method.GoName, "Cancel;")
		case StreamingKindBidiStreaming:
			g.P(nativeCGOServerBidiStreamStartCallbackName(service, method), " ", method.GoName, "Start;")
			g.P(nativeCGOServerBidiStreamSendCallbackName(service, method), " ", method.GoName, "Send;")
			g.P(nativeCGOServerBidiStreamRecvCallbackName(service, method), " ", method.GoName, "Recv;")
			g.P(nativeCGOServerBidiStreamCloseSendCallbackName(service, method), " ", method.GoName, "CloseSend;")
			g.P(nativeCGOServerBidiStreamDoneCallbackName(service, method), " ", method.GoName, "Done;")
			g.P(nativeCGOServerBidiStreamCancelCallbackName(service, method), " ", method.GoName, "Cancel;")
		}
	}
	g.P("} ", service.GoName, "CGONativeServerCallbacks;")
	g.P()
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("static inline int32_t ", nativeCGOServerTrampolineName(service, method), "(", nativeCGOServerCallbackName(service, method), " callback, ", nativeCGOServerRequestName(service, method), "* input, ", nativeCGOServerResponseName(service, method), "* output) {")
			g.P("	return callback(input, output);")
			g.P("}")
			g.P()
		case StreamingKindClientStreaming:
			g.P("static inline int32_t ", nativeCGOServerClientStreamStartTrampolineName(service, method), "(", nativeCGOServerClientStreamStartCallbackName(service, method), " callback, int32_t* stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerClientStreamSendTrampolineName(service, method), "(", nativeCGOServerClientStreamSendCallbackName(service, method), " callback, int32_t stream, ", nativeCGOServerClientStreamRequestName(service, method), "* input) {")
			g.P("	return callback(stream, input);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerClientStreamFinishTrampolineName(service, method), "(", nativeCGOServerClientStreamFinishCallbackName(service, method), " callback, int32_t stream, ", nativeCGOServerClientStreamResponseName(service, method), "* output) {")
			g.P("	return callback(stream, output);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerClientStreamCancelTrampolineName(service, method), "(", nativeCGOServerClientStreamCancelCallbackName(service, method), " callback, int32_t stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
		case StreamingKindServerStreaming:
			g.P("static inline int32_t ", nativeCGOServerServerStreamStartTrampolineName(service, method), "(", nativeCGOServerServerStreamStartCallbackName(service, method), " callback, ", nativeCGOServerServerStreamRequestName(service, method), "* input, int32_t* stream) {")
			g.P("	return callback(input, stream);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerServerStreamRecvTrampolineName(service, method), "(", nativeCGOServerServerStreamRecvCallbackName(service, method), " callback, int32_t stream, ", nativeCGOServerServerStreamResponseName(service, method), "* output) {")
			g.P("	return callback(stream, output);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerServerStreamDoneTrampolineName(service, method), "(", nativeCGOServerServerStreamDoneCallbackName(service, method), " callback, int32_t stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerServerStreamCancelTrampolineName(service, method), "(", nativeCGOServerServerStreamCancelCallbackName(service, method), " callback, int32_t stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
		case StreamingKindBidiStreaming:
			g.P("static inline int32_t ", nativeCGOServerBidiStreamStartTrampolineName(service, method), "(", nativeCGOServerBidiStreamStartCallbackName(service, method), " callback, int32_t* stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerBidiStreamSendTrampolineName(service, method), "(", nativeCGOServerBidiStreamSendCallbackName(service, method), " callback, int32_t stream, ", nativeCGOServerBidiStreamRequestName(service, method), "* input) {")
			g.P("	return callback(stream, input);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerBidiStreamRecvTrampolineName(service, method), "(", nativeCGOServerBidiStreamRecvCallbackName(service, method), " callback, int32_t stream, ", nativeCGOServerBidiStreamResponseName(service, method), "* output) {")
			g.P("	return callback(stream, output);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), "(", nativeCGOServerBidiStreamCloseSendCallbackName(service, method), " callback, int32_t stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerBidiStreamDoneTrampolineName(service, method), "(", nativeCGOServerBidiStreamDoneCallbackName(service, method), " callback, int32_t stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
			g.P("static inline int32_t ", nativeCGOServerBidiStreamCancelTrampolineName(service, method), "(", nativeCGOServerBidiStreamCancelCallbackName(service, method), " callback, int32_t stream) {")
			g.P("	return callback(stream);")
			g.P("}")
			g.P()
		}
	}
	g.P("*/")
}

func renderCGONativeServerCStruct(g *protogen.GeneratedFile, name string, fields []FieldPlan, output bool) {
	g.P("typedef struct ", name, " {")
	for _, field := range fields {
		renderCGONativeServerCField(g, field, output)
	}
	g.P("} ", name, ";")
	g.P()
}

func renderCGONativeServerCField(g *protogen.GeneratedFile, field FieldPlan, output bool) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("int8_t ", field.GoName, ";")
	case NativeABIShapeRepeated, NativeABIShapeBoolByteBufferWrapper:
		g.P("uintptr_t ", field.GoName, "Ptr;")
		g.P("int32_t ", field.GoName, "Len;")
		if output {
			g.P("int32_t ", field.GoName, "Ownership;")
		}
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindEnum:
			g.P("int32_t ", field.GoName, ";")
		case FieldKindSignedInt64:
			g.P("int64_t ", field.GoName, ";")
		case FieldKindFloat:
			g.P("float ", field.GoName, ";")
		case FieldKindDouble:
			g.P("double ", field.GoName, ";")
		case FieldKindString, FieldKindBytes:
			g.P("uintptr_t ", field.GoName, "Ptr;")
			g.P("int32_t ", field.GoName, "Len;")
			if output {
				g.P("int32_t ", field.GoName, "Ownership;")
			}
		default:
			g.P("uintptr_t ", field.GoName, ";")
		}
	default:
		g.P("uintptr_t ", field.GoName, ";")
	}
}

func renderCGONativeServerAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, callbacksName, adapterName string, errorNames nativeServerCGOErrorNames, servicePackage string) {
	g.P("type ", adapterName, " struct {")
	g.P("callbacks C.", callbacksName)
	g.P("}")
	g.P()

	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
		if !ok {
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod, errorNames)
			continue
		}
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGONativeServerUnaryAdapter(g, service, adapterName, method, errorNames)
		case StreamingKindClientStreaming:
			renderCGONativeServerClientStreamAdapter(g, service, adapterName, method, errorNames, servicePackage)
		case StreamingKindServerStreaming:
			renderCGONativeServerServerStreamAdapter(g, service, adapterName, method, errorNames, servicePackage)
		case StreamingKindBidiStreaming:
			renderCGONativeServerBidiStreamAdapter(g, service, adapterName, method, errorNames, servicePackage)
		default:
			renderCGONativeServerStreamingFallback(g, adapterName, runtimeMethod, errorNames)
		}
	}

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderCGONativeServerRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerResponseCleanup(g, service, method)
		case StreamingKindClientStreaming:
			renderCGONativeServerClientStreamRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerClientStreamResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerClientStreamResponseCleanup(g, service, method)
		case StreamingKindServerStreaming:
			renderCGONativeServerServerStreamRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerServerStreamResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerServerStreamResponseCleanup(g, service, method)
		case StreamingKindBidiStreaming:
			renderCGONativeServerBidiStreamRequestEncoder(g, service, method, errorNames)
			renderCGONativeServerBidiStreamResponseDecoder(g, service, method, errorNames)
			renderCGONativeServerBidiStreamResponseCleanup(g, service, method)
		}
	}
	renderCGONativeErrorIDHelper(g, service)
}

func renderCGONativeServerUnaryAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("callback := a.callbacks.", method.GoName)
	g.P("if callback == nil {")
	g.P("return nil, ", errorNames.UnaryCallbackMissing)
	g.P("}")
	g.P("input, cleanup, err := ", nativeCGOServerRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("output := &C.", nativeCGOServerResponseName(service, method), "{}")
	g.P("errID := int32(C.", nativeCGOServerTrampolineName(service, method), "(callback, input, output))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.callbacks.", method.GoName, "Start == nil || a.callbacks.", method.GoName, "Send == nil || a.callbacks.", method.GoName, "Finish == nil || a.callbacks.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", errorNames.StreamNotImplemented)
	g.P("}")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", nativeCGOServerClientStreamStartTrampolineName(service, method), "(a.callbacks.", method.GoName, "Start, &stream))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeClientStreamSession{callbacks: a.callbacks, stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeClientStreamSession struct {")
	g.P("callbacks C.", service.GoName, "CGONativeServerCallbacks")
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerClientStreamSend(g, service, method, errorNames)
	renderCGONativeServerClientStreamFinish(g, service, method, errorNames)
	renderCGONativeServerClientStreamCancel(g, service, method)
}

func renderCGONativeServerClientStreamSend(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Send(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") error {")
	g.P("input, cleanup, err := ", nativeCGOServerClientStreamRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("errID := int32(C.", nativeCGOServerClientStreamSendTrampolineName(service, method), "(s.callbacks.", method.GoName, "Send, s.stream, input))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamFinish(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Finish(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("output := &C.", nativeCGOServerClientStreamResponseName(service, method), "{}")
	g.P("errID := int32(C.", nativeCGOServerClientStreamFinishTrampolineName(service, method), "(s.callbacks.", method.GoName, "Finish, s.stream, output))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerClientStreamResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamCancel(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", nativeCGOServerClientStreamCancelTrampolineName(service, method), "(s.callbacks.", method.GoName, "Cancel, s.stream))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", sessionName, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.callbacks.", method.GoName, "Start == nil || a.callbacks.", method.GoName, "Recv == nil || a.callbacks.", method.GoName, "Done == nil || a.callbacks.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", errorNames.StreamNotImplemented)
	g.P("}")
	g.P("input, cleanup, err := ", nativeCGOServerServerStreamRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", nativeCGOServerServerStreamStartTrampolineName(service, method), "(a.callbacks.", method.GoName, "Start, input, &stream))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeServerStreamSession{callbacks: a.callbacks, stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeServerStreamSession struct {")
	g.P("callbacks C.", service.GoName, "CGONativeServerCallbacks")
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerServerStreamRecv(g, service, method, errorNames)
	renderCGONativeServerServerStreamDone(g, service, method)
	renderCGONativeServerServerStreamCancel(g, service, method)
}

func renderCGONativeServerServerStreamRecv(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("output := &C.", nativeCGOServerServerStreamResponseName(service, method), "{}")
	g.P("errID := int32(C.", nativeCGOServerServerStreamRecvTrampolineName(service, method), "(s.callbacks.", method.GoName, "Recv, s.stream, output))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerServerStreamResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamDone(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Done(ctx context.Context) error {")
	g.P("errID := int32(C.", nativeCGOServerServerStreamDoneTrampolineName(service, method), "(s.callbacks.", method.GoName, "Done, s.stream))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamCancel(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", nativeCGOServerServerStreamCancelTrampolineName(service, method), "(s.callbacks.", method.GoName, "Cancel, s.stream))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan, errorNames nativeServerCGOErrorNames, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("if a == nil {")
	g.P("return nil, ", errorNames.CallbacksNil)
	g.P("}")
	g.P("if a.callbacks.", method.GoName, "Start == nil || a.callbacks.", method.GoName, "Send == nil || a.callbacks.", method.GoName, "Recv == nil || a.callbacks.", method.GoName, "CloseSend == nil || a.callbacks.", method.GoName, "Done == nil || a.callbacks.", method.GoName, "Cancel == nil {")
	g.P("return nil, ", errorNames.StreamNotImplemented)
	g.P("}")
	g.P("var stream C.int32_t")
	g.P("errID := int32(C.", nativeCGOServerBidiStreamStartTrampolineName(service, method), "(a.callbacks.", method.GoName, "Start, &stream))")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "CGONativeBidiStreamSession{callbacks: a.callbacks, stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "CGONativeBidiStreamSession struct {")
	g.P("callbacks C.", service.GoName, "CGONativeServerCallbacks")
	g.P("stream C.int32_t")
	g.P("}")
	g.P()
	renderCGONativeServerBidiStreamSend(g, service, method, errorNames)
	renderCGONativeServerBidiStreamRecv(g, service, method, errorNames)
	renderCGONativeServerBidiStreamCloseSend(g, service, method)
	renderCGONativeServerBidiStreamDone(g, service, method)
	renderCGONativeServerBidiStreamCancel(g, service, method)
}

func renderCGONativeServerBidiStreamSend(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Send(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") error {")
	g.P("input, cleanup, err := ", nativeCGOServerBidiStreamRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("errID := int32(C.", nativeCGOServerBidiStreamSendTrampolineName(service, method), "(s.callbacks.", method.GoName, "Send, s.stream, input))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamRecv(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("output := &C.", nativeCGOServerBidiStreamResponseName(service, method), "{}")
	g.P("errID := int32(C.", nativeCGOServerBidiStreamRecvTrampolineName(service, method), "(s.callbacks.", method.GoName, "Recv, s.stream, output))")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerBidiStreamResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamCloseSend(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	g.P("errID := int32(C.", nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), "(s.callbacks.", method.GoName, "CloseSend, s.stream))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamDone(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Done(ctx context.Context) error {")
	g.P("errID := int32(C.", nativeCGOServerBidiStreamDoneTrampolineName(service, method), "(s.callbacks.", method.GoName, "Done, s.stream))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamCancel(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	receiver := lowerInitial(service.GoName) + method.GoName + "CGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("errID := int32(C.", nativeCGOServerBidiStreamCancelTrampolineName(service, method), "(s.callbacks.", method.GoName, "Cancel, s.stream))")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerStreamingFallback(g *protogen.GeneratedFile, adapterName string, method runtimeAdapterMethod, errorNames nativeServerCGOErrorNames) {
	g.P("func (a *", adapterName, ") ", method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult, " {")
	if method.Streaming {
		g.P("return nil, ", errorNames.StreamNotImplemented)
	} else if method.AdapterResult == " error" {
		g.P("return ", errorNames.StreamNotImplemented)
	} else {
		g.P("return nil, ", errorNames.UnaryCallbackMissing)
	}
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	requestName := nativeCGOServerRequestName(service, method)
	g.P("func ", nativeCGOServerRequestEncoderName(service, method), "(req ", nativeGoMessageType(g, method.Request), ") (*C.", requestName, ", func(), error) {")
	g.P("if req == nil {")
	g.P(`return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")`)
	g.P("}")
	g.P("input := &C.", requestName, "{}")
	g.P("var pinned []uintptr")
	g.P("cleanup := func() {")
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
	g.P("}")
	for _, field := range method.NativeContract.RequestFields {
		renderCGONativeServerRequestFieldEncode(g, field, errorNames)
	}
	g.P("return input, cleanup, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerRequestFieldEncode(g *protogen.GeneratedFile, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("if req.", field.GoName, " {")
		g.P("input.", field.GoName, " = 1")
		g.P("}")
	case NativeABIShapeBoolByteBufferWrapper:
		g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(req.", field.GoName, "))")
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return nil, func() {}, err")
		g.P("}")
		g.P(field.GoName, "Bytes := make([]byte, len(req.", field.GoName, "))")
		g.P("for i := range req.", field.GoName, " {")
		g.P("if req.", field.GoName, "[i] {")
		g.P(field.GoName, "Bytes[i] = 1")
		g.P("}")
		g.P("}")
		g.P(field.GoName, "Ptr, err := rpcruntime.PinBytes(", field.GoName, "Bytes)")
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return nil, func() {}, err")
		g.P("}")
		g.P("if ", field.GoName, "Ptr != 0 {")
		g.P("pinned = append(pinned, ", field.GoName, "Ptr)")
		g.P("}")
		g.P("input.", field.GoName, "Ptr = C.uintptr_t(", field.GoName, "Ptr)")
		g.P("input.", field.GoName, "Len = C.int32_t(", field.GoName, "Len)")
	case NativeABIShapeRepeated:
		g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(req.", field.GoName, "))")
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return nil, func() {}, err")
		g.P("}")
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindFloat, FieldKindDouble:
			g.P(field.GoName, "Ptr, err := rpcruntime.PinSlice(req.", field.GoName, ")")
		case FieldKindEnum:
			g.P(field.GoName, "Values := make([]int32, len(req.", field.GoName, "))")
			g.P("for i := range req.", field.GoName, " {")
			g.P(field.GoName, "Values[i] = int32(req.", field.GoName, "[i])")
			g.P("}")
			g.P(field.GoName, "Ptr, err := rpcruntime.PinSlice(", field.GoName, "Values)")
		default:
			g.P("cleanup()")
			g.P("return nil, func() {}, ", errorNames.UnsupportedField)
		}
		g.P("if err != nil {")
		g.P("cleanup()")
		g.P("return nil, func() {}, err")
		g.P("}")
		g.P("if ", field.GoName, "Ptr != 0 {")
		g.P("pinned = append(pinned, ", field.GoName, "Ptr)")
		g.P("}")
		g.P("input.", field.GoName, "Ptr = C.uintptr_t(", field.GoName, "Ptr)")
		g.P("input.", field.GoName, "Len = C.int32_t(", field.GoName, "Len)")
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32:
			g.P("input.", field.GoName, " = C.int32_t(req.", field.GoName, ")")
		case FieldKindSignedInt64:
			g.P("input.", field.GoName, " = C.int64_t(req.", field.GoName, ")")
		case FieldKindFloat:
			g.P("input.", field.GoName, " = C.float(req.", field.GoName, ")")
		case FieldKindDouble:
			g.P("input.", field.GoName, " = C.double(req.", field.GoName, ")")
		case FieldKindEnum:
			g.P("input.", field.GoName, " = C.int32_t(req.", field.GoName, ")")
		case FieldKindString:
			g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(req.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P("_, ", field.GoName, "Ptr, err := rpcruntime.PinString(req.", field.GoName, ")")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P("if ", field.GoName, "Ptr != 0 {")
			g.P("pinned = append(pinned, ", field.GoName, "Ptr)")
			g.P("}")
			g.P("input.", field.GoName, "Ptr = C.uintptr_t(", field.GoName, "Ptr)")
			g.P("input.", field.GoName, "Len = C.int32_t(", field.GoName, "Len)")
		case FieldKindBytes:
			g.P(field.GoName, "Len, err := rpcruntime.LengthToInt32(len(req.", field.GoName, "))")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P(field.GoName, "Ptr, err := rpcruntime.PinBytes(req.", field.GoName, ")")
			g.P("if err != nil {")
			g.P("cleanup()")
			g.P("return nil, func() {}, err")
			g.P("}")
			g.P("if ", field.GoName, "Ptr != 0 {")
			g.P("pinned = append(pinned, ", field.GoName, "Ptr)")
			g.P("}")
			g.P("input.", field.GoName, "Ptr = C.uintptr_t(", field.GoName, "Ptr)")
			g.P("input.", field.GoName, "Len = C.int32_t(", field.GoName, "Len)")
		default:
			g.P("cleanup()")
			g.P("return nil, func() {}, ", errorNames.UnsupportedField)
		}
	default:
		g.P("cleanup()")
		g.P("return nil, func() {}, ", errorNames.UnsupportedField)
	}
}

func renderCGONativeServerResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	responseName := nativeCGOServerResponseName(service, method)
	g.P("func ", nativeCGOServerResponseDecoderName(service, method), "(output *C.", responseName, ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if output == nil {")
	g.P(`return nil, errors.New("rpccgo: cgo native server response output is nil")`)
	g.P("}")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.ResponseFields {
		renderCGONativeServerResponseFieldDecode(g, field, errorNames)
	}
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerResponseFieldDecode(g *protogen.GeneratedFile, field FieldPlan, errorNames nativeServerCGOErrorNames) {
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		g.P("resp.", field.GoName, " = output.", field.GoName, " != 0")
	case NativeABIShapeBoolByteBufferWrapper:
		g.P("if _, err := rpcruntime.LengthFromInt32(int32(output.", field.GoName, "Len)); err != nil {")
		g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
		g.P("}")
		g.P(field.GoName, ", err := rpcruntime.NewRpcBoolRepeatChecked((*byte)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
		g.P("if err != nil {")
		g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
		g.P("}")
		g.P("resp.", field.GoName, " = ", field.GoName, ".SafeSlice()")
	case NativeABIShapeRepeated:
		g.P("if _, err := rpcruntime.LengthFromInt32(int32(output.", field.GoName, "Len)); err != nil {")
		g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
		g.P("}")
		switch field.Kind {
		case FieldKindSignedInt32:
			g.P(field.GoName, ", err := rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
			g.P("if err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P("resp.", field.GoName, " = ", field.GoName, ".SafeSlice()")
		case FieldKindSignedInt64:
			g.P(field.GoName, ", err := rpcruntime.NewRpcRepeatChecked((*int64)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
			g.P("if err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P("resp.", field.GoName, " = ", field.GoName, ".SafeSlice()")
		case FieldKindFloat:
			g.P(field.GoName, ", err := rpcruntime.NewRpcRepeatChecked((*float32)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
			g.P("if err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P("resp.", field.GoName, " = ", field.GoName, ".SafeSlice()")
		case FieldKindDouble:
			g.P(field.GoName, ", err := rpcruntime.NewRpcRepeatChecked((*float64)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
			g.P("if err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P("resp.", field.GoName, " = ", field.GoName, ".SafeSlice()")
		case FieldKindEnum:
			g.P(field.GoName, ", err := rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
			g.P("if err != nil {")
			g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
			g.P("}")
			g.P(field.GoName, "Raw := ", field.GoName, ".SafeSlice()")
			g.P("resp.", field.GoName, " = make([]", nativeGoEnumType(g, field), ", len(", field.GoName, "Raw))")
			g.P("for i := range ", field.GoName, "Raw {")
			g.P("resp.", field.GoName, "[i] = ", nativeGoEnumType(g, field), "(", field.GoName, "Raw[i])")
			g.P("}")
		default:
			g.P("return nil, ", errorNames.UnsupportedField)
		}
	case NativeABIShapeScalar:
		switch field.Kind {
		case FieldKindSignedInt32:
			g.P("resp.", field.GoName, " = int32(output.", field.GoName, ")")
		case FieldKindSignedInt64:
			g.P("resp.", field.GoName, " = int64(output.", field.GoName, ")")
		case FieldKindFloat:
			g.P("resp.", field.GoName, " = float32(output.", field.GoName, ")")
		case FieldKindDouble:
			g.P("resp.", field.GoName, " = float64(output.", field.GoName, ")")
		case FieldKindEnum:
			g.P("resp.", field.GoName, " = ", nativeGoEnumType(g, field), "(int32(output.", field.GoName, "))")
		case FieldKindString:
			renderCGONativeServerResponseTextDecode(g, field, "String", "SafeString")
		case FieldKindBytes:
			renderCGONativeServerResponseTextDecode(g, field, "Bytes", "SafeBytes")
		default:
			g.P("return nil, ", errorNames.UnsupportedField)
		}
	default:
		g.P("return nil, ", errorNames.UnsupportedField)
	}
}

func renderCGONativeServerResponseTextDecode(g *protogen.GeneratedFile, field FieldPlan, wrapper, safeMethod string) {
	g.P("if _, err := rpcruntime.LengthFromInt32(int32(output.", field.GoName, "Len)); err != nil {")
	g.P(`return nil, fmt.Errorf("`, field.FullName, `: %w", err)`)
	g.P("}")
	g.P(field.GoName, " := rpcruntime.NewRpc", wrapper, "((*byte)(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr))), int32(output.", field.GoName, "Len), false)")
	g.P("resp.", field.GoName, " = ", field.GoName, ".", safeMethod, "()")
}

func renderCGONativeServerRegistration(g *protogen.GeneratedFile, service ServicePlan, callbacksName, adapterName string, errorNames nativeServerCGOErrorNames, servicePackage string) {
	g.P("func Register", service.GoName, "CGONativeServer(callbacks *C.", callbacksName, ") (rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter], error) {")
	g.P("if callbacks == nil {")
	g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.CallbacksNil)
	g.P("}")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("if callbacks.", method.GoName, " == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.UnaryCallbackMissing)
			g.P("}")
		case StreamingKindClientStreaming:
			g.P("if callbacks.", method.GoName, "Start == nil || callbacks.", method.GoName, "Send == nil || callbacks.", method.GoName, "Finish == nil || callbacks.", method.GoName, "Cancel == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.StreamNotImplemented)
			g.P("}")
		case StreamingKindServerStreaming:
			g.P("if callbacks.", method.GoName, "Start == nil || callbacks.", method.GoName, "Recv == nil || callbacks.", method.GoName, "Done == nil || callbacks.", method.GoName, "Cancel == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.StreamNotImplemented)
			g.P("}")
		case StreamingKindBidiStreaming:
			g.P("if callbacks.", method.GoName, "Start == nil || callbacks.", method.GoName, "Send == nil || callbacks.", method.GoName, "Recv == nil || callbacks.", method.GoName, "CloseSend == nil || callbacks.", method.GoName, "Done == nil || callbacks.", method.GoName, "Cancel == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.StreamNotImplemented)
			g.P("}")
		}
	}
	g.P("callbacksCopy := *callbacks")
	g.P("return ", servicePackage, "Register", service.GoName, "CGONativeActiveServer(rpcruntime.ServerKindCGONative, &", adapterName, "{callbacks: callbacksCopy})")
	g.P("}")
	g.P()
}

func renderCGONativeServerResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", nativeCGOServerResponseCleanupName(service, method), "(output *C.", nativeCGOServerResponseName(service, method), ") error {")
	g.P("if output == nil {")
	g.P("return nil")
	g.P("}")
	g.P("var cleanupErr error")
	for _, field := range method.NativeContract.ResponseFields {
		if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
		if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
	}
	g.P("return cleanupErr")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	requestName := nativeCGOServerClientStreamRequestName(service, method)
	g.P("func ", nativeCGOServerClientStreamRequestEncoderName(service, method), "(req ", nativeGoMessageType(g, method.Request), ") (*C.", requestName, ", func(), error) {")
	g.P("if req == nil {")
	g.P(`return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")`)
	g.P("}")
	g.P("input := &C.", requestName, "{}")
	g.P("var pinned []uintptr")
	g.P("cleanup := func() {")
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
	g.P("}")
	for _, field := range method.NativeContract.RequestFields {
		renderCGONativeServerRequestFieldEncode(g, field, errorNames)
	}
	g.P("return input, cleanup, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	responseName := nativeCGOServerClientStreamResponseName(service, method)
	g.P("func ", nativeCGOServerClientStreamResponseDecoderName(service, method), "(output *C.", responseName, ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if output == nil {")
	g.P(`return nil, errors.New("rpccgo: cgo native server response output is nil")`)
	g.P("}")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.ResponseFields {
		renderCGONativeServerResponseFieldDecode(g, field, errorNames)
	}
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerClientStreamResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(output *C.", nativeCGOServerClientStreamResponseName(service, method), ") error {")
	g.P("if output == nil {")
	g.P("return nil")
	g.P("}")
	g.P("var cleanupErr error")
	for _, field := range method.NativeContract.ResponseFields {
		if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
		if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
	}
	g.P("return cleanupErr")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	requestName := nativeCGOServerServerStreamRequestName(service, method)
	g.P("func ", nativeCGOServerServerStreamRequestEncoderName(service, method), "(req ", nativeGoMessageType(g, method.Request), ") (*C.", requestName, ", func(), error) {")
	g.P("if req == nil {")
	g.P(`return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")`)
	g.P("}")
	g.P("input := &C.", requestName, "{}")
	g.P("var pinned []uintptr")
	g.P("cleanup := func() {")
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
	g.P("}")
	for _, field := range method.NativeContract.RequestFields {
		renderCGONativeServerRequestFieldEncode(g, field, errorNames)
	}
	g.P("return input, cleanup, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	responseName := nativeCGOServerServerStreamResponseName(service, method)
	g.P("func ", nativeCGOServerServerStreamResponseDecoderName(service, method), "(output *C.", responseName, ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if output == nil {")
	g.P(`return nil, errors.New("rpccgo: cgo native server response output is nil")`)
	g.P("}")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.ResponseFields {
		renderCGONativeServerResponseFieldDecode(g, field, errorNames)
	}
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerServerStreamResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(output *C.", nativeCGOServerServerStreamResponseName(service, method), ") error {")
	g.P("if output == nil {")
	g.P("return nil")
	g.P("}")
	g.P("var cleanupErr error")
	for _, field := range method.NativeContract.ResponseFields {
		if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
		if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
	}
	g.P("return cleanupErr")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamRequestEncoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	requestName := nativeCGOServerBidiStreamRequestName(service, method)
	g.P("func ", nativeCGOServerBidiStreamRequestEncoderName(service, method), "(req ", nativeGoMessageType(g, method.Request), ") (*C.", requestName, ", func(), error) {")
	g.P("if req == nil {")
	g.P(`return nil, func() {}, errors.New("rpccgo: cgo native server request is nil")`)
	g.P("}")
	g.P("input := &C.", requestName, "{}")
	g.P("var pinned []uintptr")
	g.P("cleanup := func() {")
	g.P("for i := len(pinned) - 1; i >= 0; i-- {")
	g.P("rpcruntime.Release(pinned[i])")
	g.P("}")
	g.P("}")
	for _, field := range method.NativeContract.RequestFields {
		renderCGONativeServerRequestFieldEncode(g, field, errorNames)
	}
	g.P("return input, cleanup, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamResponseDecoder(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, errorNames nativeServerCGOErrorNames) {
	responseName := nativeCGOServerBidiStreamResponseName(service, method)
	g.P("func ", nativeCGOServerBidiStreamResponseDecoderName(service, method), "(output *C.", responseName, ") (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("if output == nil {")
	g.P(`return nil, errors.New("rpccgo: cgo native server response output is nil")`)
	g.P("}")
	g.P("resp := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Response.GoName, GoImportPath: protogen.GoImportPath(method.Response.GoImportPath)}), "{}")
	for _, field := range method.NativeContract.ResponseFields {
		renderCGONativeServerResponseFieldDecode(g, field, errorNames)
	}
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerBidiStreamResponseCleanup(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	g.P("func ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(output *C.", nativeCGOServerBidiStreamResponseName(service, method), ") error {")
	g.P("if output == nil {")
	g.P("return nil")
	g.P("}")
	g.P("var cleanupErr error")
	for _, field := range method.NativeContract.ResponseFields {
		if field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes) {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
		if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
			g.P("if output.", field.GoName, "Ownership > 0 && output.", field.GoName, "Ptr != 0 {")
			g.P("if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.", field.GoName, "Ptr)), true, \"", field.FullName, "\"); err != nil {")
			g.P("cleanupErr = errors.Join(cleanupErr, err)")
			g.P("}")
			g.P("output.", field.GoName, "Ptr = 0")
			g.P("output.", field.GoName, "Len = 0")
			g.P("output.", field.GoName, "Ownership = 0")
			g.P("}")
		}
	}
	g.P("return cleanupErr")
	g.P("}")
	g.P()
}

func renderCGONativeServerGoHelper(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, callbacksName string, errorNames nativeServerCGOErrorNames, servicePackage string) {
	helperName := service.GoName + "GoCGONativeServerCallbacks"
	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}
	g.P("type ", helperName, " struct {")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(method.GoName, " func(ctx context.Context, input *C.", nativeCGOServerRequestName(service, method), ", output *C.", nativeCGOServerResponseName(service, method), ") int32")
		case StreamingKindClientStreaming:
			g.P(method.GoName, "Start func(ctx context.Context, stream *C.int32_t) int32")
			g.P(method.GoName, "Send func(ctx context.Context, stream C.int32_t, input *C.", nativeCGOServerClientStreamRequestName(service, method), ") int32")
			g.P(method.GoName, "Finish func(ctx context.Context, stream C.int32_t, output *C.", nativeCGOServerClientStreamResponseName(service, method), ") int32")
			g.P(method.GoName, "Cancel func(ctx context.Context, stream C.int32_t) int32")
		case StreamingKindServerStreaming:
			g.P(method.GoName, "Start func(ctx context.Context, input *C.", nativeCGOServerServerStreamRequestName(service, method), ", stream *C.int32_t) int32")
			g.P(method.GoName, "Recv func(ctx context.Context, stream C.int32_t, output *C.", nativeCGOServerServerStreamResponseName(service, method), ") int32")
			g.P(method.GoName, "Done func(ctx context.Context, stream C.int32_t) int32")
			g.P(method.GoName, "Cancel func(ctx context.Context, stream C.int32_t) int32")
		case StreamingKindBidiStreaming:
			g.P(method.GoName, "Start func(ctx context.Context, stream *C.int32_t) int32")
			g.P(method.GoName, "Send func(ctx context.Context, stream C.int32_t, input *C.", nativeCGOServerBidiStreamRequestName(service, method), ") int32")
			g.P(method.GoName, "Recv func(ctx context.Context, stream C.int32_t, output *C.", nativeCGOServerBidiStreamResponseName(service, method), ") int32")
			g.P(method.GoName, "CloseSend func(ctx context.Context, stream C.int32_t) int32")
			g.P(method.GoName, "Done func(ctx context.Context, stream C.int32_t) int32")
			g.P(method.GoName, "Cancel func(ctx context.Context, stream C.int32_t) int32")
		}
	}
	g.P("}")
	g.P()
	g.P("func Register", service.GoName, "GoCGONativeServerForTesting(callbacks *", helperName, ") (rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter], error) {")
	g.P("if callbacks == nil {")
	g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.CallbacksNil)
	g.P("}")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("if callbacks.", method.GoName, " == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.UnaryCallbackMissing)
			g.P("}")
		case StreamingKindClientStreaming:
			g.P("if callbacks.", method.GoName, "Start == nil || callbacks.", method.GoName, "Send == nil || callbacks.", method.GoName, "Finish == nil || callbacks.", method.GoName, "Cancel == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.StreamNotImplemented)
			g.P("}")
		case StreamingKindServerStreaming:
			g.P("if callbacks.", method.GoName, "Start == nil || callbacks.", method.GoName, "Recv == nil || callbacks.", method.GoName, "Done == nil || callbacks.", method.GoName, "Cancel == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.StreamNotImplemented)
			g.P("}")
		case StreamingKindBidiStreaming:
			g.P("if callbacks.", method.GoName, "Start == nil || callbacks.", method.GoName, "Send == nil || callbacks.", method.GoName, "Recv == nil || callbacks.", method.GoName, "CloseSend == nil || callbacks.", method.GoName, "Done == nil || callbacks.", method.GoName, "Cancel == nil {")
			g.P("return rpcruntime.AdapterSnapshot[", servicePackage, service.GoName, "NativeAdapter]{}, ", errorNames.StreamNotImplemented)
			g.P("}")
		}
	}
	g.P("return ", servicePackage, "Register", service.GoName, "CGONativeActiveServer(rpcruntime.ServerKindCGONative, &", lowerInitial(service.GoName), "GoCGONativeAdapter{callbacks: callbacks})")
	g.P("}")
	g.P()
	g.P("type ", lowerInitial(service.GoName), "GoCGONativeAdapter struct {")
	g.P("callbacks *", helperName)
	g.P("}")
	g.P()
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P("func (a *", lowerInitial(service.GoName), "GoCGONativeAdapter) ", method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", nativeGoMessageType(g, method.Response), ", error) {")
			g.P("input, cleanup, err := ", nativeCGOServerRequestEncoderName(service, method), "(req)")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("defer cleanup()")
			g.P("output := &C.", nativeCGOServerResponseName(service, method), "{}")
			g.P("errID := a.callbacks.", method.GoName, "(ctx, input, output)")
			g.P("if errID != 0 {")
			g.P("cleanupErr := ", nativeCGOServerResponseCleanupName(service, method), "(output)")
			g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
			g.P("if cleanupErr != nil {")
			g.P("return nil, errors.Join(callbackErr, cleanupErr)")
			g.P("}")
			g.P("return nil, callbackErr")
			g.P("}")
			g.P("resp, err := ", nativeCGOServerResponseDecoderName(service, method), "(output)")
			g.P("cleanupErr := ", nativeCGOServerResponseCleanupName(service, method), "(output)")
			g.P("if cleanupErr != nil {")
			g.P("if err != nil {")
			g.P("return nil, errors.Join(err, cleanupErr)")
			g.P("}")
			g.P("return nil, cleanupErr")
			g.P("}")
			g.P("if err != nil {")
			g.P("return nil, err")
			g.P("}")
			g.P("return resp, nil")
			g.P("}")
			g.P()
		case StreamingKindClientStreaming:
			renderGoCGONativeServerClientStreamAdapter(g, service, method, servicePackage)
		case StreamingKindServerStreaming:
			renderGoCGONativeServerServerStreamAdapter(g, service, method, servicePackage)
		case StreamingKindBidiStreaming:
			renderGoCGONativeServerBidiStreamAdapter(g, service, method, servicePackage)
		}
	}
	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
		if ok && (method.Streaming == StreamingKindUnary || method.Streaming == StreamingKindClientStreaming || method.Streaming == StreamingKindServerStreaming || method.Streaming == StreamingKindBidiStreaming) {
			continue
		}
		renderCGONativeServerStreamingFallback(g, lowerInitial(service.GoName)+"GoCGONativeAdapter", runtimeMethod, errorNames)
	}
}

func renderGoCGONativeServerClientStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	adapterName := lowerInitial(service.GoName) + "GoCGONativeAdapter"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("var stream C.int32_t")
	g.P("errID := a.callbacks.", method.GoName, "Start(ctx, &stream)")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoCGONativeClientStreamSession{callbacks: a.callbacks, stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoCGONativeClientStreamSession struct {")
	g.P("callbacks *", service.GoName, "GoCGONativeServerCallbacks")
	g.P("stream C.int32_t")
	g.P("}")
	g.P()

	receiver := lowerInitial(service.GoName) + method.GoName + "GoCGONativeClientStreamSession"
	g.P("func (s *", receiver, ") Send(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") error {")
	g.P("input, cleanup, err := ", nativeCGOServerClientStreamRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("errID := s.callbacks.", method.GoName, "Send(ctx, s.stream, input)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Finish(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("output := &C.", nativeCGOServerClientStreamResponseName(service, method), "{}")
	g.P("errID := s.callbacks.", method.GoName, "Finish(ctx, s.stream, output)")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerClientStreamResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerClientStreamResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("errID := s.callbacks.", method.GoName, "Cancel(ctx, s.stream)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderGoCGONativeServerServerStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	adapterName := lowerInitial(service.GoName) + "GoCGONativeAdapter"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", sessionName, ", error) {")
	g.P("input, cleanup, err := ", nativeCGOServerServerStreamRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("var stream C.int32_t")
	g.P("errID := a.callbacks.", method.GoName, "Start(ctx, input, &stream)")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoCGONativeServerStreamSession{callbacks: a.callbacks, stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoCGONativeServerStreamSession struct {")
	g.P("callbacks *", service.GoName, "GoCGONativeServerCallbacks")
	g.P("stream C.int32_t")
	g.P("}")
	g.P()

	receiver := lowerInitial(service.GoName) + method.GoName + "GoCGONativeServerStreamSession"
	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("output := &C.", nativeCGOServerServerStreamResponseName(service, method), "{}")
	g.P("errID := s.callbacks.", method.GoName, "Recv(ctx, s.stream, output)")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerServerStreamResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerServerStreamResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Done(ctx context.Context) error {")
	g.P("errID := s.callbacks.", method.GoName, "Done(ctx, s.stream)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("errID := s.callbacks.", method.GoName, "Cancel(ctx, s.stream)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderGoCGONativeServerBidiStreamAdapter(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage string) {
	sessionName := servicePackage + service.GoName + method.GoName + "NativeStreamSession"
	adapterName := lowerInitial(service.GoName) + "GoCGONativeAdapter"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("var stream C.int32_t")
	g.P("errID := a.callbacks.", method.GoName, "Start(ctx, &stream)")
	g.P("if errID != 0 {")
	g.P("return nil, ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoCGONativeBidiStreamSession{callbacks: a.callbacks, stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoCGONativeBidiStreamSession struct {")
	g.P("callbacks *", service.GoName, "GoCGONativeServerCallbacks")
	g.P("stream C.int32_t")
	g.P("}")
	g.P()

	receiver := lowerInitial(service.GoName) + method.GoName + "GoCGONativeBidiStreamSession"
	g.P("func (s *", receiver, ") Send(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") error {")
	g.P("input, cleanup, err := ", nativeCGOServerBidiStreamRequestEncoderName(service, method), "(req)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("defer cleanup()")
	g.P("errID := s.callbacks.", method.GoName, "Send(ctx, s.stream, input)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Recv(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error) {")
	g.P("output := &C.", nativeCGOServerBidiStreamResponseName(service, method), "{}")
	g.P("errID := s.callbacks.", method.GoName, "Recv(ctx, s.stream, output)")
	g.P("if errID != 0 {")
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(output)")
	g.P("callbackErr := ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("if cleanupErr != nil {")
	g.P("return nil, errors.Join(callbackErr, cleanupErr)")
	g.P("}")
	g.P("return nil, callbackErr")
	g.P("}")
	g.P("resp, err := ", nativeCGOServerBidiStreamResponseDecoderName(service, method), "(output)")
	g.P("cleanupErr := ", nativeCGOServerBidiStreamResponseCleanupName(service, method), "(output)")
	g.P("if cleanupErr != nil {")
	g.P("if err != nil {")
	g.P("return nil, errors.Join(err, cleanupErr)")
	g.P("}")
	g.P("return nil, cleanupErr")
	g.P("}")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	g.P("errID := s.callbacks.", method.GoName, "CloseSend(ctx, s.stream)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Done(ctx context.Context) error {")
	g.P("errID := s.callbacks.", method.GoName, "Done(ctx, s.stream)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("errID := s.callbacks.", method.GoName, "Cancel(ctx, s.stream)")
	g.P("if errID != 0 {")
	g.P("return ", nativeCGOServerErrorIDHelperName(service), "(errID)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCGONativeServerErrorStoreExport(g *protogen.GeneratedFile, service ServicePlan) {
	exportName := "Store" + service.GoName + "CGONativeServerErrorTextForExport"
	g.P("//export ", exportName)
	g.P("func ", exportName, "(text *C.char, textLen C.int32_t) C.int32_t {")
	g.P("length, err := rpcruntime.LengthFromInt32(int32(textLen))")
	g.P("if err != nil {")
	g.P(`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: cgo native server error text: %w", err)))`)
	g.P("}")
	g.P("if text == nil && length != 0 {")
	g.P(`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: cgo native server error text pointer is nil")))`)
	g.P("}")
	g.P("var data []byte")
	g.P("if length != 0 {")
	g.P("data = unsafe.Slice((*byte)(unsafe.Pointer(text)), length)")
	g.P("}")
	g.P("return C.int32_t(rpcruntime.StoreError(errors.New(string(data))))")
	g.P("}")
	g.P()
}

func renderCGONativeErrorIDHelper(g *protogen.GeneratedFile, service ServicePlan) {
	g.P("func ", nativeCGOServerErrorIDHelperName(service), "(errID int32) error {")
	g.P("if errID == 0 {")
	g.P("return nil")
	g.P("}")
	g.P("text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))")
	g.P("if ok {")
	g.P("if ptr != 0 {")
	g.P("defer rpcruntime.Release(ptr)")
	g.P("}")
	g.P("return errors.New(string(text))")
	g.P("}")
	g.P(`return fmt.Errorf("rpccgo: cgo native server callback returned unknown error id %d", errID)`)
	g.P("}")
	g.P()
}

func nativeCGOServerRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryRequest"
}

func nativeCGOServerResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeUnaryRequest"
}

func nativeCGOServerResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeUnaryResponse"
}

func nativeCGOServerCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeUnaryCallback"
}

func nativeCGOServerTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeUnaryCallback"
}

func nativeCGOServerErrorIDHelperName(service ServicePlan) string {
	return lowerInitial(service.GoName) + "CGONativeServerErrorFromID"
}

func nativeCGOServerClientStreamRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamRequest"
}

func nativeCGOServerClientStreamResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerClientStreamRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeClientStreamRequest"
}

func nativeCGOServerClientStreamResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerClientStreamResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeClientStreamResponse"
}

func nativeCGOServerClientStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamStartCallback"
}

func nativeCGOServerClientStreamSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamSendCallback"
}

func nativeCGOServerClientStreamFinishCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamFinishCallback"
}

func nativeCGOServerClientStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeClientStreamCancelCallback"
}

func nativeCGOServerClientStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamStartCallback"
}

func nativeCGOServerClientStreamSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamSendCallback"
}

func nativeCGOServerClientStreamFinishTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamFinishCallback"
}

func nativeCGOServerClientStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeClientStreamCancelCallback"
}

func nativeCGOServerServerStreamRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamRequest"
}

func nativeCGOServerServerStreamResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerServerStreamRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeServerStreamRequest"
}

func nativeCGOServerServerStreamResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerServerStreamResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeServerStreamResponse"
}

func nativeCGOServerServerStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamStartCallback"
}

func nativeCGOServerServerStreamRecvCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamRecvCallback"
}

func nativeCGOServerServerStreamDoneCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamDoneCallback"
}

func nativeCGOServerServerStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeServerStreamCancelCallback"
}

func nativeCGOServerServerStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamStartCallback"
}

func nativeCGOServerServerStreamRecvTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamRecvCallback"
}

func nativeCGOServerServerStreamDoneTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamDoneCallback"
}

func nativeCGOServerServerStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeServerStreamCancelCallback"
}

func nativeCGOServerBidiStreamRequestName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamRequest"
}

func nativeCGOServerBidiStreamResponseName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerBidiStreamRequestEncoderName(service ServicePlan, method MethodPlan) string {
	return "encode" + service.GoName + method.GoName + "CGONativeBidiStreamRequest"
}

func nativeCGOServerBidiStreamResponseDecoderName(service ServicePlan, method MethodPlan) string {
	return "decode" + service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerBidiStreamResponseCleanupName(service ServicePlan, method MethodPlan) string {
	return "cleanup" + service.GoName + method.GoName + "CGONativeBidiStreamResponse"
}

func nativeCGOServerBidiStreamStartCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamStartCallback"
}

func nativeCGOServerBidiStreamSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamSendCallback"
}

func nativeCGOServerBidiStreamRecvCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamRecvCallback"
}

func nativeCGOServerBidiStreamCloseSendCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamCloseSendCallback"
}

func nativeCGOServerBidiStreamDoneCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamDoneCallback"
}

func nativeCGOServerBidiStreamCancelCallbackName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "CGONativeBidiStreamCancelCallback"
}

func nativeCGOServerBidiStreamStartTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamStartCallback"
}

func nativeCGOServerBidiStreamSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamSendCallback"
}

func nativeCGOServerBidiStreamRecvTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamRecvCallback"
}

func nativeCGOServerBidiStreamCloseSendTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamCloseSendCallback"
}

func nativeCGOServerBidiStreamDoneTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamDoneCallback"
}

func nativeCGOServerBidiStreamCancelTrampolineName(service ServicePlan, method MethodPlan) string {
	return "call" + service.GoName + method.GoName + "CGONativeBidiStreamCancelCallback"
}

func nativeServerCGONeedsUnsafe(service ServicePlan) bool {
	return true
}

type nativeServerCGOErrorNames struct {
	CallbacksNil         string
	UnaryCallbackMissing string
	UnsupportedField     string
	StreamNotImplemented string
}

func nativeServerCGOErrorNamesFor(service ServicePlan) nativeServerCGOErrorNames {
	prefix := lowerInitial(service.GoName)
	return nativeServerCGOErrorNames{
		CallbacksNil:         prefix + "CGONativeServerCallbacksNil",
		UnaryCallbackMissing: prefix + "CGONativeServerUnaryCallbackMissing",
		UnsupportedField:     prefix + "CGONativeServerUnsupportedField",
		StreamNotImplemented: prefix + "CGONativeServerStreamNotImplemented",
	}
}

func validateNativeServerCGOSymbols(plan FilePlan, service ServicePlan) error {
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
		if otherService.FullName != service.FullName && otherService.NativeFileFamily.CGONativeServer.Enabled {
			addNativeServerCGOGeneratedSymbols(seen, otherService)
		}
	}
	addGenerated := func(symbol, source string) error {
		if symbol == "" {
			return nil
		}
		if previous, exists := seen[symbol]; exists {
			if previous != source {
				return fmt.Errorf("native server cgo symbol %s for %s collides with %s", symbol, source, previous)
			}
			return nil
		}
		if protobufSymbol, exists := protobufSymbols[symbol]; exists {
			return fmt.Errorf("native server cgo symbol %s for %s collides with protobuf %s %s", symbol, source, protobufSymbol.Kind, protobufSymbol.FullName)
		}
		seen[symbol] = source
		return nil
	}

	errorNames := nativeServerCGOErrorNamesFor(service)
	for symbol, source := range map[string]string{
		service.GoName + "CGONativeServerCallbacks":                    errorNames.CallbacksNil,
		service.GoName + "GoCGONativeServerCallbacks":                  service.FullName + " go helper callbacks",
		lowerInitial(service.GoName) + "CGONativeAdapter":              service.FullName + " adapter",
		lowerInitial(service.GoName) + "GoCGONativeAdapter":            service.FullName + " go helper adapter",
		"Register" + service.GoName + "CGONativeServer":                service.FullName + " registration",
		"Register" + service.GoName + "GoCGONativeServerForTesting":    service.FullName + " go helper registration",
		"Store" + service.GoName + "CGONativeServerErrorTextForExport": service.FullName + " error text export",
		nativeCGOServerErrorIDHelperName(service):                      service.FullName + " error id helper",
		errorNames.CallbacksNil:                                        errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                                errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                                    errorNames.UnsupportedField,
		errorNames.StreamNotImplemented:                                errorNames.StreamNotImplemented,
	} {
		if err := addGenerated(symbol, source); err != nil {
			return err
		}
	}
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary && method.Streaming != StreamingKindClientStreaming && method.Streaming != StreamingKindServerStreaming && method.Streaming != StreamingKindBidiStreaming {
			continue
		}
		requestName := nativeCGOServerRequestName(service, method)
		responseName := nativeCGOServerResponseName(service, method)
		if method.Streaming == StreamingKindClientStreaming {
			requestName = nativeCGOServerClientStreamRequestName(service, method)
			responseName = nativeCGOServerClientStreamResponseName(service, method)
		} else if method.Streaming == StreamingKindServerStreaming {
			requestName = nativeCGOServerServerStreamRequestName(service, method)
			responseName = nativeCGOServerServerStreamResponseName(service, method)
		} else if method.Streaming == StreamingKindBidiStreaming {
			requestName = nativeCGOServerBidiStreamRequestName(service, method)
			responseName = nativeCGOServerBidiStreamResponseName(service, method)
		}
		for _, item := range []struct {
			symbol string
			source string
		}{
			{requestName, method.FullName + " cgo request"},
			{responseName, method.FullName + " cgo response"},
		} {
			if err := addGenerated(item.symbol, item.source); err != nil {
				return err
			}
		}
		if method.Streaming == StreamingKindUnary {
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerCallbackName(service, method), method.FullName + " cgo callback"},
				{nativeCGOServerTrampolineName(service, method), method.FullName + " cgo trampoline"},
				{nativeCGOServerRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		} else if method.Streaming == StreamingKindClientStreaming {
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerClientStreamStartCallbackName(service, method), method.FullName + " cgo stream start callback"},
				{nativeCGOServerClientStreamSendCallbackName(service, method), method.FullName + " cgo stream send callback"},
				{nativeCGOServerClientStreamFinishCallbackName(service, method), method.FullName + " cgo stream finish callback"},
				{nativeCGOServerClientStreamCancelCallbackName(service, method), method.FullName + " cgo stream cancel callback"},
				{nativeCGOServerClientStreamStartTrampolineName(service, method), method.FullName + " cgo stream start trampoline"},
				{nativeCGOServerClientStreamSendTrampolineName(service, method), method.FullName + " cgo stream send trampoline"},
				{nativeCGOServerClientStreamFinishTrampolineName(service, method), method.FullName + " cgo stream finish trampoline"},
				{nativeCGOServerClientStreamCancelTrampolineName(service, method), method.FullName + " cgo stream cancel trampoline"},
				{nativeCGOServerClientStreamRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerClientStreamResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerClientStreamResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		} else if method.Streaming == StreamingKindServerStreaming {
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerServerStreamStartCallbackName(service, method), method.FullName + " cgo stream start callback"},
				{nativeCGOServerServerStreamRecvCallbackName(service, method), method.FullName + " cgo stream recv callback"},
				{nativeCGOServerServerStreamDoneCallbackName(service, method), method.FullName + " cgo stream done callback"},
				{nativeCGOServerServerStreamCancelCallbackName(service, method), method.FullName + " cgo stream cancel callback"},
				{nativeCGOServerServerStreamStartTrampolineName(service, method), method.FullName + " cgo stream start trampoline"},
				{nativeCGOServerServerStreamRecvTrampolineName(service, method), method.FullName + " cgo stream recv trampoline"},
				{nativeCGOServerServerStreamDoneTrampolineName(service, method), method.FullName + " cgo stream done trampoline"},
				{nativeCGOServerServerStreamCancelTrampolineName(service, method), method.FullName + " cgo stream cancel trampoline"},
				{nativeCGOServerServerStreamRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerServerStreamResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerServerStreamResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		} else {
			for _, item := range []struct {
				symbol string
				source string
			}{
				{nativeCGOServerBidiStreamStartCallbackName(service, method), method.FullName + " cgo stream start callback"},
				{nativeCGOServerBidiStreamSendCallbackName(service, method), method.FullName + " cgo stream send callback"},
				{nativeCGOServerBidiStreamRecvCallbackName(service, method), method.FullName + " cgo stream recv callback"},
				{nativeCGOServerBidiStreamCloseSendCallbackName(service, method), method.FullName + " cgo stream close send callback"},
				{nativeCGOServerBidiStreamDoneCallbackName(service, method), method.FullName + " cgo stream done callback"},
				{nativeCGOServerBidiStreamCancelCallbackName(service, method), method.FullName + " cgo stream cancel callback"},
				{nativeCGOServerBidiStreamStartTrampolineName(service, method), method.FullName + " cgo stream start trampoline"},
				{nativeCGOServerBidiStreamSendTrampolineName(service, method), method.FullName + " cgo stream send trampoline"},
				{nativeCGOServerBidiStreamRecvTrampolineName(service, method), method.FullName + " cgo stream recv trampoline"},
				{nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), method.FullName + " cgo stream close send trampoline"},
				{nativeCGOServerBidiStreamDoneTrampolineName(service, method), method.FullName + " cgo stream done trampoline"},
				{nativeCGOServerBidiStreamCancelTrampolineName(service, method), method.FullName + " cgo stream cancel trampoline"},
				{nativeCGOServerBidiStreamRequestEncoderName(service, method), method.FullName + " request encoder"},
				{nativeCGOServerBidiStreamResponseDecoderName(service, method), method.FullName + " response decoder"},
				{nativeCGOServerBidiStreamResponseCleanupName(service, method), method.FullName + " response cleanup"},
			} {
				if err := addGenerated(item.symbol, item.source); err != nil {
					return err
				}
			}
		}
		if err := validateNativeClientStructFields(requestName, method.NativeContract.RequestFields, nativeClientOutputFieldSymbols); err != nil {
			return err
		}
		if err := validateNativeClientStructFields(responseName, method.NativeContract.ResponseFields, nativeClientInputFieldSymbols); err != nil {
			return err
		}
	}
	if err := validateNativeServerCGOCallbackFields(service); err != nil {
		return err
	}
	return nil
}

func validateNativeServerCGOCallbackFields(service ServicePlan) error {
	seen := make(map[string]string)
	add := func(field, source string) error {
		if previous, exists := seen[field]; exists {
			return fmt.Errorf("native server cgo callback field %s for %s collides with %s", field, source, previous)
		}
		seen[field] = source
		return nil
	}
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			if err := add(method.GoName, method.FullName+" unary callback"); err != nil {
				return err
			}
		case StreamingKindClientStreaming:
			for _, suffix := range []string{"Start", "Send", "Finish", "Cancel"} {
				if err := add(method.GoName+suffix, method.FullName+" client stream "+suffix+" callback"); err != nil {
					return err
				}
			}
		case StreamingKindServerStreaming:
			for _, suffix := range []string{"Start", "Recv", "Done", "Cancel"} {
				if err := add(method.GoName+suffix, method.FullName+" server stream "+suffix+" callback"); err != nil {
					return err
				}
			}
		case StreamingKindBidiStreaming:
			for _, suffix := range []string{"Start", "Send", "Recv", "CloseSend", "Done", "Cancel"} {
				if err := add(method.GoName+suffix, method.FullName+" bidi stream "+suffix+" callback"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addNativeServerCGOGeneratedSymbols(seen map[string]string, service ServicePlan) {
	add := func(symbol, source string) {
		if symbol == "" {
			return
		}
		if _, exists := seen[symbol]; !exists {
			seen[symbol] = source
		}
	}
	errorNames := nativeServerCGOErrorNamesFor(service)
	for symbol, source := range map[string]string{
		service.GoName + "CGONativeServerCallbacks":                    errorNames.CallbacksNil,
		service.GoName + "GoCGONativeServerCallbacks":                  service.FullName + " go helper callbacks",
		lowerInitial(service.GoName) + "CGONativeAdapter":              service.FullName + " adapter",
		lowerInitial(service.GoName) + "GoCGONativeAdapter":            service.FullName + " go helper adapter",
		"Register" + service.GoName + "CGONativeServer":                service.FullName + " registration",
		"Register" + service.GoName + "GoCGONativeServerForTesting":    service.FullName + " go helper registration",
		"Store" + service.GoName + "CGONativeServerErrorTextForExport": service.FullName + " error text export",
		nativeCGOServerErrorIDHelperName(service):                      service.FullName + " error id helper",
		errorNames.CallbacksNil:                                        errorNames.CallbacksNil,
		errorNames.UnaryCallbackMissing:                                errorNames.UnaryCallbackMissing,
		errorNames.UnsupportedField:                                    errorNames.UnsupportedField,
		errorNames.StreamNotImplemented:                                errorNames.StreamNotImplemented,
	} {
		add(symbol, source)
	}
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			add(nativeCGOServerRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerCallbackName(service, method), method.FullName+" cgo callback")
			add(nativeCGOServerTrampolineName(service, method), method.FullName+" cgo trampoline")
			add(nativeCGOServerRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerResponseCleanupName(service, method), method.FullName+" response cleanup")
		case StreamingKindClientStreaming:
			add(nativeCGOServerClientStreamRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerClientStreamResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerClientStreamStartCallbackName(service, method), method.FullName+" cgo stream start callback")
			add(nativeCGOServerClientStreamSendCallbackName(service, method), method.FullName+" cgo stream send callback")
			add(nativeCGOServerClientStreamFinishCallbackName(service, method), method.FullName+" cgo stream finish callback")
			add(nativeCGOServerClientStreamCancelCallbackName(service, method), method.FullName+" cgo stream cancel callback")
			add(nativeCGOServerClientStreamStartTrampolineName(service, method), method.FullName+" cgo stream start trampoline")
			add(nativeCGOServerClientStreamSendTrampolineName(service, method), method.FullName+" cgo stream send trampoline")
			add(nativeCGOServerClientStreamFinishTrampolineName(service, method), method.FullName+" cgo stream finish trampoline")
			add(nativeCGOServerClientStreamCancelTrampolineName(service, method), method.FullName+" cgo stream cancel trampoline")
			add(nativeCGOServerClientStreamRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerClientStreamResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerClientStreamResponseCleanupName(service, method), method.FullName+" response cleanup")
		case StreamingKindServerStreaming:
			add(nativeCGOServerServerStreamRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerServerStreamResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerServerStreamStartCallbackName(service, method), method.FullName+" cgo stream start callback")
			add(nativeCGOServerServerStreamRecvCallbackName(service, method), method.FullName+" cgo stream recv callback")
			add(nativeCGOServerServerStreamDoneCallbackName(service, method), method.FullName+" cgo stream done callback")
			add(nativeCGOServerServerStreamCancelCallbackName(service, method), method.FullName+" cgo stream cancel callback")
			add(nativeCGOServerServerStreamStartTrampolineName(service, method), method.FullName+" cgo stream start trampoline")
			add(nativeCGOServerServerStreamRecvTrampolineName(service, method), method.FullName+" cgo stream recv trampoline")
			add(nativeCGOServerServerStreamDoneTrampolineName(service, method), method.FullName+" cgo stream done trampoline")
			add(nativeCGOServerServerStreamCancelTrampolineName(service, method), method.FullName+" cgo stream cancel trampoline")
			add(nativeCGOServerServerStreamRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerServerStreamResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerServerStreamResponseCleanupName(service, method), method.FullName+" response cleanup")
		case StreamingKindBidiStreaming:
			add(nativeCGOServerBidiStreamRequestName(service, method), method.FullName+" cgo request")
			add(nativeCGOServerBidiStreamResponseName(service, method), method.FullName+" cgo response")
			add(nativeCGOServerBidiStreamStartCallbackName(service, method), method.FullName+" cgo stream start callback")
			add(nativeCGOServerBidiStreamSendCallbackName(service, method), method.FullName+" cgo stream send callback")
			add(nativeCGOServerBidiStreamRecvCallbackName(service, method), method.FullName+" cgo stream recv callback")
			add(nativeCGOServerBidiStreamCloseSendCallbackName(service, method), method.FullName+" cgo stream close send callback")
			add(nativeCGOServerBidiStreamDoneCallbackName(service, method), method.FullName+" cgo stream done callback")
			add(nativeCGOServerBidiStreamCancelCallbackName(service, method), method.FullName+" cgo stream cancel callback")
			add(nativeCGOServerBidiStreamStartTrampolineName(service, method), method.FullName+" cgo stream start trampoline")
			add(nativeCGOServerBidiStreamSendTrampolineName(service, method), method.FullName+" cgo stream send trampoline")
			add(nativeCGOServerBidiStreamRecvTrampolineName(service, method), method.FullName+" cgo stream recv trampoline")
			add(nativeCGOServerBidiStreamCloseSendTrampolineName(service, method), method.FullName+" cgo stream close send trampoline")
			add(nativeCGOServerBidiStreamDoneTrampolineName(service, method), method.FullName+" cgo stream done trampoline")
			add(nativeCGOServerBidiStreamCancelTrampolineName(service, method), method.FullName+" cgo stream cancel trampoline")
			add(nativeCGOServerBidiStreamRequestEncoderName(service, method), method.FullName+" request encoder")
			add(nativeCGOServerBidiStreamResponseDecoderName(service, method), method.FullName+" response decoder")
			add(nativeCGOServerBidiStreamResponseCleanupName(service, method), method.FullName+" response cleanup")
		}
	}
}
