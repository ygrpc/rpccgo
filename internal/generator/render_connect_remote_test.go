package generator

import "testing"

func TestRenderRuntimeEmitsConnectRemoteClientActiveServer(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		`connect "connectrpc.com/connect"`,
		`proto "google.golang.org/protobuf/proto"`,
		"func RegisterAllServiceConnectRemoteServer(client AllServiceClient) (rpcruntime.AdapterSnapshot[AllServiceClient], error) {",
		"snapshot, err := allServiceActiveSlot.Store(rpcruntime.ServerKindConnectRemote, rpcruntime.ServerContractMessage, client)",
		"return rpcruntime.AdapterSnapshot[AllServiceClient]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: client}, nil",
		"case rpcruntime.ServerKindConnectRemote:",
		"client, ok := snapshot.Adapter.(AllServiceClient)",
		"messageResp, err := client.Unary(ctx, messageReq)",
		"return newallServiceClientStreamConnectRemoteMessageStreamSession(ctx, client)",
		"streamCtx, cancel := context.WithCancel(ctx)",
		"stream, err := client.ClientStream(streamCtx)",
		"stream *connect.ClientStreamForClientSimple[AllRequest, AllReply]",
		"cancel context.CancelFunc",
		"s.cancel()",
		"defer s.cancel()",
		"return newallServiceServerStreamConnectRemoteMessageStreamSession(ctx, client, req)",
		"stream, err := client.ServerStream(streamCtx, request)",
		"return newallServiceBidiStreamConnectRemoteMessageStreamSession(ctx, client)",
		"stream, err := client.BidiStream(streamCtx)",
		"stream *connect.BidiStreamForClientSimple[AllRequest, AllReply]",
		"return s.stream.CloseRequest()",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile, "type AllServiceConnectRemoteServer struct", "NewAllServiceConnectRemoteServer", `http "net/http"`, "panic(", "ClientModel", "closeConnectRemoteConn", "connect.NewClient", "CallUnary", "CallClientStream", "CallBidiStream")
}
