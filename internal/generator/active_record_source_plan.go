package generator

import "fmt"

type ActiveRecordOrigin string

const (
	ActiveRecordOriginGo  ActiveRecordOrigin = "go"
	ActiveRecordOriginCGO ActiveRecordOrigin = "cgo"
)

type ActiveRecordContract string

const (
	ActiveRecordContractNative  ActiveRecordContract = "native"
	ActiveRecordContractMessage ActiveRecordContract = "message"
)

type ActiveRecordTransport string

const (
	ActiveRecordTransportNone    ActiveRecordTransport = "none"
	ActiveRecordTransportConnect ActiveRecordTransport = "connect"
	ActiveRecordTransportGRPC    ActiveRecordTransport = "grpc"
)

type ActiveRecordMode string

const (
	ActiveRecordModeLocal  ActiveRecordMode = "local"
	ActiveRecordModeRemote ActiveRecordMode = "remote"
)

type ActiveRecordSourcePlan struct {
	Origin    ActiveRecordOrigin
	Contract  ActiveRecordContract
	Transport ActiveRecordTransport
	Mode      ActiveRecordMode
}

func activeRecordSourcesForService(service ServicePlan) []ActiveRecordSourcePlan {
	selection := activeRecordSourceSelectionForService(service)

	sources := []ActiveRecordSourcePlan{
		{
			Origin:    ActiveRecordOriginCGO,
			Contract:  ActiveRecordContractMessage,
			Transport: ActiveRecordTransportNone,
			Mode:      ActiveRecordModeLocal,
		},
	}

	if selection.NativeEnabled {
		sources = append([]ActiveRecordSourcePlan{
			{
				Origin:    ActiveRecordOriginGo,
				Contract:  ActiveRecordContractNative,
				Transport: ActiveRecordTransportNone,
				Mode:      ActiveRecordModeLocal,
			},
			{
				Origin:    ActiveRecordOriginCGO,
				Contract:  ActiveRecordContractNative,
				Transport: ActiveRecordTransportNone,
				Mode:      ActiveRecordModeLocal,
			},
		}, sources...)
	}

	switch selection.MessageTransport {
	case MessageTransportConnect:
		sources = append(sources,
			ActiveRecordSourcePlan{
				Origin:    ActiveRecordOriginGo,
				Contract:  ActiveRecordContractMessage,
				Transport: ActiveRecordTransportConnect,
				Mode:      ActiveRecordModeLocal,
			},
			ActiveRecordSourcePlan{
				Origin:    ActiveRecordOriginGo,
				Contract:  ActiveRecordContractMessage,
				Transport: ActiveRecordTransportConnect,
				Mode:      ActiveRecordModeRemote,
			},
		)
	case MessageTransportGRPC:
		sources = append(sources,
			ActiveRecordSourcePlan{
				Origin:    ActiveRecordOriginGo,
				Contract:  ActiveRecordContractMessage,
				Transport: ActiveRecordTransportGRPC,
				Mode:      ActiveRecordModeLocal,
			},
			ActiveRecordSourcePlan{
				Origin:    ActiveRecordOriginGo,
				Contract:  ActiveRecordContractMessage,
				Transport: ActiveRecordTransportGRPC,
				Mode:      ActiveRecordModeRemote,
			},
		)
	}

	return sources
}

func activeRecordSourceSelectionForService(service ServicePlan) ServiceGenerationSelection {
	if service.Generation.HasIdentity() {
		return service.Generation
	}
	return ServiceGenerationSelection{MessageTransport: MessageTransportConnect}
}

func ValidateActiveRecordSourcePlan(source ActiveRecordSourcePlan) error {
	for _, allowed := range validActiveRecordSourcePlans {
		if source == allowed {
			return nil
		}
	}
	return fmt.Errorf("unknown active record source origin=%q contract=%q transport=%q mode=%q", source.Origin, source.Contract, source.Transport, source.Mode)
}

var validActiveRecordSourcePlans = []ActiveRecordSourcePlan{
	{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractNative, Transport: ActiveRecordTransportNone, Mode: ActiveRecordModeLocal},
	{Origin: ActiveRecordOriginCGO, Contract: ActiveRecordContractNative, Transport: ActiveRecordTransportNone, Mode: ActiveRecordModeLocal},
	{Origin: ActiveRecordOriginCGO, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportNone, Mode: ActiveRecordModeLocal},
	{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportConnect, Mode: ActiveRecordModeLocal},
	{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportConnect, Mode: ActiveRecordModeRemote},
	{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportGRPC, Mode: ActiveRecordModeLocal},
	{Origin: ActiveRecordOriginGo, Contract: ActiveRecordContractMessage, Transport: ActiveRecordTransportGRPC, Mode: ActiveRecordModeRemote},
}
