package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

// GenerateDart parses the protoc plugin request into a Dart generation plan
// without emitting generated files.
func GenerateDart(plugin *protogen.Plugin) (GenerationPlan, error) {
	if plugin == nil {
		return GenerationPlan{}, fmt.Errorf("dart generator plugin is nil")
	}
	plan, err := buildGenerationPlan(plugin, GeneratorConfig{CGODir: defaultCGODir})
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := ValidateGenerationPlan(plan); err != nil {
		return GenerationPlan{}, err
	}
	return plan, nil
}

// GenerateDartWithOptions builds a Dart generation plan and renders files into
// the plugin response.
func GenerateDartWithOptions(plugin *protogen.Plugin) (GenerationPlan, error) {
	plan, err := GenerateDart(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := renderDartGeneratedFiles(plugin, plan); err != nil {
		return GenerationPlan{}, err
	}
	return plan, nil
}

// DartProtogenOptions returns protogen options configured for the Dart FFI plugin.
func DartProtogenOptions() protogen.Options {
	return protogen.Options{
		ParamFunc: parseRPCCGODartParameter,
	}
}

func parseRPCCGODartParameter(name, value string) error {
	return fmt.Errorf("unknown rpccgo dart parameter %q", name)
}
