package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	streamingMethods := runtimeStreamingMethods(runtimeMethods)
	codecEnabled := service.CodecEnabled

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	adapterName := service.GoName + "NativeAdapter"
	messageAdapterName := service.GoName + "MessageAdapter"
	activeAdapterName := service.GoName + "ActiveAdapter"
	dispatcherName := lowerInitial(service.GoName) + "Dispatcher"

	g.P("type ", adapterName, " interface {")
	for _, method := range runtimeMethods {
		g.P(method.AdapterName, "(ctx context.Context", method.AdapterArgs, ")", method.AdapterResult)
	}
	g.P("}")
	g.P()

	renderRuntimeMessageAdapter(g, service, messageAdapterName, runtimeMethods)
	renderRuntimeActiveAdapter(g, activeAdapterName, adapterName, messageAdapterName)

	for _, method := range streamingMethods {
		renderRuntimeSessionInterface(g, method)
		renderRuntimeMessageSessionInterface(g, method)
	}

	g.P("var ", dispatcherName, " rpcruntime.Dispatcher[", activeAdapterName, "]")
	g.P("var ", lowerInitial(service.GoName), `NativeContractMismatchErr = errors.New("rpccgo: native contract mismatch: active server is message and native/message converter is not enabled")`)
	g.P("var ", lowerInitial(service.GoName), `MessageContractMismatchErr = errors.New("rpccgo: message contract mismatch: active server is native and native/message converter is not enabled")`)
	g.P()

	g.P("func ", service.GoName, "DispatcherForRuntime() *rpcruntime.Dispatcher[", activeAdapterName, "] {")
	g.P("return &", dispatcherName)
	g.P("}")
	g.P()

	g.P("func register", service.GoName, "ActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("snapshot, err := ", dispatcherName, ".Register(kind, rpcruntime.ServerContractNative, ", activeAdapterName, "{Native: adapter})")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", adapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil")
	g.P("}")
	g.P()

	g.P("func register", service.GoName, "MessageActiveServer(kind rpcruntime.ServerKind, adapter ", messageAdapterName, ") (rpcruntime.AdapterSnapshot[", messageAdapterName, "], error) {")
	g.P("snapshot, err := ", dispatcherName, ".Register(kind, rpcruntime.ServerContractMessage, ", activeAdapterName, "{Message: adapter})")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{}, err")
	g.P("}")
	g.P("return rpcruntime.AdapterSnapshot[", messageAdapterName, "]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil")
	g.P("}")
	g.P()

	renderRuntimeCGOBridge(g, service.GoName, adapterName, activeAdapterName, dispatcherName, runtimeMethods, codecEnabled)
	renderRuntimeMessageCGOBridge(g, service.GoName, messageAdapterName, activeAdapterName, dispatcherName, runtimeMethods, codecEnabled)

	return nil
}

type runtimeAdapterMethod struct {
	SourceFullName string
	AdapterName    string
	AdapterArgs    string
	AdapterResult  string
	MethodGoName   string
	SessionName    string
	NativeArgs     string
	NativeReturns  string
	NativeZero     string
	NativeErrZero  string
	NativeArgNames string
	NativeNames    string
	NativeVarDecls []string
	StreamingKind  StreamingKind
	Streaming      bool
}

func buildRuntimeAdapterMethods(g *protogen.GeneratedFile, service ServicePlan) ([]runtimeAdapterMethod, error) {
	if len(service.Methods) == 0 {
		return []runtimeAdapterMethod{
			{AdapterName: "DispatchUnary", AdapterResult: " error", MethodGoName: "DispatchUnary", SessionName: service.GoName + "DispatchUnaryNativeStreamSession"},
			{AdapterName: "StartClientStream", AdapterResult: " (" + service.GoName + "ClientStreamNativeStreamSession, error)", MethodGoName: "ClientStream", SessionName: service.GoName + "ClientStreamNativeStreamSession", Streaming: true},
			{AdapterName: "StartServerStream", AdapterResult: " (" + service.GoName + "ServerStreamNativeStreamSession, error)", MethodGoName: "ServerStream", SessionName: service.GoName + "ServerStreamNativeStreamSession", Streaming: true},
			{AdapterName: "StartBidiStream", AdapterResult: " (" + service.GoName + "BidiStreamNativeStreamSession, error)", MethodGoName: "BidiStream", SessionName: service.GoName + "BidiStreamNativeStreamSession", Streaming: true},
		}, nil
	}

	methods := make([]runtimeAdapterMethod, 0, len(service.Methods))
	seen := make(map[string]string, len(service.Methods))
	for _, method := range service.Methods {
		rendered, err := runtimeAdapterMethodFor(g, service, method)
		if err != nil {
			return nil, err
		}
		if previous, exists := seen[rendered.AdapterName]; exists {
			return nil, fmt.Errorf("runtime adapter method %s for %s collides with %s", rendered.AdapterName, method.FullName, previous)
		}
		seen[rendered.AdapterName] = method.FullName
		methods = append(methods, rendered)
	}
	return methods, nil
}

