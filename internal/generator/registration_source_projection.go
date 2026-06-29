package generator

import "fmt"

type runtimeRegistrationKind string

const (
	runtimeRegistrationKindNative           runtimeRegistrationKind = "native"
	runtimeRegistrationKindCGONativeForward runtimeRegistrationKind = "cgo_native_forward"
	runtimeRegistrationKindMessage          runtimeRegistrationKind = "message"
	runtimeRegistrationKindTransportMessage runtimeRegistrationKind = "transport_message"
)

type runtimeServerKindExpr string

const (
	runtimeServerKindGoNative      runtimeServerKindExpr = "rpcruntime.ServerKindGoNative"
	runtimeServerKindCGONative     runtimeServerKindExpr = "rpcruntime.ServerKindCGONative"
	runtimeServerKindCGOMessage    runtimeServerKindExpr = "rpcruntime.ServerKindCGOMessage"
	runtimeServerKindConnect       runtimeServerKindExpr = "rpcruntime.ServerKindConnect"
	runtimeServerKindGRPC          runtimeServerKindExpr = "rpcruntime.ServerKindGRPC"
	runtimeServerKindConnectRemote runtimeServerKindExpr = "rpcruntime.ServerKindConnectRemote"
	runtimeServerKindGRPCRemote    runtimeServerKindExpr = "rpcruntime.ServerKindGRPCRemote"
)

type registrationSourceProjection struct {
	registrationKind runtimeRegistrationKind
	registerName     string
	inputName        string
	inputType        string
	nilErr           string
	sourceExpr       string
	serverKind       runtimeServerKindExpr
	label            string
}

// ProjectRegistrationSource derives renderer names, input types, and server kind for a registration source.
func ProjectRegistrationSource(service ServicePlan, source RegistrationSourceKind) (registrationSourceProjection, error) {
	if err := validateRegistrationSourceKind(source); err != nil {
		return registrationSourceProjection{}, err
	}

	serviceName := service.GoName
	switch source {
	case RegistrationSourceGoNative:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindNative,
			registerName:     "register" + serviceName + "GoNativeServer",
			inputName:        "server",
			inputType:        serviceName + "NativeServer",
			nilErr:           serviceName + "NativeServerUnavailableErr",
			sourceExpr:       "server",
			serverKind:       runtimeServerKindGoNative,
			label:            "go native",
		}, nil
	case RegistrationSourceCGONative:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindCGONativeForward,
			registerName:     "Register" + serviceName + "CGONativeServer",
			inputName:        "server",
			inputType:        serviceName + "NativeServer",
			nilErr:           serviceName + "NativeServerUnavailableErr",
			sourceExpr:       "server",
			serverKind:       runtimeServerKindCGONative,
			label:            "cgo native",
		}, nil
	case RegistrationSourceCGOMessage:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindMessage,
			registerName:     "register" + serviceName + "CGOMessageServer",
			inputName:        "server",
			inputType:        serviceName + "CGOMessageServer",
			nilErr:           serviceName + "MessageServerUnavailableErr",
			sourceExpr:       "server",
			serverKind:       runtimeServerKindCGOMessage,
			label:            "cgo message",
		}, nil
	case RegistrationSourceConnectHandler:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindTransportMessage,
			registerName:     "Register" + serviceName + "ConnectHandler",
			inputName:        "handler",
			inputType:        serviceName + "Handler",
			nilErr:           serviceName + "MessageServerUnavailableErr",
			sourceExpr:       "handler",
			serverKind:       runtimeServerKindConnect,
			label:            "connect handler",
		}, nil
	case RegistrationSourceConnectRemote:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindTransportMessage,
			registerName:     "Register" + serviceName + "ConnectRemoteServer",
			inputName:        "client",
			inputType:        serviceName + "Client",
			nilErr:           serviceName + "MessageServerUnavailableErr",
			sourceExpr:       "client",
			serverKind:       runtimeServerKindConnectRemote,
			label:            "connect remote",
		}, nil
	case RegistrationSourceGRPCServer:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindTransportMessage,
			registerName:     "Register" + serviceName + "GRPCServer",
			inputName:        "server",
			inputType:        serviceName + "Server",
			nilErr:           serviceName + "MessageServerUnavailableErr",
			sourceExpr:       "server",
			serverKind:       runtimeServerKindGRPC,
			label:            "grpc server",
		}, nil
	case RegistrationSourceGRPCRemote:
		return registrationSourceProjection{
			registrationKind: runtimeRegistrationKindTransportMessage,
			registerName:     "Register" + serviceName + "GRPCRemoteServer",
			inputName:        "client",
			inputType:        serviceName + "Client",
			nilErr:           serviceName + "MessageServerUnavailableErr",
			sourceExpr:       "client",
			serverKind:       runtimeServerKindGRPCRemote,
			label:            "grpc remote",
		}, nil
	default:
		return registrationSourceProjection{}, fmt.Errorf("unknown registration source projection %q", source)
	}
}

func registrationTransportMessageStreamConstructor(service ServicePlan, method runtimeMethodProjection, projection registrationSourceProjection) (string, bool, error) {
	if projection.registrationKind != runtimeRegistrationKindTransportMessage {
		return "", false, fmt.Errorf("registration source %q is not a transport message registration", projection.label)
	}

	var constructor string
	switch projection.serverKind {
	case runtimeServerKindConnect:
		constructor = "new" + connectDirectMessageSessionName(service.GoName, method)
	case runtimeServerKindConnectRemote:
		constructor = "new" + connectRemoteMessageSessionName(service.GoName, method)
	case runtimeServerKindGRPC:
		constructor = "new" + grpcDirectMessageSessionName(service.GoName, method)
	case runtimeServerKindGRPCRemote:
		constructor = "new" + grpcRemoteMessageSessionName(service.GoName, method)
	default:
		return "", false, fmt.Errorf("registration source %q does not define a transport stream constructor", projection.label)
	}

	return constructor, projection.serverKind == runtimeServerKindConnectRemote || projection.serverKind == runtimeServerKindGRPCRemote || method.Stream.Shape == runtimeStreamServer, nil
}
