package integration

import (
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestStage8EmptyInputNormalization(t *testing.T) {
	plugin := newStage8EmptyInputPlugin(t)
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const messageClientFile = "test/v1/cgo/empty_input.greeter.client.message.cgo.rpccgo.go"
	assertIntegrationGeneratedContentContains(t, plugin, messageClientFile, "if ptr == 0 || length == 0 {")
	assertIntegrationGeneratedContentContains(t, plugin, messageClientFile, "return nil, nil")

	const nativeClientFile = "test/v1/cgo/empty_input.greeter.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		"if input.NamePtr == 0 || input.NameLen == 0 {",
		"Name = rpcruntime.EmptyRpcString()",
		"if input.PayloadPtr == 0 || input.PayloadLen == 0 {",
		"Payload = rpcruntime.EmptyRpcBytes()",
		"if input.ScoresPtr == 0 || input.ScoresLen == 0 {",
		"Scores = rpcruntime.EmptyRpcRepeat[int32]()",
		"if input.FlagsPtr == 0 || input.FlagsLen == 0 {",
		"Flags = rpcruntime.EmptyRpcBoolRepeat()",
		"input.NameOwnership > 0",
		"input.PayloadOwnership > 0",
		"input.ScoresOwnership > 0",
		"input.FlagsOwnership > 0",
	} {
		assertIntegrationGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
}

func newStage8EmptyInputPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"test/v1/empty_input.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("test/v1/empty_input.proto"),
			Package: proto.String("test.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String("example.com/stage8empty/test/v1;testv1"),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("EmptyInputRequest"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("scores", 3, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
						fieldDescriptor("flags", 4, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					},
				},
				{
					Name: proto.String("EmptyInputReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:       proto.String("Unary"),
					InputType:  proto.String(".test.v1.EmptyInputRequest"),
					OutputType: proto.String(".test.v1.EmptyInputReply"),
				}},
			}},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
				Path:            []int32{6, 0},
				Span:            []int32{0, 0, 0},
				LeadingComments: proto.String("@rpccgo: msg-connect|native\n"),
			}}},
		}},
	}
	plugin, err := generator.ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}
