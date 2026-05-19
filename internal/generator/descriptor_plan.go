package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func BuildDescriptorPlan(file *protogen.File) (FilePlan, error) {
	if file == nil {
		return FilePlan{}, fmt.Errorf("protogen file is nil")
	}

	plan := FilePlan{
		GoPackageName:           string(file.GoPackageName),
		GoImportPath:            string(file.GoImportPath),
		ProtoPath:               file.Desc.Path(),
		GeneratedFilenamePrefix: file.GeneratedFilenamePrefix,
		TopLevelSymbols:         buildTopLevelSymbolPlans(file),
		Services:                make([]ServicePlan, 0, len(file.Services)),
	}
	for _, service := range file.Services {
		servicePlan, err := buildServiceDescriptorPlan(service)
		if err != nil {
			return FilePlan{}, err
		}
		plan.Services = append(plan.Services, servicePlan)
	}
	return plan, nil
}

func buildTopLevelSymbolPlans(file *protogen.File) []TopLevelSymbolPlan {
	return buildFileSymbolPlans(file)
}

func buildPackageLevelSymbolPlans(files []*protogen.File, goImportPath string) []TopLevelSymbolPlan {
	var symbols []TopLevelSymbolPlan
	for _, file := range files {
		if file == nil || string(file.GoImportPath) != goImportPath {
			continue
		}
		symbols = append(symbols, buildFileSymbolPlans(file)...)
	}
	return symbols
}

func buildFileSymbolPlans(file *protogen.File) []TopLevelSymbolPlan {
	symbols := make([]TopLevelSymbolPlan, 0, len(file.Messages)+len(file.Enums)+len(file.Services))
	for _, message := range file.Messages {
		symbols = appendMessageSymbolPlans(symbols, message)
	}
	for _, enum := range file.Enums {
		symbols = appendEnumSymbolPlan(symbols, enum)
	}
	for _, service := range file.Services {
		symbols = append(symbols, TopLevelSymbolPlan{
			GoName:   service.GoName,
			FullName: string(service.Desc.FullName()),
			Kind:     TopLevelSymbolKindService,
		})
	}
	return symbols
}

func appendMessageSymbolPlans(symbols []TopLevelSymbolPlan, message *protogen.Message) []TopLevelSymbolPlan {
	symbols = append(symbols, TopLevelSymbolPlan{
		GoName:   message.GoIdent.GoName,
		FullName: string(message.Desc.FullName()),
		Kind:     TopLevelSymbolKindMessage,
	})
	for _, nested := range message.Messages {
		symbols = appendMessageSymbolPlans(symbols, nested)
	}
	for _, enum := range message.Enums {
		symbols = appendEnumSymbolPlan(symbols, enum)
	}
	return symbols
}

func appendEnumSymbolPlan(symbols []TopLevelSymbolPlan, enum *protogen.Enum) []TopLevelSymbolPlan {
	return append(symbols, TopLevelSymbolPlan{
		GoName:   enum.GoIdent.GoName,
		FullName: string(enum.Desc.FullName()),
		Kind:     TopLevelSymbolKindEnum,
	})
}

func buildServiceDescriptorPlan(service *protogen.Service) (ServicePlan, error) {
	if service == nil {
		return ServicePlan{}, fmt.Errorf("protogen service is nil")
	}

	adapters, err := ParseServiceRPCCGOOptions(string(service.Comments.Leading))
	if err != nil {
		return ServicePlan{}, fmt.Errorf("service %s: %w", service.Desc.FullName(), err)
	}

	plan := ServicePlan{
		Name:       string(service.Desc.Name()),
		GoName:     service.GoName,
		FullName:   string(service.Desc.FullName()),
		Adapters:   adapters,
		Methods:    make([]MethodPlan, 0, len(service.Methods)),
		NeedsCodec: serviceNeedsNativeMessageCodec(),
	}
	for _, method := range service.Methods {
		methodPlan, err := buildMethodDescriptorPlan(service, method, plan.GoName, plan.NeedsCodec)
		if err != nil {
			return ServicePlan{}, err
		}
		plan.Methods = append(plan.Methods, methodPlan)
	}
	return plan, nil
}

func serviceNeedsNativeMessageCodec() bool {
	return true
}

func buildMethodDescriptorPlan(service *protogen.Service, method *protogen.Method, serviceName string, needsCodec bool) (MethodPlan, error) {
	plan := MethodPlan{
		Name:       string(method.Desc.Name()),
		GoName:     method.GoName,
		FullName:   string(method.Desc.FullName()),
		Streaming:  StreamingKindOf(method.Desc.IsStreamingClient(), method.Desc.IsStreamingServer()),
		NeedsCodec: needsCodec,
		Request: MethodIOPlan{
			GoName:       method.Input.GoIdent.GoName,
			GoImportPath: string(method.Input.GoIdent.GoImportPath),
			FullName:     string(method.Input.Desc.FullName()),
		},
		Response: MethodIOPlan{
			GoName:       method.Output.GoIdent.GoName,
			GoImportPath: string(method.Output.GoIdent.GoImportPath),
			FullName:     string(method.Output.Desc.FullName()),
		},
	}
	facts, err := BuildContractPlan(service, method, plan)
	if err != nil {
		return MethodPlan{}, err
	}
	return BuildStreamingPlan(plan, facts, serviceName)
}
