package integration

import (
	"strings"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestMessageStage4AAcceptanceGeneratedDirectPath(t *testing.T) {
	plugin := newMessageStage4ATestPlugin(t)

	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	assertIntegrationGeneratedFilenames(t, plugin, []string{
		"test/v1/message_stage4a.greeter.runtime.rpccgo.go",
		"test/v1/cgo/message_stage4a.greeter.client.cgo.rpccgo.go",
		"test/v1/cgo/message_stage4a.greeter.server.message.cgo.rpccgo.go",
		"test/v1/cgo/message_stage4a.greeter.client.message.cgo.rpccgo.go",
	})
	assertIntegrationNoGeneratedFilenameContains(t, plugin, ".codec.", ".connect.", ".grpc.", ".remote.")

	const runtimeFile = "test/v1/message_stage4a.greeter.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type GreeterMessageAdapter interface {",
		"UnaryMessage(ctx context.Context, req []byte) ([]byte, error)",
		"StartUploadMessage(ctx context.Context) (GreeterUploadMessageStreamSession, error)",
		"StartListMessage(ctx context.Context, req []byte) (GreeterListMessageStreamSession, error)",
		"StartChatMessage(ctx context.Context) (GreeterChatMessageStreamSession, error)",
		"native/message converter is not enabled",
		"rpcruntime.StreamHandle",
	} {
		assertIntegrationGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}

	const clientFile = "test/v1/cgo/message_stage4a.greeter.client.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		"func CallGreeterUnaryMessageUnary",
		"func StartGreeterUploadMessageClientStream",
		"func SendGreeterUploadMessageClientStream",
		"func FinishGreeterUploadMessageClientStream",
		"func StartGreeterListMessageServerStream",
		"func ReadGreeterListMessageServerStream",
		"func DoneGreeterListMessageServerStream",
		"func StartGreeterChatMessageBidiStream",
		"func SendGreeterChatMessageBidiStream",
		"func CloseSendGreeterChatMessageBidiStream",
		"func DoneGreeterChatMessageBidiStream",
		"proto.Unmarshal",
		"rpcruntime.StoreError",
	} {
		assertIntegrationGeneratedContentContains(t, plugin, clientFile, fragment)
	}

	const serverFile = "test/v1/cgo/message_stage4a.greeter.server.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		"typedef struct GreeterCGOMessageServerCallbacks {",
		"GreeterUnaryCGOMessageUnaryCallback Unary;",
		"GreeterUploadCGOMessageClientStreamStartCallback UploadStart;",
		"GreeterListCGOMessageServerStreamRecvCallback ListRecv;",
		"GreeterChatCGOMessageBidiStreamCloseSendCallback ChatCloseSend;",
		"func RegisterGreeterCGOMessageServer",
		"return v1.RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindCGOMessage",
	} {
		assertIntegrationGeneratedContentContains(t, plugin, serverFile, fragment)
	}
}

func newMessageStage4ATestPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"test/v1/message_stage4a.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("test/v1/message_stage4a.proto"),
			Package: proto.String("test.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String("example.com/messagestage4a/test/v1;testv1"),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{Name: proto.String("MessageRequest")},
				{Name: proto.String("MessageReply")},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{
					messageStage4AMethod("Unary", false, false),
					messageStage4AMethod("Upload", true, false),
					messageStage4AMethod("List", false, true),
					messageStage4AMethod("Chat", true, true),
				},
			}},
		}},
	}
	plugin, err := generator.ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func messageStage4AMethod(name string, clientStreaming, serverStreaming bool) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:            proto.String(name),
		InputType:       proto.String(".test.v1.MessageRequest"),
		OutputType:      proto.String(".test.v1.MessageReply"),
		ClientStreaming: proto.Bool(clientStreaming),
		ServerStreaming: proto.Bool(serverStreaming),
	}
}

func assertIntegrationGeneratedFilenames(t *testing.T, plugin *protogen.Plugin, want []string) {
	t.Helper()
	files := plugin.Response().GetFile()
	if len(files) != len(want) {
		t.Fatalf("generated files = %v, want %v", integrationGeneratedFilenames(plugin), want)
	}
	for i, file := range files {
		if got := file.GetName(); got != want[i] {
			t.Fatalf("generated file %d = %q, want %q; all files: %v", i, got, want[i], integrationGeneratedFilenames(plugin))
		}
	}
}

func assertIntegrationNoGeneratedFilenameContains(t *testing.T, plugin *protogen.Plugin, fragments ...string) {
	t.Helper()
	for _, name := range integrationGeneratedFilenames(plugin) {
		for _, fragment := range fragments {
			if strings.Contains(name, fragment) {
				t.Fatalf("generated filename %q contains forbidden fragment %q", name, fragment)
			}
		}
	}
}

func assertIntegrationGeneratedContentContains(t *testing.T, plugin *protogen.Plugin, filename, fragment string) {
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
	t.Fatalf("generated file %q not found; all files: %v", filename, integrationGeneratedFilenames(plugin))
}

func integrationGeneratedFilenames(plugin *protogen.Plugin) []string {
	files := plugin.Response().GetFile()
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.GetName())
	}
	return names
}
