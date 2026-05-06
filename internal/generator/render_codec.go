package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func RenderCodecFiles(plugin *protogen.Plugin, plan FilePlan) error {
	if plugin == nil {
		return fmt.Errorf("generator plugin is nil")
	}

	for _, service := range plan.Services {
		file := BuildCodecFilePlan(plan, service)
		if !file.Enabled {
			continue
		}
		renderCodecFile(plugin, plan, service, file)
	}
	return nil
}

func renderCodecFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`errors "errors"`)
	g.P(`fmt "fmt"`)
	g.P(`proto "google.golang.org/protobuf/proto"`)
	g.P(")")
	g.P()
	g.P("// rpccgo native message codec stage file for ", service.GoName)
	g.P()
	g.P("var ", lowerInitial(service.GoName), `NativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")`)
	g.P()

	for _, method := range service.Methods {
		renderCodecMethodStubs(g, service, method)
	}
}

func renderCodecMethodStubs(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	requestType := nativeRuntimeMessageType(g, method.Request)
	responseType := nativeRuntimeMessageType(g, method.Response)

	g.P("func ", codecMessageToNativeRequestName(service, method), "(data []byte) (", requestType, ", error) {")
	g.P("var msg ", strings.TrimPrefix(requestType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return &msg, nil")
	g.P("}")
	g.P()

	g.P("func ", codecNativeRequestToMessageName(service, method), "(req ", requestType, ") ([]byte, error) {")
	g.P("if req == nil {")
	g.P(`return nil, errors.New("rpccgo: native request is nil")`)
	g.P("}")
	g.P("data, err := proto.Marshal(req)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()

	g.P("func ", codecMessageToNativeResponseName(service, method), "(data []byte) (", responseType, ", error) {")
	g.P("var msg ", strings.TrimPrefix(responseType, "*"))
	g.P("if err := proto.Unmarshal(data, &msg); err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)`)
	g.P("}")
	g.P("return &msg, nil")
	g.P("}")
	g.P()

	g.P("func ", codecNativeResponseToMessageName(service, method), "(resp ", responseType, ") ([]byte, error) {")
	g.P("if resp == nil {")
	g.P(`return nil, errors.New("rpccgo: native response is nil")`)
	g.P("}")
	g.P("data, err := proto.Marshal(resp)")
	g.P("if err != nil {")
	g.P(`return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)`)
	g.P("}")
	g.P("return data, nil")
	g.P("}")
	g.P()
}

func codecMessageToNativeRequestName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "MessageToNativeRequest"
}

func codecNativeRequestToMessageName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "NativeToMessageRequest"
}

func codecMessageToNativeResponseName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "MessageToNativeResponse"
}

func codecNativeResponseToMessageName(service ServicePlan, method MethodPlan) string {
	return "convert" + service.GoName + method.GoName + "NativeToMessageResponse"
}
