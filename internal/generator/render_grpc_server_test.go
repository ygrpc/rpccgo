package generator

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRenderGRPCServerFileDoesNotImportIOWithoutClientStreaming(t *testing.T) {
	file := completeServicePlanTestFile()
	file.SourceCodeInfo = completeServicePlanServiceComments([]string{
		"",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-grpc\n",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: msg-grpc|native\n",
		"@rpccgo: native\n",
	})
	allService := file.Service[5]
	allService.Method = []*descriptorpb.MethodDescriptorProto{
		allService.Method[0],
		allService.Method[2],
		allService.Method[3],
	}
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const grpcFile = "test/v1/complete_service_plan.all_service.server.grpc.rpccgo.go"
	assertGeneratedContentContains(t, plugin, grpcFile, `rpcruntime "rpccgo/rpcruntime"`)
	assertGeneratedFileContentDoesNotContain(t, plugin, grpcFile, `io "io"`)
}

func TestRenderGRPCServerFileEmitsServiceDesc(t *testing.T) {
	file := completeServicePlanTestFile()
	file.SourceCodeInfo = completeServicePlanServiceComments([]string{
		"",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-grpc\n",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: msg-grpc|native\n",
		"@rpccgo: native\n",
	})
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
		"respData, err := InvokeAllServiceMessageUnary(ctx, reqData)",
		"func allServiceClientStreamGRPC(stream grpc.ClientStreamingServer[AllRequest, AllReply]) error {",
		"handle, err := StartAllServiceMessageClientStream(stream.Context())",
		"lifecycle := NewAllServiceClientStreamMessageStream(handle)",
		"respData, err := lifecycle.Finish(stream.Context())",
		"func allServiceServerStreamGRPC(req *AllRequest, stream grpc.ServerStreamingServer[AllReply]) error {",
		"handle, err := StartAllServiceMessageServerStream(stream.Context(), reqData)",
		"lifecycle := NewAllServiceServerStreamMessageStream(handle)",
		"return lifecycle.Recv(stream.Context())",
		"return lifecycle.Done(stream.Context())",
		"func allServiceBidiStreamGRPC(stream grpc.BidiStreamingServer[AllRequest, AllReply]) error {",
		"handle, err := StartAllServiceMessageBidiStream(stream.Context())",
		"lifecycle := NewAllServiceBidiStreamMessageStream(handle)",
		"return lifecycle.CloseSend(stream.Context())",
		"return allServiceClientStreamGRPC(&grpc.GenericServerStream[AllRequest, AllReply]{ServerStream: stream})",
	} {
		assertGeneratedContentContains(t, plugin, grpcFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, grpcFile,
		"connectrpc.com/connect",
		".remote.",
		"panic(",
		"rpcruntime.DispatcherStreamSend[",
		"rpcruntime.DispatcherStreamReceive[",
		"rpcruntime.DispatcherStreamFinish[",
		"rpcruntime.DispatcherStreamDone[",
		"rpcruntime.DispatcherStreamCancel[",
		"rpcruntime.DispatcherStreamCloseSend[",
	)
}
