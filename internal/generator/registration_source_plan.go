package generator

import "fmt"

// RegistrationOrigin identifies whether a registered server source comes from Go or cgo.
type RegistrationOrigin string

// Registration origins supported by generated server registration helpers.
const (
	RegistrationOriginGo  RegistrationOrigin = "go"
	RegistrationOriginCGO RegistrationOrigin = "cgo"
)

// RegistrationContract identifies whether a source provides the native or message contract.
type RegistrationContract string

// Registration contracts accepted by generated registration helpers.
const (
	RegistrationContractNative  RegistrationContract = "native"
	RegistrationContractMessage RegistrationContract = "message"
)

// RegistrationTransport identifies the transport attached to a registration source.
type RegistrationTransport string

// Registration transports supported by registration source planning.
const (
	RegistrationTransportNone    RegistrationTransport = "none"
	RegistrationTransportConnect RegistrationTransport = "connect"
	RegistrationTransportGRPC    RegistrationTransport = "grpc"
)

// RegistrationMode identifies whether a registration source is local or remote.
type RegistrationMode string

// Registration modes supported by generated registration helpers.
const (
	RegistrationModeLocal  RegistrationMode = "local"
	RegistrationModeRemote RegistrationMode = "remote"
)

// RegistrationSourcePlan records the four-axis identity of one supported server source.
type RegistrationSourcePlan struct {
	Origin    RegistrationOrigin
	Contract  RegistrationContract
	Transport RegistrationTransport
	Mode      RegistrationMode
}

func registrationSourcesForService(service ServicePlan) []RegistrationSourcePlan {
	selection := registrationSourceSelectionForService(service)

	sources := []RegistrationSourcePlan{
		{
			Origin:    RegistrationOriginCGO,
			Contract:  RegistrationContractMessage,
			Transport: RegistrationTransportNone,
			Mode:      RegistrationModeLocal,
		},
	}

	if selection.NativeEnabled {
		sources = append([]RegistrationSourcePlan{
			{
				Origin:    RegistrationOriginGo,
				Contract:  RegistrationContractNative,
				Transport: RegistrationTransportNone,
				Mode:      RegistrationModeLocal,
			},
			{
				Origin:    RegistrationOriginCGO,
				Contract:  RegistrationContractNative,
				Transport: RegistrationTransportNone,
				Mode:      RegistrationModeLocal,
			},
		}, sources...)
	}

	switch selection.MessageTransport {
	case MessageTransportConnect:
		sources = append(sources,
			RegistrationSourcePlan{
				Origin:    RegistrationOriginGo,
				Contract:  RegistrationContractMessage,
				Transport: RegistrationTransportConnect,
				Mode:      RegistrationModeLocal,
			},
			RegistrationSourcePlan{
				Origin:    RegistrationOriginGo,
				Contract:  RegistrationContractMessage,
				Transport: RegistrationTransportConnect,
				Mode:      RegistrationModeRemote,
			},
		)
	case MessageTransportGRPC:
		sources = append(sources,
			RegistrationSourcePlan{
				Origin:    RegistrationOriginGo,
				Contract:  RegistrationContractMessage,
				Transport: RegistrationTransportGRPC,
				Mode:      RegistrationModeLocal,
			},
			RegistrationSourcePlan{
				Origin:    RegistrationOriginGo,
				Contract:  RegistrationContractMessage,
				Transport: RegistrationTransportGRPC,
				Mode:      RegistrationModeRemote,
			},
		)
	}

	return sources
}

func registrationSourceSelectionForService(service ServicePlan) ServiceGenerationSelection {
	if service.Generation.HasIdentity() {
		return service.Generation
	}
	return ServiceGenerationSelection{MessageTransport: MessageTransportConnect}
}

// ValidateRegistrationSourcePlan rejects registration source axis combinations without generation semantics.
func ValidateRegistrationSourcePlan(source RegistrationSourcePlan) error {
	for _, allowed := range validRegistrationSourcePlans {
		if source == allowed {
			return nil
		}
	}
	return fmt.Errorf("unknown registration source origin=%q contract=%q transport=%q mode=%q", source.Origin, source.Contract, source.Transport, source.Mode)
}

var validRegistrationSourcePlans = []RegistrationSourcePlan{
	{Origin: RegistrationOriginGo, Contract: RegistrationContractNative, Transport: RegistrationTransportNone, Mode: RegistrationModeLocal},
	{Origin: RegistrationOriginCGO, Contract: RegistrationContractNative, Transport: RegistrationTransportNone, Mode: RegistrationModeLocal},
	{Origin: RegistrationOriginCGO, Contract: RegistrationContractMessage, Transport: RegistrationTransportNone, Mode: RegistrationModeLocal},
	{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportConnect, Mode: RegistrationModeLocal},
	{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportConnect, Mode: RegistrationModeRemote},
	{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportGRPC, Mode: RegistrationModeLocal},
	{Origin: RegistrationOriginGo, Contract: RegistrationContractMessage, Transport: RegistrationTransportGRPC, Mode: RegistrationModeRemote},
}
