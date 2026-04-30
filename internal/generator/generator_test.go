package generator

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateBuildsBasicFilePlans(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("Generate() returned %d plans, want 1", len(plans))
	}
	plan := plans[0]
	if plan.ProtoPath != "test/v1/greeter.proto" {
		t.Fatalf("ProtoPath = %q, want %q", plan.ProtoPath, "test/v1/greeter.proto")
	}
	if plan.GoPackageName != "testv1" {
		t.Fatalf("GoPackageName = %q, want %q", plan.GoPackageName, "testv1")
	}
	if plan.GoImportPath != "example.com/test/v1" {
		t.Fatalf("GoImportPath = %q, want %q", plan.GoImportPath, "example.com/test/v1")
	}
	if plan.GeneratedFilenamePrefix != "test/v1/greeter" {
		t.Fatalf("GeneratedFilenamePrefix = %q, want %q", plan.GeneratedFilenamePrefix, "test/v1/greeter")
	}
	if plan.CGODir != "cgo" {
		t.Fatalf("CGODir = %q, want %q", plan.CGODir, "cgo")
	}
	if len(plan.Services) != 1 {
		t.Fatalf("Services = %d, want 1", len(plan.Services))
	}
	service := plan.Services[0]
	if service.Name != "Greeter" || service.GoName != "Greeter" || service.FullName != "test.v1.Greeter" {
		t.Fatalf("Service identity = (%q, %q, %q), want Greeter metadata", service.Name, service.GoName, service.FullName)
	}
	if !service.Adapters.Has(AdapterTokenMessageConnect) || len(service.Adapters.Tokens) != 1 {
		t.Fatalf("Service adapters = %#v, want default msg-connect", service.Adapters.Tokens)
	}
	if len(service.Methods) != 1 {
		t.Fatalf("Methods = %d, want 1", len(service.Methods))
	}
	assertGeneratedFilePlan(t, service.NativeFileFamily.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, service.NativeFileFamily.NativeServer, "test/v1/greeter.greeter.server.native.rpccgo.go", false)
	assertGeneratedFilePlan(t, service.NativeFileFamily.CGONativeServer, "test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go", false)
	assertGeneratedFilePlan(t, service.NativeFileFamily.CGONativeClient, "test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go", true)
	if len(plugin.Response().GetFile()) != 0 {
		t.Fatalf("Generate() may attach file family plans, but must not emit files during plan-only generation")
	}
}

func TestGenerateWithNativeRendererEmitsNativeStageFiles(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("GenerateWithOptions() returned %d plans, want 1", len(plans))
	}
	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go",
	})
	assertNoGeneratedFilenameContains(t, plugin, ".connect.", ".grpc.", ".message.", ".remote.")
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "rpccgo service runtime stage file for Greeter")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go", "rpccgo native stage file for Greeter go native server")
	assertGeneratedContentContains(t, plugin, "test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go", "rpccgo native stage file for Greeter cgo native server")
	assertGeneratedContentContains(t, plugin, "test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go", "rpccgo native stage file for Greeter cgo native client")
}

func TestGenerateWithNativeRendererSkipsNativeServerForMessageOnlyService(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go",
	})
	assertNoGeneratedFilenameContains(t, plugin, ".server.native.", ".server.cgo.", ".connect.", ".grpc.", ".message.", ".remote.")
	assertGeneratedContentDoesNotContain(t, plugin, "go native server", "cgo native server", "connectrpc.com/connect", "google.golang.org/grpc")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "rpccgo service runtime stage file for Greeter")
	assertGeneratedContentContains(t, plugin, "test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go", "rpccgo native stage file for Greeter cgo native client")
}

func TestGenerateWithNativeRendererUsesNonSourceRelativeGeneratedPrefix(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "", file)

	plans, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	if got, want := plans[0].GeneratedFilenamePrefix, "example.com/test/v1/greeter"; got != want {
		t.Fatalf("GeneratedFilenamePrefix = %q, want %q", got, want)
	}
	assertGeneratedFilenames(t, plugin, []string{
		"example.com/test/v1/greeter.greeter.runtime.rpccgo.go",
		"example.com/test/v1/greeter.greeter.server.native.rpccgo.go",
		"example.com/test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go",
		"example.com/test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go",
	})
}

func TestGenerateAcceptsCGODirParameter(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", file)

	plans, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	if got, want := plans[0].CGODir, "../cmd/rpc"; got != want {
		t.Fatalf("CGODir = %q, want %q", got, want)
	}
	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.server.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
	})
}

func TestPluginOptionsRejectEmptyCGODirParameter(t *testing.T) {
	request := newTestCodeGeneratorRequest("cgo_dir=", simpleTestFile())

	_, err := ProtogenOptions().New(request)
	if err == nil {
		t.Fatal("ProtogenOptions().New() error = nil, want empty cgo_dir error")
	}
	if !strings.Contains(err.Error(), "cgo_dir") || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("ProtogenOptions().New() error = %q, want empty cgo_dir error", err.Error())
	}
}

func TestPluginOptionsRejectAbsoluteCGODirParameter(t *testing.T) {
	request := newTestCodeGeneratorRequest("cgo_dir=/tmp/rpc", simpleTestFile())

	_, err := ProtogenOptions().New(request)
	if err == nil {
		t.Fatal("ProtogenOptions().New() error = nil, want absolute cgo_dir error")
	}
	if !strings.Contains(err.Error(), "cgo_dir") || !strings.Contains(err.Error(), "relative") {
		t.Fatalf("ProtogenOptions().New() error = %q, want relative cgo_dir error", err.Error())
	}
}

