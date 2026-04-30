package generator

import (
	"strings"
	"testing"
)

func TestRenderMessageFileFamilyPlanMessageAdapterService(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go", true)
	assertMessageFileFamilyDoesNotUseAdapterOrCodecFiles(t, got)
}

func TestRenderMessageFileFamilyPlanNativeOnlyStillEnablesMessageClient(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenNative}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go", false)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go", true)
}

func TestRenderMessageFileFamilyPlanUsesConfiguredCGODir(t *testing.T) {
	file := FilePlan{
		GeneratedFilenamePrefix: "test/v1/greeter",
		CGODir:                  "../cmd/rpc",
	}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageGRPC}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/cmd/rpc/greeter.greeter.server.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/cmd/rpc/greeter.greeter.client.cgo.rpccgo.go", true)
}

func TestAttachMessageFileFamilyPlan(t *testing.T) {
	file := FilePlan{
		GeneratedFilenamePrefix: "test/v1/greeter",
		Services: []ServicePlan{
			{
				GoName:   "Greeter",
				Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}},
			},
		},
	}

	AttachMessageFileFamilyPlan(&file)

	assertGeneratedFilePlan(t, file.Services[0].MessageFileFamily.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, file.Services[0].MessageFileFamily.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, file.Services[0].MessageFileFamily.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go", true)
}

func assertMessageFileFamilyDoesNotUseAdapterOrCodecFiles(t *testing.T, got MessageFileFamilyPlan) {
	t.Helper()

	for _, file := range []GeneratedFilePlan{got.Runtime, got.CGOMessageServer, got.CGOMessageClient} {
		assertFilenameDoesNotContain(t, file.Filename, ".connect.", ".grpc.", ".remote.", ".codec.", ".message.")
	}
}

func assertFilenameDoesNotContain(t *testing.T, filename string, fragments ...string) {
	t.Helper()

	for _, fragment := range fragments {
		if strings.Contains(filename, fragment) {
			t.Fatalf("generated filename %q contains forbidden fragment %q", filename, fragment)
		}
	}
}
