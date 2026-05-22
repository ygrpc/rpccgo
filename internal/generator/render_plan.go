package generator

import "fmt"

type MethodRenderPlan struct {
	CallPath   CallPathPlan
	Session    SessionRenderPlan
	Terminal   TerminalRenderPlan
	Conversion ConversionRenderPlan
	Symbols    RenderSymbolsPlan
	Errors     RenderErrorsPlan
}

type CallPathPlan struct {
	NativeUnary   CallPathRoutePlan
	MessageUnary  CallPathRoutePlan
	NativeStream  CallPathRoutePlan
	MessageStream CallPathRoutePlan
}

type CallPathRoutePlan struct {
	RouteKind                 CallPathRouteKind
	NeedsCodec                bool
	NeedsNativeConversion     bool
	NeedsMessageConversion    bool
	NeedsMissingAdapterGuard  bool
	NeedsUnknownContractGuard bool
	NativeAdapterMethod       string
	MessageAdapterMethod      string
	NativeSessionMethod       string
	MessageSessionMethod      string
	NativeWrapperType         string
	MessageWrapperType        string
}

type CallPathRouteKind string

const (
	CallPathRouteKindUnset   CallPathRouteKind = "unset"
	CallPathRouteKindNative  CallPathRouteKind = "native"
	CallPathRouteKindMessage CallPathRouteKind = "message"
)

type SessionRenderPlan struct {
	Kind       SessionKind
	Operations []SessionOperationPlan
}

type SessionKind string

const (
	SessionKindNone   SessionKind = "none"
	SessionKindClient SessionKind = "client_streaming"
	SessionKindServer SessionKind = "server_streaming"
	SessionKindBidi   SessionKind = "bidi_streaming"
)

type SessionOperationPlan struct {
	Kind             SessionOperationKind
	Enabled          bool
	NativeIO         MethodIOShapePlan
	MessageIO        MethodIOShapePlan
	RequiresCodec    bool
	RequiresTerminal bool
}

type SessionOperationKind string

const (
	SessionOperationStart     SessionOperationKind = "start"
	SessionOperationSend      SessionOperationKind = "send"
	SessionOperationReceive   SessionOperationKind = "receive"
	SessionOperationFinish    SessionOperationKind = "finish"
	SessionOperationDone      SessionOperationKind = "done"
	SessionOperationCloseSend SessionOperationKind = "close_send"
	SessionOperationCancel    SessionOperationKind = "cancel"
)

type MethodIOShapePlan struct {
	Request  []FieldPlan
	Response []FieldPlan
}

type TerminalRenderPlan struct {
	Kind                    TerminalKind
	Operation               SessionOperationKind
	ReleasesHandle          bool
	RequiresResponseConvert bool
	AllowsCancel            bool
	AllowsCloseSend         bool
}

type TerminalKind string

const (
	TerminalKindUnset  TerminalKind = "unset"
	TerminalKindFinish TerminalKind = "finish"
	TerminalKindDone   TerminalKind = "done"
)

type ConversionRenderPlan struct {
	NativeToMessage ConversionShapePlan
	MessageToNative ConversionShapePlan
}

type ConversionShapePlan struct {
	Kind      ConversionKind
	Direction ConversionDirection
	Enabled   bool
	Native    MethodIOShapePlan
	Message   MethodIOShapePlan
}

type ConversionKind string

const (
	ConversionKindUnset  ConversionKind = "unset"
	ConversionKindDecode ConversionKind = "decode"
	ConversionKindEncode ConversionKind = "encode"
)

type ConversionDirection string

const (
	ConversionDirectionNativeToMessage ConversionDirection = "native_to_message"
	ConversionDirectionMessageToNative ConversionDirection = "message_to_native"
)

type RenderSymbolsPlan struct {
	NativeAdapterMethod  string
	MessageAdapterMethod string
	NativeSessionType    string
	MessageSessionType   string
	ActiveRouterMethod   string
	NativeWrapperType    string
	MessageWrapperType   string
}

type RenderErrorsPlan struct {
	NativeAdapterUnavailableErr  string
	MessageAdapterUnavailableErr string
	UnknownActiveContractErr     string
	NativeMessageConverterErr    string
	Role                         string
	Category                     string
}

