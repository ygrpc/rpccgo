package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func RenderNativeStageFiles(plugin *protogen.Plugin, plan FilePlan) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}

	for _, service := range plan.Services {
		family := service.NativeFileFamily
		files := []GeneratedFilePlan{
			family.Runtime,
			family.NativeServer,
			family.CGONativeServer,
			family.CGONativeClient,
		}
		for _, file := range files {
			if !file.Enabled {
				continue
			}
			renderNativeStageFile(plugin, plan, service, file)
		}
	}
	return nil
}

func renderNativeStageFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// ", nativeStageMarker(service, file))
}

func nativeStageMarker(service ServicePlan, file GeneratedFilePlan) string {
	name := file.Filename
	parts := []string{"rpccgo native stage file for", service.GoName}
	switch {
	case strings.Contains(name, ".runtime.rpccgo.go"):
		parts = append(parts, "runtime")
	case strings.Contains(name, ".server.native.rpccgo.go"):
		parts = append(parts, "go native server")
	case strings.Contains(name, ".server.cgo.rpccgo.go"):
		parts = append(parts, "cgo native server")
	case strings.Contains(name, ".client.cgo.rpccgo.go"):
		parts = append(parts, "cgo native client")
	default:
		parts = append(parts, "unknown")
	}
	return strings.Join(parts, " ")
}
