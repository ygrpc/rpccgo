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

func TestRunEmitsRuntimeGlue(t *testing.T) {
	plugin := newMainTestPlugin(t, "paths=source_relative", mainTestFile())

	if err := run(plugin); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	assertMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", `rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"`)
	assertMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", `const greeterServiceID rpcruntime.ServiceID = "test.v1.Greeter"`)
	assertMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "func ClearGreeterServer() error")
	assertMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "err := rpcruntime.RegisterServer(greeterServiceID, rpcruntime.RegisteredServer{")
	assertMainGeneratedContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "var greeterStreamRegistry rpcruntime.StreamRegistry")
	assertMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "func RegisterGreeterConnectHandler(handler GreeterHandler) error")
}

func TestRunEmitsMessageDirectPathForDefaultService(t *testing.T) {
	file := mainTestFile()
	file.SourceCodeInfo = nil
	plugin := newMainTestPlugin(t, "paths=source_relative", file)

	if err := run(plugin); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	assertMainGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go", "type GreeterCGOMessageServer interface {")
	assertMainGeneratedContentContains(t, plugin, "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go", "//export rpccgo_msg_testv1_Greeter_register")
	assertMainGeneratedContentContains(t, plugin, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go", "//export rpccgo_msg_testv1_Greeter_SayHello")
}

func newMainTestPlugin(t *testing.T, parameter string, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()

	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(parameter),
		ProtoFile:      files,
		FileToGenerate: []string{files[0].GetName()},
	}
	plugin, err := generator.ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func mainTestFile() *descriptorpb.FileDescriptorProto {
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
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{
			Location: []*descriptorpb.SourceCodeInfo_Location{
				{
					Path:            []int32{6, 0},
					Span:            []int32{0, 0, 0},
					LeadingComments: proto.String("@rpccgo: native\n"),
				},
			},
		},
	}
}

func assertMainGeneratedContentContains(t *testing.T, plugin *protogen.Plugin, filename string, fragment string) {
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

func assertMainGeneratedContentDoesNotContain(t *testing.T, plugin *protogen.Plugin, filename string, fragment string) {
	t.Helper()

	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != filename {
			continue
		}
		if strings.Contains(file.GetContent(), fragment) {
			t.Fatalf("generated file %q content contains forbidden fragment %q: %q", filename, fragment, file.GetContent())
		}
		return
	}
	t.Fatalf("generated file %q not found", filename)
}
