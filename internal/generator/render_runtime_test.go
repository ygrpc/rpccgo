package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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
		"Unary(ctx context.Context, req *AllRequest) (*AllReply, error)",
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
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime go test failed: %v\n%s", err, out)
	}
}

func TestRenderRuntimeGeneratedSourceCompilesWithImportedMessages(t *testing.T) {
	common := importedNativeCommonFile()
	service := importedNativeServiceFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", service.GetName(), common, service)

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
	target := filepath.Join(tmp, "common/v1/common_stubs.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir common stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(`package commonv1

type CommonRequest struct {
	Name string
}

type CommonReply struct {
	Payload []byte
}
`), 0o644); err != nil {
		t.Fatalf("write common stubs: %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime with imported messages go test failed: %v\n%s", err, out)
	}
}

func importedNativeCommonFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("common/v1/common.proto"),
		Package: proto.String("common.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/generated/common/v1;commonv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("CommonRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
			{
				Name: proto.String("CommonReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("payload", 1, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
		},
	}
}

func importedNativeServiceFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/v1/imported_native.proto"),
		Package:    proto.String("test.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"common/v1/common.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/generated/test/v1;testv1"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("ImportedNative"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Call", ".common.v1.CommonRequest", ".common.v1.CommonReply", false, false),
				},
			},
		},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{
			{
				Path:            []int32{6, 0},
				Span:            []int32{0, 0, 0},
				LeadingComments: proto.String("@rpccgo: native\n"),
			},
		}},
	}
}
