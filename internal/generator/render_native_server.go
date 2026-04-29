package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderNativeServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	runtimeMethods, err := buildRuntimeAdapterMethods(service)
	if err != nil {
		return err
	}

	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
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

	serverName := service.GoName + "NativeServer"
	adapterName := lowerInitial(service.GoName) + "GoNativeAdapter"

	renderGoNativeServerInterface(g, service, serverName)
	renderGoNativeStreamInterfaces(g, service)
	renderGoNativeAdapter(g, service, runtimeMethods, serverName, adapterName)
	renderGoNativeRegistration(g, service, serverName, adapterName)
	return nil
}

func renderGoNativeServerInterface(g *protogen.GeneratedFile, service ServicePlan, serverName string) {
	g.P("type ", serverName, " interface {")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", nativeGoMessageType(g, method.Response), ", error)")
		case StreamingKindClientStreaming:
			g.P(method.GoName, "(ctx context.Context) (", service.GoName, method.GoName, "NativeClientStream, error)")
		case StreamingKindServerStreaming:
			g.P(method.GoName, "(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") (", service.GoName, method.GoName, "NativeServerStream, error)")
		case StreamingKindBidiStreaming:
			g.P(method.GoName, "(ctx context.Context) (", service.GoName, method.GoName, "NativeBidiStream, error)")
		}
	}
	g.P("}")
	g.P()
}

func renderGoNativeStreamInterfaces(g *protogen.GeneratedFile, service ServicePlan) {
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindClientStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeClientStream interface {")
			g.P("Recv(ctx context.Context) (", nativeGoMessageType(g, method.Request), ", error)")
			g.P("Finish(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error)")
			g.P("Cancel(ctx context.Context) error")
			g.P("}")
			g.P()
		case StreamingKindServerStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeServerStream interface {")
			g.P("Recv(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error)")
			g.P("Cancel(ctx context.Context) error")
			g.P("}")
			g.P()
		case StreamingKindBidiStreaming:
			g.P("type ", service.GoName, method.GoName, "NativeBidiStream interface {")
			g.P("Send(ctx context.Context, req ", nativeGoMessageType(g, method.Request), ") error")
			g.P("Recv(ctx context.Context) (", nativeGoMessageType(g, method.Response), ", error)")
			g.P("CloseSend(ctx context.Context) error")
			g.P("Cancel(ctx context.Context) error")
			g.P("}")
			g.P()
		}
	}
}

func renderGoNativeAdapter(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, serverName, adapterName string) {
	g.P("type ", adapterName, " struct {")
	g.P("server ", serverName)
	g.P("}")
	g.P()

	byName := make(map[string]MethodPlan, len(service.Methods))
	for _, method := range service.Methods {
		byName[method.GoName] = method
	}

	for _, runtimeMethod := range methods {
		method, ok := byName[runtimeMethod.MethodGoName]
		if !ok {
			renderGoNativeFallbackAdapterMethod(g, adapterName, runtimeMethod)
			continue
		}
		switch method.Streaming {
		case StreamingKindUnary:
			renderGoNativeUnaryAdapterMethod(g, adapterName, method)
		case StreamingKindClientStreaming:
			renderGoNativeClientStreamAdapterMethod(g, service, adapterName, method)
		case StreamingKindServerStreaming:
			renderGoNativeServerStreamAdapterMethod(g, service, adapterName, method)
		case StreamingKindBidiStreaming:
			renderGoNativeBidiStreamAdapterMethod(g, service, adapterName, method)
		}
	}
}

func renderGoNativeUnaryAdapterMethod(g *protogen.GeneratedFile, adapterName string, method MethodPlan) {
	g.P("func (a *", adapterName, ") ", method.GoName, "(ctx context.Context) error {")
	g.P("_, err := a.server.", method.GoName, "(ctx, nil)")
	g.P("return err")
	g.P("}")
	g.P()
}

func renderGoNativeClientStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("stream, err := a.server.", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoNativeClientStreamSession{stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoNativeClientStreamSession struct {")
	g.P("stream ", service.GoName, method.GoName, "NativeClientStream")
	g.P("}")
	g.P()
	renderNoopSend(g, lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession")
	g.P("func (s *", lowerInitial(service.GoName), method.GoName, "GoNativeClientStreamSession) Finish(ctx context.Context) error {")
	g.P("_, err := s.stream.Finish(ctx)")
	g.P("return err")
	g.P("}")
	g.P()
	renderNoopCloseSend(g, lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession")
	renderCancelForwarder(g, lowerInitial(service.GoName)+method.GoName+"GoNativeClientStreamSession")
}

func renderGoNativeServerStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("stream, err := a.server.", method.GoName, "(ctx, nil)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoNativeServerStreamSession{stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoNativeServerStreamSession struct {")
	g.P("stream ", service.GoName, method.GoName, "NativeServerStream")
	g.P("}")
	g.P()
	renderNoopSend(g, lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession")
	renderNoopFinish(g, lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession")
	renderNoopCloseSend(g, lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession")
	renderCancelForwarder(g, lowerInitial(service.GoName)+method.GoName+"GoNativeServerStreamSession")
}

func renderGoNativeBidiStreamAdapterMethod(g *protogen.GeneratedFile, service ServicePlan, adapterName string, method MethodPlan) {
	sessionName := service.GoName + method.GoName + "NativeStreamSession"
	g.P("func (a *", adapterName, ") Start", method.GoName, "(ctx context.Context) (", sessionName, ", error) {")
	g.P("stream, err := a.server.", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", lowerInitial(service.GoName), method.GoName, "GoNativeBidiStreamSession{stream: stream}, nil")
	g.P("}")
	g.P()

	g.P("type ", lowerInitial(service.GoName), method.GoName, "GoNativeBidiStreamSession struct {")
	g.P("stream ", service.GoName, method.GoName, "NativeBidiStream")
	g.P("}")
	g.P()
	renderNoopSend(g, lowerInitial(service.GoName)+method.GoName+"GoNativeBidiStreamSession")
	renderNoopFinish(g, lowerInitial(service.GoName)+method.GoName+"GoNativeBidiStreamSession")
	g.P("func (s *", lowerInitial(service.GoName), method.GoName, "GoNativeBidiStreamSession) CloseSend(ctx context.Context) error {")
	g.P("return s.stream.CloseSend(ctx)")
	g.P("}")
	g.P()
	renderCancelForwarder(g, lowerInitial(service.GoName)+method.GoName+"GoNativeBidiStreamSession")
}

func renderGoNativeFallbackAdapterMethod(g *protogen.GeneratedFile, adapterName string, method runtimeAdapterMethod) {
	g.P("func (a *", adapterName, ") ", method.AdapterName, "(ctx context.Context)", method.AdapterResult, " {")
	if method.Streaming {
		g.P(`return nil, errors.New("rpccgo native server method is not implemented")`)
	} else {
		g.P(`return errors.New("rpccgo native server method is not implemented")`)
	}
	g.P("}")
	g.P()
}

func renderNoopSend(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Send(ctx context.Context) error {")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderNoopFinish(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Finish(ctx context.Context) error {")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderNoopCloseSend(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") CloseSend(ctx context.Context) error {")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderCancelForwarder(g *protogen.GeneratedFile, receiver string) {
	g.P("func (s *", receiver, ") Cancel(ctx context.Context) error {")
	g.P("return s.stream.Cancel(ctx)")
	g.P("}")
	g.P()
}

func renderGoNativeRegistration(g *protogen.GeneratedFile, service ServicePlan, serverName, adapterName string) {
	g.P("func Register", service.GoName, "GoNativeServer(server ", serverName, ") (rpcruntime.AdapterSnapshot[", service.GoName, "NativeAdapter], error) {")
	g.P("if server == nil {")
	g.P(`return rpcruntime.AdapterSnapshot[`, service.GoName, `NativeAdapter]{}, errors.New("rpccgo: `, service.GoName, ` go native server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "ActiveServer(rpcruntime.ServerKindGoNative, &", adapterName, "{server: server})")
	g.P("}")
	g.P()
}

func nativeGoMessageType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}
