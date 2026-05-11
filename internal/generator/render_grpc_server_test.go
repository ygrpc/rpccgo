package generator

import "testing"

func TestRenderGRPCServerFileEmitsServiceDesc(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const grpcFile = "test/v1/complete_service_plan.all_service.server.grpc.rpccgo.go"
	for _, fragment := range []string{
		`grpc "google.golang.org/grpc"`,
		`codes "google.golang.org/grpc/codes"`,
		`status "google.golang.org/grpc/status"`,
		`proto "google.golang.org/protobuf/proto"`,
		"func RegisterAllServiceGRPCServer(registrar grpc.ServiceRegistrar) error {",
		"registrar.RegisterService(&AllServiceGRPCServiceDesc, struct{}{})",
		"type AllServiceGRPCHandler interface{}",
		"var AllServiceGRPCServiceDesc = grpc.ServiceDesc{",
		`ServiceName: "test.v1.AllService"`,
		"HandlerType: (*AllServiceGRPCHandler)(nil),",
		"Methods: []grpc.MethodDesc{",
		"MethodName: \"Unary\"",
		"Streams: []grpc.StreamDesc{",
		"StreamName:    \"ClientStream\"",
		"StreamName:    \"ServerStream\"",
		"StreamName:    \"BidiStream\"",
		"ClientStreams: true",
		"ServerStreams: true",
		"func allServiceUnaryGRPC(ctx context.Context, req *AllRequest) (*AllReply, error) {",
		"respData, err := NewAllServiceCGOMessageClientBridge().Unary(ctx, reqData)",
		"func allServiceClientStreamGRPC(stream grpc.ClientStreamingServer[AllRequest, AllReply]) error {",
		"handle, err := bridge.StartClientStream(stream.Context())",
		"func allServiceServerStreamGRPC(req *AllRequest, stream grpc.ServerStreamingServer[AllReply]) error {",
		"handle, err := bridge.StartServerStream(stream.Context(), reqData)",
		"func allServiceBidiStreamGRPC(stream grpc.BidiStreamingServer[AllRequest, AllReply]) error {",
		"handle, err := bridge.StartBidiStream(stream.Context())",
		"return allServiceClientStreamGRPC(&grpc.GenericServerStream[AllRequest, AllReply]{ServerStream: stream})",
	} {
		assertGeneratedContentContains(t, plugin, grpcFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, grpcFile, "connectrpc.com/connect", ".remote.", "panic(")
}
