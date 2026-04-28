package generator

import "fmt"

func BuildStreamingPlan(method MethodPlan) (MethodPlan, error) {
	lifecycle, err := expectedLifecyclePlan(method.Streaming)
	if err != nil {
		return MethodPlan{}, fmt.Errorf("method %s: %w", methodPlanName(method), err)
	}
	method.Lifecycle = lifecycle

	if err := ValidateStreamingLifecyclePlan(method); err != nil {
		return MethodPlan{}, err
	}
	return method, nil
}

func ValidateStreamingLifecyclePlan(method MethodPlan) error {
	want, err := expectedLifecyclePlan(method.Streaming)
	if err != nil {
		return fmt.Errorf("method %s: %w", methodPlanName(method), err)
	}
	if method.Lifecycle != want {
		return fmt.Errorf("method %s %s: invalid lifecycle matrix: got %#v, want %#v",
			methodPlanName(method), streamingKindName(method.Streaming), method.Lifecycle, want)
	}
	return nil
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
