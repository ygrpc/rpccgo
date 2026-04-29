package generator

import "testing"

func TestRenderNativeFileFamilyPlanNativeEnabledService(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative}},
	}

	got := BuildNativeFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.NativeServer, "test/v1/greeter.greeter.server.native.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGONativeServer, "test/v1/greeter.greeter.server.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGONativeClient, "test/v1/greeter.greeter.client.cgo.rpccgo.go", false)
}

func TestRenderNativeFileFamilyPlanMessageOnlyService(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}},
	}

	got := BuildNativeFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.Runtime, "test/v1/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.NativeServer, "test/v1/greeter.greeter.server.native.rpccgo.go", false)
	assertGeneratedFilePlan(t, got.CGONativeServer, "test/v1/greeter.greeter.server.cgo.rpccgo.go", false)
	assertGeneratedFilePlan(t, got.CGONativeClient, "test/v1/greeter.greeter.client.cgo.rpccgo.go", false)
}

func TestRenderNativeFileFamilyPlanUsesGeneratedFilenamePrefix(t *testing.T) {
	file := FilePlan{
		ProtoPath:               "proto/source/path/greeter.proto",
		GeneratedFilenamePrefix: "gen/output/greeter",
	}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenNative}},
	}

	got := BuildNativeFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.Runtime, "gen/output/greeter.greeter.runtime.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.NativeServer, "gen/output/greeter.greeter.server.native.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGONativeServer, "gen/output/greeter.greeter.server.cgo.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.CGONativeClient, "gen/output/greeter.greeter.client.cgo.rpccgo.go", false)
}

func assertGeneratedFilePlan(t *testing.T, got GeneratedFilePlan, wantFilename string, wantEnabled bool) {
	t.Helper()

	if got.Filename != wantFilename || got.Enabled != wantEnabled {
		t.Fatalf("GeneratedFilePlan = %#v, want filename %q enabled %v", got, wantFilename, wantEnabled)
	}
}
