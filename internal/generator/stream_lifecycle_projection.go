package generator

import "fmt"

type StreamLifecycleProjectionPlan struct {
	Streaming             bool
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
	RequiresCodec         bool
}

func ProjectStreamLifecycle(lifecycle StreamLifecycleContractPlan, needsCodec bool) (StreamLifecycleProjectionPlan, error) {
	plan := StreamLifecycleProjectionPlan{
		Streaming:             !lifecycle.IsZero(),
		CanSend:               lifecycle.CanSend,
		CanRecv:               lifecycle.CanRecv,
		CanCloseSend:          lifecycle.CanCloseSend,
		FinishReturnsResponse: lifecycle.FinishReturnsResponse,
		RequiresCodec:         needsCodec,
	}
	if err := validateStreamLifecycleCapabilities(lifecycle); err != nil {
		return StreamLifecycleProjectionPlan{}, err
	}
	return plan, nil
}

func validateStreamLifecycleCapabilities(lifecycle StreamLifecycleContractPlan) error {
	switch lifecycle {
	case StreamLifecycleContractPlan{}:
		return nil
	case StreamLifecycleContractPlan{CanSend: true, FinishReturnsResponse: true}:
		return nil
	case StreamLifecycleContractPlan{CanRecv: true}:
		return nil
	case StreamLifecycleContractPlan{CanSend: true, CanRecv: true, CanCloseSend: true}:
		return nil
	default:
		return fmt.Errorf("invalid stream lifecycle capabilities")
	}
}
