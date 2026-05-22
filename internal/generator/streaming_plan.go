package generator

import "fmt"

func BuildStreamingPlan(method MethodPlan, serviceName string) (MethodPlan, error) {
	method, err := AttachMethodLifecyclePlan(method)
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

func AttachMethodLifecyclePlan(method MethodPlan) (MethodPlan, error) {
	lifecycle, err := expectedLifecyclePlan(method.Streaming)
	if err != nil {
		return MethodPlan{}, err
	}
	method.Contract.Lifecycle = lifecycle
	return method, nil
}

func expectedLifecyclePlan(streaming StreamingKind) (StreamLifecycleContractPlan, error) {
	op := func(kind StreamLifecycleOperationKind) StreamLifecycleOperationPlan {
		return StreamLifecycleOperationPlan{Kind: kind}
	}
	switch streaming {
	case StreamingKindUnary:
		return StreamLifecycleContractPlan{}, nil
	case StreamingKindClientStreaming:
		return StreamLifecycleContractPlan{
			Operations: []StreamLifecycleOperationPlan{
				op(StreamLifecycleOperationStart),
				op(StreamLifecycleOperationSend),
				op(StreamLifecycleOperationFinish),
				op(StreamLifecycleOperationCancel),
			},
			CancelFinalizes: true,
			TerminalKind:    LifecycleTerminalFinishResult,
		}, nil
	case StreamingKindServerStreaming:
		return StreamLifecycleContractPlan{
			Operations: []StreamLifecycleOperationPlan{
				op(StreamLifecycleOperationStart),
				op(StreamLifecycleOperationReceive),
				op(StreamLifecycleOperationDone),
				op(StreamLifecycleOperationCancel),
			},
			CancelFinalizes: true,
			TerminalKind:    LifecycleTerminalOnDone,
		}, nil
	case StreamingKindBidiStreaming:
		return StreamLifecycleContractPlan{
			Operations: []StreamLifecycleOperationPlan{
				op(StreamLifecycleOperationStart),
				op(StreamLifecycleOperationSend),
				op(StreamLifecycleOperationReceive),
				op(StreamLifecycleOperationCloseSend),
				op(StreamLifecycleOperationDone),
				op(StreamLifecycleOperationCancel),
			},
			CancelFinalizes: true,
			TerminalKind:    LifecycleTerminalOnDone,
		}, nil
	default:
		return StreamLifecycleContractPlan{}, fmt.Errorf("unknown streaming kind %d", streaming)
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