func BuildMethodRenderPlan(method MethodPlan, facts methodContractFacts, serviceName string) (MethodRenderPlan, error) {
	lifecycle, err := expectedLifecyclePlan(method.Streaming)
	if err != nil {
		return MethodRenderPlan{}, err
	}
	ops, sessionKind, terminal, err := renderSessionShape(lifecycle, facts)
	if err != nil {
		return MethodRenderPlan{}, err
	}

	nativeAdapterMethod := method.GoName
	if method.Streaming != StreamingKindUnary {
		nativeAdapterMethod = "Start" + method.GoName
	}
	messageAdapterMethod := nativeAdapterMethod + "Message"
	nativeSessionType := ""
	messageSessionType := ""
	if sessionKind != SessionKindNone {
		nativeSessionType = serviceName + method.GoName + "NativeStreamSession"
		messageSessionType = serviceName + method.GoName + "MessageStreamSession"
	}
	nativeWrapperType := lowerInitial(serviceName) + method.GoName + "NativeToMessageStreamSession"
	messageWrapperType := lowerInitial(serviceName) + method.GoName + "MessageToNativeStreamSession"
	shape := MethodRenderPlan{
		Session:  SessionRenderPlan{Kind: sessionKind, Operations: ops},
		Terminal: terminal,
		Conversion: ConversionRenderPlan{
			NativeToMessage: ConversionShapePlan{
				Kind:      ConversionKindEncode,
				Direction: ConversionDirectionNativeToMessage,
				Enabled:   method.NeedsCodec,
				Native:    MethodIOShapePlan{Request: facts.NativeContract.RequestFields, Response: facts.NativeContract.ResponseFields},
				Message:   MethodIOShapePlan{Request: facts.RequestBody, Response: facts.ResponseBody},
			},
			MessageToNative: ConversionShapePlan{
				Kind:      ConversionKindDecode,
				Direction: ConversionDirectionMessageToNative,
				Enabled:   method.NeedsCodec,
				Native:    MethodIOShapePlan{Request: facts.NativeContract.RequestFields, Response: facts.NativeContract.ResponseFields},
				Message:   MethodIOShapePlan{Request: facts.RequestBody, Response: facts.ResponseBody},
			},
		},
		Symbols: RenderSymbolsPlan{
			NativeAdapterMethod:  nativeAdapterMethod,
			MessageAdapterMethod: messageAdapterMethod,
			NativeSessionType:    nativeSessionType,
			MessageSessionType:   messageSessionType,
			ActiveRouterMethod:   method.GoName,
			NativeWrapperType:    nativeWrapperType,
			MessageWrapperType:   messageWrapperType,
		},
		Errors: RenderErrorsPlan{
			NativeAdapterUnavailableErr:  serviceName + "NativeAdapterUnavailableErr",
			MessageAdapterUnavailableErr: serviceName + "MessageAdapterUnavailableErr",
			UnknownActiveContractErr:     serviceName + "UnknownActiveContractErr",
			NativeMessageConverterErr:    serviceName + "NativeMessageConverterUnavailableErr",
			Role:                         "active_router",
			Category:                     "routing",
		},
	}
	shape.CallPath = renderCallPath(method, shape.Symbols)
	if err := validateMethodRenderPlan(MethodPlan{Name: method.Name, GoName: method.GoName, FullName: method.FullName, Streaming: method.Streaming, RenderShape: shape}); err != nil {
		return MethodRenderPlan{}, err
	}
	return shape, nil
}

