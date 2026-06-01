package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeRegistrations(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, codecEnabled bool, activeName string) {
	serviceName := service.GoName
	ctx := runtimeRegistrationRenderContext{
		service:        service,
		nativeAdapter:  lowerInitial(serviceName) + "NativeServerAdapter",
		messageAdapter: lowerInitial(serviceName) + "MessageServerAdapter",
		methods:        methods,
		codecEnabled:   codecEnabled,
		activeName:     activeName,
		recordName:     lowerInitial(serviceName) + "ActiveServerRecord",
	}

	for _, source := range activeRecordSourcesForService(service) {
		renderRuntimeRegistrationForSource(g, ctx, source)
	}
}

type runtimeRegistrationRenderContext struct {
	service        ServicePlan
	nativeAdapter  string
	messageAdapter string
	methods        []runtimeAdapterMethod
	codecEnabled   bool
	activeName     string
	recordName     string
}

func renderRuntimeRegistrationForSource(g *protogen.GeneratedFile, ctx runtimeRegistrationRenderContext, source ActiveRecordSourcePlan) {
	serviceName := ctx.service.GoName

	switch {
	case source.AliasGoNativeRegistration:
		g.P("func ", source.RegisterName, "(", source.InputName, " ", source.InputType, ") error {")
		g.P("return register", serviceName, "GoNativeServer(", source.SourceExpr, ")")
		g.P("}")
		g.P()
	case source.RecordRenderer == ActiveRecordRendererNative:
		g.P("func ", source.RegisterName, "(", source.InputName, " ", source.InputType, ") error {")
		g.P("if ", source.InputName, " == nil { return ", source.NilErr, " }")
		g.P("adapter := &", ctx.nativeAdapter, "{server: ", source.SourceExpr, "}")
		renderRuntimeNativeRecord(g, ctx.service, ctx.methods, ctx.codecEnabled, ctx.activeName, ctx.recordName, "adapter")
		g.P("}")
		g.P()
	case source.RecordRenderer == ActiveRecordRendererMessage:
		g.P("func ", source.RegisterName, "(", source.InputName, " ", source.InputType, ") error {")
		g.P("if ", source.InputName, " == nil { return ", source.NilErr, " }")
		g.P("adapter := &", ctx.messageAdapter, "{server: ", source.SourceExpr, "}")
		renderRuntimeMessageRecord(g, ctx.service, ctx.methods, ctx.codecEnabled, ctx.activeName, ctx.recordName, "adapter")
		g.P("}")
		g.P()
	case source.RecordRenderer == ActiveRecordRendererTransportMessage:
		g.P("func ", source.RegisterName, "(", source.InputName, " ", source.InputType, ") error {")
		g.P("if ", source.InputName, " == nil { return ", source.NilErr, " }")
		renderRuntimeTransportMessageRecord(g, ctx.service, ctx.methods, ctx.codecEnabled, ctx.activeName, ctx.recordName, source.SourceExpr, source.Label)
		g.P("}")
		g.P()
	default:
		panic("unknown active record source")
	}
}
