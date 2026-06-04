package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeBindingTypes(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection) {
	nativeBindingName := lowerInitial(service.GoName) + "NativeActiveBinding"
	messageBindingName := lowerInitial(service.GoName) + "MessageActiveBinding"
	renderRuntimeNativeBindingType(g, service, methods, nativeBindingName)
	renderRuntimeMessageBindingType(g, service, methods, messageBindingName)
}

func renderRuntimeNativeBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, bindingName string) {
	g.P("// ", bindingName, " is the immutable native active closure set.")
	g.P("type ", bindingName, " struct {")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("invoke", method.Identity.GoName, " func(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ")")
			continue
		}
		nativeSession := runtimeStreamNativeSessionName(service.GoName, method)
		if method.Stream.StartAcceptsRequest {
			g.P("start", method.Identity.GoName, " func(ctx context.Context", method.Native.Args, ") (*", nativeSession, ", error)")
			continue
		}
		g.P("start", method.Identity.GoName, " func(ctx context.Context) (*", nativeSession, ", error)")
	}
	g.P("}")
	g.P()
	if service.Generation.NativeEnabled {
		renderRuntimeNativeActiveContractMethods(g, service.GoName, bindingName, methods)
	}
}

func renderRuntimeMessageBindingType(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, bindingName string) {
	g.P("// ", bindingName, " is the immutable message active closure set.")
	g.P("type ", bindingName, " struct {")
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("invoke", method.Identity.GoName, " func(ctx context.Context, req []byte) ([]byte, error)")
			continue
		}
		messageSession := runtimeStreamMessageSessionName(service.GoName, method)
		if method.Stream.StartAcceptsRequest {
			g.P("start", method.Identity.GoName, " func(ctx context.Context, req []byte) (*", messageSession, ", error)")
			continue
		}
		g.P("start", method.Identity.GoName, " func(ctx context.Context) (*", messageSession, ", error)")
	}
	g.P("}")
	g.P()
	renderRuntimeMessageActiveContractMethods(g, service.GoName, bindingName, methods)
}

func renderRuntimeNativeActiveContractMethods(g *protogen.GeneratedFile, serviceName, bindingName string, methods []runtimeMethodProjection) {
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ") (", method.Native.Returns, ") {")
			g.P("return a.invoke", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
			g.P("}")
			g.P()
			continue
		}
		switch method.Stream.Shape {
		case runtimeStreamClient:
			renderRuntimeNativeActiveClientStreamContractMethod(g, serviceName, bindingName, method)
		case runtimeStreamServer:
			renderRuntimeNativeActiveServerStreamContractMethod(g, serviceName, bindingName, method)
		case runtimeStreamBidi:
			renderRuntimeNativeActiveBidiStreamContractMethod(g, serviceName, bindingName, method)
		}
	}
}

func renderRuntimeMessageActiveContractMethods(g *protogen.GeneratedFile, serviceName, bindingName string, methods []runtimeMethodProjection) {
	for _, method := range methods {
		if !method.Stream.Streaming {
			g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context, req []byte) ([]byte, error) {")
			g.P("return a.invoke", method.Identity.GoName, "(ctx, req)")
			g.P("}")
			g.P()
			continue
		}
		switch method.Stream.Shape {
		case runtimeStreamClient:
			renderRuntimeMessageActiveClientStreamContractMethod(g, serviceName, bindingName, method)
		case runtimeStreamServer:
			renderRuntimeMessageActiveServerStreamContractMethod(g, serviceName, bindingName, method)
		case runtimeStreamBidi:
			renderRuntimeMessageActiveBidiStreamContractMethod(g, serviceName, bindingName, method)
		}
	}
}

