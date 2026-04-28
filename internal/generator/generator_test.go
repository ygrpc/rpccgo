package generator

import (
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
	if len(plugin.Response().GetFile()) != 0 {
		t.Fatalf("Generate() must not emit runtime files during Stage 1 planning")
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
