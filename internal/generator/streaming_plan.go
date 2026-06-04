package generator

import "fmt"

func BuildStreamingPlan(method MethodPlan, serviceName string) (MethodPlan, error) {
	method, err := AttachMethodStreamCapabilityPlan(method)
	if err != nil {
		return MethodPlan{}, err
	}
	renderPlan, err := BuildMethodRenderPlan(method, serviceName)
	if err != nil {
		return MethodPlan{}, err
	}
	method.RenderPlan = renderPlan
	if err := ValidateMethodContractPlan(method); err != nil {
		return MethodPlan{}, err
	}
	if err := ValidateMethodRenderPlan(method); err != nil {
		return MethodPlan{}, err
	}
	return method, nil
}

func AttachMethodStreamCapabilityPlan(method MethodPlan) (MethodPlan, error) {
	capability, err := expectedStreamCapabilityPlan(method.Streaming)
	if err != nil {
		return MethodPlan{}, err
	}
	method.Contract.Stream = capability
	return method, nil
}

func expectedStreamCapabilityPlan(streaming StreamingKind) (StreamCapabilityContractPlan, error) {
	switch streaming {
	case StreamingKindUnary:
		return StreamCapabilityContractPlan{}, nil
	case StreamingKindClientStreaming:
		return StreamCapabilityContractPlan{
			CanSend:               true,
			FinishReturnsResponse: true,
		}, nil
	case StreamingKindServerStreaming:
		return StreamCapabilityContractPlan{
			CanRecv: true,
		}, nil
	case StreamingKindBidiStreaming:
		return StreamCapabilityContractPlan{
			CanSend:      true,
			CanRecv:      true,
			CanCloseSend: true,
		}, nil
	default:
		return StreamCapabilityContractPlan{}, fmt.Errorf("unknown streaming kind %d", streaming)
	}
}

func streamingKindName(streaming StreamingKind) string {
	switch streaming {
	case StreamingKindUnary:
		return "unary"
	case StreamingKindClientStreaming:
		return "client_streaming"
	case StreamingKindServerStreaming:
		return "server_streaming"
	case StreamingKindBidiStreaming:
		return "bidi_streaming"
	default:
		return fmt.Sprintf("unknown_streaming_kind_%d", streaming)
	}
}

func methodPlanName(method MethodPlan) string {
	if method.FullName != "" {
		return method.FullName
	}
	if method.Name != "" {
		return method.Name
	}
	return "<unknown>"
}
