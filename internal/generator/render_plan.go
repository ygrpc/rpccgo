package generator

import "fmt"

type MethodRenderPlan struct {
	CallPath  CallPathPlan
	Lifecycle StreamLifecycleProjectionPlan
	Symbols   RenderSymbolsPlan
	Errors    RenderErrorsPlan
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
	NativeServerUnavailableErr  string
	MessageServerUnavailableErr string
	UnknownActiveContractErr    string
	NativeMessageConverterErr   string
	Role                        string
	Category                    string
}

func BuildMethodRenderPlan(method MethodPlan, serviceName string) (MethodRenderPlan, error) {
	lifecycle, err := ProjectStreamLifecycle(method.Contract.Lifecycle, method.Contract.RenderInputs.NeedsCodec)
	if err != nil {
		return MethodRenderPlan{}, err
	}

	nativeAdapterMethod := method.GoName
	if method.Streaming != StreamingKindUnary {
		nativeAdapterMethod = "Start" + method.GoName
	}
	messageAdapterMethod := nativeAdapterMethod
	nativeSessionType := ""
	messageSessionType := ""
	if lifecycle.Streaming {
		nativeSessionType = serviceName + method.GoName + "NativeStreamSession"
		messageSessionType = serviceName + method.GoName + "MessageStreamSession"
	}
	nativeWrapperType := lowerInitial(serviceName) + method.GoName + "NativeToMessageStreamSession"
	messageWrapperType := lowerInitial(serviceName) + method.GoName + "MessageToNativeStreamSession"
	shape := MethodRenderPlan{
		Lifecycle: lifecycle,
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
			NativeServerUnavailableErr:  serviceName + "NativeServerUnavailableErr",
			MessageServerUnavailableErr: serviceName + "MessageServerUnavailableErr",
			UnknownActiveContractErr:    serviceName + "UnknownActiveContractErr",
			NativeMessageConverterErr:   serviceName + "NativeMessageConverterUnavailableErr",
			Role:                        "active_router",
			Category:                    "routing",
		},
	}
	shape.CallPath = renderCallPath(method, shape.Symbols)
	method.RenderPlan = shape
	if err := validateMethodRenderPlan(method); err != nil {
		return MethodRenderPlan{}, err
	}
	return shape, nil
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
	expected, err := expectedLifecyclePlan(method.Streaming)
	if err != nil {
		return fmt.Errorf("method %s has unknown streaming kind %d", methodPlanName(method), method.Streaming)
	}
	if lifecycle != expected {
		return fmt.Errorf("method %s streaming lifecycle capabilities do not match descriptor", methodPlanName(method))
	}
	return nil
}

func ValidateMethodRenderPlan(method MethodPlan) error {
	return validateMethodRenderPlan(method)
}

func validateMethodRenderPlan(method MethodPlan) error {
	shape := method.RenderPlan
	expectedLifecycle, err := ProjectStreamLifecycle(method.Contract.Lifecycle, method.Contract.RenderInputs.NeedsCodec)
	if err != nil {
		return fmt.Errorf("method %s render lifecycle is invalid: %w", methodPlanName(method), err)
	}
	if shape.Lifecycle != expectedLifecycle {
		return fmt.Errorf("method %s render lifecycle does not match contract capabilities", methodPlanName(method))
	}
	if method.Streaming == StreamingKindUnary {
		if shape.Lifecycle.Streaming {
			return fmt.Errorf("method %s unary render lifecycle must not stream", methodPlanName(method))
		}
	} else {
		if !shape.Lifecycle.Streaming {
			return fmt.Errorf("method %s streaming render lifecycle is missing", methodPlanName(method))
		}
	}
	if shape.Symbols.NativeAdapterMethod == "" || shape.Symbols.MessageAdapterMethod == "" {
		return fmt.Errorf("method %s render symbols are incomplete", methodPlanName(method))
	}
	if method.Streaming != StreamingKindUnary && (shape.Symbols.NativeSessionType == "" || shape.Symbols.MessageSessionType == "") {
		return fmt.Errorf("method %s render session symbols are incomplete", methodPlanName(method))
	}
	if shape.Errors.NativeServerUnavailableErr == "" || shape.Errors.MessageServerUnavailableErr == "" || shape.Errors.UnknownActiveContractErr == "" || shape.Errors.NativeMessageConverterErr == "" {
		return fmt.Errorf("method %s render errors are incomplete", methodPlanName(method))
	}
	return nil
}
