package generator

import "fmt"

// RegistrationSourceKind identifies one supported registered server source.
type RegistrationSourceKind string

// Registration source kinds supported by generated server registration helpers.
const (
	RegistrationSourceGoNative       RegistrationSourceKind = "go_native"
	RegistrationSourceCGONative      RegistrationSourceKind = "cgo_native"
	RegistrationSourceCGOMessage     RegistrationSourceKind = "cgo_message"
	RegistrationSourceConnectHandler RegistrationSourceKind = "connect_handler"
	RegistrationSourceConnectRemote  RegistrationSourceKind = "connect_remote"
	RegistrationSourceGRPCServer     RegistrationSourceKind = "grpc_server"
	RegistrationSourceGRPCRemote     RegistrationSourceKind = "grpc_remote"
)

func registrationSourcesForService(service ServicePlan) []RegistrationSourceKind {
	selection := registrationSourceSelectionForService(service)

	sources := []RegistrationSourceKind{RegistrationSourceCGOMessage}

	if selection.NativeEnabled {
		sources = append([]RegistrationSourceKind{RegistrationSourceGoNative, RegistrationSourceCGONative}, sources...)
	}

	switch selection.MessageTransport {
	case MessageTransportConnect:
		sources = append(sources, RegistrationSourceConnectHandler, RegistrationSourceConnectRemote)
	case MessageTransportGRPC:
		sources = append(sources, RegistrationSourceGRPCServer, RegistrationSourceGRPCRemote)
	}

	return sources
}

func registrationSourceSelectionForService(service ServicePlan) ServiceGenerationSelection {
	if service.Generation.HasIdentity() {
		return service.Generation
	}
	return ServiceGenerationSelection{MessageTransport: MessageTransportConnect}
}

func validateRegistrationSourceKind(source RegistrationSourceKind) error {
	switch source {
	case RegistrationSourceGoNative,
		RegistrationSourceCGONative,
		RegistrationSourceCGOMessage,
		RegistrationSourceConnectHandler,
		RegistrationSourceConnectRemote,
		RegistrationSourceGRPCServer,
		RegistrationSourceGRPCRemote:
		return nil
	default:
		return fmt.Errorf("unknown registration source %q", source)
	}
}
