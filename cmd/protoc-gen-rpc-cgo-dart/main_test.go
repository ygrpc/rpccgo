package main

import (
	"strings"
	"testing"

	"github.com/ygrpc/rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestRunEmitsDartFFIClient(t *testing.T) {
	plugin := newDartMainTestPlugin(t, "paths=source_relative,dart_package=rpccgo_test", dartMainTestFile())

	if err := run(plugin); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	assertDartMainGeneratedContentContains(t, plugin, "rpccgo.dart", "export 'test/v1/greeter.greeter.rpccgo.dart';")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "@ffi.DefaultAsset('package:rpccgo_test/gen/rpccgo.dart')")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "const GreeterRpccgoClient();")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "({pb.HelloReply? value, String? error}) SayHello(pb.HelloRequest request) {")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "symbol: 'rpccgoMsgTestv1GreeterSayHello'")
}

func newDartMainTestPlugin(t *testing.T, parameter string, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()

	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(parameter),
		ProtoFile:      files,
		FileToGenerate: []string{files[0].GetName()},
	}
	plugin, err := generator.DartProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func dartMainTestFile() *descriptorpb.FileDescriptorProto {
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

func assertDartMainGeneratedContentContains(t *testing.T, plugin *protogen.Plugin, filename string, fragment string) {
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
	t.Fatalf("generated file %q not found", filename)
}
