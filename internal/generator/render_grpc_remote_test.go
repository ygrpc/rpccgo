package generator

import "testing"

func TestRenderGRPCRemoteFileEmitsMessageAdapter(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const remoteFile = "test/v1/stage1_acceptance.all_service.remote.grpc.rpccgo.go"
	for _, fragment := range []string{
		`grpc "google.golang.org/grpc"`,
		`proto "google.golang.org/protobuf/proto"`,
		"type AllServiceGRPCRemoteServer struct {",
		"conn grpc.ClientConnInterface",
		"func NewAllServiceGRPCRemoteServer(conn grpc.ClientConnInterface) (*AllServiceGRPCRemoteServer, error) {",
		"func RegisterAllServiceGRPCRemoteServer(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"return RegisterAllServiceCGOMessageActiveServer(rpcruntime.ServerKindGRPCRemote, adapter)",
		"func (s *AllServiceGRPCRemoteServer) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"err := s.conn.Invoke(ctx, AllServiceUnaryGRPCFullMethodName, request, response)",
		"func (s *AllServiceGRPCRemoteServer) StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error) {",
		"streamCtx, cancel := context.WithCancel(ctx)",
		"stream, err := s.conn.NewStream(streamCtx, &grpc.StreamDesc{ClientStreams: true}, AllServiceClientStreamGRPCFullMethodName)",
		"cancel context.CancelFunc",
		"cancel: cancel",
		"s.cancel()",
		"return s.stream.CloseSend()",
		"func (s *AllServiceGRPCRemoteServer) StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error) {",
		"stream, err := s.conn.NewStream(streamCtx, &grpc.StreamDesc{ServerStreams: true}, AllServiceServerStreamGRPCFullMethodName)",
		"func (s *AllServiceGRPCRemoteServer) StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error) {",
		"stream, err := s.conn.NewStream(streamCtx, &grpc.StreamDesc{ClientStreams: true, ServerStreams: true}, AllServiceBidiStreamGRPCFullMethodName)",
	} {
		assertGeneratedContentContains(t, plugin, remoteFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, remoteFile, "connectrpc.com/connect", "panic(", "ClientModel")
}
