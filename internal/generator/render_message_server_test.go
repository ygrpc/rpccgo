package generator

import (
	os "os"
	exec "os/exec"
	filepath "path/filepath"
	strings "strings"
	testing "testing"
)

func TestRenderMessageServerDefinesUnimplementedHelper(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const messageServerFile = "test/v1/complete_service_plan.all_service.server.message.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceCGOMessageServer interface {",
		"type UnimplementedAllServiceCGOMessageServer struct{}",
		`errors.New("rpccgo: AllService.Unary cgo message server method is not implemented")`,
		"func (UnimplementedAllServiceCGOMessageServer) Unary(ctx context.Context, req *AllRequest) (*AllReply, error) {",
		"func (UnimplementedAllServiceCGOMessageServer) ClientStream(ctx context.Context, stream rpcruntime.ClientStreamingServer[*AllRequest]) (*AllReply, error) {",
		"func (UnimplementedAllServiceCGOMessageServer) ServerStream(ctx context.Context, req *AllRequest, stream rpcruntime.ServerStreamingServer[*AllReply]) error {",
		"func (UnimplementedAllServiceCGOMessageServer) BidiStream(ctx context.Context, stream rpcruntime.BidiStreamingServer[*AllRequest, *AllReply]) error {",
		"func RegisterAllServiceCGOMessageServer(server AllServiceCGOMessageServer) error {",
		"return registerAllServiceCGOMessageServer(server)",
		"func registerAllServiceCGOMessageServer(server AllServiceCGOMessageServer) error {",
		"Kind:   rpcruntime.ServerKindCGOMessage,",
		"Server: server,",
		"func startAllServiceCGOMessageClientStream(ctx context.Context, server AllServiceCGOMessageServer) (rpcruntime.ClientStreamingClient[*AllRequest, *AllReply], error)",
		"func startAllServiceCGOMessageServerStream(ctx context.Context, server AllServiceCGOMessageServer, req *AllRequest) (rpcruntime.ServerStreamingClient[*AllReply], error)",
		"func startAllServiceCGOMessageBidiStream(ctx context.Context, server AllServiceCGOMessageServer) (rpcruntime.BidiStreamingClient[*AllRequest, *AllReply], error)",
		"rpcruntime.NewClientStreaming[*AllRequest, *AllReply]",
		"rpcruntime.NewServerStreaming[*AllReply]",
		"rpcruntime.NewBidiStreaming[*AllRequest, *AllReply]",
		"NilRequest:",
		"NilResponse:",
		"resp, err := server.ClientStream(streamCtx, stream)",
		"stream.Complete(resp, err)",
		"err := server.BidiStream(streamCtx, stream)",
	} {
		assertGeneratedContentContains(t, plugin, messageServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, messageServerFile,
		"type allServiceCGOMessageEntry struct {",
		"func (a *allServiceCGOMessageEntry)",
		"rpcruntime.AdapterSnapshot",
		"type allServiceMessageBinding struct {",
		"defer close(session.responses)",
		"closeResponses",
		"responsesMu",
		"responsesClosed",
		"doneRequested",
		"s.err = nil",
		"s.finishCancel()\n\ts.cancel()\n\ts.acknowledgeReceived()",
		"type allServiceClientStreamMessageServerClientStreamSessionRequest struct {",
		"type allServiceServerStreamMessageServerServerStreamSessionResponse struct {",
		"type allServiceBidiStreamMessageServerBidiStreamSessionRequest struct {",
		"func (s *allServiceServerStreamMessageServerServerStreamSession) Finish(ctx context.Context) error {\n\ts.finishCancel()",
		"s.closeSendOnce.Do(func() { close(s.sendDone) })",
		"s.acknowledgeReceived()",
		"MessageServerClientStreamSession",
		"MessageServerServerStreamSession",
		"MessageServerBidiStreamSession",
		"MessageServerBidiStreamFacade",
	)
}

func TestRenderMessageServerRejectsUnimplementedHelperCollision(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{GoPackageName: "testv1", GoImportPath: "example.com/test/v1"}
	service := ServicePlan{
		Name:     "Greeter",
		GoName:   "Greeter",
		FullName: "test.v1.Greeter",
		Methods: []MethodPlan{{
			Name:      "SayHello",
			GoName:    "SayHello",
			FullName:  "test.v1.Greeter.SayHello",
			Streaming: StreamingKindUnary,
			Request:   MethodIOPlan{GoName: "UnimplementedGreeterCGOMessageServer", GoImportPath: "example.com/test/v1", FullName: "test.v1.UnimplementedGreeterCGOMessageServer"},
			Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
		}},
	}
	file := GeneratedArtifactPlan{Kind: GeneratedArtifactKindMessageServer, Filename: "test/v1/collision.server.message.rpccgo.go"}

	err := renderMessageServerFile(plugin, plan, service, file)
	if err == nil {
		t.Fatal("renderMessageServerFile() error = nil, want unimplemented helper collision")
	}
	if got := err.Error(); !strings.Contains(got, "UnimplementedGreeterCGOMessageServer") || !strings.Contains(got, "collides") {
		t.Fatalf("renderMessageServerFile() error = %q, want unimplemented helper collision", got)
	}
}

func TestRenderMessageServerUnimplementedHelperSupportsPartialImplementation(t *testing.T) {
	file := simpleTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".server.message.rpccgo.go")
	})
	writeMessageServerRuntimeStub(t, tmp)
	writePartialMessageServerBehaviorTest(t, tmp)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated partial message server go test failed: %v\n%s", err, out)
	}
}

func writePartialMessageServerBehaviorTest(t *testing.T, root string) {
	t.Helper()

	const content = `package generated_test

import (
	context "context"
	strings "strings"
	testing "testing"

	testv1 "example.com/test/v1"
)

type partialGreeterMessageServer struct {
	testv1.UnimplementedGreeterCGOMessageServer
}

func TestPartialMessageServerUsesUnimplementedFallback(t *testing.T) {
	var server testv1.GreeterCGOMessageServer = partialGreeterMessageServer{}
	resp, err := server.SayHello(context.Background(), &testv1.HelloRequest{})
	if err == nil {
		t.Fatal("SayHello() error = nil, want unimplemented error")
	}
	if resp != nil {
		t.Fatalf("SayHello() response = %v, want nil", resp)
	}
	if got := err.Error(); !strings.Contains(got, "rpccgo: Greeter.SayHello cgo message server method is not implemented") {
		t.Fatalf("SayHello() error = %q, want method-specific unimplemented error", got)
	}
}
`
	target := filepath.Join(root, "partial_message_server_test.go")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write partial message server behavior test: %v", err)
	}
}

func writeMessageServerRuntimeStub(t *testing.T, root string) {
	t.Helper()

	const content = `package testv1

import (
	errors "errors"

	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

const greeterServiceID rpcruntime.ServiceID = "test.v1.Greeter"

var GreeterMessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")

type HelloRequest struct{}

type HelloReply struct{}
`
	target := filepath.Join(root, "test/v1/message_server_runtime_stub.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir message server runtime stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write message server runtime stub: %v", err)
	}
}
