package generator

import "testing"

func TestRenderStageFilesSkipsConnectServerFile(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	assertNoGeneratedFilenameContains(t, plugin, ".server.connect.rpccgo.go")
}
