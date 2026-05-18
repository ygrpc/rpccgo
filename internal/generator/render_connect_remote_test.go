package generator

import "testing"

func TestRenderConnectRemoteFileEmitsMessageAdapter(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const remoteFile = "test/v1/complete_service_plan.all_service.remote.connect.rpccgo.go"
	for _, fragment := range []string{
		`connect "connectrpc.com/connect"`,
		`proto "google.golang.org/protobuf/proto"`,
		"type AllServiceConnectRemoteServer struct {",
		"client AllServiceClient",
		"func NewAllServiceConnectRemoteServer(client AllServiceClient) (*AllServiceConnectRemoteServer, error) {",
		`return nil, errors.New("rpccgo: connect remote client is nil")`,
		"func RegisterAllServiceConnectRemoteServer(client AllServiceClient) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"return RegisterAllServiceCGOMessageActiveServer(rpcruntime.ServerKindConnectRemote, adapter)",
		"func (s *AllServiceConnectRemoteServer) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"resp, err := s.client.Unary(ctx, request)",
		"func (s *AllServiceConnectRemoteServer) StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error) {",
		"streamCtx, cancel := context.WithCancel(ctx)",
		"stream, err := s.client.ClientStream(streamCtx)",
		"stream *connect.ClientStreamForClientSimple[AllRequest, AllReply]",
		"cancel: cancel",
		"cancel context.CancelFunc",
		"s.cancel()",
		"defer s.cancel()",
		"func (s *AllServiceConnectRemoteServer) StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error) {",
		"stream, err := s.client.ServerStream(streamCtx, request)",
		"func (s *AllServiceConnectRemoteServer) StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error) {",
		"stream, err := s.client.BidiStream(streamCtx)",
		"stream *connect.BidiStreamForClientSimple[AllRequest, AllReply]",
		"return s.stream.CloseRequest()",
	} {
		assertGeneratedContentContains(t, plugin, remoteFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, remoteFile, `http "net/http"`, `grpc "google.golang.org/grpc"`, "panic(", "ClientModel", "closeConnectRemoteConn", "connect.NewClient", "CallUnary", "CallClientStream", "CallBidiStream")
}
