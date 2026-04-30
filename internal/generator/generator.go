package generator

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const defaultCGODir = "cgo"

type GenerateOptions struct {
	RenderNativeStageFiles  bool
	RenderMessageStageFiles bool
	RenderStageFiles        bool
}

type GeneratorConfig struct {
	CGODir string
}

var renderNativeStageFiles = RenderNativeStageFiles
var renderMessageStageFiles = RenderMessageStageFiles
var renderStageFiles = RenderStageFiles

// Generate parses the protoc plugin request into planning data, including file
// family plans, without emitting generated files.
func Generate(plugin *protogen.Plugin) ([]FilePlan, error) {
	return GenerateWithOptions(plugin, GenerateOptions{})
}

func GenerateWithOptions(plugin *protogen.Plugin, options GenerateOptions) ([]FilePlan, error) {
	if plugin == nil {
		return nil, fmt.Errorf("generator plugin is nil")
	}
	config, err := generatorConfigFromPlugin(plugin)
	if err != nil {
		return nil, err
	}

	plans := make([]FilePlan, 0, len(plugin.Files))
	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}
		plan, err := BuildDescriptorPlan(file)
		if err != nil {
			return nil, err
		}
		plan.CGODir = config.CGODir
		AttachNativeFileFamilyPlan(&plan)
		AttachMessageFileFamilyPlan(&plan)
		plans = append(plans, plan)
	}
	for i := range plans {
		plans[i].TopLevelSymbols = buildPackageLevelSymbolPlans(plugin.Files, plans[i].GoImportPath)
	}
	if options.RenderNativeStageFiles {
		for _, plan := range plans {
			if err := renderNativeStageFiles(plugin, plan); err != nil {
				return nil, err
			}
		}
	}
	if options.RenderMessageStageFiles {
		for _, plan := range plans {
			if err := renderMessageStageFiles(plugin, plan); err != nil {
				return nil, err
			}
		}
	}
	if options.RenderStageFiles {
		for _, plan := range plans {
			if err := renderStageFiles(plugin, plan); err != nil {
				return nil, err
			}
		}
	}
	return plans, nil
}

func ProtogenOptions() protogen.Options {
	return protogen.Options{
		ParamFunc: parseRPCCGOParameter,
	}
}

func parseRPCCGOParameter(name, value string) error {
	switch name {
	case "cgo_dir":
		_, err := cleanCGODir(value)
		return err
	default:
		return fmt.Errorf("unknown rpccgo parameter %q", name)
	}
}

func generatorConfigFromPlugin(plugin *protogen.Plugin) (GeneratorConfig, error) {
	config := GeneratorConfig{CGODir: defaultCGODir}
	if plugin.Request == nil {
		return config, nil
	}
	for _, param := range strings.Split(plugin.Request.GetParameter(), ",") {
		if param == "" {
			continue
		}
		name, value, hasValue := strings.Cut(param, "=")
		if name != "cgo_dir" {
			continue
		}
		if !hasValue {
			value = ""
		}
		cleaned, err := cleanCGODir(value)
		if err != nil {
			return GeneratorConfig{}, err
		}
		config.CGODir = cleaned
	}
	return config, nil
}

func cleanCGODir(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("cgo_dir must not be empty")
	}
	if filepath.IsAbs(value) || path.IsAbs(value) {
		return "", fmt.Errorf("cgo_dir must be relative to the protobuf Go package output directory")
	}
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "." {
		return "", fmt.Errorf("cgo_dir must not be empty")
	}
	return cleaned, nil
}
