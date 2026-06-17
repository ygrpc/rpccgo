package generator

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	defaultJNICPPDir    = "cpp/rpccgo"
	defaultJNIKotlinDir = "kotlin"
)

var jniClassPattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*(\.[A-Za-z_$][A-Za-z0-9_$]*)+$`)

// JNIGeneratorConfig stores protoc-gen-rpc-cgo-jni options after parameter parsing.
type JNIGeneratorConfig struct {
	JNIClass     string
	RPCCGOHeader string
	CPPDir       string
	KotlinDir    string
}

// GenerateJNI parses the protoc plugin request into a JNI generation plan
// without emitting generated files.
func GenerateJNI(plugin *protogen.Plugin) (GenerationPlan, error) {
	if plugin == nil {
		return GenerationPlan{}, fmt.Errorf("jni generator plugin is nil")
	}
	if _, err := jniGeneratorConfigFromPlugin(plugin); err != nil {
		return GenerationPlan{}, err
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

// GenerateJNIWithOptions builds a JNI generation plan and renders files into
// the plugin response.
func GenerateJNIWithOptions(plugin *protogen.Plugin) (GenerationPlan, error) {
	plan, err := GenerateJNI(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}
	config, err := jniGeneratorConfigFromPlugin(plugin)
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := renderJNIGeneratedFiles(plugin, plan, config); err != nil {
		return GenerationPlan{}, err
	}
	return plan, nil
}

// JNIProtogenOptions returns protogen options configured for the Android JNI plugin.
func JNIProtogenOptions() protogen.Options {
	return protogen.Options{
		ParamFunc: parseRPCCGOJNIParameter,
	}
}

func parseRPCCGOJNIParameter(name, value string) error {
	switch name {
	case "jni_class":
		return validateJNIClass(value)
	case "rpccgo_header":
		return validateRPCCGOHeader(value)
	case "cpp_dir":
		_, err := cleanJNIRelativeDir("cpp_dir", value)
		return err
	case "kotlin_dir":
		_, err := cleanJNIRelativeDir("kotlin_dir", value)
		return err
	default:
		return fmt.Errorf("unknown rpccgo jni parameter %q", name)
	}
}

func jniGeneratorConfigFromPlugin(plugin *protogen.Plugin) (JNIGeneratorConfig, error) {
	if plugin == nil {
		return JNIGeneratorConfig{}, fmt.Errorf("jni generator plugin is nil")
	}
	config := JNIGeneratorConfig{
		CPPDir:    defaultJNICPPDir,
		KotlinDir: defaultJNIKotlinDir,
	}
	if plugin.Request == nil {
		return JNIGeneratorConfig{}, fmt.Errorf("jni_class parameter is required")
	}
	var seenJNIClass bool
	var seenHeader bool
	for _, param := range strings.Split(plugin.Request.GetParameter(), ",") {
		if param == "" {
			continue
		}
		name, value, hasValue := strings.Cut(param, "=")
		if !hasValue {
			value = ""
		}
		switch name {
		case "jni_class":
			if err := validateJNIClass(value); err != nil {
				return JNIGeneratorConfig{}, err
			}
			config.JNIClass = value
			seenJNIClass = true
		case "rpccgo_header":
			if err := validateRPCCGOHeader(value); err != nil {
				return JNIGeneratorConfig{}, err
			}
			config.RPCCGOHeader = value
			seenHeader = true
		case "cpp_dir":
			cleaned, err := cleanJNIRelativeDir("cpp_dir", value)
			if err != nil {
				return JNIGeneratorConfig{}, err
			}
			config.CPPDir = cleaned
		case "kotlin_dir":
			cleaned, err := cleanJNIRelativeDir("kotlin_dir", value)
			if err != nil {
				return JNIGeneratorConfig{}, err
			}
			config.KotlinDir = cleaned
		}
	}
	if !seenJNIClass {
		return JNIGeneratorConfig{}, fmt.Errorf("jni_class parameter is required")
	}
	if !seenHeader {
		return JNIGeneratorConfig{}, fmt.Errorf("rpccgo_header parameter is required")
	}
	return config, nil
}

func validateJNIClass(value string) error {
	if !jniClassPattern.MatchString(value) {
		return fmt.Errorf("jni_class must be a fully-qualified Java class name")
	}
	return nil
}

func validateRPCCGOHeader(value string) error {
	if value == "" {
		return fmt.Errorf("rpccgo_header must not be empty")
	}
	if filepath.IsAbs(value) || path.IsAbs(value) || strings.Contains(strings.ReplaceAll(value, "\\", "/"), "/") {
		return fmt.Errorf("rpccgo_header must be a header filename")
	}
	if !strings.HasSuffix(value, ".h") {
		return fmt.Errorf("rpccgo_header must name a .h file")
	}
	return nil
}

func cleanJNIRelativeDir(name, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	if filepath.IsAbs(value) || path.IsAbs(value) {
		return "", fmt.Errorf("%s must be relative to the JNI output directory", name)
	}
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "." {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	return cleaned, nil
}
