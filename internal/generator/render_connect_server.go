package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderConnectServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	if serviceHasStreamingMethod(service) {
		g.P(`io "io"`)
		g.P(`rpcruntime "rpccgo/rpcruntime"`)
	}
	if serviceHasBidiStreamingMethod(service) {
		g.P(`sync "sync"`)
	}
	g.P(`http "net/http"`)
	g.P(`connect "connectrpc.com/connect"`)
	g.P(`proto "google.golang.org/protobuf/proto"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	g.P("const ", service.GoName, `ConnectServiceName = "`, service.FullName, `"`)
	g.P("const ", service.GoName, `ConnectServicePathPrefix = "/`, service.FullName, `/"`)
	for _, method := range service.Methods {
		g.P("const ", connectProcedureConstName(service, method), ` = "/`, service.FullName, `/`, method.Name, `"`)
	}
	g.P()

	g.P("func New", service.GoName, "ConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {")
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			g.P(connectProcedureHandlerName(service, method), " := connect.NewUnaryHandler(", connectProcedureConstName(service, method), ", ", connectImplementationName(service, method), ", options...)")
		case StreamingKindClientStreaming:
			g.P(connectProcedureHandlerName(service, method), " := connect.NewClientStreamHandler(", connectProcedureConstName(service, method), ", ", connectImplementationName(service, method), ", options...)")
		case StreamingKindServerStreaming:
			g.P(connectProcedureHandlerName(service, method), " := connect.NewServerStreamHandler(", connectProcedureConstName(service, method), ", ", connectImplementationName(service, method), ", options...)")
		case StreamingKindBidiStreaming:
			g.P(connectProcedureHandlerName(service, method), " := connect.NewBidiStreamHandler(", connectProcedureConstName(service, method), ", ", connectImplementationName(service, method), ", options...)")
		}
	}
	g.P("return ", service.GoName, "ConnectServicePathPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {")
	g.P("switch r.URL.Path {")
	for _, method := range service.Methods {
		g.P("case ", connectProcedureConstName(service, method), ":")
		g.P(connectProcedureHandlerName(service, method), ".ServeHTTP(w, r)")
	}
	g.P("default:")
	g.P("http.NotFound(w, r)")
	g.P("}")
	g.P("})")
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderConnectUnaryImplementation(g, service, method)
		case StreamingKindClientStreaming:
			renderConnectClientStreamingImplementation(g, service, method)
		case StreamingKindServerStreaming:
			renderConnectServerStreamingImplementation(g, service, method)
		case StreamingKindBidiStreaming:
			renderConnectBidiStreamingImplementation(g, service, method)
		}
	}

	return nil
}

func renderConnectUnaryImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeName := service.GoName + "CGOMessageClientBridge"
	g.P("func ", connectImplementationName(service, method), "(ctx context.Context, req *connect.Request[", reqType, "]) (*connect.Response[", respType, "], error) {")
	g.P("if req == nil || req.Msg == nil {")
	g.P(`return nil, errors.New("rpccgo: connect request is nil")`)
	g.P("}")
	g.P("reqData, err := proto.Marshal(req.Msg)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect request protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("respData, err := New", bridgeName, "().", method.GoName, "(ctx, reqData)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect response protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return connect.NewResponse(resp), nil")
	g.P("}")
	g.P()
}

func renderConnectClientStreamingImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeType := service.GoName + "CGOMessageClientBridge"
	g.P("func ", connectImplementationName(service, method), "(ctx context.Context, stream *connect.ClientStream[", reqType, "]) (*connect.Response[", respType, "], error) {")
	g.P("bridge := New", bridgeType, "()")
	g.P("handle, err := bridge.Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	renderDispatcherStreamLoadSession(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", `nil, errors.New("rpccgo: connect message stream handle is invalid")`, "handle")
	g.P("for stream.Receive() {")
	g.P("reqData, err := proto.Marshal(stream.Msg())")
	g.P("if err != nil {")
	renderDispatcherStreamCancelIfPresent(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", "handle", "ctx")
	g.P(`return nil, fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("if err := session.Send(ctx, reqData); err != nil {")
	renderDispatcherStreamCancelIfPresent(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", "handle", "ctx")
	g.P("return nil, err")
	g.P("}")
	g.P("}")
	g.P("if err := stream.Err(); err != nil {")
	renderDispatcherStreamCancelIfPresent(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", "handle", "ctx")
	g.P("return nil, err")
	g.P("}")
	renderDispatcherStreamTakeSession(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", `nil, errors.New("rpccgo: connect message stream handle is invalid")`, "handle", "terminal")
	g.P("respData, err := terminal.Finish(ctx)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return connect.NewResponse(resp), nil")
	g.P("}")
	g.P()
}

func renderConnectServerStreamingImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeType := service.GoName + "CGOMessageClientBridge"
	g.P("func ", connectImplementationName(service, method), "(ctx context.Context, req *connect.Request[", reqType, "], stream *connect.ServerStream[", respType, "]) error {")
	g.P("if req == nil || req.Msg == nil {")
	g.P(`return errors.New("rpccgo: connect request is nil")`)
	g.P("}")
	g.P("reqData, err := proto.Marshal(req.Msg)")
	g.P("if err != nil {")
	g.P(`return fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("bridge := New", bridgeType, "()")
	g.P("handle, err := bridge.Start", method.GoName, "(ctx, reqData)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	renderDispatcherStreamLoadSession(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", `errors.New("rpccgo: connect message stream handle is invalid")`, "handle")
	g.P("for {")
	g.P("respData, err := session.Recv(ctx)")
	g.P("if err != nil {")
	renderDispatcherStreamTakeSession(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", `errors.New("rpccgo: connect message stream handle is invalid")`, "handle", "terminal")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return terminal.Done(ctx)")
	g.P("}")
	g.P("_ = terminal.Cancel(ctx)")
	g.P("return err")
	g.P("}")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	renderDispatcherStreamCancelIfPresent(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", "handle", "ctx")
	g.P(`return fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("if err := stream.Send(resp); err != nil {")
	renderDispatcherStreamCancelIfPresent(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", "handle", "ctx")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P()
}

func renderConnectBidiStreamingImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeType := service.GoName + "CGOMessageClientBridge"
	g.P("func ", connectImplementationName(service, method), "(ctx context.Context, stream *connect.BidiStream[", reqType, ", ", respType, "]) error {")
	g.P("bridge := New", bridgeType, "()")
	g.P("handle, err := bridge.Start", method.GoName, "(ctx)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	renderDispatcherStreamLoadSession(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", `errors.New("rpccgo: connect message stream handle is invalid")`, "handle")
	renderDispatcherStreamFinishOnce(g, service.GoName, service.GoName+method.GoName+"MessageStreamSession", `errors.New("rpccgo: connect message stream handle is invalid")`, "handle", "ctx")
	g.P("receiveErrCh := make(chan error, 1)")
	g.P("sendErrCh := make(chan error, 1)")
	g.P("go func() {")
	g.P("for {")
	g.P("req, err := stream.Receive()")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("receiveErrCh <- session.CloseSend(ctx)")
	g.P("return")
	g.P("}")
	g.P("receiveErrCh <- err")
	g.P("return")
	g.P("}")
	g.P("reqData, err := proto.Marshal(req)")
	g.P("if err != nil {")
	g.P(`receiveErrCh <- fmt.Errorf("rpccgo: connect bidi request protobuf marshal failed: %w", err)`)
	g.P("return")
	g.P("}")
	g.P("if err := session.Send(ctx, reqData); err != nil {")
	g.P("receiveErrCh <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("go func() {")
	g.P("for {")
	g.P("respData, err := session.Recv(ctx)")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("sendErrCh <- finish(true)")
	g.P("return")
	g.P("}")
	g.P("sendErrCh <- err")
	g.P("return")
	g.P("}")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`sendErrCh <- fmt.Errorf("rpccgo: connect bidi response protobuf unmarshal failed: %w", err)`)
	g.P("return")
	g.P("}")
	g.P("if err := stream.Send(resp); err != nil {")
	g.P("sendErrCh <- err")
	g.P("return")
	g.P("}")
	g.P("}")
	g.P("}()")
	g.P("for receiveErrCh != nil || sendErrCh != nil {")
	g.P("select {")
	g.P("case err := <-receiveErrCh:")
	g.P("receiveErrCh = nil")
	g.P("if err != nil {")
	g.P("_ = finish(false)")
	g.P("return err")
	g.P("}")
	g.P("case err := <-sendErrCh:")
	g.P("sendErrCh = nil")
	g.P("if err != nil {")
	g.P("_ = finish(false)")
	g.P("return err")
	g.P("}")
	g.P("if receiveErrCh == nil {")
	g.P("return nil")
	g.P("}")
	g.P("}")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

func serviceHasStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			return true
		}
	}
	return false
}

func serviceHasBidiStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindBidiStreaming {
			return true
		}
	}
	return false
}

func connectProcedureConstName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "ConnectProcedure"
}

func connectProcedureHandlerName(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName + "ConnectHandler"
}

func connectImplementationName(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + "Connect" + method.GoName
}

func qualifiedMethodType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}
