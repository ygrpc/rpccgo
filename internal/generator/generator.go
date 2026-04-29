package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

type GenerateOptions struct {
	RenderNativeStageFiles bool
}

// Generate parses the protoc plugin request into planning data without
// emitting generated files.
func Generate(plugin *protogen.Plugin) ([]FilePlan, error) {
	return GenerateWithOptions(plugin, GenerateOptions{})
}

func GenerateWithOptions(plugin *protogen.Plugin, options GenerateOptions) ([]FilePlan, error) {
	if plugin == nil {
		return nil, fmt.Errorf("generator plugin is nil")
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
		AttachNativeFileFamilyPlan(&plan)
		plans = append(plans, plan)
		if options.RenderNativeStageFiles {
			if err := RenderNativeStageFiles(plugin, plan); err != nil {
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
	return fmt.Errorf("unknown rpccgo parameter %q", name)
}
