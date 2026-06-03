package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeRegistrations(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, currentBindingName, activeServerName, nativeBindingName, messageBindingName, nativeCallerBindingName, messageCallerBindingName string) error {
	ctx := runtimeRegistrationRenderContext{
		service:                  service,
		methods:                  methods,
		currentBindingName:       currentBindingName,
		nativeBindingName:        nativeBindingName,
		messageBindingName:       messageBindingName,
		activeServerName:         activeServerName,
		nativeCallerBindingName:  nativeCallerBindingName,
		messageCallerBindingName: messageCallerBindingName,
	}

	for _, source := range registrationSourcesForService(service) {
		if err := renderRuntimeRegistrationForSource(g, ctx, source); err != nil {
			return err
		}
	}
	return nil
}

type runtimeRegistrationRenderContext struct {
	service                  ServicePlan
	methods                  []runtimeMethodProjection
	currentBindingName       string
	nativeBindingName        string
	messageBindingName       string
	activeServerName         string
	nativeCallerBindingName  string
	messageCallerBindingName string
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
		g.P("serverBinding := &", ctx.nativeBindingName, "{server: ", projection.sourceExpr, "}")
		renderRuntimeNativeBinding(g, ctx.service, ctx.methods, ctx.currentBindingName, ctx.activeServerName, ctx.nativeCallerBindingName, ctx.messageCallerBindingName, "serverBinding")
		g.P("}")
		g.P()
	case runtimeRegistrationBindingKindMessage:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		g.P("serverBinding := &", ctx.messageBindingName, "{server: ", projection.sourceExpr, "}")
		renderRuntimeMessageBinding(g, ctx.service, ctx.methods, ctx.currentBindingName, ctx.activeServerName, ctx.nativeCallerBindingName, ctx.messageCallerBindingName, "serverBinding")
		g.P("}")
		g.P()
	case runtimeRegistrationBindingKindTransportMessage:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		if err := renderRuntimeTransportMessageBinding(g, ctx.service, ctx.methods, ctx.currentBindingName, ctx.activeServerName, ctx.nativeCallerBindingName, ctx.messageCallerBindingName, projection.sourceExpr, projection); err != nil {
			return err
		}
		g.P("}")
		g.P()
	default:
		return fmt.Errorf("unknown runtime registration binding kind %q", projection.bindingKind)
	}
	return nil
}
