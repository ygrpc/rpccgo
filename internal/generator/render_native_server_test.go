package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderNativeServerDefinesInterfaceAdapterAndRegistration(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/stage1_acceptance.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceNativeServer interface {",
		"Unary(ctx context.Context, req *AllRequest) (*AllReply, error)",
		"ClientStream(ctx context.Context) (AllServiceClientStreamNativeClientStream, error)",
		"ServerStream(ctx context.Context, req *AllRequest) (AllServiceServerStreamNativeServerStream, error)",
		"BidiStream(ctx context.Context) (AllServiceBidiStreamNativeBidiStream, error)",
		"type allServiceGoNativeAdapter struct {",
		"server AllServiceNativeServer",
		"func (a *allServiceGoNativeAdapter) Unary(ctx context.Context) error {",
		"_, err := a.server.Unary(ctx, nil)",
		"func (a *allServiceGoNativeAdapter) StartClientStream(ctx context.Context) (AllServiceClientStreamNativeStreamSession, error) {",
		"func (a *allServiceGoNativeAdapter) StartServerStream(ctx context.Context) (AllServiceServerStreamNativeStreamSession, error) {",
		"func (a *allServiceGoNativeAdapter) StartBidiStream(ctx context.Context) (AllServiceBidiStreamNativeStreamSession, error) {",
		"func RegisterAllServiceGoNativeServer(server AllServiceNativeServer) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		`errors.New("rpccgo: AllService go native server is nil")`,
		"return registerAllServiceActiveServer(rpcruntime.ServerKindGoNative, &allServiceGoNativeAdapter{server: server})",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf")
}

func TestRenderNativeServerDefinesStreamingMethodSignatures(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/stage1_acceptance.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceClientStreamNativeClientStream interface {",
		"Recv(ctx context.Context) (*AllRequest, error)",
		"Finish(ctx context.Context) (*AllReply, error)",
		"type AllServiceServerStreamNativeServerStream interface {",
		"Recv(ctx context.Context) (*AllReply, error)",
		"type AllServiceBidiStreamNativeBidiStream interface {",
		"Send(ctx context.Context, req *AllRequest) error",
		"CloseSend(ctx context.Context) error",
		"Cancel(ctx context.Context) error",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
}

func TestRenderNativeServerGeneratedSourceCompiles(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !strings.Contains(name, ".runtime.rpccgo.go") && !strings.Contains(name, ".server.native.rpccgo.go") {
			continue
		}
		target := filepath.Join(tmp, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir generated dir: %v", err)
		}
		if err := os.WriteFile(target, []byte(generated.GetContent()), 0o644); err != nil {
			t.Fatalf("write generated file %s: %v", name, err)
		}
	}
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated native server go test failed: %v\n%s", err, out)
	}
}

func writeNativeServerCompileStubs(t *testing.T, root string) {
	t.Helper()

	const content = `package testv1

type AllRequest struct{}
type AllReply struct{}
type ConnectNativeRequest struct{}
type ConnectNativeReply struct{}
type NativeOnlyRequest struct{}
type NativeOnlyReply struct{}
`
	target := filepath.Join(root, "test/v1/stage1_acceptance_stubs_test.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write native server compile stubs: %v", err)
	}
}
