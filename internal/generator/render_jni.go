package generator

import (
	"fmt"
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderJNIGeneratedFiles(plugin *protogen.Plugin, plan GenerationPlan, config JNIGeneratorConfig) error {
	if plugin == nil {
		return fmt.Errorf("jni generator plugin is nil")
	}
	var renderedCommon bool
	for _, pkg := range plan.Packages {
		for _, file := range pkg.Files {
			if len(file.Services) == 0 {
				continue
			}
			if !renderedCommon {
				renderJNICPPCommonHeaderFile(plugin, config)
				renderJNICPPCommonFile(plugin, config)
				renderedCommon = true
			}
			for _, service := range file.Services {
				renderJNICPPFile(plugin, file, service, config)
			}
			renderJNIKotlinFile(plugin, file, file.Services, config)
		}
	}
	return nil
}

func jniCPPCommonHeaderFilename(config JNIGeneratorConfig) string {
	return path.Join(config.CPPDir, "rpccgo.jni.h")
}

func jniCPPCommonFilename(config JNIGeneratorConfig) string {
	return path.Join(config.CPPDir, "rpccgo.jni.cpp")
}

func jniCPPServiceFilename(file FilePlan, service ServicePlan, config JNIGeneratorConfig) string {
	return path.Join(config.CPPDir, fmt.Sprintf("%s.%s.jni.cpp", path.Base(file.GeneratedFilenamePrefix), lowerSnakeCase(service.GoName)))
}

func jniKotlinFilename(config JNIGeneratorConfig) string {
	pkg, className := jniClassPackageAndSimpleName(config.JNIClass)
	return path.Join(config.KotlinDir, path.Join(strings.Split(pkg, ".")...), className+".kt")
}

func jniClassPackageAndSimpleName(jniClass string) (string, string) {
	lastDot := strings.LastIndex(jniClass, ".")
	if lastDot < 0 {
		return "", jniClass
	}
	return jniClass[:lastDot], jniClass[lastDot+1:]
}

func rpccgoKotlinMessageType(message MethodIOPlan) string {
	pkg, name := rpccgoKotlinMessagePackageAndName(message)
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

func rpccgoKotlinMessagePackageAndName(message MethodIOPlan) (string, string) {
	lastDot := strings.LastIndex(message.FullName, ".")
	if lastDot < 0 {
		return "", message.GoName
	}
	return message.FullName[:lastDot], message.GoName
}
