package generator

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	defaultDartNativeAssetName = "rpccgo.dart"
	defaultDartNativeAssetPath = "gen/rpccgo.dart"
)

var dartPackagePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// DartGeneratorConfig stores protoc-gen-rpc-cgo-dart options after parameter parsing.
type DartGeneratorConfig struct {
	DartPackage string
}

// GenerateDart parses the protoc plugin request into a Dart generation plan
// without emitting generated files.
func GenerateDart(plugin *protogen.Plugin) (GenerationPlan, error) {
	if plugin == nil {
		return GenerationPlan{}, fmt.Errorf("dart generator plugin is nil")
	}
	if _, err := dartGeneratorConfigFromPlugin(plugin); err != nil {
		return GenerationPlan{}, err
	}
	return generateDartPlan(plugin)
}

// GenerateDartWithOptions builds a Dart generation plan and renders files into
// the plugin response.
func GenerateDartWithOptions(plugin *protogen.Plugin) (GenerationPlan, error) {
	config, err := dartGeneratorConfigFromPlugin(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}
	plan, err := generateDartPlan(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := renderDartGeneratedFiles(plugin, plan, config); err != nil {
		return GenerationPlan{}, err
	}
	return plan, nil
}

func generateDartPlan(plugin *protogen.Plugin) (GenerationPlan, error) {
	plan, err := buildGenerationPlan(plugin, GeneratorConfig{CGODir: defaultCGODir})
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := ValidateGenerationPlan(plan); err != nil {
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
	switch name {
	case "dart_package":
		return validateDartPackage(value)
	default:
		return fmt.Errorf("unknown rpccgo dart parameter %q", name)
	}
}

func dartGeneratorConfigFromPlugin(plugin *protogen.Plugin) (DartGeneratorConfig, error) {
	if plugin == nil {
		return DartGeneratorConfig{}, fmt.Errorf("dart generator plugin is nil")
	}
	if plugin.Request == nil {
		return DartGeneratorConfig{}, fmt.Errorf("dart_package parameter is required")
	}

	var config DartGeneratorConfig
	var foundPackage bool
	for _, param := range strings.Split(plugin.Request.GetParameter(), ",") {
		if param == "" {
			continue
		}
		name, value, hasValue := strings.Cut(param, "=")
		if name != "dart_package" {
			continue
		}
		if !hasValue {
			value = ""
		}
		if err := validateDartPackage(value); err != nil {
			return DartGeneratorConfig{}, err
		}
		config.DartPackage = value
		foundPackage = true
	}
	if !foundPackage {
		return DartGeneratorConfig{}, fmt.Errorf("dart_package parameter is required")
	}
	return config, nil
}

func validateDartPackage(value string) error {
	if value == "" {
		return fmt.Errorf("dart_package must not be empty")
	}
	if !dartPackagePattern.MatchString(value) {
		return fmt.Errorf("dart_package %q must be a valid Dart package name", value)
	}
	return nil
}