func runtimeAdapterMethodFor(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) (runtimeAdapterMethod, error) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	nativeArgs := nativeGoRequestParams(g, method.NativeContract.RequestFields)
	nativeReturns := nativeGoResponseReturns(g, method.NativeContract.ResponseFields)
	nativeZero := nativeGoZeroReturns(method.NativeContract.ResponseFields, `errors.New("rpccgo native server method is not implemented")`)
	nativeErrZero := nativeGoZeroReturns(method.NativeContract.ResponseFields, "err")
	nativeArgNames := nativeGoRequestArgNames(method.NativeContract.RequestFields)
	nativeResultNames := nativeGoResponseResultNames(method.NativeContract.ResponseFields)
	nativeVarDecls := nativeGoResponseResultVarDecls(g, method.NativeContract.ResponseFields)
	rendered := runtimeAdapterMethod{
		SourceFullName: method.FullName,
		MethodGoName:   method.GoName,
		SessionName:    sessionName,
		NativeArgs:     nativeArgs,
		NativeReturns:  nativeReturns,
		NativeZero:     nativeZero,
		NativeErrZero:  nativeErrZero,
		NativeArgNames: nativeArgNames,
		NativeNames:    nativeResultNames,
		NativeVarDecls: nativeVarDecls,
		StreamingKind:  method.Streaming,
	}
	switch method.Streaming {
	case StreamingKindUnary:
		rendered.AdapterName = method.GoName
		rendered.AdapterArgs = nativeArgs
		rendered.AdapterResult = " (" + nativeReturns + ")"
	case StreamingKindClientStreaming:
		rendered.AdapterName = "Start" + method.GoName
		rendered.AdapterResult = " (" + sessionName + ", error)"
		rendered.Streaming = true
	case StreamingKindServerStreaming:
		rendered.AdapterName = "Start" + method.GoName
		rendered.AdapterArgs = nativeArgs
		rendered.AdapterResult = " (" + sessionName + ", error)"
		rendered.Streaming = true
	case StreamingKindBidiStreaming:
		rendered.AdapterName = "Start" + method.GoName
		rendered.AdapterResult = " (" + sessionName + ", error)"
		rendered.Streaming = true
	default:
		return runtimeAdapterMethod{}, fmt.Errorf("%s has unknown streaming kind %d", method.FullName, method.Streaming)
	}
	return rendered, nil
}

func nativeRuntimeMessageType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}

func runtimeStreamingMethods(methods []runtimeAdapterMethod) []runtimeAdapterMethod {
	streaming := make([]runtimeAdapterMethod, 0, len(methods))
	for _, method := range methods {
		if method.Streaming {
			streaming = append(streaming, method)
		}
	}
	return streaming
}

func renderRuntimeSessionInterface(g *protogen.GeneratedFile, method runtimeAdapterMethod) {
	g.P("type ", method.SessionName, " interface {")
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("Send(ctx context.Context", method.NativeArgs, ") error")
		g.P("Finish(ctx context.Context) (", method.NativeReturns, ")")
		g.P("Cancel(ctx context.Context) error")
	case StreamingKindServerStreaming:
		g.P("Recv(ctx context.Context) (", method.NativeReturns, ")")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	case StreamingKindBidiStreaming:
		g.P("Send(ctx context.Context", method.NativeArgs, ") error")
		g.P("Recv(ctx context.Context) (", method.NativeReturns, ")")
		g.P("CloseSend(ctx context.Context) error")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	default:
		g.P("Cancel(ctx context.Context) error")
	}
	g.P("}")
	g.P()
}

