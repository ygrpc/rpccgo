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
		GoPackageName: string(file.GoPackageName),
		GoImportPath:  string(file.GoImportPath),
		ProtoPath:     file.Desc.Path(),
		Services:      make([]ServicePlan, 0, len(file.Services)),
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

func buildServiceDescriptorPlan(service *protogen.Service) (ServicePlan, error) {
	if service == nil {
		return ServicePlan{}, fmt.Errorf("protogen service is nil")
	}

	adapters, err := ParseServiceRPCCGOOptions(string(service.Comments.Leading))
	if err != nil {
		return ServicePlan{}, fmt.Errorf("service %s: %w", service.Desc.FullName(), err)
	}

	plan := ServicePlan{
		Name:     string(service.Desc.Name()),
		GoName:   service.GoName,
		FullName: string(service.Desc.FullName()),
		Adapters: adapters,
		Methods:  make([]MethodPlan, 0, len(service.Methods)),
	}
	for _, method := range service.Methods {
		methodPlan, err := buildMethodDescriptorPlan(service, method)
		if err != nil {
			return ServicePlan{}, err
		}
		plan.Methods = append(plan.Methods, methodPlan)
	}
	return plan, nil
}

func buildMethodDescriptorPlan(service *protogen.Service, method *protogen.Method) (MethodPlan, error) {
	plan := MethodPlan{
		Name:      string(method.Desc.Name()),
		GoName:    method.GoName,
		FullName:  string(method.Desc.FullName()),
		Streaming: StreamingKindOf(method.Desc.IsStreamingClient(), method.Desc.IsStreamingServer()),
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
	return BuildContractPlan(service, method, plan)
}
