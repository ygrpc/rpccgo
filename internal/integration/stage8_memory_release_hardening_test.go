package integration

import (
	"testing"

	"rpccgo/internal/generator"
)

func TestStage8MemoryReleaseHardening(t *testing.T) {
	t.Run("native request decode keeps ownership-aware release", func(t *testing.T) {
		plugin := newNativeUnaryTestPlugin(t)
		if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
			t.Fatalf("GenerateWithOptions() error = %v", err)
		}

		const nativeClientFile = "test/v1/cgo/native_unary.greeter.client.cgo.rpccgo.go"
		for _, fragment := range []string{
			"input.NameOwnership > 0",
			"input.PayloadOwnership > 0",
			"if err := Name.Release(); err != nil {",
			"if err := Payload.Release(); err != nil {",
			"rpcruntime.Release(PayloadPtr)",
			"rpcruntime.Release(NotePtr)",
		} {
			assertIntegrationGeneratedContentContains(t, plugin, nativeClientFile, fragment)
		}
	})

	t.Run("message server error text lifecycle keeps take-and-release", func(t *testing.T) {
		plugin := newMessageStage4ATestPlugin(t)
		if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
			t.Fatalf("GenerateWithOptions() error = %v", err)
		}

		const messageServerFile = "test/v1/cgo/message_stage4a.greeter.server.message.cgo.rpccgo.go"
		for _, fragment := range []string{
			"rpcruntime.TakeErrorText",
			"if ptr != 0 {",
			"defer rpcruntime.Release(ptr)",
			"unknown error id",
		} {
			assertIntegrationGeneratedContentContains(t, plugin, messageServerFile, fragment)
		}
	})
}