func renderRuntimeMessageAdapter(g *protogen.GeneratedFile, service ServicePlan, adapterName string, methods []runtimeAdapterMethod) {
	g.P("type ", adapterName, " interface {")
	for _, method := range methods {
		switch method.StreamingKind {
		case StreamingKindUnary:
			g.P(method.AdapterName, "Message(ctx context.Context, req []byte) ([]byte, error)")
		case StreamingKindClientStreaming:
			g.P("Start", method.MethodGoName, "Message(ctx context.Context) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		case StreamingKindServerStreaming:
			g.P("Start", method.MethodGoName, "Message(ctx context.Context, req []byte) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		case StreamingKindBidiStreaming:
			g.P("Start", method.MethodGoName, "Message(ctx context.Context) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		}
	}
	g.P("}")
	g.P()
}

func renderRuntimeActiveAdapter(g *protogen.GeneratedFile, activeAdapterName, nativeAdapterName, messageAdapterName string) {
	g.P("type ", activeAdapterName, " struct {")
	g.P("Native ", nativeAdapterName)
	g.P("Message ", messageAdapterName)
	g.P("}")
	g.P()
}

func renderRuntimeMessageSessionInterface(g *protogen.GeneratedFile, method runtimeAdapterMethod) {
	sessionName := methodMessageSessionName(method)
	g.P("type ", sessionName, " interface {")
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("Send(ctx context.Context, req []byte) error")
		g.P("Finish(ctx context.Context) ([]byte, error)")
		g.P("Cancel(ctx context.Context) error")
	case StreamingKindServerStreaming:
		g.P("Recv(ctx context.Context) ([]byte, error)")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	case StreamingKindBidiStreaming:
		g.P("Send(ctx context.Context, req []byte) error")
		g.P("Recv(ctx context.Context) ([]byte, error)")
		g.P("CloseSend(ctx context.Context) error")
		g.P("Done(ctx context.Context) error")
		g.P("Cancel(ctx context.Context) error")
	default:
		g.P("Cancel(ctx context.Context) error")
	}
	g.P("}")
	g.P()
}

func renderRuntimeCGOBridge(g *protogen.GeneratedFile, serviceName, adapterName, activeAdapterName, dispatcherName string, methods []runtimeAdapterMethod, codecEnabled bool) {
	bridgeName := serviceName + "CGONativeClientBridge"
	g.P("type ", bridgeName, " struct{}")
	g.P()

	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func (", bridgeName, ") ", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (", method.NativeReturns, ") {")
		for _, decl := range method.NativeVarDecls {
			g.P(decl)
		}
		g.P("err := ", dispatcherName, ".Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) error {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		if method.NativeNames == "" {
			g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		} else {
			g.P("var callErr error")
			g.P(method.NativeNames, ", callErr = snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			g.P("return callErr")
		}
		g.P("case rpcruntime.ServerContractMessage:")
		if codecEnabled {
			g.P("if snapshot.Adapter.Message == nil {")
			g.P("return ", lowerInitial(serviceName), "NativeContractMismatchErr")
			g.P("}")
			g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("messageResp, err := snapshot.Adapter.Message.", method.AdapterName, "Message(ctx, messageReq)")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			if method.NativeNames == "" {
				g.P("return ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
			} else {
				g.P("var callErr error")
				g.P(method.NativeNames, ", callErr = ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
				g.P("return callErr")
			}
		} else {
			g.P("return ", lowerInitial(serviceName), "NativeContractMismatchErr")
		}
		g.P("default:")
		g.P("return ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("if err != nil {")
		g.P("return ", method.NativeErrZero)
		g.P("}")
		if method.NativeNames == "" {
			g.P("return nil")
		} else {
			g.P("return ", method.NativeNames, ", nil")
		}
		g.P("}")
		g.P()
		_ = codecEnabled
	}

	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		renderRuntimeCGOStreamBridge(g, serviceName, bridgeName, dispatcherName, method, codecEnabled)
	}

	g.P("func New", serviceName, "CGONativeClientBridge() ", bridgeName, " {")
	g.P("return ", bridgeName, "{}")
	g.P("}")
	g.P()

	g.P("func Register", serviceName, "CGONativeActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return register", serviceName, "ActiveServer(kind, adapter)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageCGOBridge(g *protogen.GeneratedFile, serviceName, adapterName, activeAdapterName, dispatcherName string, methods []runtimeAdapterMethod, codecEnabled bool) {
	bridgeName := serviceName + "CGOMessageClientBridge"
	g.P("type ", bridgeName, " struct{}")
	g.P()

	for _, method := range methods {
		if method.Streaming {
			continue
		}
		g.P("func (", bridgeName, ") ", method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error) {")
		g.P("var resp []byte")
		g.P("err := ", dispatcherName, ".Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) error {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("var callErr error")
		g.P("resp, callErr = snapshot.Adapter.Message.", method.AdapterName, "Message(ctx, req)")
		g.P("return callErr")
		g.P("case rpcruntime.ServerContractNative:")
		if codecEnabled {
			g.P("if snapshot.Adapter.Native == nil {")
			g.P("return ", lowerInitial(serviceName), "MessageContractMismatchErr")
			g.P("}")
			g.P("return ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
			if method.NativeNames == "" {
				g.P("err := snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			} else {
				g.P(method.NativeNames, ", err := snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
			}
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("messageResp, err := ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
			g.P("if err != nil {")
			g.P("return err")
			g.P("}")
			g.P("resp = messageResp")
			g.P("return nil")
			g.P("})")
		} else {
			g.P("return ", lowerInitial(serviceName), "MessageContractMismatchErr")
		}
		g.P("default:")
		g.P("return ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return resp, nil")
		g.P("}")
		g.P()
	}

	for _, method := range methods {
		if !method.Streaming {
			continue
		}
		renderRuntimeMessageCGOStreamBridge(g, serviceName, bridgeName, adapterName, activeAdapterName, dispatcherName, method, codecEnabled)
	}

	g.P("func New", serviceName, "CGOMessageClientBridge() ", bridgeName, " {")
	g.P("return ", bridgeName, "{}")
	g.P("}")
	g.P()

	g.P("func Register", serviceName, "CGOMessageActiveServer(kind rpcruntime.ServerKind, adapter ", adapterName, ") (rpcruntime.AdapterSnapshot[", adapterName, "], error) {")
	g.P("return register", serviceName, "MessageActiveServer(kind, adapter)")
	g.P("}")
	g.P()
}

func renderRuntimeMessageCGOStreamBridge(g *protogen.GeneratedFile, serviceName, bridgeName, adapterName, activeAdapterName, dispatcherName string, method runtimeAdapterMethod, codecEnabled bool) {
	if !codecEnabled {
		renderRuntimeMessageCGOStreamBridgeDirect(g, serviceName, bridgeName, adapterName, activeAdapterName, dispatcherName, method)
		return
	}
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("handle, err := ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("nativeSession, err := snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return &", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}, nil")
		g.P("default:")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return handle, nil")
		g.P("}")
		g.P()
	case StreamingKindServerStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
		g.P("handle, err := ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx, req)")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("var session any")
		g.P("err := ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
		g.P("nativeSession, err := snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("session = &", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}")
		g.P("return nil")
		g.P("})")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return session, nil")
		g.P("default:")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return handle, nil")
		g.P("}")
		g.P()
	case StreamingKindBidiStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("handle, err := ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("nativeSession, err := snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return &", nativeToMessageStreamWrapperName(serviceName, method), "{native: nativeSession}, nil")
		g.P("default:")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return handle, nil")
		g.P("}")
		g.P()
	}

	renderNativeToMessageStreamWrapper(g, serviceName, method)
}

func renderRuntimeMessageCGOStreamBridgeDirect(g *protogen.GeneratedFile, serviceName, bridgeName, _ string, activeAdapterName, dispatcherName string, method runtimeAdapterMethod) {
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("handle, err := ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
		g.P("if snapshot.Contract != rpcruntime.ServerContractMessage || snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
		g.P("})")
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return handle, nil")
		g.P("}")
		g.P()
	case StreamingKindServerStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {")
		g.P("handle, err := ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
		g.P("if snapshot.Contract != rpcruntime.ServerContractMessage || snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx, req)")
		g.P("})")
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return handle, nil")
		g.P("}")
		g.P()
	case StreamingKindBidiStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("handle, err := ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", activeAdapterName, "]) (any, error) {")
		g.P("if snapshot.Contract != rpcruntime.ServerContractMessage || snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "MessageContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
		g.P("})")
		g.P("if err != nil {")
		g.P("return 0, err")
		g.P("}")
		g.P("return handle, nil")
		g.P("}")
		g.P()
	}

}

func renderNativeToMessageStreamWrapper(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	wrapperName := nativeToMessageStreamWrapperName(serviceName, method)
	g.P("type ", wrapperName, " struct {")
	g.P("native ", method.SessionName)
	g.P("}")
	g.P()
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		renderNativeToMessageSend(g, serviceName, method, wrapperName)
		renderNativeToMessageFinish(g, serviceName, method, wrapperName)
		renderNativeToMessageCancel(g, wrapperName)
	case StreamingKindServerStreaming:
		renderNativeToMessageRecv(g, serviceName, method, wrapperName)
		renderNativeToMessageDone(g, wrapperName)
		renderNativeToMessageCancel(g, wrapperName)
	case StreamingKindBidiStreaming:
		renderNativeToMessageSend(g, serviceName, method, wrapperName)
		renderNativeToMessageRecv(g, serviceName, method, wrapperName)
		renderNativeToMessageCloseSend(g, wrapperName)
		renderNativeToMessageDone(g, wrapperName)
		renderNativeToMessageCancel(g, wrapperName)
	}
}

func renderNativeToMessageSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Send(ctx context.Context, req []byte) error {")
	g.P("return ", codecMessageToNativeRequestName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(req, func(", strings.TrimPrefix(method.NativeArgs, ", "), ") error {")
	g.P("return s.native.Send(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
	g.P("})")
	g.P("}")
	g.P()
}

func renderNativeToMessageFinish(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) ([]byte, error) {")
	if method.NativeNames == "" {
		g.P("err := s.native.Finish(ctx)")
	} else {
		g.P(method.NativeNames, ", err := s.native.Finish(ctx)")
	}
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
	g.P("}")
	g.P()
}

func renderNativeToMessageRecv(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) ([]byte, error) {")
	if method.NativeNames == "" {
		g.P("err := s.native.Recv(ctx)")
	} else {
		g.P(method.NativeNames, ", err := s.native.Recv(ctx)")
	}
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return ", codecNativeResponseToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeNames, ")")
	g.P("}")
	g.P()
}

func renderNativeToMessageCloseSend(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("return s.native.CloseSend(ctx)")
	g.P("}")
	g.P()
}

func renderNativeToMessageDone(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("return s.native.Done(ctx)")
	g.P("}")
	g.P()
}

func renderNativeToMessageCancel(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("return s.native.Cancel(ctx)")
	g.P("}")
	g.P()
}

func renderRuntimeCGOStreamBridge(g *protogen.GeneratedFile, serviceName, bridgeName, dispatcherName string, method runtimeAdapterMethod, codecEnabled bool) {
	if !codecEnabled {
		renderRuntimeCGOStreamBridgeDirect(g, serviceName, bridgeName, dispatcherName, method)
		return
	}
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("return ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", serviceName, "ActiveAdapter]) (any, error) {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("messageSession, err := snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return &", messageToNativeStreamWrapperName(serviceName, method), "{message: messageSession}, nil")
		g.P("default:")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("}")
		g.P()
	case StreamingKindServerStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
		g.P("return ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", serviceName, "ActiveAdapter]) (any, error) {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("messageSession, err := snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx, messageReq)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return &", messageToNativeStreamWrapperName(serviceName, method), "{message: messageSession}, nil")
		g.P("default:")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("}")
		g.P()
	case StreamingKindBidiStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("return ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", serviceName, "ActiveAdapter]) (any, error) {")
		g.P("switch snapshot.Contract {")
		g.P("case rpcruntime.ServerContractNative:")
		g.P("if snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
		g.P("case rpcruntime.ServerContractMessage:")
		g.P("if snapshot.Adapter.Message == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("messageSession, err := snapshot.Adapter.Message.Start", method.MethodGoName, "Message(ctx)")
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P("return &", messageToNativeStreamWrapperName(serviceName, method), "{message: messageSession}, nil")
		g.P("default:")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("})")
		g.P("}")
		g.P()
	}

	renderMessageToNativeStreamWrapper(g, serviceName, method)
}

func renderRuntimeCGOStreamBridgeDirect(g *protogen.GeneratedFile, serviceName, bridgeName, dispatcherName string, method runtimeAdapterMethod) {
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("return ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", serviceName, "ActiveAdapter]) (any, error) {")
		g.P("if snapshot.Contract != rpcruntime.ServerContractNative || snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
		g.P("})")
		g.P("}")
		g.P()
	case StreamingKindServerStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context", method.NativeArgs, ") (rpcruntime.StreamHandle, error) {")
		g.P("return ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", serviceName, "ActiveAdapter]) (any, error) {")
		g.P("if snapshot.Contract != rpcruntime.ServerContractNative || snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx", nativeGoCallSuffix(method.NativeArgNames), ")")
		g.P("})")
		g.P("}")
		g.P()
	case StreamingKindBidiStreaming:
		g.P("func (", bridgeName, ") Start", method.MethodGoName, "(ctx context.Context) (rpcruntime.StreamHandle, error) {")
		g.P("return ", dispatcherName, ".StartStream(func(snapshot rpcruntime.AdapterSnapshot[", serviceName, "ActiveAdapter]) (any, error) {")
		g.P("if snapshot.Contract != rpcruntime.ServerContractNative || snapshot.Adapter.Native == nil {")
		g.P("return nil, ", lowerInitial(serviceName), "NativeContractMismatchErr")
		g.P("}")
		g.P("return snapshot.Adapter.Native.", method.AdapterName, "(ctx)")
		g.P("})")
		g.P("}")
		g.P()
	}

}

func renderMessageToNativeStreamWrapper(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod) {
	wrapperName := messageToNativeStreamWrapperName(serviceName, method)
	g.P("type ", wrapperName, " struct {")
	g.P("message ", methodMessageSessionName(method))
	g.P("}")
	g.P()
	switch method.StreamingKind {
	case StreamingKindClientStreaming:
		renderMessageToNativeSend(g, serviceName, method, wrapperName)
		renderMessageToNativeFinish(g, serviceName, method, wrapperName)
		renderMessageToNativeCancel(g, wrapperName)
	case StreamingKindServerStreaming:
		renderMessageToNativeRecv(g, serviceName, method, wrapperName)
		renderMessageToNativeDone(g, wrapperName)
		renderMessageToNativeCancel(g, wrapperName)
	case StreamingKindBidiStreaming:
		renderMessageToNativeSend(g, serviceName, method, wrapperName)
		renderMessageToNativeRecv(g, serviceName, method, wrapperName)
		renderMessageToNativeCloseSend(g, wrapperName)
		renderMessageToNativeDone(g, wrapperName)
		renderMessageToNativeCancel(g, wrapperName)
	}
}

func renderMessageToNativeSend(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Send(ctx context.Context", method.NativeArgs, ") error {")
	g.P("messageReq, err := ", codecNativeRequestToMessageName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(", method.NativeArgNames, ")")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("return s.message.Send(ctx, messageReq)")
	g.P("}")
	g.P()
}

func renderMessageToNativeFinish(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Finish(ctx context.Context) (", method.NativeReturns, ") {")
	g.P("messageResp, err := s.message.Finish(ctx)")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("return ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
	g.P("}")
	g.P()
}

func renderMessageToNativeRecv(g *protogen.GeneratedFile, serviceName string, method runtimeAdapterMethod, wrapperName string) {
	g.P("func (s *", wrapperName, ") Recv(ctx context.Context) (", method.NativeReturns, ") {")
	g.P("messageResp, err := s.message.Recv(ctx)")
	g.P("if err != nil {")
	g.P("return ", method.NativeErrZero)
	g.P("}")
	g.P("return ", codecMessageToNativeResponseName(ServicePlan{GoName: serviceName}, MethodPlan{GoName: method.MethodGoName}), "(messageResp)")
	g.P("}")
	g.P()
}

func renderMessageToNativeCloseSend(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") CloseSend(ctx context.Context) error {")
	g.P("return s.message.CloseSend(ctx)")
	g.P("}")
	g.P()
}

func renderMessageToNativeDone(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Done(ctx context.Context) error {")
	g.P("return s.message.Done(ctx)")
	g.P("}")
	g.P()
}

func renderMessageToNativeCancel(g *protogen.GeneratedFile, wrapperName string) {
	g.P("func (s *", wrapperName, ") Cancel(ctx context.Context) error {")
	g.P("return s.message.Cancel(ctx)")
	g.P("}")
	g.P()
}

func methodMessageSessionName(method runtimeAdapterMethod) string {
	return strings.Replace(method.SessionName, "NativeStreamSession", "MessageStreamSession", 1)
}

func nativeToMessageStreamWrapperName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "NativeToMessageStreamSession"
}

func messageToNativeStreamWrapperName(serviceName string, method runtimeAdapterMethod) string {
	return lowerInitial(serviceName) + method.MethodGoName + "MessageToNativeStreamSession"
}
