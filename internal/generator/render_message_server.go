package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
	if err := validateMessageServerSymbols(service); err != nil {
		return err
	}
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))
	runtimeMethods, err := buildRuntimeMethodProjectionsWithMessageTypes(g, service, true)
	if err != nil {
		return err
	}

	serverName := service.GoName + "CGOMessageServer"
	streamingMethods := runtimeStreamingMethodProjections(runtimeMethods)

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if serviceHasServerStreamingMethod(service) {
		g.P(`io "io"`)
	}
	if serviceHasStreamingMethod(service) {
		g.P(`fmt "fmt"`)
		if messageServerNeedsGoRuntime(service) {
			g.P(`goruntime "runtime"`)
		}
	}
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()
	if service.DocComment == "" {
		renderDoc(g, serverName, "defines the cgo message server contract for "+service.GoName+".")
		g.P("type ", serverName, " interface {")
	} else {
		renderDocLine(g, service.DocComment, "type ", serverName, " interface {")
	}
	for _, method := range runtimeMethods {
		switch {
		case !method.Stream.Streaming:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, req *", method.Message.RequestType, ") (*", method.Message.ResponseType, ", error)")
		case method.Stream.CanSend && method.Stream.FinishReturnsResponse:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, stream rpcruntime.ClientStreamingServer[*", method.Message.RequestType, "]) (*", method.Message.ResponseType, ", error)")
		case method.Stream.CanRecv && !method.Stream.CanSend:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, req *", method.Message.RequestType, ", stream rpcruntime.ServerStreamingServer[*", method.Message.ResponseType, "]) error")
		case method.Stream.CanSend && method.Stream.CanRecv && method.Stream.CanCloseSend:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, stream rpcruntime.BidiStreamingServer[*", method.Message.RequestType, ", *", method.Message.ResponseType, "]) error")
		}
	}
	g.P("}")
	g.P()
	for _, method := range streamingMethods {
		renderRuntimeMessageStreamFacade(g, service.GoName, method, service.Generation.NativeEnabled)
	}
	renderUnimplementedCGOMessageServer(g, service, runtimeMethods)
	renderDoc(g, "Register"+service.GoName+"CGOMessageServer", "registers a cgo message server as the current server for "+service.GoName+".")
	g.P("func Register", service.GoName, "CGOMessageServer(server ", serverName, ") error {")
	g.P("if server == nil {")
	g.P("_ = register", service.GoName, "CGOMessageServer(server)")
	g.P(`return errors.New("rpccgo: `, service.GoName, ` cgo message server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "CGOMessageServer(server)")
	g.P("}")
	g.P()
	if err := renderCGOMessageServerRuntimeRegistration(g, service); err != nil {
		return err
	}
	renderMessageStartHelpers(g, service, runtimeMethods, serverName)
	return nil
}

func messageServerNeedsGoRuntime(service ServicePlan) bool {
	if !service.Generation.NativeEnabled {
		return false
	}
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindClientStreaming || method.Streaming == StreamingKindBidiStreaming {
			return true
		}
	}
	return false
}

func renderCGOMessageServerRuntimeRegistration(g *protogen.GeneratedFile, service ServicePlan) error {
	projection, err := ProjectRegistrationSource(service, RegistrationSourceCGOMessage)
	if err != nil {
		return err
	}
	renderRuntimeServerRegistration(g, lowerInitial(service.GoName)+"ServiceID", projection)
	return nil
}

func renderUnimplementedCGOMessageServer(g *protogen.GeneratedFile, service ServicePlan, runtimeMethods []runtimeMethodProjection) {
	serverName := "Unimplemented" + service.GoName + "CGOMessageServer"
	renderDoc(g, serverName, "provides default unimplemented cgo message server methods for "+service.GoName+".")
	g.P("type ", serverName, " struct{}")
	g.P()
	for _, method := range runtimeMethods {
		errExpr := `errors.New("rpccgo: ` + service.GoName + "." + method.Identity.GoName + ` cgo message server method is not implemented")`
		renderDoc(g, method.Identity.GoName, "returns an unimplemented error for the "+service.GoName+" cgo message "+method.Identity.GoName+" method.")
		switch {
		case !method.Stream.Streaming:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, req *", method.Message.RequestType, ") (*", method.Message.ResponseType, ", error) {")
			g.P("return nil, ", errExpr)
			g.P("}")
		case method.Stream.CanSend && method.Stream.FinishReturnsResponse:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, stream rpcruntime.ClientStreamingServer[*", method.Message.RequestType, "]) (*", method.Message.ResponseType, ", error) {")
			g.P("return nil, ", errExpr)
			g.P("}")
		case method.Stream.CanRecv && !method.Stream.CanSend:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, req *", method.Message.RequestType, ", stream rpcruntime.ServerStreamingServer[*", method.Message.ResponseType, "]) error {")
			g.P("return ", errExpr)
			g.P("}")
		case method.Stream.CanSend && method.Stream.CanRecv && method.Stream.CanCloseSend:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, stream rpcruntime.BidiStreamingServer[*", method.Message.RequestType, ", *", method.Message.ResponseType, "]) error {")
			g.P("return ", errExpr)
			g.P("}")
		}
		g.P()
	}
}

func validateMessageServerSymbols(service ServicePlan) error {
	seen := make(map[string]string)
	messageTypes := make(map[string]string)
	for _, method := range service.Methods {
		if method.Request.GoName != "" {
			messageTypes[method.Request.GoName] = method.FullName + " request"
		}
		if method.Response.GoName != "" {
			messageTypes[method.Response.GoName] = method.FullName + " response"
		}
	}

	addGenerated := func(symbol, source string) error {
		if symbol == "" {
			return nil
		}
		if previous, exists := seen[symbol]; exists {
			return fmt.Errorf("message server symbol %s for %s collides with %s", symbol, source, previous)
		}
		if messageSource, exists := messageTypes[symbol]; exists {
			return fmt.Errorf("message server symbol %s for %s collides with protobuf message type from %s", symbol, source, messageSource)
		}
		seen[symbol] = source
		return nil
	}

	if err := addGenerated(service.GoName+"CGOMessageServer", service.FullName+" cgo message server interface"); err != nil {
		return err
	}
	if err := addGenerated("Unimplemented"+service.GoName+"CGOMessageServer", service.FullName+" unimplemented cgo message server helper"); err != nil {
		return err
	}
	if err := addGenerated("Register"+service.GoName+"CGOMessageServer", service.FullName+" cgo message server registration"); err != nil {
		return err
	}
	return nil
}
