package generator

import "fmt"

// MethodRenderPlan records renderer-facing call paths, stream operations, symbols, and errors for one method.
type MethodRenderPlan struct {
	CallPath CallPathPlan
	Stream   StreamCapabilityProjectionPlan
	Symbols  RenderSymbolsPlan
	Errors   RenderErrorsPlan
}

// CallPathPlan records native and message routing paths for unary and streaming calls.
type CallPathPlan struct {
	NativeUnary   CallPathRoutePlan
	MessageUnary  CallPathRoutePlan
	NativeStream  CallPathRoutePlan
	MessageStream CallPathRoutePlan
}

// CallPathRoutePlan describes one generated dispatch route and any conversion or guard it needs.
type CallPathRoutePlan struct {
	RouteKind                 CallPathRouteKind
	NeedsNativeConversion     bool
	NeedsMessageConversion    bool
	NeedsMissingEntryGuard    bool
	NeedsUnknownContractGuard bool
	NativeEntryMethod         string
	MessageEntryMethod        string
	NativeSessionMethod       string
	MessageSessionMethod      string
	NativeWrapperType         string
	MessageWrapperType        string
}

// CallPathRouteKind identifies whether a generated dispatch route targets native or message code.
type CallPathRouteKind string

// Call path route kinds used by runtime renderer projections.
const (
	CallPathRouteKindUnset   CallPathRouteKind = "unset"
	CallPathRouteKindNative  CallPathRouteKind = "native"
	CallPathRouteKindMessage CallPathRouteKind = "message"
)

// RenderSymbolsPlan records generated symbol names derived for one method.
type RenderSymbolsPlan struct {
	NativeEntryMethod    string
	MessageEntryMethod   string
	NativeAdapterMethod  string
	MessageAdapterMethod string
	NativeSessionType    string
	MessageSessionType   string
	ActiveRouterMethod   string
	NativeWrapperType    string
	MessageWrapperType   string
}

// RenderErrorsPlan records generated error symbol names and error context labels.
type RenderErrorsPlan struct {
	NativeServerUnavailableErr  string
	MessageServerUnavailableErr string
	UnknownActiveContractErr    string
	Role                        string
	Category                    string
}

// BuildMethodRenderPlan projects a method contract plan into renderer-facing symbols and call paths.
func BuildMethodRenderPlan(method MethodPlan, serviceName string) (MethodRenderPlan, error) {
	capability, err := ProjectStreamCapability(method.Contract.Stream, true)
	if err != nil {
		return MethodRenderPlan{}, err
	}

	nativeEntryMethod := method.GoName
	if method.Streaming != StreamingKindUnary {
		nativeEntryMethod = "Start" + method.GoName
	}
	messageEntryMethod := nativeEntryMethod
	nativeSessionType := ""
	messageSessionType := ""
	if capability.Streaming {
		nativeSessionType = serviceName + method.GoName + "NativeStreamSession"
		messageSessionType = serviceName + method.GoName + "MessageStreamSession"
	}
	nativeWrapperType := lowerInitial(serviceName) + method.GoName + "NativeToMessageStreamSession"
	messageWrapperType := lowerInitial(serviceName) + method.GoName + "MessageToNativeStreamSession"
	shape := MethodRenderPlan{
		Stream: capability,
		Symbols: RenderSymbolsPlan{
			NativeEntryMethod:    nativeEntryMethod,
			MessageEntryMethod:   messageEntryMethod,
			NativeAdapterMethod:  nativeEntryMethod,
			MessageAdapterMethod: messageEntryMethod,
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
			Role:                        "entry",
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
	native := CallPathRoutePlan{RouteKind: CallPathRouteKindNative, NeedsNativeConversion: true, NeedsMessageConversion: true, NeedsMissingEntryGuard: true, NeedsUnknownContractGuard: true, NativeEntryMethod: symbols.NativeEntryMethod, MessageEntryMethod: symbols.MessageEntryMethod, NativeSessionMethod: symbols.NativeEntryMethod, MessageSessionMethod: symbols.MessageEntryMethod, NativeWrapperType: symbols.NativeWrapperType, MessageWrapperType: symbols.MessageWrapperType}
	message := native
	message.RouteKind = CallPathRouteKindMessage
	if method.Streaming == StreamingKindUnary {
		return CallPathPlan{NativeUnary: native, MessageUnary: message}
	}
	return CallPathPlan{NativeStream: native, MessageStream: message}
}

// ValidateMethodContractPlan checks that a method contract matches its descriptor-derived streaming shape.
func ValidateMethodContractPlan(method MethodPlan) error {
	if !method.Contract.Message.RequestType.HasIdentity() || !method.Contract.Message.ResponseType.HasIdentity() {
		return fmt.Errorf("method %s message contract is incomplete", methodPlanName(method))
	}
	capability := method.Contract.Stream
	if method.Streaming == StreamingKindUnary {
		if !capability.IsZero() {
			return fmt.Errorf("method %s unary capability must be empty", methodPlanName(method))
		}
		return nil
	}
	expected, err := expectedStreamCapabilityPlan(method.Streaming)
	if err != nil {
		return fmt.Errorf("method %s has unknown streaming kind %d", methodPlanName(method), method.Streaming)
	}
	if capability != expected {
		return fmt.Errorf("method %s streaming capabilities do not match descriptor", methodPlanName(method))
	}
	return nil
}

// ValidateMethodRenderPlan checks that a method render plan is complete and matches its contract plan.
func ValidateMethodRenderPlan(method MethodPlan) error {
	return validateMethodRenderPlan(method)
}

func validateMethodRenderPlan(method MethodPlan) error {
	shape := method.RenderPlan
	expectedStreamCapability, err := ProjectStreamCapability(method.Contract.Stream, true)
	if err != nil {
		return fmt.Errorf("method %s render capability is invalid: %w", methodPlanName(method), err)
	}
	if shape.Stream != expectedStreamCapability {
		return fmt.Errorf("method %s render capability does not match contract capabilities", methodPlanName(method))
	}
	if method.Streaming == StreamingKindUnary {
		if shape.Stream.Streaming {
			return fmt.Errorf("method %s unary render capability must not stream", methodPlanName(method))
		}
	} else {
		if !shape.Stream.Streaming {
			return fmt.Errorf("method %s streaming render capability is missing", methodPlanName(method))
		}
	}
	if shape.Symbols.NativeEntryMethod == "" || shape.Symbols.MessageEntryMethod == "" {
		return fmt.Errorf("method %s render symbols are incomplete", methodPlanName(method))
	}
	if method.Streaming != StreamingKindUnary && (shape.Symbols.NativeSessionType == "" || shape.Symbols.MessageSessionType == "") {
		return fmt.Errorf("method %s render session symbols are incomplete", methodPlanName(method))
	}
	if shape.Errors.NativeServerUnavailableErr == "" || shape.Errors.MessageServerUnavailableErr == "" || shape.Errors.UnknownActiveContractErr == "" {
		return fmt.Errorf("method %s render errors are incomplete", methodPlanName(method))
	}
	return nil
}
