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
	runtimeMethods, err := buildRuntimeMethodProjectionsWithMessageTypes(g, service, false)
	if err != nil {
		return err
	}

	serverName := service.GoName + "CGOMessageServer"

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if serviceHasStreamingMethod(service) {
		g.P(`io "io"`)
		if serviceHasClientStreamingMethod(service) || serviceHasBidiStreamingMethod(service) {
			g.P(`sync "sync"`)
		}
	}
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()
	renderDocLine(g, service.DocComment, "type ", serverName, " interface {")
	for _, method := range runtimeMethods {
		switch {
		case !method.Stream.Streaming:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, req []byte) ([]byte, error)")
		case method.Stream.CanSend && method.Stream.FinishReturnsResponse:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, stream ", service.GoName, method.Identity.GoName, "MessageClientStream) ([]byte, error)")
		case method.Stream.CanRecv && !method.Stream.CanSend:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, req []byte, stream ", service.GoName, method.Identity.GoName, "MessageServerStream) error")
		case method.Stream.CanSend && method.Stream.CanRecv && method.Stream.CanCloseSend:
			renderDocLine(g, method.Identity.DocComment, method.Identity.GoName, "(ctx context.Context, stream ", service.GoName, method.Identity.GoName, "MessageBidiStream) error")
		}
	}
	g.P("}")
	g.P()
	renderCGOMessageServerStreamInterfaces(g, service, runtimeMethods)
	renderUnimplementedCGOMessageServer(g, service, runtimeMethods)
	g.P("func Register", service.GoName, "CGOMessageServer(server ", serverName, ") error {")
	g.P("if server == nil {")
	g.P(`return errors.New("rpccgo: `, service.GoName, ` cgo message server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "CGOMessageServer(server)")
	g.P("}")
	g.P()
	renderMessageEntry(g, service, runtimeMethods, serverName, lowerInitial(service.GoName)+"CGOMessageEntry")
	return nil
}

func renderCGOMessageServerStreamInterfaces(g *protogen.GeneratedFile, service ServicePlan, runtimeMethods []runtimeMethodProjection) {
	for _, method := range runtimeMethods {
		switch {
		case method.Stream.CanSend && method.Stream.FinishReturnsResponse:
			g.P("type ", service.GoName, method.Identity.GoName, "MessageClientStream interface {")
			g.P("Recv(ctx context.Context) ([]byte, error)")
			g.P("}")
			g.P()
		case method.Stream.CanRecv && !method.Stream.CanSend:
			g.P("type ", service.GoName, method.Identity.GoName, "MessageServerStream interface {")
			g.P("Send(ctx context.Context, resp []byte) error")
			g.P("}")
			g.P()
		case method.Stream.CanSend && method.Stream.CanRecv && method.Stream.CanCloseSend:
			g.P("type ", service.GoName, method.Identity.GoName, "MessageBidiStream interface {")
			g.P("Recv(ctx context.Context) ([]byte, error)")
			g.P("Send(ctx context.Context, resp []byte) error")
			g.P("}")
			g.P()
		}
	}
}

func renderUnimplementedCGOMessageServer(g *protogen.GeneratedFile, service ServicePlan, runtimeMethods []runtimeMethodProjection) {
	serverName := "Unimplemented" + service.GoName + "CGOMessageServer"
	g.P("type ", serverName, " struct{}")
	g.P()
	for _, method := range runtimeMethods {
		errExpr := `errors.New("rpccgo: ` + service.GoName + "." + method.Identity.GoName + ` cgo message server method is not implemented")`
		switch {
		case !method.Stream.Streaming:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, req []byte) ([]byte, error) {")
			g.P("return nil, ", errExpr)
			g.P("}")
		case method.Stream.CanSend && method.Stream.FinishReturnsResponse:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, stream ", service.GoName, method.Identity.GoName, "MessageClientStream) ([]byte, error) {")
			g.P("return nil, ", errExpr)
			g.P("}")
		case method.Stream.CanRecv && !method.Stream.CanSend:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, req []byte, stream ", service.GoName, method.Identity.GoName, "MessageServerStream) error {")
			g.P("return ", errExpr)
			g.P("}")
		case method.Stream.CanSend && method.Stream.CanRecv && method.Stream.CanCloseSend:
			g.P("func (", serverName, ") ", method.Identity.GoName, "(ctx context.Context, stream ", service.GoName, method.Identity.GoName, "MessageBidiStream) error {")
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
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindClientStreaming:
			if err := addGenerated(service.GoName+method.GoName+"MessageClientStream", method.FullName+" cgo message client stream interface"); err != nil {
				return err
			}
		case StreamingKindServerStreaming:
			if err := addGenerated(service.GoName+method.GoName+"MessageServerStream", method.FullName+" cgo message server stream interface"); err != nil {
				return err
			}
		case StreamingKindBidiStreaming:
			if err := addGenerated(service.GoName+method.GoName+"MessageBidiStream", method.FullName+" cgo message bidi stream interface"); err != nil {
				return err
			}
		}
	}
	return nil
}
