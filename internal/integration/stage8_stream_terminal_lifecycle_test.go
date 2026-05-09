package integration

import (
	"strings"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func TestStage8StreamTerminalLifecycle(t *testing.T) {
	t.Run("cgo clients keep load/take terminal boundaries", func(t *testing.T) {
		plugin := newMessageStage4ATestPlugin(t)
		if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
			t.Fatalf("GenerateWithOptions() error = %v", err)
		}

		const messageClientFile = "test/v1/cgo/message_stage4a.greeter.client.message.cgo.rpccgo.go"
		for _, fragment := range []string{
			"LoadUploadMessageStream",
			"TakeUploadMessageStream",
			"LoadListMessageStream",
			"TakeListMessageStream",
			"LoadChatMessageStream",
			"TakeChatMessageStream",
			"CloseSendGreeterChatMessageBidiStream",
			`rpccgo: message client stream handle is invalid`,
		} {
			assertIntegrationGeneratedContentContains(t, plugin, messageClientFile, fragment)
		}
	})

	t.Run("remote adapters keep cancel local to stream session", func(t *testing.T) {
		plugin := newRemoteTransportStage6TestPlugin(t, "local/v1/stage8_lifecycle.proto", "example.com/stage8/lifecycle/v1;lifecyclev1")
		if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
			t.Fatalf("GenerateWithOptions() error = %v", err)
		}

		const connectRemoteFile = "local/v1/stage8_lifecycle.greeter.remote.connect.rpccgo.go"
		for _, fragment := range []string{
			"s.cancel()",
			"return s.stream.CloseRequest()",
			"return s.stream.Close()",
		} {
			assertIntegrationGeneratedContentContains(t, plugin, connectRemoteFile, fragment)
		}
		assertStage8GeneratedFileContentDoesNotContain(t, plugin, connectRemoteFile, "closeConnectRemoteConn")
	})
}

func assertStage8GeneratedFileContentDoesNotContain(t *testing.T, plugin *protogen.Plugin, filename string, fragments ...string) {
	t.Helper()
	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != filename {
			continue
		}
		content := file.GetContent()
		for _, fragment := range fragments {
			if strings.Contains(content, fragment) {
				t.Fatalf("generated file %q unexpectedly contains %q", filename, fragment)
			}
		}
		return
	}
	t.Fatalf("generated file %q not found", filename)
}
