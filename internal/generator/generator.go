package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

// Generate parses the protoc plugin request into Stage 1 planning data.
func Generate(plugin *protogen.Plugin) ([]FilePlan, error) {
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
		plans = append(plans, plan)
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
