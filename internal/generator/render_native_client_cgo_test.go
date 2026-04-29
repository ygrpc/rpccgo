package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
)

func TestRenderNativeClientCGODefinesUnaryExportSurface(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/stage1_acceptance.all_service.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		`rpcruntime "rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"type AllServiceUnaryNativeUnaryInput struct {",
		"NamePtr       uintptr",
		"NameLen       int32",
		"NameOwnership int32",
		"Enabled       int8",
		"Child         uintptr",
		"type AllServiceUnaryNativeUnaryOutput struct {",
		"Accepted   int8",
		"PayloadPtr uintptr",
		"PayloadLen int32",
		"func CallAllServiceUnaryNativeUnary(ctx context.Context, input *AllServiceUnaryNativeUnaryInput, output *AllServiceUnaryNativeUnaryOutput) int32 {",
		"err = allServiceDispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[AllServiceNativeAdapter]) error {",
		"resp, callErr = snapshot.Adapter.Unary(ctx, req)",
		"return int32(rpcruntime.StoreError(err))",
		"return int32(rpcruntime.StoreError(errors.New(\"rpccgo: native unary client input is nil\")))",
		"return int32(rpcruntime.StoreError(errors.New(\"rpccgo: native unary client output is nil\")))",
		"return int32(rpcruntime.StoreError(errors.New(\"rpccgo: native unary server returned nil response\")))",
		"req.Name = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0).SafeString()",
		"req.Enabled = input.Enabled != 0",
		"return nil, allServiceNativeClientUnsupportedField",
		"output.Accepted = 1",
		"ptr, err := rpcruntime.PinBytes(resp.Payload)",
		"length, err := rpcruntime.LengthToInt32(len(resp.Payload))",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf")
}

func TestRenderNativeClientCGOGeneratedSourceCompiles(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
	})
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated native client go test failed: %v\n%s", err, out)
	}
}

func writeNativeGeneratedModule(t *testing.T, root string, plugin *protogen.Plugin, include func(string) bool) {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !include(name) {
			continue
		}
		target := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir generated dir: %v", err)
		}
		if err := os.WriteFile(target, []byte(generated.GetContent()), 0o644); err != nil {
			t.Fatalf("write generated file %s: %v", name, err)
		}
	}
}
