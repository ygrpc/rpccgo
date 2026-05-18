package generator

import "testing"

func TestRenderGRPCRemoteFileEmitsMessageAdapter(t *testing.T) {
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

	const remoteFile = "test/v1/complete_service_plan.all_service.remote.grpc.rpccgo.go"
	for _, fragment := range []string{
		`grpc "google.golang.org/grpc"`,
		`proto "google.golang.org/protobuf/proto"`,
		"type AllServiceGRPCRemoteServer struct {",
		"client AllServiceClient",
		"func NewAllServiceGRPCRemoteServer(client AllServiceClient) (*AllServiceGRPCRemoteServer, error) {",
		`return nil, errors.New("rpccgo: grpc remote client is nil")`,
		"func RegisterAllServiceGRPCRemoteServer(client AllServiceClient) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"return RegisterAllServiceCGOMessageActiveServer(rpcruntime.ServerKindGRPCRemote, adapter)",
		"func (s *AllServiceGRPCRemoteServer) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"response, err := s.client.Unary(ctx, request)",
		"func (s *AllServiceGRPCRemoteServer) StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error) {",
		"streamCtx, cancel := context.WithCancel(ctx)",
		"stream, err := s.client.ClientStream(streamCtx)",
		"stream grpc.ClientStreamingClient[AllRequest, AllReply]",
		"cancel context.CancelFunc",
		"cancel: cancel",
		"s.cancel()",
		"return s.stream.CloseSend()",
		"func (s *AllServiceGRPCRemoteServer) StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error) {",
		"stream, err := s.client.ServerStream(streamCtx, request)",
		"stream grpc.ServerStreamingClient[AllReply]",
		"func (s *AllServiceGRPCRemoteServer) StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error) {",
		"stream, err := s.client.BidiStream(streamCtx)",
		"stream grpc.BidiStreamingClient[AllRequest, AllReply]",
	} {
		assertGeneratedContentContains(t, plugin, remoteFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, remoteFile, "connectrpc.com/connect", "panic(", "ClientModel", "ClientConnInterface", ".Invoke(", ".NewStream(", "GRPCFullMethodName")
}
