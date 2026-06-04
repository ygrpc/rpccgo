package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeRegistrations(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, currentNativeBindingName, currentMessageBindingName, nativeBindingName, messageBindingName, nativeActiveBindingName, messageActiveBindingName string) error {
	ctx := runtimeRegistrationRenderContext{
		service:                   service,
		methods:                   methods,
		currentNativeBindingName:  currentNativeBindingName,
		currentMessageBindingName: currentMessageBindingName,
		nativeBindingName:         nativeBindingName,
		messageBindingName:        messageBindingName,
		nativeActiveBindingName:   nativeActiveBindingName,
		messageActiveBindingName:  messageActiveBindingName,
	}

	for _, source := range registrationSourcesForService(service) {
		if err := renderRuntimeRegistrationForSource(g, ctx, source); err != nil {
			return err
		}
	}
	return nil
}

type runtimeRegistrationRenderContext struct {
	service                   ServicePlan
	methods                   []runtimeMethodProjection
	currentNativeBindingName  string
	currentMessageBindingName string
	nativeBindingName         string
	messageBindingName        string
	nativeActiveBindingName   string
	messageActiveBindingName  string
}

func renderRuntimeRegistrationForSource(g *protogen.GeneratedFile, ctx runtimeRegistrationRenderContext, source RegistrationSourcePlan) error {
	serviceName := ctx.service.GoName
	projection, err := ProjectRegistrationSource(ctx.service, source)
	if err != nil {
		return err
	}

	switch projection.bindingKind {
	case runtimeRegistrationBindingKindCGONativeForward:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("return register", serviceName, "GoNativeServer(", projection.sourceExpr, ")")
		g.P("}")
		g.P()
	case runtimeRegistrationBindingKindNative:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		renderRuntimeServerBindingLiteral(g, ctx.nativeBindingName, projection.sourceExpr, ctx.methods, "serverBinding")
		renderRuntimeNativeBinding(g, ctx.service, ctx.methods, ctx.currentNativeBindingName, ctx.nativeActiveBindingName, "serverBinding")
		g.P("}")
		g.P()
	case runtimeRegistrationBindingKindMessage:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		renderRuntimeServerBindingLiteral(g, ctx.messageBindingName, projection.sourceExpr, ctx.methods, "serverBinding")
		renderRuntimeMessageBinding(g, ctx.service, ctx.methods, ctx.currentMessageBindingName, ctx.messageActiveBindingName, "serverBinding")
		g.P("}")
		g.P()
	case runtimeRegistrationBindingKindTransportMessage:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		if err := renderRuntimeTransportMessageBinding(g, ctx.service, ctx.methods, ctx.currentMessageBindingName, ctx.messageActiveBindingName, projection.sourceExpr, projection); err != nil {
			return err
		}
		g.P("}")
		g.P()
	default:
		return fmt.Errorf("unknown runtime registration binding kind %q", projection.bindingKind)
	}
	return nil
}

func renderRuntimeServerBindingLiteral(g *protogen.GeneratedFile, bindingName, sourceExpr string, methods []runtimeMethodProjection, targetName string) {
	g.P(targetName, " := &", bindingName, "{")
	for _, method := range methods {
		g.P(lowerInitial(method.Identity.GoName), ": ", sourceExpr, ".", method.Identity.GoName, ",")
	}
	g.P("}")
}