func renderSessionShape(lifecycle LifecyclePlan, facts methodContractFacts) ([]SessionOperationPlan, SessionKind, TerminalRenderPlan, error) {
	nativeIO := MethodIOShapePlan{Request: facts.NativeContract.RequestFields, Response: facts.NativeContract.ResponseFields}
	messageIO := MethodIOShapePlan{Request: facts.RequestBody, Response: facts.ResponseBody}
	op := func(kind SessionOperationKind, terminal bool) SessionOperationPlan {
		return SessionOperationPlan{Kind: kind, Enabled: true, NativeIO: nativeIO, MessageIO: messageIO, RequiresCodec: true, RequiresTerminal: terminal}
	}
	if !lifecycle.HasStart {
		return nil, SessionKindNone, TerminalRenderPlan{}, nil
	}
	if lifecycle.HasSend && lifecycle.HasFinish {
		return []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationSend, false), op(SessionOperationFinish, true), op(SessionOperationCancel, true)}, SessionKindClient, TerminalRenderPlan{Kind: TerminalKindFinish, Operation: SessionOperationFinish, ReleasesHandle: true, RequiresResponseConvert: true, AllowsCancel: lifecycle.HasCancel}, nil
	}
	if lifecycle.HasOnRead && lifecycle.HasCloseSend {
		return []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationSend, false), op(SessionOperationReceive, false), op(SessionOperationCloseSend, false), op(SessionOperationDone, true), op(SessionOperationCancel, true)}, SessionKindBidi, TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true, AllowsCancel: lifecycle.HasCancel, AllowsCloseSend: true}, nil
	}
	if lifecycle.HasOnRead && lifecycle.HasOnDone {
		return []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationReceive, false), op(SessionOperationDone, true), op(SessionOperationCancel, true)}, SessionKindServer, TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true, AllowsCancel: lifecycle.HasCancel}, nil
	}
	return nil, "", TerminalRenderPlan{}, fmt.Errorf("invalid lifecycle plan")
}

func renderCallPath(method MethodPlan, symbols RenderSymbolsPlan) CallPathPlan {
	native := CallPathRoutePlan{RouteKind: CallPathRouteKindNative, NeedsCodec: method.NeedsCodec, NeedsNativeConversion: true, NeedsMessageConversion: true, NeedsMissingAdapterGuard: true, NeedsUnknownContractGuard: true, NativeAdapterMethod: symbols.NativeAdapterMethod, MessageAdapterMethod: symbols.MessageAdapterMethod, NativeSessionMethod: symbols.NativeAdapterMethod, MessageSessionMethod: symbols.MessageAdapterMethod, NativeWrapperType: symbols.NativeWrapperType, MessageWrapperType: symbols.MessageWrapperType}
	message := native
	message.RouteKind = CallPathRouteKindMessage
	if method.Streaming == StreamingKindUnary {
		return CallPathPlan{NativeUnary: native, MessageUnary: message}
	}
	return CallPathPlan{NativeStream: native, MessageStream: message}
}

func ValidateMethodRenderPlan(method MethodPlan) error {
	return validateMethodRenderPlan(method)
}

func validateMethodRenderPlan(method MethodPlan) error {
	shape := method.RenderShape
	if method.Streaming == StreamingKindUnary {
		if shape.Session.Kind != SessionKindNone || len(shape.Session.Operations) != 0 {
			return fmt.Errorf("method %s unary render session must be none", methodPlanName(method))
		}
		if shape.Terminal.Kind != "" {
			return fmt.Errorf("method %s unary render terminal must be empty", methodPlanName(method))
		}
	} else {
		if shape.Session.Kind == SessionKindNone || len(shape.Session.Operations) == 0 {
			return fmt.Errorf("method %s streaming render session operations are missing", methodPlanName(method))
		}
		if shape.Terminal.Kind == "" || shape.Terminal.Operation == "" || !shape.Terminal.ReleasesHandle {
			return fmt.Errorf("method %s streaming render terminal is incomplete", methodPlanName(method))
		}
	}
	if shape.Symbols.NativeAdapterMethod == "" || shape.Symbols.MessageAdapterMethod == "" {
		return fmt.Errorf("method %s render symbols are incomplete", methodPlanName(method))
	}
	if method.Streaming != StreamingKindUnary && (shape.Symbols.NativeSessionType == "" || shape.Symbols.MessageSessionType == "") {
		return fmt.Errorf("method %s render session symbols are incomplete", methodPlanName(method))
	}
	if shape.Errors.NativeAdapterUnavailableErr == "" || shape.Errors.MessageAdapterUnavailableErr == "" || shape.Errors.UnknownActiveContractErr == "" || shape.Errors.NativeMessageConverterErr == "" {
		return fmt.Errorf("method %s render errors are incomplete", methodPlanName(method))
	}
	return nil
}
