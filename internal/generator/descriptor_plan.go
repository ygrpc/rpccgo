package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func BuildDescriptorPlan(file *protogen.File) (FilePlan, error) {
	plan, err := BuildFileDescriptorPlan(file)
	if err != nil {
		return FilePlan{}, err
	}
	if err := AttachMethodContractPlans(&plan, file); err != nil {
		return FilePlan{}, err
	}
	if err := AttachMethodLifecyclePlans(&plan); err != nil {
		return FilePlan{}, err
	}
	if err := AttachMethodRenderPlans(&plan); err != nil {
		return FilePlan{}, err
	}
	return plan, nil
}

func BuildFileDescriptorPlan(file *protogen.File) (FilePlan, error) {
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

func AttachMethodContractPlans(plan *FilePlan, file *protogen.File) error {
	if plan == nil {
		return fmt.Errorf("file plan is nil")
	}
	if file == nil {
		return fmt.Errorf("protogen file is nil")
	}
	if len(plan.Services) != len(file.Services) {
		return fmt.Errorf("file plan service count does not match descriptor service count")
	}
	for si := range plan.Services {
		service := file.Services[si]
		if len(plan.Services[si].Methods) != len(service.Methods) {
			return fmt.Errorf("service %s method count does not match descriptor method count", plan.Services[si].FullName)
		}
		for mi := range plan.Services[si].Methods {
			method := plan.Services[si].Methods[mi]
			contract, err := BuildContractPlan(service, service.Methods[mi], method)
			if err != nil {
				return err
			}
			plan.Services[si].Methods[mi].Contract = contract
		}
	}
	return nil
}

func AttachMethodLifecyclePlans(plan *FilePlan) error {
	if plan == nil {
		return fmt.Errorf("file plan is nil")
	}
	for si := range plan.Services {
		for mi := range plan.Services[si].Methods {
			method, err := AttachMethodLifecyclePlan(plan.Services[si].Methods[mi])
			if err != nil {
				return err
			}
			plan.Services[si].Methods[mi] = method
		}
	}
	return nil
}

func AttachMethodRenderPlans(plan *FilePlan) error {
	if plan == nil {
		return fmt.Errorf("file plan is nil")
	}
	for si := range plan.Services {
		for mi := range plan.Services[si].Methods {
			method := plan.Services[si].Methods[mi]
			renderPlan, err := BuildMethodRenderPlan(method, plan.Services[si].GoName)
			if err != nil {
				return err
			}
			method.RenderPlan = renderPlan
			if err := ValidateMethodContractPlan(method); err != nil {
				return err
			}
			if err := ValidateMethodRenderPlan(method); err != nil {
				return err
			}
			plan.Services[si].Methods[mi] = method
		}
	}
	return nil
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

	generation, err := ParseServiceRPCCGOOptions(string(service.Comments.Leading))
	if err != nil {
		return ServicePlan{}, fmt.Errorf("service %s: %w", service.Desc.FullName(), err)
	}

	plan := ServicePlan{
		Name:       string(service.Desc.Name()),
		GoName:     service.GoName,
		FullName:   string(service.Desc.FullName()),
		DocComment: protoDocComment(string(service.Comments.Leading)),
		Generation: generation,
		Methods:    make([]MethodPlan, 0, len(service.Methods)),
	}
	for _, method := range service.Methods {
		methodPlan := buildMethodDescriptorPlan(method)
		plan.Methods = append(plan.Methods, methodPlan)
	}
	return plan, nil
}

func buildMethodDescriptorPlan(method *protogen.Method) MethodPlan {
	return MethodPlan{
		Name:       string(method.Desc.Name()),
		GoName:     method.GoName,
		FullName:   string(method.Desc.FullName()),
		DocComment: protoDocComment(string(method.Comments.Leading)),
		Streaming:  StreamingKindOf(method.Desc.IsStreamingClient(), method.Desc.IsStreamingServer()),
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
}
