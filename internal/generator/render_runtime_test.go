package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRuntimeGlueImportsRPCRuntimeOnly(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/greeter.greeter.runtime.rpccgo.go"
	assertGeneratedContentContains(t, plugin, runtimeFile, `rpcruntime "rpccgo/rpcruntime"`)
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc")
}

func TestRenderRuntimeGlueDefinesServiceDispatcherAndRegistration(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/stage1_acceptance.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceNativeAdapter interface {",
		"Unary(ctx context.Context) error",
		"StartClientStream(ctx context.Context) (AllServiceClientStreamNativeStreamSession, error)",
		"StartServerStream(ctx context.Context) (AllServiceServerStreamNativeStreamSession, error)",
		"StartBidiStream(ctx context.Context) (AllServiceBidiStreamNativeStreamSession, error)",
		"type AllServiceClientStreamNativeStreamSession interface {",
		"type AllServiceServerStreamNativeStreamSession interface {",
		"type AllServiceBidiStreamNativeStreamSession interface {",
		"Send(ctx context.Context) error",
		"Finish(ctx context.Context) error",
		"CloseSend(ctx context.Context) error",
		"Cancel(ctx context.Context) error",
		"var allServiceDispatcher rpcruntime.Dispatcher[AllServiceNativeAdapter]",
		"func registerAllServiceActiveServer(kind rpcruntime.ServerKind, adapter AllServiceNativeAdapter) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"return allServiceDispatcher.Register(kind, rpcruntime.ServerContractNative, adapter)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueUsesRPCRuntimeStreamHandleAndHelpers(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/stage1_acceptance.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func loadAllServiceClientStreamNativeStream(handle rpcruntime.StreamHandle) (AllServiceClientStreamNativeStreamSession, bool) {",
		"return rpcruntime.LoadDispatcherStream[AllServiceNativeAdapter, AllServiceClientStreamNativeStreamSession](&allServiceDispatcher, handle)",
		"func takeAllServiceClientStreamNativeStream(handle rpcruntime.StreamHandle) (AllServiceClientStreamNativeStreamSession, bool) {",
		"return rpcruntime.TakeDispatcherStream[AllServiceNativeAdapter, AllServiceClientStreamNativeStreamSession](&allServiceDispatcher, handle)",
		"func deleteAllServiceClientStreamNativeStream(handle rpcruntime.StreamHandle) bool {",
		"return rpcruntime.DeleteDispatcherStream[AllServiceNativeAdapter](&allServiceDispatcher, handle)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "rpcruntime.Handle", " handle Handle", "handle Handle")
}

func TestRenderRuntimeRejectsUnknownStreamingKind(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:   "Greeter",
			GoName: "Greeter",
			Methods: []MethodPlan{{
				Name:      "Mystery",
				GoName:    "Mystery",
				FullName:  "test.v1.Greeter.Mystery",
				Streaming: StreamingKind(99),
			}},
			NativeFileFamily: NativeFileFamilyPlan{
				Runtime: GeneratedFilePlan{Filename: "test/v1/greeter.greeter.runtime.rpccgo.go", Enabled: true},
			},
		}},
	}

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want unknown streaming kind error")
	}
	if got := err.Error(); !strings.Contains(got, "Mystery") || !strings.Contains(got, "unknown streaming kind") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want method name and unknown streaming kind", got)
	}
}

func TestRenderRuntimeRejectsAdapterMethodSymbolCollision(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:   "Greeter",
			GoName: "Greeter",
			Methods: []MethodPlan{
				{Name: "StartFoo", GoName: "StartFoo", FullName: "test.v1.Greeter.StartFoo", Streaming: StreamingKindUnary},
				{Name: "Foo", GoName: "Foo", FullName: "test.v1.Greeter.Foo", Streaming: StreamingKindClientStreaming},
			},
			NativeFileFamily: NativeFileFamilyPlan{
				Runtime: GeneratedFilePlan{Filename: "test/v1/greeter.greeter.runtime.rpccgo.go", Enabled: true},
			},
		}},
	}

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want adapter method collision error")
	}
	if got := err.Error(); !strings.Contains(got, "StartFoo") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want colliding adapter method name", got)
	}
}

func TestRenderRuntimeGeneratedSourceCompiles(t *testing.T) {
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
		if !strings.Contains(name, ".runtime.rpccgo.go") {
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

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime go test failed: %v\n%s", err, out)
	}
}
