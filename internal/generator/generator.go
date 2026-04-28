package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

// Generate parses the protoc plugin request into Stage 1 planning data.
//
// Task 3 intentionally stops at file-level planning. Later Stage 1 tasks fill
// service, method, contract, and lifecycle metadata from the same protogen
// request without changing the plugin entry contract.
func Generate(plugin *protogen.Plugin) ([]FilePlan, error) {
	if plugin == nil {
		return nil, fmt.Errorf("generator plugin is nil")
	}

	plans := make([]FilePlan, 0, len(plugin.Files))
	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}
		plans = append(plans, FilePlan{
			GoPackageName: string(file.GoPackageName),
			GoImportPath:  string(file.GoImportPath),
			ProtoPath:     file.Desc.Path(),
		})
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