func renderRuntimeNativeActiveClientStreamContractMethod(g *protogen.GeneratedFile, serviceName, bindingName string, method runtimeMethodProjection) {
	streamType := serviceName + method.Identity.GoName + "NativeClientStream"
	g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context, stream ", streamType, ") (", method.Native.Returns, ") {")
	g.P("session, err := a.start", method.Identity.GoName, "(ctx)")
	g.P("if err != nil { return ", method.Native.ErrZero, " }")
	g.P("for {")
	renderRuntimeNativeContractRecvAssign(g, method, "stream")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) { return session.finish(ctx) }")
	g.P("_ = session.cancel(ctx)")
	g.P("return ", method.Native.ErrZero)
	g.P("}")
	g.P("if err := session.send(ctx", nativeGoCallSuffix(method.Native.ArgNames), "); err != nil {")
	g.P("_ = session.cancel(ctx)")
	g.P("return ", method.Native.ErrZero)
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeActiveServerStreamContractMethod(g *protogen.GeneratedFile, serviceName, bindingName string, method runtimeMethodProjection) {
	streamType := serviceName + method.Identity.GoName + "NativeServerStream"
	g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context", method.Native.Args, ", stream ", streamType, ") error {")
	g.P("session, err := a.start", method.Identity.GoName, "(ctx", nativeGoCallSuffix(method.Native.ArgNames), ")")
	g.P("if err != nil { return err }")
	g.P("for {")
	renderRuntimeNativeContractRecvAssign(g, method, "session")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) { return session.finish(ctx) }")
	g.P("_ = session.cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("if err := stream.Send(ctx", nativeGoCallSuffix(method.Native.ResultNames), "); err != nil {")
	g.P("if errors.Is(err, io.EOF) { return session.finish(ctx) }")
	g.P("_ = session.cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeNativeActiveBidiStreamContractMethod(g *protogen.GeneratedFile, serviceName, bindingName string, method runtimeMethodProjection) {
	streamType := serviceName + method.Identity.GoName + "NativeBidiStream"
	g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context, stream ", streamType, ") error {")
	g.P("session, err := a.start", method.Identity.GoName, "(ctx)")
	g.P("if err != nil { return err }")
	g.P("errs := make(chan error, 2)")
	g.P("go func() {")
	g.P("for {")
	renderRuntimeNativeContractRecvAssign(g, method, "stream")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.closeSend(ctx)")
	g.P("return")
	g.P("}")
	g.P("if err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if err := session.send(ctx", nativeGoCallSuffix(method.Native.ArgNames), "); err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("go func() {")
	g.P("for {")
	renderRuntimeNativeContractRecvAssign(g, method, "session")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.finish(ctx)")
	g.P("return")
	g.P("}")
	g.P("if err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if err := stream.Send(ctx", nativeGoCallSuffix(method.Native.ResultNames), "); err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.finish(ctx)")
	g.P("return")
	g.P("}")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("for range 2 {")
	g.P("if err := <-errs; err != nil {")
	g.P("_ = session.cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderRuntimeMessageActiveClientStreamContractMethod(g *protogen.GeneratedFile, serviceName, bindingName string, method runtimeMethodProjection) {
	streamType := serviceName + method.Identity.GoName + "MessageClientStream"
	g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context, stream ", streamType, ") ([]byte, error) {")
	g.P("session, err := a.start", method.Identity.GoName, "(ctx)")
	g.P("if err != nil { return nil, err }")
	g.P("for {")
	g.P("req, err := stream.Recv(ctx)")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) { return session.finish(ctx) }")
	g.P("_ = session.cancel(ctx)")
	g.P("return nil, err")
	g.P("}")
	g.P("if err := session.send(ctx, req); err != nil {")
	g.P("_ = session.cancel(ctx)")
	g.P("return nil, err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageActiveServerStreamContractMethod(g *protogen.GeneratedFile, serviceName, bindingName string, method runtimeMethodProjection) {
	streamType := serviceName + method.Identity.GoName + "MessageServerStream"
	g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context, req []byte, stream ", streamType, ") error {")
	g.P("session, err := a.start", method.Identity.GoName, "(ctx, req)")
	g.P("if err != nil { return err }")
	g.P("for {")
	g.P("resp, err := session.recv(ctx)")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) { return session.finish(ctx) }")
	g.P("_ = session.cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("if err := stream.Send(ctx, resp); err != nil {")
	g.P("if errors.Is(err, io.EOF) { return session.finish(ctx) }")
	g.P("_ = session.cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderRuntimeMessageActiveBidiStreamContractMethod(g *protogen.GeneratedFile, serviceName, bindingName string, method runtimeMethodProjection) {
	streamType := serviceName + method.Identity.GoName + "MessageBidiStream"
	g.P("func (a *", bindingName, ") ", method.Identity.GoName, "(ctx context.Context, stream ", streamType, ") error {")
	g.P("session, err := a.start", method.Identity.GoName, "(ctx)")
	g.P("if err != nil { return err }")
	g.P("errs := make(chan error, 2)")
	g.P("go func() {")
	g.P("for {")
	g.P("req, err := stream.Recv(ctx)")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.closeSend(ctx)")
	g.P("return")
	g.P("}")
	g.P("if err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if err := session.send(ctx, req); err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("go func() {")
	g.P("for {")
	g.P("resp, err := session.recv(ctx)")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.finish(ctx)")
	g.P("return")
	g.P("}")
	g.P("if err != nil {")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("if err := stream.Send(ctx, resp); err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("errs <- session.finish(ctx)")
	g.P("return")
	g.P("}")
	g.P("errs <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("for range 2 {")
	g.P("if err := <-errs; err != nil {")
	g.P("_ = session.cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func renderRuntimeNativeContractRecvAssign(g *protogen.GeneratedFile, method runtimeMethodProjection, receiver string) {
	if method.Native.ArgNames == "" && receiver == "stream" {
		g.P("err := ", receiver, ".Recv(ctx)")
		return
	}
	if method.Native.ResultNames == "" && receiver == "session" {
		g.P("err := ", receiver, ".recv(ctx)")
		return
	}
	if receiver == "stream" {
		g.P(method.Native.ArgNames, ", err := ", receiver, ".Recv(ctx)")
		return
	}
	g.P(method.Native.ResultNames, ", err := ", receiver, ".recv(ctx)")
}
