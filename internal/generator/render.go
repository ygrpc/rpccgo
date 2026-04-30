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
			if file == family.Runtime {
				if err := renderRuntimeFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			if file == family.NativeServer {
				if err := renderNativeServerFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			if file == family.CGONativeServer {
				if err := renderNativeServerCGOFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			if file == family.CGONativeClient {
				if err := renderNativeClientCGOFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			renderNativeStageFile(plugin, plan, service, file)
		}
	}
	return nil
}

func RenderStageFiles(plugin *protogen.Plugin, plan FilePlan) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}

	for _, service := range plan.Services {
		servicePlan := plan
		servicePlan.Services = []ServicePlan{service}
		if service.Adapters.Has(AdapterTokenNative) {
			if err := RenderNativeStageFiles(plugin, servicePlan); err != nil {
				return err
			}
			continue
		}
		if err := RenderMessageStageFiles(plugin, servicePlan); err != nil {
			return err
		}
	}
	return nil
}

func RenderMessageStageFiles(plugin *protogen.Plugin, plan FilePlan) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}

	for _, service := range plan.Services {
		family := service.MessageFileFamily
		files := []GeneratedFilePlan{
			family.Runtime,
			family.CGOMessageServer,
			family.CGOMessageClient,
		}
		for _, file := range files {
			if !file.Enabled {
				continue
			}
			if file == family.Runtime {
				if err := renderRuntimeFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			if file == family.CGOMessageServer {
				if err := renderMessageServerCGOFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			if file == family.CGOMessageClient {
				if err := renderMessageClientCGOFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			renderMessageStageFile(plugin, plan, service, file)
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

func renderMessageStageFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	packageName := plan.GoPackageName
	if file == service.MessageFileFamily.CGOMessageServer || file == service.MessageFileFamily.CGOMessageClient {
		packageName = "main"
	}
	g.P("package ", packageName)
	g.P()
	g.P("// ", messageStageMarker(service, file))
}

func nativeStageMarker(service ServicePlan, file GeneratedFilePlan) string {
	name := file.Filename
	switch {
	case strings.Contains(name, ".runtime.rpccgo.go"):
		return strings.Join([]string{"rpccgo service runtime stage file for", service.GoName}, " ")
	case strings.Contains(name, ".server.native.rpccgo.go"):
		return strings.Join([]string{"rpccgo native stage file for", service.GoName, "go native server"}, " ")
	case strings.Contains(name, ".server.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo native stage file for", service.GoName, "cgo native server"}, " ")
	case strings.Contains(name, ".client.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo native stage file for", service.GoName, "cgo native client"}, " ")
	default:
		return strings.Join([]string{"rpccgo service stage file for", service.GoName, "unknown"}, " ")
	}
}

func messageStageMarker(service ServicePlan, file GeneratedFilePlan) string {
	name := file.Filename
	switch {
	case strings.Contains(name, ".runtime.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct stage file for", service.GoName, "runtime"}, " ")
	case strings.Contains(name, ".server.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct stage file for", service.GoName, "cgo message server callbacks"}, " ")
	case strings.Contains(name, ".client.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct stage file for", service.GoName, "cgo message client"}, " ")
	default:
		return strings.Join([]string{"rpccgo message direct stage file for", service.GoName, "unknown"}, " ")
	}
}
