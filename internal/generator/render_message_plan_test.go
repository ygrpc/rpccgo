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
	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go", true)
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
	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go", false)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go", true)
}

func TestRenderMessageFileFamilyPlanUsesDistinctCGOFilenamesForStageMerge(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go", true)
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
	assertGeneratedFilePlan(t, got.CGOMessageServer, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGOMessageClient, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go", true)
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
	assertGeneratedFilePlan(t, file.Services[0].MessageFileFamily.CGOMessageServer, "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, file.Services[0].MessageFileFamily.CGOMessageClient, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go", true)
}

func TestRenderMessageFileFamilyPlanDisablesLocalTransportAdapters(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.ConnectServer, "test/v1/greeter.greeter.server.connect.rpccgo.go", false)
	assertGeneratedFilePlan(t, got.GRPCServer, "test/v1/greeter.greeter.server.grpc.rpccgo.go", false)
}

func TestRenderMessageFileFamilyPlanDisablesRemoteTransportAdapterFiles(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.ConnectRemote, "test/v1/greeter.greeter.remote.connect.rpccgo.go", false)
	assertGeneratedFilePlan(t, got.GRPCRemote, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", false)
}

func TestRenderMessageFileFamilyPlanNeverEnablesRemoteTransportAdapterFiles(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}

	connectOnly := BuildMessageFileFamilyPlan(file, ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}},
	})
	assertGeneratedFilePlan(t, connectOnly.ConnectRemote, "test/v1/greeter.greeter.remote.connect.rpccgo.go", false)
	assertGeneratedFilePlan(t, connectOnly.GRPCRemote, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", false)

	grpcOnly := BuildMessageFileFamilyPlan(file, ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageGRPC}},
	})
	assertGeneratedFilePlan(t, grpcOnly.ConnectRemote, "test/v1/greeter.greeter.remote.connect.rpccgo.go", false)
	assertGeneratedFilePlan(t, grpcOnly.GRPCRemote, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", false)
}

func assertMessageFileFamilyDoesNotUseAdapterOrCodecFiles(t *testing.T, got MessageFileFamilyPlan) {
	t.Helper()

	for _, file := range []GeneratedFilePlan{got.Runtime, got.CGOMessageServer, got.CGOMessageClient, got.ConnectServer, got.GRPCServer, got.ConnectRemote, got.GRPCRemote} {
		assertFilenameDoesNotContain(t, file.Filename, ".adapter.", ".codec.")
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
