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
		`http "net/http"`,
		`proto "google.golang.org/protobuf/proto"`,
		"type AllServiceConnectRemoteServer struct {",
		"func NewAllServiceConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (*AllServiceConnectRemoteServer, error) {",
		"func RegisterAllServiceConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"return RegisterAllServiceCGOMessageActiveServer(rpcruntime.ServerKindConnectRemote, adapter)",
		"func (s *AllServiceConnectRemoteServer) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"resp, err := s.unary.CallUnary(ctx, connect.NewRequest(request))",
		"func (s *AllServiceConnectRemoteServer) StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error) {",
		"streamCtx, cancel := context.WithCancel(ctx)",
		"stream := s.clientStream.CallClientStream(streamCtx)",
		"cancel: cancel",
		"cancel context.CancelFunc",
		"s.cancel()",
		"defer s.cancel()",
		"func (s *AllServiceConnectRemoteServer) StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error) {",
		"stream, err := s.serverStream.CallServerStream(streamCtx, connect.NewRequest(request))",
		"func (s *AllServiceConnectRemoteServer) StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error) {",
		"stream := s.bidiStream.CallBidiStream(streamCtx)",
		"conn, err := s.stream.Conn()",
		"return conn.CloseRequest()",
	} {
		assertGeneratedContentContains(t, plugin, remoteFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, remoteFile, `grpc "google.golang.org/grpc"`, "panic(", "ClientModel", "closeConnectRemoteConn")
}
