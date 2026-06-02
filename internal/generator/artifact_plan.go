package generator

import (
	"fmt"
	"path"
)

func AttachServiceArtifactPlans(file *FilePlan) {
	if file == nil {
		return
	}
	for i := range file.Services {
		file.Services[i].Artifacts = BuildServiceArtifactPlans(*file, file.Services[i])
	}
}

func BuildServiceArtifactPlans(file FilePlan, service ServicePlan) []GeneratedArtifactPlan {
	serviceName := lowerSnakeCase(service.GoName)
	prefix := file.GeneratedFilenamePrefix
	cgoPrefix := path.Join(path.Dir(prefix), cgoDirForFilePlan(file), path.Base(prefix))

	artifacts := []GeneratedArtifactPlan{
		{Kind: GeneratedArtifactKindRuntime, Filename: fmt.Sprintf("%s.%s.runtime.rpccgo.go", prefix, serviceName)},
		{Kind: GeneratedArtifactKindCodec, Filename: fmt.Sprintf("%s.%s.codec.rpccgo.go", prefix, serviceName)},
		{Kind: GeneratedArtifactKindMessageServer, Filename: fmt.Sprintf("%s.%s.server.message.rpccgo.go", prefix, serviceName)},
		{Kind: GeneratedArtifactKindCGOMessageServer, Filename: fmt.Sprintf("%s.%s.server.message.cgo.rpccgo.go", cgoPrefix, serviceName)},
		{Kind: GeneratedArtifactKindCGOMessageClient, Filename: fmt.Sprintf("%s.%s.client.message.cgo.rpccgo.go", cgoPrefix, serviceName)},
	}
	if service.Generation.NativeEnabled {
		artifacts = append(artifacts,
			GeneratedArtifactPlan{Kind: GeneratedArtifactKindNativeServer, Filename: fmt.Sprintf("%s.%s.server.native.rpccgo.go", prefix, serviceName)},
			GeneratedArtifactPlan{Kind: GeneratedArtifactKindCGONativeServer, Filename: fmt.Sprintf("%s.%s.server.native.cgo.rpccgo.go", cgoPrefix, serviceName)},
			GeneratedArtifactPlan{Kind: GeneratedArtifactKindCGONativeClient, Filename: fmt.Sprintf("%s.%s.client.native.cgo.rpccgo.go", cgoPrefix, serviceName)},
		)
	}
	return artifacts
}

func BuildSharedArtifactPlans(pkg PackagePlan) []GeneratedArtifactPlan {
	if !packageHasCGOArtifacts(pkg) || len(pkg.Files) == 0 {
		return nil
	}
	dir := path.Dir(pkg.Files[0].GeneratedFilenamePrefix)
	return []GeneratedArtifactPlan{
		{
			Kind:     GeneratedArtifactKindSharedCGOExports,
			Filename: path.Join(dir, cgoDirForPackagePlan(pkg), "rpccgo.exports.cgo.rpccgo.go"),
		},
	}
}

func BuildCodecFilePlan(file FilePlan, service ServicePlan) GeneratedArtifactPlan {
	if artifact, ok := service.Artifact(GeneratedArtifactKindCodec); ok {
		return artifact
	}
	return GeneratedArtifactPlan{
		Kind:     GeneratedArtifactKindCodec,
		Filename: fmt.Sprintf("%s.%s.codec.rpccgo.go", file.GeneratedFilenamePrefix, lowerSnakeCase(service.GoName)),
	}
}

func packageHasCGOArtifacts(pkg PackagePlan) bool {
	for _, file := range pkg.Files {
		for _, service := range file.Services {
			for _, artifact := range service.Artifacts {
				switch artifact.Kind {
				case GeneratedArtifactKindCGONativeServer, GeneratedArtifactKindCGONativeClient, GeneratedArtifactKindCGOMessageServer, GeneratedArtifactKindCGOMessageClient:
					return true
				}
			}
		}
	}
	return false
}

func cgoDirForFilePlan(file FilePlan) string {
	if file.CGODir == "" {
		return defaultCGODir
	}
	return file.CGODir
}

func cgoDirForPackagePlan(pkg PackagePlan) string {
	if pkg.CGODir == "" {
		return defaultCGODir
	}
	return pkg.CGODir
}
