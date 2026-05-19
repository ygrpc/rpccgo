package generator

import "fmt"

func BuildStreamingPlan(method MethodPlan, facts methodContractFacts, serviceName string) (MethodPlan, error) {
	renderShape, err := BuildMethodRenderPlan(method, facts, serviceName)
	if err != nil {
		return MethodPlan{}, err
	}
	method.RenderShape = renderShape
	if err := ValidateMethodRenderPlan(method); err != nil {
		return MethodPlan{}, err
	}
	return method, nil
}

func expectedLifecyclePlan(streaming StreamingKind) (LifecyclePlan, error) {
	switch streaming {
	case StreamingKindUnary:
		return LifecyclePlan{}, nil
	case StreamingKindClientStreaming:
		return LifecyclePlan{
			HasStart:        true,
			HasSend:         true,
			HasFinish:       true,
			HasCancel:       true,
			CancelFinalizes: true,
			TerminalKind:    LifecycleTerminalFinishResult,
		}, nil
	case StreamingKindServerStreaming:
		return LifecyclePlan{
			HasStart:        true,
			HasCancel:       true,
			CancelFinalizes: true,
			HasOnRead:       true,
			HasOnDone:       true,
			TerminalKind:    LifecycleTerminalOnDone,
		}, nil
	case StreamingKindBidiStreaming:
		return LifecyclePlan{
			HasStart:        true,
			HasSend:         true,
			HasCloseSend:    true,
			HasCancel:       true,
			CancelFinalizes: true,
			HasOnRead:       true,
			HasOnDone:       true,
			TerminalKind:    LifecycleTerminalOnDone,
		}, nil
	default:
		return LifecyclePlan{}, fmt.Errorf("unknown streaming kind %d", streaming)
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
