package generator

import "testing"

func TestBuildServiceArtifactPlansNativeEnabledService(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:     "Greeter",
		Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
	}

	got := BuildServiceArtifactPlans(file, service)

	assertArtifact(t, got, GeneratedArtifactKindRuntime, "test/v1/greeter.greeter.runtime.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindCodec, "test/v1/greeter.greeter.codec.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindMessageServer, "test/v1/greeter.greeter.server.message.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindCGOMessageServer, "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindCGOMessageClient, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindNativeServer, "test/v1/greeter.greeter.server.native.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindCGONativeServer, "test/v1/cgo/greeter.greeter.server.native.cgo.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindCGONativeClient, "test/v1/cgo/greeter.greeter.client.native.cgo.rpccgo.go")
}

func TestBuildServiceArtifactPlansMessageOnlyServiceOmitsNativeArtifacts(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:     "Greeter",
		Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect},
	}

	got := BuildServiceArtifactPlans(file, service)

	assertArtifactMissing(t, got, GeneratedArtifactKindNativeServer)
	assertArtifactMissing(t, got, GeneratedArtifactKindCGONativeServer)
	assertArtifactMissing(t, got, GeneratedArtifactKindCGONativeClient)
	assertArtifact(t, got, GeneratedArtifactKindCGOMessageClient, "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go")
}

func TestBuildServiceArtifactPlansUsesConfiguredCGODir(t *testing.T) {
	file := FilePlan{
		GeneratedFilenamePrefix: "test/v1/greeter",
		CGODir:                  "../cmd/rpc",
	}
	service := ServicePlan{
		GoName:     "Greeter",
		Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
	}

	got := BuildServiceArtifactPlans(file, service)

	assertArtifact(t, got, GeneratedArtifactKindCGONativeServer, "test/cmd/rpc/greeter.greeter.server.native.cgo.rpccgo.go")
	assertArtifact(t, got, GeneratedArtifactKindCGOMessageServer, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go")
}

func TestBuildSharedArtifactPlansUsesPackageLevelExportsFilename(t *testing.T) {
	pkg := PackagePlan{
		CGODir: "cgo",
		Files: []FilePlan{{
			GeneratedFilenamePrefix: "test/v1/greeter",
			Services: []ServicePlan{{
				Artifacts: []GeneratedArtifactPlan{
					{Kind: GeneratedArtifactKindCGOMessageClient, Filename: "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go"},
				},
			}},
		}},
	}

	got := BuildSharedArtifactPlans(pkg)

	if len(got) != 1 {
		t.Fatalf("shared artifacts = %d, want 1", len(got))
	}
	if got[0].Kind != GeneratedArtifactKindSharedCGOExports || got[0].Filename != "test/v1/cgo/rpccgo.exports.cgo.rpccgo.go" {
		t.Fatalf("shared artifact = %#v, want package-level exports", got[0])
	}
}

func assertArtifact(t *testing.T, artifacts []GeneratedArtifactPlan, kind GeneratedArtifactKind, wantFilename string) {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			if artifact.Filename != wantFilename {
				t.Fatalf("artifact %q filename = %q, want %q", kind, artifact.Filename, wantFilename)
			}
			return
		}
	}
	t.Fatalf("artifact %q not found in %#v", kind, artifacts)
}

func assertArtifactMissing(t *testing.T, artifacts []GeneratedArtifactPlan, kind GeneratedArtifactKind) {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			t.Fatalf("artifact %q present: %#v", kind, artifact)
		}
	}
}
