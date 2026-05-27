package generator

import (
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRenderStageFilesSkipsGRPCServerFileWithoutClientStreaming(t *testing.T) {
	file := completeServicePlanTestFile()
	file.SourceCodeInfo = completeServicePlanServiceComments([]string{
		"",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-grpc\n",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: msg-grpc|native\n",
		"@rpccgo: native\n",
	})
	allService := file.Service[5]
	allService.Method = []*descriptorpb.MethodDescriptorProto{
		allService.Method[0],
		allService.Method[2],
		allService.Method[3],
	}
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	assertNoGeneratedFilenameContains(t, plugin, ".server.grpc.rpccgo.go")
}

func TestRenderStageFilesSkipsGRPCServerFile(t *testing.T) {
	file := completeServicePlanTestFile()
	file.SourceCodeInfo = completeServicePlanServiceComments([]string{
		"",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-grpc\n",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: msg-grpc|native\n",
		"@rpccgo: native\n",
	})
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	assertNoGeneratedFilenameContains(t, plugin, ".server.grpc.rpccgo.go")
}
