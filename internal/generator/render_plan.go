package generator

import "fmt"

type MethodRenderPlan struct {
	CallPath CallPathPlan
	Session  SessionRenderPlan
	Terminal TerminalRenderPlan
	Symbols  RenderSymbolsPlan
	Errors   RenderErrorsPlan
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

func BuildMethodRenderPlan(method MethodPlan, serviceName string) (MethodRenderPlan, error) {
	ops, sessionKind, terminal, err := renderSessionShape(method.Contract.Lifecycle, method.Contract.RenderInputs.NeedsCodec)
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
	method.RenderPlan = shape
	if err := validateMethodRenderPlan(method); err != nil {
		return MethodRenderPlan{}, err
	}
	return shape, nil
}

func renderSessionShape(lifecycle StreamLifecycleContractPlan, needsCodec bool) ([]SessionOperationPlan, SessionKind, TerminalRenderPlan, error) {
	op := func(kind SessionOperationKind, terminal bool) SessionOperationPlan {
		return SessionOperationPlan{Kind: kind, Enabled: true, RequiresCodec: needsCodec, RequiresTerminal: terminal}
	}
	if !lifecycle.HasOperation(StreamLifecycleOperationStart) {
		return nil, SessionKindNone, TerminalRenderPlan{}, nil
	}
	hasSend := lifecycle.HasOperation(StreamLifecycleOperationSend)
	hasReceive := lifecycle.HasOperation(StreamLifecycleOperationReceive)
	hasFinish := lifecycle.HasOperation(StreamLifecycleOperationFinish)
	hasDone := lifecycle.HasOperation(StreamLifecycleOperationDone)
	hasCloseSend := lifecycle.HasOperation(StreamLifecycleOperationCloseSend)
	hasCancel := lifecycle.HasOperation(StreamLifecycleOperationCancel)
	if hasSend && hasFinish {
		ops := []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationSend, false), op(SessionOperationFinish, true)}
		if hasCancel {
			ops = append(ops, op(SessionOperationCancel, true))
		}
		return ops, SessionKindClient, TerminalRenderPlan{Kind: TerminalKindFinish, Operation: SessionOperationFinish, ReleasesHandle: true, RequiresResponseConvert: true, AllowsCancel: hasCancel}, nil
	}
	if hasReceive && hasCloseSend && hasDone {
		ops := []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationSend, false), op(SessionOperationReceive, false), op(SessionOperationCloseSend, false), op(SessionOperationDone, true)}
		if hasCancel {
			ops = append(ops, op(SessionOperationCancel, true))
		}
		return ops, SessionKindBidi, TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true, AllowsCancel: hasCancel, AllowsCloseSend: true}, nil
	}
	if hasReceive && hasDone {
		ops := []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationReceive, false), op(SessionOperationDone, true)}
		if hasCancel {
			ops = append(ops, op(SessionOperationCancel, true))
		}
		return ops, SessionKindServer, TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true, AllowsCancel: hasCancel}, nil
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

func ValidateMethodContractPlan(method MethodPlan) error {
	if !method.Contract.Message.RequestType.HasIdentity() || !method.Contract.Message.ResponseType.HasIdentity() {
		return fmt.Errorf("method %s message contract is incomplete", methodPlanName(method))
	}
	if method.Contract.RenderInputs.NeedsCodec != method.NeedsCodec {
		return fmt.Errorf("method %s render inputs do not match method codec requirement", methodPlanName(method))
	}
	lifecycle := method.Contract.Lifecycle
	if method.Streaming == StreamingKindUnary {
		if !lifecycle.IsZero() {
			return fmt.Errorf("method %s unary lifecycle must be empty", methodPlanName(method))
		}
		return nil
	}
	if !lifecycle.HasOperation(StreamLifecycleOperationStart) {
		return fmt.Errorf("method %s streaming lifecycle is incomplete", methodPlanName(method))
	}
	switch method.Streaming {
	case StreamingKindClientStreaming:
		if lifecycle.TerminalKind != LifecycleTerminalFinishResult || !lifecycle.HasOperation(StreamLifecycleOperationFinish) {
			return fmt.Errorf("method %s client streaming lifecycle must finish with result", methodPlanName(method))
		}
	case StreamingKindServerStreaming, StreamingKindBidiStreaming:
		if lifecycle.TerminalKind != LifecycleTerminalOnDone || !lifecycle.HasOperation(StreamLifecycleOperationDone) {
			return fmt.Errorf("method %s streaming lifecycle must terminate on done", methodPlanName(method))
		}
	default:
		return fmt.Errorf("method %s has unknown streaming kind %d", methodPlanName(method), method.Streaming)
	}
	return nil
}

func ValidateMethodRenderPlan(method MethodPlan) error {
	return validateMethodRenderPlan(method)
}

func validateMethodRenderPlan(method MethodPlan) error {
	shape := method.RenderPlan
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
