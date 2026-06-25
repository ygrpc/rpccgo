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
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "// ignore_for_file: non_constant_identifier_names")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "const GreeterRpccgoClient();")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "({pb.HelloReply? value, String? error}) SayHello(pb.HelloRequest request) {")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "({GreeterListStream? value, String? error}) ListStart(pb.HelloRequest request) {")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "({GreeterListStream? value, String? error}) ListStartCallback(pb.HelloRequest request, {required void Function(pb.HelloReply value) onRecv, required void Function(String? error) onDone}) {")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "({GreeterChatStream? value, String? error}) ChatStartCallback({required void Function(pb.HelloReply value) onRecv, required void Function(String? error) onDone}) {")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "typedef _RpccgoMessageOnRecvCAbi = ffi.Void Function(ffi.Int32 stream, ffi.UintPtr responsePtr, ffi.Int32 responseLen);")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "typedef _RpccgoMessageOnDoneCAbi = ffi.Void Function(ffi.Int32 stream, ffi.Int32 errID);")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "ffi.NativeCallable<_RpccgoMessageOnRecvCAbi>.listener((int stream, int responsePtr, int responseLen)")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "ffi.NativeCallable<_RpccgoMessageOnDoneCAbi>.listener((int stream, int errID)")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "return (value: null, error: 'rpccgo: stream receive is owned by callback receive mode');")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "String? Close() {")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "symbol: 'rpccgoMsgTestv1GreeterListClose'")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "symbol: 'rpccgoMsgTestv1GreeterChatClose'")
	assertDartMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "symbol: 'rpccgoMsgTestv1GreeterSayHello'")
	assertDartMainGeneratedContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "isolateGroupBound")
	assertDartMainGeneratedContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "exceptionalReturn")
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
					{
						Name:            proto.String("List"),
						InputType:       proto.String(".test.v1.HelloRequest"),
						OutputType:      proto.String(".test.v1.HelloReply"),
						ServerStreaming: proto.Bool(true),
					},
					{
						Name:            proto.String("Chat"),
						InputType:       proto.String(".test.v1.HelloRequest"),
						OutputType:      proto.String(".test.v1.HelloReply"),
						ClientStreaming: proto.Bool(true),
						ServerStreaming: proto.Bool(true),
					},
				},
			},
		},
	}
}

func assertDartMainGeneratedContentContains(t *testing.T, plugin *protogen.Plugin, filename string, fragment string) {
	t.Helper()

	content := dartMainGeneratedContent(t, plugin, filename)
	if !strings.Contains(content, fragment) {
		t.Fatalf("generated file %q content missing %q: %q", filename, fragment, content)
	}
}

func assertDartMainGeneratedContentDoesNotContain(t *testing.T, plugin *protogen.Plugin, filename string, fragment string) {
	t.Helper()

	content := dartMainGeneratedContent(t, plugin, filename)
	if strings.Contains(content, fragment) {
		t.Fatalf("generated file %q content contains %q: %q", filename, fragment, content)
	}
}

func dartMainGeneratedContent(t *testing.T, plugin *protogen.Plugin, filename string) string {
	t.Helper()

	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != filename {
			continue
		}
		return file.GetContent()
	}
	t.Fatalf("generated file %q not found", filename)
	return ""
}
