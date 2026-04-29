package generator

import "fmt"

func AttachNativeFileFamilyPlan(file *FilePlan) {
	if file == nil {
		return
	}
	for i := range file.Services {
		file.Services[i].NativeFileFamily = BuildNativeFileFamilyPlan(*file, file.Services[i])
	}
}

func BuildNativeFileFamilyPlan(file FilePlan, service ServicePlan) NativeFileFamilyPlan {
	serviceName := lowerSnakeCase(service.GoName)
	prefix := file.GeneratedFilenamePrefix

	enabledNativeServer := service.Adapters.Has(AdapterTokenNative)
	return NativeFileFamilyPlan{
		Runtime: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.runtime.rpccgo.go", prefix, serviceName),
			Enabled:  true,
		},
		NativeServer: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.server.native.rpccgo.go", prefix, serviceName),
			Enabled:  enabledNativeServer,
		},
		CGONativeServer: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.server.cgo.rpccgo.go", prefix, serviceName),
			Enabled:  enabledNativeServer,
		},
		CGONativeClient: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.client.cgo.rpccgo.go", prefix, serviceName),
			Enabled:  true,
		},
	}
}
