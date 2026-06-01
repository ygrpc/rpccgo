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

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const messageServerFile = "test/v1/complete_service_plan.all_service.server.message.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceCGOMessageServer interface {",
		"type UnimplementedAllServiceCGOMessageServer struct{}",
		`errors.New("rpccgo: AllService.Unary cgo message server method is not implemented")`,
		"func (UnimplementedAllServiceCGOMessageServer) Unary(ctx context.Context, req []byte) ([]byte, error) {",
		"type AllServiceClientStreamMessageClientStream interface {",
		"type AllServiceServerStreamMessageServerStream interface {",
		"type AllServiceBidiStreamMessageBidiStream interface {",
		"func (UnimplementedAllServiceCGOMessageServer) ClientStream(ctx context.Context, stream AllServiceClientStreamMessageClientStream) ([]byte, error) {",
		"func (UnimplementedAllServiceCGOMessageServer) ServerStream(ctx context.Context, req []byte, stream AllServiceServerStreamMessageServerStream) error {",
		"func (UnimplementedAllServiceCGOMessageServer) BidiStream(ctx context.Context, stream AllServiceBidiStreamMessageBidiStream) error {",
		"func RegisterAllServiceCGOMessageServer(server AllServiceCGOMessageServer) error {",
		"return registerAllServiceCGOMessageServer(server)",
	} {
		assertGeneratedContentContains(t, plugin, messageServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, messageServerFile,
		"rpcruntime.AdapterSnapshot", `rpcruntime "rpccgo/rpcruntime"`,
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
	file := GeneratedFilePlan{Filename: "test/v1/collision.server.message.rpccgo.go", Enabled: true}

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

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderMessageStageFiles: true})
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
	resp, err := server.SayHello(context.Background(), []byte("request"))
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

func registerGreeterCGOMessageServer(server GreeterCGOMessageServer) error {
	return nil
}
`
	target := filepath.Join(root, "test/v1/message_server_runtime_stub.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir message server runtime stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write message server runtime stub: %v", err)
	}
}
