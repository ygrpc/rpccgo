package generator

import "fmt"

type runtimeRegistrationBindingKind string

const (
	runtimeRegistrationBindingKindNative           runtimeRegistrationBindingKind = "native"
	runtimeRegistrationBindingKindCGONativeForward runtimeRegistrationBindingKind = "cgo_native_forward"
	runtimeRegistrationBindingKindMessage          runtimeRegistrationBindingKind = "message"
	runtimeRegistrationBindingKindTransportMessage runtimeRegistrationBindingKind = "transport_message"
)

type transportMessageStreamConstructorShape string

const (
	transportMessageStreamConstructorShapeNone          transportMessageStreamConstructorShape = ""
	transportMessageStreamConstructorShapeConnectLocal  transportMessageStreamConstructorShape = "connect_local"
	transportMessageStreamConstructorShapeConnectRemote transportMessageStreamConstructorShape = "connect_remote"
	transportMessageStreamConstructorShapeGRPCLocal     transportMessageStreamConstructorShape = "grpc_local"
	transportMessageStreamConstructorShapeGRPCRemote    transportMessageStreamConstructorShape = "grpc_remote"
)

type registrationSourceProjection struct {
	bindingKind                          runtimeRegistrationBindingKind
	registerName                         string
	inputName                            string
	inputType                            string
	nilErr                               string
	sourceExpr                           string
	label                                string
	transportStreamConstructorShape      transportMessageStreamConstructorShape
	transportStreamConstructorReturnsErr bool
}

func ProjectRegistrationSource(service ServicePlan, source RegistrationSourcePlan) (registrationSourceProjection, error) {
	if err := ValidateRegistrationSourcePlan(source); err != nil {
		return registrationSourceProjection{}, err
	}

	serviceName := service.GoName
	switch source {
	case RegistrationSourcePlan{Origin: RegistrationOriginGo, Contract: RegistrationContractNative, Transport: RegistrationTransportNone, Mode: RegistrationModeLocal}:
		return registrationSourceProjection{
			bindingKind:  runtimeRegistrationBindingKindNative,
			registerName: "register" + serviceName + "GoNativeServer",
			inputName:    "server",
			inputType:    serviceName + "NativeServer",
			nilErr:       serviceName + "NativeServerUnavailableErr",
			sourceExpr:   "server",
			label:        "go native",
		}, nil
	case RegistrationSourcePlan{Origin: RegistrationOriginCGO, Contract: RegistrationContractNative, Transport: RegistrationTransportNone, Mode: RegistrationModeLocal}:
		return registrationSourceProjection{
			bindingKind:  runtimeRegistrationBindingKindCGONativeForward,
			registerName: "Register" + serviceName + "CGONativeServer",
			inputName:    "server",
			inputType:    serviceName + "NativeServer",
			nilErr:       serviceName + "NativeServerUnavailableErr",
			sourceExpr:   "server",
			label:        "cgo native",
		}, nil
	case RegistrationSourcePlan{Origin: RegistrationOriginCGO, Contract: RegistrationContractMessage, Transport: RegistrationTransportNone, Mode: RegistrationModeLocal}:
		return registrationSourceProjection{
			bindingKind:  runtimeRegistrationBindingKindMessage,
			registerName: "register" + serviceName + "CGOMessageServer",
			inputName:    "server",
			inputType:    serviceName + "CGOMessageServer",
			nilErr:       serviceName + "MessageServerUnavailableErr",
			sourceExpr:   "server",
			label:        "cgo message",
		}, nil
	case RegistrationSourcePlan{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportConnect, Mode: RegistrationModeLocal}:
		return registrationSourceProjection{
			bindingKind:                     runtimeRegistrationBindingKindTransportMessage,
			registerName:                    "Register" + serviceName + "ConnectHandler",
			inputName:                       "handler",
			inputType:                       serviceName + "Handler",
			nilErr:                          serviceName + "MessageServerUnavailableErr",
			sourceExpr:                      "handler",
			label:                           "connect handler",
			transportStreamConstructorShape: transportMessageStreamConstructorShapeConnectLocal,
		}, nil
	case RegistrationSourcePlan{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportConnect, Mode: RegistrationModeRemote}:
		return registrationSourceProjection{
			bindingKind:                          runtimeRegistrationBindingKindTransportMessage,
			registerName:                         "Register" + serviceName + "ConnectRemoteServer",
			inputName:                            "client",
			inputType:                            serviceName + "Client",
			nilErr:                               serviceName + "MessageServerUnavailableErr",
			sourceExpr:                           "client",
			label:                                "connect remote",
			transportStreamConstructorShape:      transportMessageStreamConstructorShapeConnectRemote,
			transportStreamConstructorReturnsErr: true,
		}, nil
	case RegistrationSourcePlan{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportGRPC, Mode: RegistrationModeLocal}:
		return registrationSourceProjection{
			bindingKind:                     runtimeRegistrationBindingKindTransportMessage,
			registerName:                    "Register" + serviceName + "GRPCServer",
			inputName:                       "server",
			inputType:                       serviceName + "Server",
			nilErr:                          serviceName + "MessageServerUnavailableErr",
			sourceExpr:                      "server",
			label:                           "grpc server",
			transportStreamConstructorShape: transportMessageStreamConstructorShapeGRPCLocal,
		}, nil
	case RegistrationSourcePlan{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportGRPC, Mode: RegistrationModeRemote}:
		return registrationSourceProjection{
			bindingKind:                          runtimeRegistrationBindingKindTransportMessage,
			registerName:                         "Register" + serviceName + "GRPCRemoteServer",
			inputName:                            "client",
			inputType:                            serviceName + "Client",
			nilErr:                               serviceName + "MessageServerUnavailableErr",
			sourceExpr:                           "client",
			label:                                "grpc remote",
			transportStreamConstructorShape:      transportMessageStreamConstructorShapeGRPCRemote,
			transportStreamConstructorReturnsErr: true,
		}, nil
	default:
		return registrationSourceProjection{}, fmt.Errorf("unknown registration source projection origin=%q contract=%q transport=%q mode=%q", source.Origin, source.Contract, source.Transport, source.Mode)
	}
}

func registrationTransportMessageStreamConstructor(service ServicePlan, method runtimeAdapterMethod, projection registrationSourceProjection) (string, bool, error) {
	if projection.bindingKind != runtimeRegistrationBindingKindTransportMessage {
		return "", false, fmt.Errorf("registration source %q is not a transport message binding", projection.label)
	}

	var constructor string
	switch projection.transportStreamConstructorShape {
	case transportMessageStreamConstructorShapeConnectLocal:
		constructor = "new" + connectDirectMessageSessionName(service.GoName, method)
	case transportMessageStreamConstructorShapeConnectRemote:
		constructor = "new" + connectRemoteMessageSessionName(service.GoName, method)
	case transportMessageStreamConstructorShapeGRPCLocal:
		constructor = "new" + grpcDirectMessageSessionName(service.GoName, method)
	case transportMessageStreamConstructorShapeGRPCRemote:
		constructor = "new" + grpcRemoteMessageSessionName(service.GoName, method)
	default:
		return "", false, fmt.Errorf("registration source %q does not define a transport stream constructor", projection.label)
	}

	return constructor, projection.transportStreamConstructorReturnsErr || runtimeStreamShapeFor(method) == runtimeStreamServer, nil
}
