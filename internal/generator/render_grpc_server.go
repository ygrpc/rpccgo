package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderGRPCServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if serviceHasClientStreamingMethod(service) {
		g.P(`io "io"`)
	}
	if serviceHasStreamingMethod(service) {
		g.P(`rpcruntime "rpccgo/rpcruntime"`)
	}
	g.P(`grpc "google.golang.org/grpc"`)
	g.P(`codes "google.golang.org/grpc/codes"`)
	g.P(`status "google.golang.org/grpc/status"`)
	g.P(`proto "google.golang.org/protobuf/proto"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	g.P("type ", service.GoName, "GRPCHandler interface{}")
	g.P()
	g.P("func Register", service.GoName, "GRPCServer(registrar grpc.ServiceRegistrar) error {")
	g.P("if registrar == nil {")
	g.P(`return errors.New("rpccgo: grpc registrar is nil")`)
	g.P("}")
	g.P("registrar.RegisterService(&", service.GoName, "GRPCServiceDesc, struct{}{})")
	g.P("return nil")
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		if method.Streaming == StreamingKindUnary {
			g.P("const ", grpcFullMethodConstName(service, method), ` = "/`, service.FullName, `/`, method.Name, `"`)
		}
	}
	g.P()

	g.P("var ", service.GoName, "GRPCServiceDesc = grpc.ServiceDesc{")
	g.P(`ServiceName: "`, service.FullName, `",`)
	g.P("HandlerType: (*", service.GoName, "GRPCHandler)(nil),")
	g.P("Methods: []grpc.MethodDesc{")
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			continue
		}
		g.P("{")
		g.P(`MethodName: "`, method.Name, `",`)
		g.P("Handler: ", grpcUnaryHandlerName(service, method), ",")
		g.P("},")
	}
	g.P("},")
	g.P("Streams: []grpc.StreamDesc{")
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindUnary {
			continue
		}
		g.P("{")
		g.P(`StreamName: "`, method.Name, `",`)
		g.P("Handler: ", grpcStreamHandlerName(service, method), ",")
		if method.Streaming == StreamingKindClientStreaming || method.Streaming == StreamingKindBidiStreaming {
			g.P("ClientStreams: true,")
		}
		if method.Streaming == StreamingKindServerStreaming || method.Streaming == StreamingKindBidiStreaming {
			g.P("ServerStreams: true,")
		}
		g.P("},")
	}
	g.P("},")
	g.P(`Metadata: "`, plan.ProtoPath, `",`)
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderGRPCUnaryImplementation(g, service, method)
		case StreamingKindClientStreaming:
			renderGRPCClientStreamingImplementation(g, service, method)
		case StreamingKindServerStreaming:
			renderGRPCServerStreamingImplementation(g, service, method)
		case StreamingKindBidiStreaming:
			renderGRPCBidiStreamingImplementation(g, service, method)
		}
	}

	return nil
}

func renderGRPCUnaryImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeName := service.GoName + "CGOMessageClientBridge"
	g.P("func ", grpcUnaryHandlerName(service, method), "(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {")
	g.P("in := new(", reqType, ")")
	g.P("if err := dec(in); err != nil {")
	g.P(`return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc request decode failed: %v", err)`)
	g.P("}")
	g.P("handler := func(ctx context.Context, req any) (any, error) {")
	g.P("typed, ok := req.(*", reqType, ")")
	g.P("if !ok || typed == nil {")
	g.P(`return nil, status.Error(codes.InvalidArgument, "rpccgo: grpc request type mismatch")`)
	g.P("}")
	g.P("return ", grpcImplementationName(service, method), "(ctx, typed)")
	g.P("}")
	g.P("if interceptor == nil {")
	g.P("return handler(ctx, in)")
	g.P("}")
	g.P("info := &grpc.UnaryServerInfo{Server: srv, FullMethod: ", grpcFullMethodConstName(service, method), "}")
	g.P("return interceptor(ctx, in, info, handler)")
	g.P("}")
	g.P()
	g.P("func ", grpcImplementationName(service, method), "(ctx context.Context, req *", reqType, ") (*", respType, ", error) {")
	g.P("if req == nil {")
	g.P(`return nil, status.Error(codes.InvalidArgument, "rpccgo: grpc request is nil")`)
	g.P("}")
	g.P("reqData, err := proto.Marshal(req)")
	g.P("if err != nil {")
	g.P(`return nil, status.Errorf(codes.Internal, "rpccgo: grpc request protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("respData, err := New", bridgeName, "().", method.GoName, "(ctx, reqData)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`return nil, status.Errorf(codes.Internal, "rpccgo: grpc response protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("return resp, nil")
	g.P("}")
	g.P()
}

func renderGRPCClientStreamingImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeType := service.GoName + "CGOMessageClientBridge"
	g.P("func ", grpcStreamHandlerName(service, method), "(srv any, stream grpc.ServerStream) error {")
	g.P("return ", grpcImplementationName(service, method), "(&grpc.GenericServerStream[", reqType, ", ", respType, "]{ServerStream: stream})")
	g.P("}")
	g.P()
	g.P("func ", grpcImplementationName(service, method), "(stream grpc.ClientStreamingServer[", reqType, ", ", respType, "]) error {")
	g.P("bridge := New", bridgeType, "()")
	g.P("handle, err := bridge.Start", method.GoName, "(stream.Context())")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("lifecycle := New", service.GoName, method.GoName, "MessageStream(handle)")
	g.P("for {")
	g.P("req, err := stream.Recv()")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("break")
	g.P("}")
	g.P("_ = lifecycle.Cancel(stream.Context())")
	g.P("return err")
	g.P("}")
	g.P("reqData, err := proto.Marshal(req)")
	g.P("if err != nil {")
	g.P("_ = lifecycle.Cancel(stream.Context())")
	g.P(`return status.Errorf(codes.Internal, "rpccgo: grpc stream request protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("if err := lifecycle.Send(stream.Context(), reqData); err != nil {")
	g.P("_ = lifecycle.Cancel(stream.Context())")
	g.P("return err")
	g.P("}")
	g.P("}")
	g.P("respData, err := lifecycle.Finish(stream.Context())")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`return status.Errorf(codes.Internal, "rpccgo: grpc stream response protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("return stream.SendAndClose(resp)")
	g.P("}")
	g.P()
}

func renderGRPCServerStreamingImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeType := service.GoName + "CGOMessageClientBridge"
	g.P("func ", grpcStreamHandlerName(service, method), "(srv any, stream grpc.ServerStream) error {")
	g.P("m := new(", reqType, ")")
	g.P("if err := stream.RecvMsg(m); err != nil {")
	g.P("return err")
	g.P("}")
	g.P("return ", grpcImplementationName(service, method), "(m, &grpc.GenericServerStream[", reqType, ", ", respType, "]{ServerStream: stream})")
	g.P("}")
	g.P()
	g.P("func ", grpcImplementationName(service, method), "(req *", reqType, ", stream grpc.ServerStreamingServer[", respType, "]) error {")
	g.P("if req == nil {")
	g.P(`return status.Error(codes.InvalidArgument, "rpccgo: grpc request is nil")`)
	g.P("}")
	g.P("reqData, err := proto.Marshal(req)")
	g.P("if err != nil {")
	g.P(`return status.Errorf(codes.Internal, "rpccgo: grpc stream request protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("bridge := New", bridgeType, "()")
	g.P("handle, err := bridge.Start", method.GoName, "(stream.Context(), reqData)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("lifecycle := New", service.GoName, method.GoName, "MessageStream(handle)")
	g.P("return rpcruntime.RunServerStream(")
	g.P("func() ([]byte, error) {")
	g.P("return lifecycle.Recv(stream.Context())")
	g.P("},")
	g.P("func(respData []byte) error {")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`return status.Errorf(codes.Internal, "rpccgo: grpc stream response protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("return stream.Send(resp)")
	g.P("},")
	g.P("func() error {")
	g.P("return lifecycle.Done(stream.Context())")
	g.P("},")
	g.P("func() error {")
	g.P("return lifecycle.Cancel(stream.Context())")
	g.P("},")
	g.P(")")
	g.P("}")
	g.P()
}

func renderGRPCBidiStreamingImplementation(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	bridgeType := service.GoName + "CGOMessageClientBridge"
	g.P("func ", grpcStreamHandlerName(service, method), "(srv any, stream grpc.ServerStream) error {")
	g.P("return ", grpcImplementationName(service, method), "(&grpc.GenericServerStream[", reqType, ", ", respType, "]{ServerStream: stream})")
	g.P("}")
	g.P()
	g.P("func ", grpcImplementationName(service, method), "(stream grpc.BidiStreamingServer[", reqType, ", ", respType, "]) error {")
	g.P("bridge := New", bridgeType, "()")
	g.P("handle, err := bridge.Start", method.GoName, "(stream.Context())")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("lifecycle := New", service.GoName, method.GoName, "MessageStream(handle)")
	g.P("return rpcruntime.RunBidiStream(")
	g.P("func() (*", reqType, ", error) {")
	g.P("return stream.Recv()")
	g.P("},")
	g.P("func(req *", reqType, ") error {")
	g.P("reqData, err := proto.Marshal(req)")
	g.P("if err != nil {")
	g.P(`return status.Errorf(codes.Internal, "rpccgo: grpc bidi request protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("return lifecycle.Send(stream.Context(), reqData)")
	g.P("},")
	g.P("func() error {")
	g.P("return lifecycle.CloseSend(stream.Context())")
	g.P("},")
	g.P("func() ([]byte, error) {")
	g.P("return lifecycle.Recv(stream.Context())")
	g.P("},")
	g.P("func(respData []byte) error {")
	g.P("resp := new(", respType, ")")
	g.P("if err := proto.Unmarshal(respData, resp); err != nil {")
	g.P(`return status.Errorf(codes.Internal, "rpccgo: grpc bidi response protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("return stream.Send(resp)")
	g.P("},")
	g.P("func() error {")
	g.P("return lifecycle.Done(stream.Context())")
	g.P("},")
	g.P("func() error {")
	g.P("return lifecycle.Cancel(stream.Context())")
	g.P("},")
	g.P(")")
	g.P("}")
	g.P()
}

func grpcFullMethodConstName(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "GRPCFullMethodName"
}

func grpcUnaryHandlerName(service ServicePlan, method MethodPlan) string {
	return "_" + service.GoName + "_" + method.GoName + "_GRPC_Handler"
}

func grpcStreamHandlerName(service ServicePlan, method MethodPlan) string {
	return "_" + service.GoName + "_" + method.GoName + "_GRPC_StreamHandler"
}

func grpcImplementationName(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName + "GRPC"
}
