package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderGRPCRemoteFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if serviceHasStreamingMethod(service) {
		g.P(`io "io"`)
	}
	g.P(`grpc "google.golang.org/grpc"`)
	g.P(`codes "google.golang.org/grpc/codes"`)
	g.P(`status "google.golang.org/grpc/status"`)
	g.P(`proto "google.golang.org/protobuf/proto"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	for _, method := range service.Methods {
		if method.Streaming == StreamingKindUnary {
			continue
		}
		g.P("const ", grpcFullMethodConstName(service, method), ` = "/`, service.FullName, `/`, method.Name, `"`)
	}
	g.P()

	typeName := service.GoName + "GRPCRemoteServer"
	g.P("type ", typeName, " struct {")
	g.P("conn grpc.ClientConnInterface")
	g.P("}")
	g.P()

	g.P("func New", typeName, "(conn grpc.ClientConnInterface) (*", typeName, ", error) {")
	g.P("if conn == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote client connection is nil")`)
	g.P("}")
	g.P("return &", typeName, "{conn: conn}, nil")
	g.P("}")
	g.P()

	g.P("func Register", typeName, "(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[", service.GoName, "MessageAdapter], error) {")
	g.P("adapter, err := New", typeName, "(conn)")
	g.P("if err != nil {")
	g.P("return rpcruntime.AdapterSnapshot[", service.GoName, "MessageAdapter]{}, err")
	g.P("}")
	g.P("return Register", service.GoName, "CGOMessageActiveServer(rpcruntime.ServerKindGRPCRemote, adapter)")
	g.P("}")
	g.P()

	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderGRPCRemoteUnary(g, service, method, typeName)
		case StreamingKindClientStreaming:
			renderGRPCRemoteClientStreaming(g, service, method, typeName)
		case StreamingKindServerStreaming:
			renderGRPCRemoteServerStreaming(g, service, method, typeName)
		case StreamingKindBidiStreaming:
			renderGRPCRemoteBidiStreaming(g, service, method, typeName)
		}
	}

	return nil
}

func renderGRPCRemoteUnary(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, typeName string) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)

	g.P("func (s *", typeName, ") ", method.GoName, "Message(ctx context.Context, req []byte) ([]byte, error) {")
	g.P("if s == nil || s.conn == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote server is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote request protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("response := new(", respType, ")")
	g.P("err := s.conn.Invoke(ctx, ", grpcFullMethodConstName(service, method), ", request, response)")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("respData, err := proto.Marshal(response)")
	g.P("if err != nil {")
	g.P(`return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote response protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("return respData, nil")
	g.P("}")
	g.P()
}

func renderGRPCRemoteClientStreaming(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, typeName string) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	sessionType := service.GoName + method.GoName + "GRPCRemoteClientStreamSession"

	g.P("func (s *", typeName, ") Start", method.GoName, "Message(ctx context.Context) (", service.GoName, method.GoName, "MessageStreamSession, error) {")
	g.P("if s == nil || s.conn == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote server is nil")`)
	g.P("}")
	g.P("stream, err := s.conn.NewStream(ctx, &grpc.StreamDesc{ClientStreams: true}, ", grpcFullMethodConstName(service, method), ")")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", sessionType, "{stream: &grpc.GenericClientStream[", reqType, ", ", respType, "]{ClientStream: stream}}, nil")
	g.P("}")
	g.P()

	g.P("type ", sessionType, " struct {")
	g.P("stream *grpc.GenericClientStream[", reqType, ", ", respType, "]")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Send(ctx context.Context, req []byte) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return errors.New("rpccgo: grpc remote client stream is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote stream request protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("return s.stream.Send(request)")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Finish(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote client stream is nil")`)
	g.P("}")
	g.P("response, err := s.stream.CloseAndRecv()")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("respData, err := proto.Marshal(response)")
	g.P("if err != nil {")
	g.P(`return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote stream response protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("return respData, nil")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
}

func renderGRPCRemoteServerStreaming(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, typeName string) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	sessionType := service.GoName + method.GoName + "GRPCRemoteServerStreamSession"

	g.P("func (s *", typeName, ") Start", method.GoName, "Message(ctx context.Context, req []byte) (", service.GoName, method.GoName, "MessageStreamSession, error) {")
	g.P("if s == nil || s.conn == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote server is nil")`)
	g.P("}")
	g.P("stream, err := s.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, ", grpcFullMethodConstName(service, method), ")")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote request protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("client := &grpc.GenericClientStream[", reqType, ", ", respType, "]{ClientStream: stream}")
	g.P("if err := client.Send(request); err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("if err := client.CloseSend(); err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", sessionType, "{stream: client}, nil")
	g.P("}")
	g.P()

	g.P("type ", sessionType, " struct {")
	g.P("stream *grpc.GenericClientStream[", reqType, ", ", respType, "]")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote server stream is nil")`)
	g.P("}")
	g.P("response, err := s.stream.Recv()")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("return nil, err")
	g.P("}")
	g.P("respData, err := proto.Marshal(response)")
	g.P("if err != nil {")
	g.P(`return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote stream response protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("return respData, nil")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Done(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
}

func renderGRPCRemoteBidiStreaming(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, typeName string) {
	reqType := qualifiedMethodType(g, method.Request)
	respType := qualifiedMethodType(g, method.Response)
	sessionType := service.GoName + method.GoName + "GRPCRemoteBidiStreamSession"

	g.P("func (s *", typeName, ") Start", method.GoName, "Message(ctx context.Context) (", service.GoName, method.GoName, "MessageStreamSession, error) {")
	g.P("if s == nil || s.conn == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote server is nil")`)
	g.P("}")
	g.P("stream, err := s.conn.NewStream(ctx, &grpc.StreamDesc{ClientStreams: true, ServerStreams: true}, ", grpcFullMethodConstName(service, method), ")")
	g.P("if err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return &", sessionType, "{stream: &grpc.GenericClientStream[", reqType, ", ", respType, "]{ClientStream: stream}}, nil")
	g.P("}")
	g.P()

	g.P("type ", sessionType, " struct {")
	g.P("stream *grpc.GenericClientStream[", reqType, ", ", respType, "]")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Send(ctx context.Context, req []byte) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return errors.New("rpccgo: grpc remote bidi stream is nil")`)
	g.P("}")
	g.P("request := new(", reqType, ")")
	g.P("if err := proto.Unmarshal(req, request); err != nil {")
	g.P(`return status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote bidi request protobuf unmarshal failed: %v", err)`)
	g.P("}")
	g.P("return s.stream.Send(request)")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Recv(ctx context.Context) ([]byte, error) {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P(`return nil, errors.New("rpccgo: grpc remote bidi stream is nil")`)
	g.P("}")
	g.P("response, err := s.stream.Recv()")
	g.P("if err != nil {")
	g.P("if errors.Is(err, io.EOF) {")
	g.P("return nil, io.EOF")
	g.P("}")
	g.P("return nil, err")
	g.P("}")
	g.P("respData, err := proto.Marshal(response)")
	g.P("if err != nil {")
	g.P(`return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote bidi response protobuf marshal failed: %v", err)`)
	g.P("}")
	g.P("return respData, nil")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") CloseSend(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Done(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("func (s *", sessionType, ") Cancel(ctx context.Context) error {")
	g.P("_ = ctx")
	g.P("if s == nil || s.stream == nil {")
	g.P("return nil")
	g.P("}")
	g.P("return s.stream.CloseSend()")
	g.P("}")
	g.P()
}
