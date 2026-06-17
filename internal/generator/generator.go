package generator

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const defaultCGODir = "cgo"

// GeneratorConfig stores protoc-gen-rpc-cgo options after parameter parsing.
type GeneratorConfig struct {
	CGODir string
}

// Generate parses the protoc plugin request into a generation plan without
// emitting generated files.
func Generate(plugin *protogen.Plugin) (GenerationPlan, error) {
	if plugin == nil {
		return GenerationPlan{}, fmt.Errorf("generator plugin is nil")
	}
	config, err := generatorConfigFromPlugin(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}

	plan, err := buildGenerationPlan(plugin, config)
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := ValidateGenerationPlan(plan); err != nil {
		return GenerationPlan{}, err
	}
	return plan, nil
}

// GenerateWithOptions builds a generation plan and renders all generated files into the plugin response.
func GenerateWithOptions(plugin *protogen.Plugin) (GenerationPlan, error) {
	plan, err := Generate(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := renderGeneratedFiles(plugin, plan); err != nil {
		return GenerationPlan{}, err
	}
	return plan, nil
}

func buildGenerationPlan(plugin *protogen.Plugin, config GeneratorConfig) (GenerationPlan, error) {
	files, err := buildFilePlans(plugin, config)
	if err != nil {
		return GenerationPlan{}, err
	}
	return GenerationPlan{Packages: packagePlansFromFiles(files, plugin.Files, config)}, nil
}

func buildFilePlans(plugin *protogen.Plugin, config GeneratorConfig) ([]FilePlan, error) {
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
		AttachServiceArtifactPlans(&plan)
		plans = append(plans, plan)
	}
	return plans, nil
}

func packagePlansFromFiles(files []FilePlan, descriptors []*protogen.File, config GeneratorConfig) []PackagePlan {
	byImportPath := make(map[string]*PackagePlan)
	var order []string
	for _, file := range files {
		key := file.GoImportPath
		pkg := byImportPath[key]
		if pkg == nil {
			pkg = &PackagePlan{
				GoPackageName: file.GoPackageName,
				GoImportPath:  file.GoImportPath,
				CGODir:        config.CGODir,
			}
			byImportPath[key] = pkg
			order = append(order, key)
		}
		pkg.Files = append(pkg.Files, file)
	}
	sort.Strings(order)

	packages := make([]PackagePlan, 0, len(order))
	for _, key := range order {
		pkg := *byImportPath[key]
		pkg.TopLevelSymbols = buildPackageLevelSymbolPlans(descriptors, key)
		for i := range pkg.Files {
			pkg.Files[i].TopLevelSymbols = pkg.TopLevelSymbols
		}
		pkg.SharedArtifacts = BuildSharedArtifactPlans(pkg)
		packages = append(packages, pkg)
	}
	return packages
}

// ProtogenOptions returns protogen options configured with rpccgo parameter parsing.
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
		if !hasValue {
			value = ""
		}
		switch name {
		case "cgo_dir":
			cleaned, err := cleanCGODir(value)
			if err != nil {
				return GeneratorConfig{}, err
			}
			config.CGODir = cleaned
		}
	}
	return config, nil
}

func cleanCGODir(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if filepath.IsAbs(value) || path.IsAbs(value) {
		return "", fmt.Errorf("cgo_dir must be relative to the protobuf Go package output directory")
	}
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "." {
		return "", nil
	}
	return cleaned, nil
}
