package generator

import "testing"

func TestRenderRuntimeEmitsGRPCRemoteClientActiveServer(t *testing.T) {
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

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		`grpc "google.golang.org/grpc"`,
		`proto "google.golang.org/protobuf/proto"`,
		"func RegisterAllServiceGRPCRemoteServer(client AllServiceClient) (rpcruntime.AdapterSnapshot[AllServiceClient], error) {",
		"snapshot, err := allServiceActiveSlot.Store(rpcruntime.ServerKindGRPCRemote, rpcruntime.ServerContractMessage, client)",
		"return rpcruntime.AdapterSnapshot[AllServiceClient]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: client}, nil",
		"case rpcruntime.ServerKindGRPCRemote:",
		"client, ok := snapshot.Adapter.(AllServiceClient)",
		"messageResp, err := client.Unary(ctx, messageReq)",
		"return newallServiceClientStreamGRPCRemoteMessageStreamSession(ctx, client)",
		"streamCtx, cancel := context.WithCancel(ctx)",
		"stream, err := client.ClientStream(streamCtx)",
		"stream grpc.ClientStreamingClient[AllRequest, AllReply]",
		"cancel context.CancelFunc",
		"s.cancel()",
		"return s.stream.CloseSend()",
		"return newallServiceServerStreamGRPCRemoteMessageStreamSession(ctx, client, req)",
		"stream, err := client.ServerStream(streamCtx, request)",
		"stream grpc.ServerStreamingClient[AllReply]",
		"return newallServiceBidiStreamGRPCRemoteMessageStreamSession(ctx, client)",
		"stream, err := client.BidiStream(streamCtx)",
		"stream grpc.BidiStreamingClient[AllRequest, AllReply]",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile, "type AllServiceGRPCRemoteServer struct", "NewAllServiceGRPCRemoteServer", "panic(", "ClientModel", "ClientConnInterface", ".Invoke(", ".NewStream(", "GRPCFullMethodName")
}

func TestRenderRuntimeGRPCRemoteOmitsGRPCImportForUnaryOnlyService(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-grpc\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const runtimeFile = "test/v1/greeter.greeter.runtime.rpccgo.go"
	assertGeneratedContentContains(t, plugin, runtimeFile, "func RegisterGreeterGRPCRemoteServer(client GreeterClient) (rpcruntime.AdapterSnapshot[GreeterClient], error)")
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile, `grpc "google.golang.org/grpc"`)
}
