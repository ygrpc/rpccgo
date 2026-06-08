package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

// RenderGeneratedFiles writes every artifact in a validated generation plan into the plugin response.
func RenderGeneratedFiles(plugin *protogen.Plugin, plan GenerationPlan) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}
	if err := ValidateGenerationPlan(plan); err != nil {
		return err
	}
	for _, pkg := range plan.Packages {
		for _, artifact := range pkg.SharedArtifacts {
			if err := renderPackageArtifact(plugin, pkg, artifact); err != nil {
				return err
			}
		}
		for _, file := range pkg.Files {
			for _, service := range file.Services {
				if err := renderServiceArtifacts(plugin, file, service); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func renderPackageArtifact(plugin *protogen.Plugin, pkg PackagePlan, artifact GeneratedArtifactPlan) error {
	switch artifact.Kind {
	case GeneratedArtifactKindSharedCGOExports:
		renderCGOExportSupportFile(plugin, pkg, artifact)
		return nil
	case GeneratedArtifactKindSharedCGOMain:
		renderCGOMainFile(plugin, pkg, artifact)
		return nil
	default:
		return fmt.Errorf("unknown package artifact kind %q", artifact.Kind)
	}
}

func renderServiceArtifacts(plugin *protogen.Plugin, file FilePlan, service ServicePlan) error {
	for _, artifact := range service.Artifacts {
		if err := renderServiceArtifact(plugin, file, service, artifact); err != nil {
			return err
		}
	}
	return nil
}

func renderServiceArtifact(plugin *protogen.Plugin, file FilePlan, service ServicePlan, artifact GeneratedArtifactPlan) error {
	switch artifact.Kind {
	case GeneratedArtifactKindRuntime:
		return renderRuntimeFile(plugin, file, service, artifact)
	case GeneratedArtifactKindCodec:
		renderCodecFile(plugin, file, service, artifact)
		return nil
	case GeneratedArtifactKindNativeServer:
		return renderNativeServerFile(plugin, file, service, artifact)
	case GeneratedArtifactKindCGONativeServer:
		return renderNativeServerCGOFile(plugin, file, service, artifact)
	case GeneratedArtifactKindCGONativeClient:
		return renderNativeClientCGOFile(plugin, file, service, artifact)
	case GeneratedArtifactKindMessageServer:
		return renderMessageServerFile(plugin, file, service, artifact)
	case GeneratedArtifactKindCGOMessageServer:
		return renderMessageServerCGOFile(plugin, file, service, artifact)
	case GeneratedArtifactKindCGOMessageClient:
		return renderMessageClientCGOFile(plugin, file, service, artifact)
	default:
		return fmt.Errorf("unknown service artifact kind %q", artifact.Kind)
	}
}
