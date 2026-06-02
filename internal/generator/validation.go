package generator

import "fmt"

func ValidateGenerationPlan(plan GenerationPlan) error {
	packageKeys := make(map[string]bool)
	for pi, pkg := range plan.Packages {
		if err := ValidatePackagePlan(pkg); err != nil {
			return fmt.Errorf("package[%d]: %w", pi, err)
		}
		key := pkg.GoImportPath
		if packageKeys[key] {
			return fmt.Errorf("duplicate package import path %q", key)
		}
		packageKeys[key] = true
	}
	return nil
}

func ValidatePackagePlan(pkg PackagePlan) error {
	if pkg.GoPackageName == "" {
		return fmt.Errorf("go package name is empty")
	}
	if pkg.GoImportPath == "" {
		return fmt.Errorf("go import path is empty")
	}
	if pkg.CGODir == "" {
		return fmt.Errorf("cgo dir is empty")
	}
	if err := validateArtifacts(pkg.SharedArtifacts, true); err != nil {
		return err
	}
	filenames := make(map[string]bool)
	for _, artifact := range pkg.SharedArtifacts {
		if filenames[artifact.Filename] {
			return fmt.Errorf("duplicate artifact filename %q", artifact.Filename)
		}
		filenames[artifact.Filename] = true
	}
	for fi, file := range pkg.Files {
		if err := ValidateFilePlan(file); err != nil {
			return fmt.Errorf("file[%d] %s: %w", fi, file.ProtoPath, err)
		}
		for _, service := range file.Services {
			for _, artifact := range service.Artifacts {
				if filenames[artifact.Filename] {
					return fmt.Errorf("duplicate artifact filename %q", artifact.Filename)
				}
				filenames[artifact.Filename] = true
			}
		}
	}
	return nil
}

func ValidateFilePlan(file FilePlan) error {
	if file.GoPackageName == "" {
		return fmt.Errorf("go package name is empty")
	}
	if file.GoImportPath == "" {
		return fmt.Errorf("go import path is empty")
	}
	if file.ProtoPath == "" {
		return fmt.Errorf("proto path is empty")
	}
	if file.GeneratedFilenamePrefix == "" {
		return fmt.Errorf("generated filename prefix is empty")
	}
	if file.CGODir == "" {
		return fmt.Errorf("cgo dir is empty")
	}
	serviceKinds := make(map[string]map[GeneratedArtifactKind]bool)
	for si, service := range file.Services {
		if err := ValidateServicePlan(service); err != nil {
			return fmt.Errorf("service[%d] %s: %w", si, service.FullName, err)
		}
		kinds := make(map[GeneratedArtifactKind]bool)
		for _, artifact := range service.Artifacts {
			if kinds[artifact.Kind] {
				return fmt.Errorf("service %s duplicate artifact kind %q", service.FullName, artifact.Kind)
			}
			kinds[artifact.Kind] = true
		}
		serviceKinds[service.FullName] = kinds
	}
	_ = serviceKinds
	return nil
}

func ValidateServicePlan(service ServicePlan) error {
	if !service.HasIdentity() {
		return fmt.Errorf("service identity is incomplete")
	}
	if service.Generation.MessageTransport != MessageTransportConnect && service.Generation.MessageTransport != MessageTransportGRPC {
		return fmt.Errorf("invalid message transport %q", service.Generation.MessageTransport)
	}
	if err := validateArtifacts(service.Artifacts, false); err != nil {
		return err
	}
	for mi, method := range service.Methods {
		if !method.HasIdentity() {
			return fmt.Errorf("method[%d] identity is incomplete", mi)
		}
	}
	return nil
}

func validateArtifacts(artifacts []GeneratedArtifactPlan, shared bool) error {
	for _, artifact := range artifacts {
		if artifact.Filename == "" {
			return fmt.Errorf("artifact %q filename is empty", artifact.Kind)
		}
		if shared {
			if artifact.Kind != GeneratedArtifactKindSharedCGOExports {
				return fmt.Errorf("artifact kind %q is not valid for package shared artifacts", artifact.Kind)
			}
			continue
		}
		if !isServiceArtifactKind(artifact.Kind) {
			return fmt.Errorf("artifact kind %q is not valid for service artifacts", artifact.Kind)
		}
	}
	return nil
}

func isServiceArtifactKind(kind GeneratedArtifactKind) bool {
	switch kind {
	case GeneratedArtifactKindRuntime,
		GeneratedArtifactKindCodec,
		GeneratedArtifactKindNativeServer,
		GeneratedArtifactKindCGONativeServer,
		GeneratedArtifactKindCGONativeClient,
		GeneratedArtifactKindMessageServer,
		GeneratedArtifactKindCGOMessageServer,
		GeneratedArtifactKindCGOMessageClient:
		return true
	default:
		return false
	}
}
