package generator

type ActiveRecordOrigin string

const (
	ActiveRecordOriginGo  ActiveRecordOrigin = "go"
	ActiveRecordOriginCGO ActiveRecordOrigin = "cgo"
)

type ActiveRecordContract string

const (
	ActiveRecordContractNative           ActiveRecordContract = "native"
	ActiveRecordContractMessage          ActiveRecordContract = "message"
	ActiveRecordContractTransportMessage ActiveRecordContract = "transport_message"
)

type ActiveRecordTransport string

const (
	ActiveRecordTransportNone    ActiveRecordTransport = ""
	ActiveRecordTransportConnect ActiveRecordTransport = "connect"
	ActiveRecordTransportGRPC    ActiveRecordTransport = "grpc"
)

type ActiveRecordMode string

const (
	ActiveRecordModeLocal  ActiveRecordMode = "local"
	ActiveRecordModeRemote ActiveRecordMode = "remote"
)

type ActiveRecordRenderer string

const (
	ActiveRecordRendererNative           ActiveRecordRenderer = "native"
	ActiveRecordRendererMessage          ActiveRecordRenderer = "message"
	ActiveRecordRendererTransportMessage ActiveRecordRenderer = "transport_message"
)

type ActiveRecordSourcePlan struct {
	Origin    ActiveRecordOrigin
	Contract  ActiveRecordContract
	Transport ActiveRecordTransport
	Mode      ActiveRecordMode

	RecordRenderer            ActiveRecordRenderer
	AliasGoNativeRegistration bool

	SourceExpr string
	Label      string

	RegisterName string
	InputName    string
	InputType    string
	NilErr       string
}

func activeRecordSourcesForService(service ServicePlan) []ActiveRecordSourcePlan {
	serviceName := service.GoName
	nativeInputType := serviceName + "NativeServer"

	sources := []ActiveRecordSourcePlan{
		{
			Origin:         ActiveRecordOriginGo,
			Contract:       ActiveRecordContractNative,
			Transport:      ActiveRecordTransportNone,
			Mode:           ActiveRecordModeLocal,
			RecordRenderer: ActiveRecordRendererNative,
			SourceExpr:     "server",
			Label:          "go native",
			RegisterName:   "register" + serviceName + "GoNativeServer",
			InputName:      "server",
			InputType:      nativeInputType,
			NilErr:         serviceName + "NativeServerUnavailableErr",
		},
		{
			Origin:                    ActiveRecordOriginCGO,
			Contract:                  ActiveRecordContractNative,
			Transport:                 ActiveRecordTransportNone,
			Mode:                      ActiveRecordModeLocal,
			RecordRenderer:            ActiveRecordRendererNative,
			AliasGoNativeRegistration: true,
			SourceExpr:                "server",
			Label:                     "cgo native",
			RegisterName:              "Register" + serviceName + "CGONativeServer",
			InputName:                 "server",
			InputType:                 nativeInputType,
			NilErr:                    serviceName + "NativeServerUnavailableErr",
		},
		{
			Origin:         ActiveRecordOriginCGO,
			Contract:       ActiveRecordContractMessage,
			Transport:      ActiveRecordTransportNone,
			Mode:           ActiveRecordModeLocal,
			RecordRenderer: ActiveRecordRendererMessage,
			SourceExpr:     "server",
			Label:          "cgo message",
			RegisterName:   "register" + serviceName + "CGOMessageServer",
			InputName:      "server",
			InputType:      serviceName + "CGOMessageServer",
			NilErr:         serviceName + "MessageServerUnavailableErr",
		},
	}

	if service.Adapters.Has(AdapterTokenMessageConnect) {
		sources = append(sources,
			ActiveRecordSourcePlan{
				Origin:         ActiveRecordOriginGo,
				Contract:       ActiveRecordContractTransportMessage,
				Transport:      ActiveRecordTransportConnect,
				Mode:           ActiveRecordModeLocal,
				RecordRenderer: ActiveRecordRendererTransportMessage,
				SourceExpr:     "handler",
				Label:          "connect handler",
				RegisterName:   "Register" + serviceName + "ConnectHandler",
				InputName:      "handler",
				InputType:      serviceName + "Handler",
				NilErr:         serviceName + "MessageServerUnavailableErr",
			},
			ActiveRecordSourcePlan{
				Origin:         ActiveRecordOriginGo,
				Contract:       ActiveRecordContractTransportMessage,
				Transport:      ActiveRecordTransportConnect,
				Mode:           ActiveRecordModeRemote,
				RecordRenderer: ActiveRecordRendererTransportMessage,
				SourceExpr:     "client",
				Label:          "connect remote",
				RegisterName:   "Register" + serviceName + "ConnectRemoteServer",
				InputName:      "client",
				InputType:      serviceName + "Client",
				NilErr:         serviceName + "MessageServerUnavailableErr",
			},
		)
	}

	if service.Adapters.Has(AdapterTokenMessageGRPC) {
		sources = append(sources,
			ActiveRecordSourcePlan{
				Origin:         ActiveRecordOriginGo,
				Contract:       ActiveRecordContractTransportMessage,
				Transport:      ActiveRecordTransportGRPC,
				Mode:           ActiveRecordModeLocal,
				RecordRenderer: ActiveRecordRendererTransportMessage,
				SourceExpr:     "server",
				Label:          "grpc server",
				RegisterName:   "Register" + serviceName + "GRPCServer",
				InputName:      "server",
				InputType:      serviceName + "Server",
				NilErr:         serviceName + "MessageServerUnavailableErr",
			},
			ActiveRecordSourcePlan{
				Origin:         ActiveRecordOriginGo,
				Contract:       ActiveRecordContractTransportMessage,
				Transport:      ActiveRecordTransportGRPC,
				Mode:           ActiveRecordModeRemote,
				RecordRenderer: ActiveRecordRendererTransportMessage,
				SourceExpr:     "client",
				Label:          "grpc remote",
				RegisterName:   "Register" + serviceName + "GRPCRemoteServer",
				InputName:      "client",
				InputType:      serviceName + "Client",
				NilErr:         serviceName + "MessageServerUnavailableErr",
			},
		)
	}

	return sources
}