func TestGenerateWithNativeRendererPropagatesRendererError(t *testing.T) {
	wantErr := errors.New("renderer failed")
	original := renderNativeStageFiles
	renderNativeStageFiles = func(plugin *protogen.Plugin, plan FilePlan) error {
		return wantErr
	}
	t.Cleanup(func() {
		renderNativeStageFiles = original
	})

	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if !errors.Is(err, wantErr) {
		t.Fatalf("GenerateWithOptions() error = %v, want %v", err, wantErr)
	}
}

func TestGenerateAllowsStandardPathsParameter(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())

	if _, err := Generate(plugin); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestPluginOptionsAllowStandardImportMappingParameter(t *testing.T) {
	plugin := newTestPlugin(t, "Mtest/v1/greeter.proto=example.com/override/v1", simpleTestFile())

	if _, err := Generate(plugin); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got, want := string(plugin.Files[0].GoImportPath), "example.com/override/v1"; got != want {
		t.Fatalf("GoImportPath = %q, want %q", got, want)
	}
}

func TestGenerateRejectsUnknownRPCCGOParameter(t *testing.T) {
	request := newTestCodeGeneratorRequest("mode=message", simpleTestFile())

	_, err := ProtogenOptions().New(request)
	if err == nil {
		t.Fatal("ProtogenOptions().New() error = nil, want unknown parameter error")
	}
	if !strings.Contains(err.Error(), `unknown rpccgo parameter "mode"`) {
		t.Fatalf("ProtogenOptions().New() error = %q, want unknown mode parameter", err.Error())
	}
}

func newTestPlugin(t *testing.T, parameter string, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()

	request := newTestCodeGeneratorRequest(parameter, files...)
	plugin, err := ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func newTestPluginGenerating(t *testing.T, parameter, fileToGenerate string, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()

	request := newTestCodeGeneratorRequest(parameter, files...)
	request.FileToGenerate = []string{fileToGenerate}
	plugin, err := ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func newTestCodeGeneratorRequest(parameter string, files ...*descriptorpb.FileDescriptorProto) *pluginpb.CodeGeneratorRequest {
	return &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(parameter),
		ProtoFile:      files,
		FileToGenerate: []string{files[0].GetName()},
	}
}

func simpleTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/greeter.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("HelloRequest")},
			{Name: proto.String("HelloReply")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("SayHello"),
						InputType:  proto.String(".test.v1.HelloRequest"),
						OutputType: proto.String(".test.v1.HelloReply"),
					},
				},
			},
		},
	}
}

func setSimpleServiceComment(t *testing.T, file *descriptorpb.FileDescriptorProto, comment string) {
	t.Helper()

	if len(file.Service) != 1 {
		t.Fatalf("simple test file has %d services, want 1", len(file.Service))
	}
	file.SourceCodeInfo = &descriptorpb.SourceCodeInfo{
		Location: []*descriptorpb.SourceCodeInfo_Location{
			{
				Path:            []int32{6, 0},
				Span:            []int32{0, 0, 0},
				LeadingComments: proto.String(comment),
			},
		},
	}
}

func assertGeneratedFilenames(t *testing.T, plugin *protogen.Plugin, want []string) {
	t.Helper()

	files := plugin.Response().GetFile()
	if len(files) != len(want) {
		t.Fatalf("generated files = %v, want %v", generatedFilenames(plugin), want)
	}
	for i, file := range files {
		if got := file.GetName(); got != want[i] {
			t.Fatalf("generated file %d = %q, want %q; all files: %v", i, got, want[i], generatedFilenames(plugin))
		}
		content := file.GetContent()
		wantPackage := "package testv1"
		if strings.Contains(file.GetName(), "/cgo/") || strings.Contains(file.GetName(), "/cmd/rpc/") {
			wantPackage = "package main"
		}
		if !strings.Contains(content, wantPackage) {
			t.Fatalf("generated file %q missing package declaration: %q", file.GetName(), content)
		}
	}
}

func assertNoGeneratedFilenameContains(t *testing.T, plugin *protogen.Plugin, fragments ...string) {
	t.Helper()

	for _, name := range generatedFilenames(plugin) {
		for _, fragment := range fragments {
			if strings.Contains(name, fragment) {
				t.Fatalf("generated filename %q contains forbidden fragment %q; all files: %v", name, fragment, generatedFilenames(plugin))
			}
		}
	}
}

func assertGeneratedContentContains(t *testing.T, plugin *protogen.Plugin, filename string, fragment string) {
	t.Helper()

	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != filename {
			continue
		}
		if !strings.Contains(file.GetContent(), fragment) {
			t.Fatalf("generated file %q content missing %q: %q", filename, fragment, file.GetContent())
		}
		return
	}
	t.Fatalf("generated file %q not found; all files: %v", filename, generatedFilenames(plugin))
}

func assertGeneratedContentDoesNotContain(t *testing.T, plugin *protogen.Plugin, fragments ...string) {
	t.Helper()

	for _, file := range plugin.Response().GetFile() {
		for _, fragment := range fragments {
			if strings.Contains(file.GetContent(), fragment) {
				t.Fatalf("generated file %q content contains forbidden fragment %q: %q", file.GetName(), fragment, file.GetContent())
			}
		}
	}
}

func assertGeneratedFileContentDoesNotContain(t *testing.T, plugin *protogen.Plugin, filename string, fragments ...string) {
	t.Helper()

	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != filename {
			continue
		}
		for _, fragment := range fragments {
			if strings.Contains(file.GetContent(), fragment) {
				t.Fatalf("generated file %q content contains forbidden fragment %q: %q", filename, fragment, file.GetContent())
			}
		}
		return
	}
	t.Fatalf("generated file %q not found; all files: %v", filename, generatedFilenames(plugin))
}

func generatedFilenames(plugin *protogen.Plugin) []string {
	files := plugin.Response().GetFile()
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.GetName())
	}
	return names
}
