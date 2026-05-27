package generator

import (
	"fmt"
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func RenderNativeStageFiles(plugin *protogen.Plugin, plan FilePlan) error {
	return renderNativeStageFilesWithSupport(plugin, plan, map[string]bool{})
}

func renderNativeStageFilesWithSupport(plugin *protogen.Plugin, plan FilePlan, rendered map[string]bool) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}
	if err := renderCGOExportSupportFileOnce(plugin, plan, rendered); err != nil {
		return err
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
	return renderCombinedStageFiles(plugin, plan)
}

func renderCombinedStageFiles(plugin *protogen.Plugin, plan FilePlan) error {
	sharedRendered := make(map[string]bool)
	if err := renderCGOExportSupportFileOnce(plugin, plan, sharedRendered); err != nil {
		return err
	}
	for _, service := range plan.Services {
		if err := renderServiceStageFiles(plugin, plan, service, sharedRendered); err != nil {
			return err
		}
	}
	return nil
}

func renderServiceStageFiles(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, sharedRendered map[string]bool) error {
	rendered := make(map[string]bool)
	if err := renderSharedRuntimeOnce(plugin, plan, service, rendered); err != nil {
		return err
	}

	nativeService := service
	nativeService.NativeFileFamily.Runtime.Enabled = false
	nativePlan := plan
	nativePlan.Services = []ServicePlan{nativeService}
	if err := renderNativeStageFilesWithSupport(plugin, nativePlan, sharedRendered); err != nil {
		return err
	}
	markRendered(rendered, nativeService.NativeFileFamily.NativeServer)
	markRendered(rendered, nativeService.NativeFileFamily.CGONativeServer)
	markRendered(rendered, nativeService.NativeFileFamily.CGONativeClient)

	messageService := service
	messageService.MessageFileFamily.Runtime.Enabled = false
	avoidRenderedFilenames(rendered, &messageService.MessageFileFamily.CGOMessageServer, "message")
	avoidRenderedFilenames(rendered, &messageService.MessageFileFamily.CGOMessageClient, "message")
	messagePlan := plan
	messagePlan.Services = []ServicePlan{messageService}
	if err := renderMessageStageFilesWithSupport(plugin, messagePlan, sharedRendered); err != nil {
		return err
	}
	markRendered(rendered, messageService.MessageFileFamily.ConnectRemote)
	markRendered(rendered, messageService.MessageFileFamily.GRPCRemote)

	codecPlan := plan
	codecPlan.Services = []ServicePlan{service}
	return RenderCodecFiles(plugin, codecPlan)
}

func renderSharedRuntimeOnce(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, rendered map[string]bool) error {
	service.CodecEnabled = service.NeedsCodec
	runtimeFile := service.NativeFileFamily.Runtime
	if !runtimeFile.Enabled {
		runtimeFile = service.MessageFileFamily.Runtime
	}
	if !runtimeFile.Enabled {
		return nil
	}
	if rendered[runtimeFile.Filename] {
		return nil
	}
	if err := renderRuntimeFile(plugin, plan, service, runtimeFile); err != nil {
		return err
	}
	rendered[runtimeFile.Filename] = true
	return nil
}

func markRendered(rendered map[string]bool, file GeneratedFilePlan) {
	if !file.Enabled {
		return
	}
	rendered[file.Filename] = true
}

func avoidRenderedFilenames(rendered map[string]bool, file *GeneratedFilePlan, qualifier string) {
	if file == nil || !file.Enabled || !rendered[file.Filename] {
		return
	}
	file.Filename = qualifiedGeneratedFilename(file.Filename, qualifier)
}

func qualifiedGeneratedFilename(filename, qualifier string) string {
	dir, base := path.Split(filename)
	suffix := ".cgo.rpccgo.go"
	if strings.HasSuffix(base, suffix) {
		return dir + strings.TrimSuffix(base, suffix) + "." + qualifier + suffix
	}
	suffix = ".rpccgo.go"
	if strings.HasSuffix(base, suffix) {
		return dir + strings.TrimSuffix(base, suffix) + "." + qualifier + suffix
	}
	return dir + base + "." + qualifier
}

func RenderMessageStageFiles(plugin *protogen.Plugin, plan FilePlan) error {
	return renderMessageStageFilesWithSupport(plugin, plan, map[string]bool{})
}

func renderMessageStageFilesWithSupport(plugin *protogen.Plugin, plan FilePlan, rendered map[string]bool) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}
	if err := renderCGOExportSupportFileOnce(plugin, plan, rendered); err != nil {
		return err
	}

	for _, service := range plan.Services {
		family := service.MessageFileFamily
		files := []GeneratedFilePlan{
			family.Runtime,
			family.CGOMessageServer,
			family.CGOMessageClient,
			family.ConnectRemote,
			family.GRPCRemote,
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
			if file == family.ConnectRemote {
				if err := renderConnectRemoteFile(plugin, plan, service, file); err != nil {
					return err
				}
				continue
			}
			if file == family.GRPCRemote {
				if err := renderGRPCRemoteFile(plugin, plan, service, file); err != nil {
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
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// ", nativeStageMarker(service, file))
}

func renderMessageStageFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) {
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))
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
		return strings.Join([]string{"rpccgo service runtime generated file for", service.GoName}, " ")
	case strings.Contains(name, ".server.native.rpccgo.go"):
		return strings.Join([]string{"rpccgo native generated file for", service.GoName, "go native server"}, " ")
	case strings.Contains(name, ".server.native.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo native generated file for", service.GoName, "cgo native server"}, " ")
	case strings.Contains(name, ".client.native.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo native generated file for", service.GoName, "cgo native client"}, " ")
	default:
		return strings.Join([]string{"rpccgo service generated file for", service.GoName, "unknown"}, " ")
	}
}

func messageStageMarker(service ServicePlan, file GeneratedFilePlan) string {
	name := file.Filename
	switch {
	case strings.Contains(name, ".runtime.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "runtime"}, " ")
	case strings.Contains(name, ".server.message.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "cgo message server callbacks"}, " ")
	case strings.Contains(name, ".client.message.cgo.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "cgo message client"}, " ")
	case strings.Contains(name, ".server.connect.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "connect local server adapter"}, " ")
	case strings.Contains(name, ".server.grpc.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "grpc local server adapter"}, " ")
	case strings.Contains(name, ".remote.connect.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "connect remote server adapter"}, " ")
	case strings.Contains(name, ".remote.grpc.rpccgo.go"):
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "grpc remote server adapter"}, " ")
	default:
		return strings.Join([]string{"rpccgo message direct generated file for", service.GoName, "unknown"}, " ")
	}
}
