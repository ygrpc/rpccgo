package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeRegistrations(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeAdapterMethod, activeName string) error {
	serviceName := service.GoName
	ctx := runtimeRegistrationRenderContext{
		service:        service,
		nativeAdapter:  lowerInitial(serviceName) + "NativeServerAdapter",
		messageAdapter: lowerInitial(serviceName) + "MessageServerAdapter",
		methods:        methods,
		activeName:     activeName,
		recordName:     lowerInitial(serviceName) + "ActiveServerRecord",
	}

	for _, source := range activeRecordSourcesForService(service) {
		if err := renderRuntimeRegistrationForSource(g, ctx, source); err != nil {
			return err
		}
	}
	return nil
}

type runtimeRegistrationRenderContext struct {
	service        ServicePlan
	nativeAdapter  string
	messageAdapter string
	methods        []runtimeAdapterMethod
	activeName     string
	recordName     string
}

func renderRuntimeRegistrationForSource(g *protogen.GeneratedFile, ctx runtimeRegistrationRenderContext, source ActiveRecordSourcePlan) error {
	serviceName := ctx.service.GoName
	projection, err := projectRuntimeRegistrationSource(ctx.service, source)
	if err != nil {
		return err
	}

	switch projection.recordKind {
	case runtimeRegistrationRecordKindGoNativeAlias:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("return register", serviceName, "GoNativeServer(", projection.sourceExpr, ")")
		g.P("}")
		g.P()
	case runtimeRegistrationRecordKindNative:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		g.P("adapter := &", ctx.nativeAdapter, "{server: ", projection.sourceExpr, "}")
		renderRuntimeNativeRecord(g, ctx.service, ctx.methods, ctx.activeName, ctx.recordName, "adapter")
		g.P("}")
		g.P()
	case runtimeRegistrationRecordKindMessage:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		g.P("adapter := &", ctx.messageAdapter, "{server: ", projection.sourceExpr, "}")
		renderRuntimeMessageRecord(g, ctx.service, ctx.methods, ctx.activeName, ctx.recordName, "adapter")
		g.P("}")
		g.P()
	case runtimeRegistrationRecordKindTransportMessage:
		g.P("func ", projection.registerName, "(", projection.inputName, " ", projection.inputType, ") error {")
		g.P("if ", projection.inputName, " == nil { return ", projection.nilErr, " }")
		if err := renderRuntimeTransportMessageRecord(g, ctx.service, ctx.methods, ctx.activeName, ctx.recordName, projection.sourceExpr, projection.label, source); err != nil {
			return err
		}
		g.P("}")
		g.P()
	default:
		return fmt.Errorf("unknown runtime registration record kind %q", projection.recordKind)
	}
	return nil
}

type runtimeRegistrationRecordKind string

const (
	runtimeRegistrationRecordKindNative           runtimeRegistrationRecordKind = "native"
	runtimeRegistrationRecordKindGoNativeAlias    runtimeRegistrationRecordKind = "go_native_alias"
	runtimeRegistrationRecordKindMessage          runtimeRegistrationRecordKind = "message"
	runtimeRegistrationRecordKindTransportMessage runtimeRegistrationRecordKind = "transport_message"
)

type runtimeRegistrationSourceProjection struct {
	recordKind   runtimeRegistrationRecordKind
	registerName string
	inputName    string
	inputType    string
	nilErr       string
	sourceExpr   string
	label        string
}

func projectRuntimeRegistrationSource(service ServicePlan, source ActiveRecordSourcePlan) (runtimeRegistrationSourceProjection, error) {
	if err := ValidateActiveRecordSourcePlan(source); err != nil {
		return runtimeRegistrationSourceProjection{}, err
	}

	serviceName := service.GoName
	switch source {
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractNative, Transport: ActiveRecordTransportNone, Mode: ActiveRecordModeLocal}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindNative,
			registerName: "register" + serviceName + "GoNativeServer",
			inputName:    "server",
			inputType:    serviceName + "NativeServer",
			nilErr:       serviceName + "NativeServerUnavailableErr",
			sourceExpr:   "server",
			label:        "go native",
		}, nil
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginCGO, Contract: ActiveRecordContractNative, Transport: ActiveRecordTransportNone, Mode: ActiveRecordModeLocal}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindGoNativeAlias,
			registerName: "Register" + serviceName + "CGONativeServer",
			inputName:    "server",
			inputType:    serviceName + "NativeServer",
			nilErr:       serviceName + "NativeServerUnavailableErr",
			sourceExpr:   "server",
			label:        "cgo native",
		}, nil
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginCGO, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportNone, Mode: ActiveRecordModeLocal}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindMessage,
			registerName: "register" + serviceName + "CGOMessageServer",
			inputName:    "server",
			inputType:    serviceName + "CGOMessageServer",
			nilErr:       serviceName + "MessageServerUnavailableErr",
			sourceExpr:   "server",
			label:        "cgo message",
		}, nil
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportConnect, Mode: ActiveRecordModeLocal}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindTransportMessage,
			registerName: "Register" + serviceName + "ConnectHandler",
			inputName:    "handler",
			inputType:    serviceName + "Handler",
			nilErr:       serviceName + "MessageServerUnavailableErr",
			sourceExpr:   "handler",
			label:        "connect handler",
		}, nil
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportConnect, Mode: ActiveRecordModeRemote}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindTransportMessage,
			registerName: "Register" + serviceName + "ConnectRemoteServer",
			inputName:    "client",
			inputType:    serviceName + "Client",
			nilErr:       serviceName + "MessageServerUnavailableErr",
			sourceExpr:   "client",
			label:        "connect remote",
		}, nil
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportGRPC, Mode: ActiveRecordModeLocal}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindTransportMessage,
			registerName: "Register" + serviceName + "GRPCServer",
			inputName:    "server",
			inputType:    serviceName + "Server",
			nilErr:       serviceName + "MessageServerUnavailableErr",
			sourceExpr:   "server",
			label:        "grpc server",
		}, nil
	case ActiveRecordSourcePlan{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportGRPC, Mode: ActiveRecordModeRemote}:
		return runtimeRegistrationSourceProjection{
			recordKind:   runtimeRegistrationRecordKindTransportMessage,
			registerName: "Register" + serviceName + "GRPCRemoteServer",
			inputName:    "client",
			inputType:    serviceName + "Client",
			nilErr:       serviceName + "MessageServerUnavailableErr",
			sourceExpr:   "client",
			label:        "grpc remote",
		}, nil
	default:
		return runtimeRegistrationSourceProjection{}, fmt.Errorf("unknown active record source projection origin=%q contract=%q transport=%q mode=%q", source.Origin, source.Contract, source.Transport, source.Mode)
	}
}
