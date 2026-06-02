package generator

import (
	"strings"
	"testing"
)

func TestValidateGenerationPlanRejectsArtifactsOutsideSelection(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plan, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	plan.Packages[0].Files[0].Services[0].Artifacts = append(plan.Packages[0].Files[0].Services[0].Artifacts, GeneratedArtifactPlan{
		Kind:     GeneratedArtifactKindCGONativeClient,
		Filename: "test/v1/cgo/greeter.greeter.client.native.cgo.rpccgo.go",
	})

	err = ValidateGenerationPlan(plan)
	if err == nil {
		t.Fatal("ValidateGenerationPlan() error = nil, want native artifact rejected for message-only service")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected artifact") || !strings.Contains(got, string(GeneratedArtifactKindCGONativeClient)) {
		t.Fatalf("ValidateGenerationPlan() error = %q, want unexpected cgo native client artifact", got)
	}
}

func TestValidateGenerationPlanRejectsMethodRenderPlanMismatch(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())
	plan, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	plan.Packages[0].Files[0].Services[0].Methods[0].RenderPlan.Lifecycle.RequiresCodec = false

	err = ValidateGenerationPlan(plan)
	if err == nil {
		t.Fatal("ValidateGenerationPlan() error = nil, want method render plan mismatch")
	}
	if got := err.Error(); !strings.Contains(got, "render lifecycle does not match contract capabilities") {
		t.Fatalf("ValidateGenerationPlan() error = %q, want method render mismatch", got)
	}
}
