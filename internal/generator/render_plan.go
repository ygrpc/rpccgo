package generator

import "fmt"

// MethodRenderPlan records renderer-facing stream operations, symbols, and errors for one method.
type MethodRenderPlan struct {
	Stream  StreamCapabilityProjectionPlan
	Symbols RenderSymbolsPlan
	Errors  RenderErrorsPlan
}

// RenderSymbolsPlan records generated symbol names derived for one method.
type RenderSymbolsPlan struct {
	NativeEntryMethod        string
	MessageEntryMethod       string
	NativeAdapterMethod      string
	MessageAdapterMethod     string
	NativeStreamRequestType  string
	NativeStreamResponseType string
}

// RenderErrorsPlan records generated error symbol names and error context labels.
type RenderErrorsPlan struct {
	NativeServerUnavailableErr  string
	MessageServerUnavailableErr string
	UnknownActiveContractErr    string
	Role                        string
	Category                    string
}

// BuildMethodRenderPlan projects a method contract plan into renderer-facing stream operations and symbols.
func BuildMethodRenderPlan(method MethodPlan, serviceName string) (MethodRenderPlan, error) {
	capability, err := ProjectStreamCapability(method.Contract.Stream, true)
	if err != nil {
		return MethodRenderPlan{}, err
	}

	nativeEntryMethod := method.GoName
	if method.Streaming != StreamingKindUnary {
		nativeEntryMethod = method.GoName + "Start"
	}
	messageEntryMethod := nativeEntryMethod
	nativeStreamRequestType := ""
	nativeStreamResponseType := ""
	if capability.Streaming {
		nativeStreamRequestType = serviceName + method.GoName + "NativeStreamRequest"
		nativeStreamResponseType = serviceName + method.GoName + "NativeStreamResponse"
	}
	shape := MethodRenderPlan{
		Stream: capability,
		Symbols: RenderSymbolsPlan{
			NativeEntryMethod:        nativeEntryMethod,
			MessageEntryMethod:       messageEntryMethod,
			NativeAdapterMethod:      nativeEntryMethod,
			MessageAdapterMethod:     messageEntryMethod,
			NativeStreamRequestType:  nativeStreamRequestType,
			NativeStreamResponseType: nativeStreamResponseType,
		},
		Errors: RenderErrorsPlan{
			NativeServerUnavailableErr:  serviceName + "NativeServerUnavailableErr",
			MessageServerUnavailableErr: serviceName + "MessageServerUnavailableErr",
			UnknownActiveContractErr:    serviceName + "UnknownActiveContractErr",
			Role:                        "entry",
			Category:                    "routing",
		},
	}
	method.RenderPlan = shape
	if err := validateMethodRenderPlan(method); err != nil {
		return MethodRenderPlan{}, err
	}
	return shape, nil
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
	if method.Streaming != StreamingKindUnary && (shape.Symbols.NativeStreamRequestType == "" || shape.Symbols.NativeStreamResponseType == "") {
		return fmt.Errorf("method %s native stream envelope symbols are incomplete", methodPlanName(method))
	}
	if shape.Errors.NativeServerUnavailableErr == "" || shape.Errors.MessageServerUnavailableErr == "" || shape.Errors.UnknownActiveContractErr == "" {
		return fmt.Errorf("method %s render errors are incomplete", methodPlanName(method))
	}
	return nil
}
