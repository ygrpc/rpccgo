package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeRegistrations(g *protogen.GeneratedFile, service ServicePlan, serviceIDName string) error {
	ctx := runtimeRegistrationRenderContext{
		service:       service,
		serviceIDName: serviceIDName,
	}

	for _, source := range registrationSourcesForService(service) {
		if err := renderRuntimeRegistrationForSource(g, ctx, source); err != nil {
			return err
		}
	}
	return nil
}

type runtimeRegistrationRenderContext struct {
	service       ServicePlan
	serviceIDName string
}

func renderRuntimeRegistrationForSource(g *protogen.GeneratedFile, ctx runtimeRegistrationRenderContext, source RegistrationSourcePlan) error {
	projection, err := ProjectRegistrationSource(ctx.service, source)
	if err != nil {
		return err
	}
	if projection.registrationKind != runtimeRegistrationKindTransportMessage {
		return nil
	}

	switch projection.registrationKind {
	case runtimeRegistrationKindTransportMessage:
		renderRuntimeServerRegistration(g, ctx.serviceIDName, projection)
	default:
		return fmt.Errorf("unknown runtime registration kind %q", projection.registrationKind)
	}
	return nil
}

func renderRuntimeServerRegistration(g *protogen.GeneratedFile, serviceIDName string, projection registrationSourceProjection) {
	renderDoc(g, projection.registerName, "registers the supplied "+projection.label+" server as the current server for this service.")
	g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
	g.P("if ", projection.inputName, " == nil {")
	g.P("_ = rpcruntime.ClearServer(", serviceIDName, ")")
	g.P("return ", projection.nilErr)
	g.P("}")
	g.P("err := rpcruntime.RegisterServer(", serviceIDName, ", rpcruntime.RegisteredServer{")
	g.P("Kind: ", projection.serverKind, ",")
	g.P("Server: ", projection.sourceExpr, ",")
	g.P("})")
	g.P("if err != nil {")
	g.P("_ = rpcruntime.ClearServer(", serviceIDName, ")")
	g.P("return err")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}
