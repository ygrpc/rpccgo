package integration

import (
	"testing"

	"rpccgo/internal/generator"
)

func TestStage8MessageBytesHardening(t *testing.T) {
	plugin := newMessageStage4ATestPlugin(t)
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const clientFile = "test/v1/cgo/message_stage4a.greeter.client.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		`rpccgo: message request protobuf unmarshal failed`,
		`rpccgo: message response protobuf unmarshal failed`,
		"func SendGreeterUploadMessageClientStream",
		"func StartGreeterListMessageServerStream",
		"func SendGreeterChatMessageBidiStream",
	} {
		assertIntegrationGeneratedContentContains(t, plugin, clientFile, fragment)
	}

	const serverFile = "test/v1/cgo/message_stage4a.greeter.server.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		`protobuf "google.golang.org/protobuf/proto"`,
		`rpccgo: message request protobuf unmarshal failed`,
		`rpccgo: message response protobuf unmarshal failed`,
		"decodeGreeterUnaryCGOMessageResponseBytes",
		"decodeGreeterUploadCGOMessageResponseBytes",
		"decodeGreeterListCGOMessageResponseBytes",
		"decodeGreeterChatCGOMessageResponseBytes",
		"rpcruntime.TakeErrorText",
		"unknown error id",
	} {
		assertIntegrationGeneratedContentContains(t, plugin, serverFile, fragment)
	}
}
