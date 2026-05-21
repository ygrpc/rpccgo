package generator

import "testing"

func TestRenderConnectServerFileEmitsHandlers(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const connectFile = "test/v1/complete_service_plan.all_service.server.connect.rpccgo.go"
	for _, fragment := range []string{
		`connect "connectrpc.com/connect"`,
		`http "net/http"`,
		`proto "google.golang.org/protobuf/proto"`,
		"const AllServiceConnectServiceName = \"test.v1.AllService\"",
		"const AllServiceUnaryConnectProcedure = \"/test.v1.AllService/Unary\"",
		"func NewAllServiceConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {",
		"connect.NewUnaryHandler(",
		"connect.NewClientStreamHandler(",
		"connect.NewServerStreamHandler(",
		"connect.NewBidiStreamHandler(",
		"func allServiceConnectUnary(ctx context.Context, req *connect.Request[AllRequest]) (*connect.Response[AllReply], error) {",
		"respData, err := NewAllServiceCGOMessageClientBridge().Unary(ctx, reqData)",
		"func allServiceConnectClientStream(ctx context.Context, stream *connect.ClientStream[AllRequest]) (*connect.Response[AllReply], error) {",
		"handle, err := bridge.StartClientStream(ctx)",
		"lifecycle := NewAllServiceClientStreamMessageStream(handle)",
		"respData, err := lifecycle.Finish(ctx)",
		"func allServiceConnectServerStream(ctx context.Context, req *connect.Request[AllRequest], stream *connect.ServerStream[AllReply]) error {",
		"handle, err := bridge.StartServerStream(ctx, reqData)",
		"lifecycle := NewAllServiceServerStreamMessageStream(handle)",
		"return lifecycle.Recv(ctx)",
		"return lifecycle.Done(ctx)",
		"func allServiceConnectBidiStream(ctx context.Context, stream *connect.BidiStream[AllRequest, AllReply]) error {",
		"handle, err := bridge.StartBidiStream(ctx)",
		"lifecycle := NewAllServiceBidiStreamMessageStream(handle)",
		"return lifecycle.CloseSend(ctx)",
		"switch r.URL.Path {",
		"case AllServiceUnaryConnectProcedure:",
	} {
		assertGeneratedContentContains(t, plugin, connectFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, connectFile,
		"google.golang.org/grpc",
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
