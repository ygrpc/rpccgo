package generator

import "fmt"

// StreamCapabilityProjectionPlan records renderer-facing stream operations and codec requirements.
type StreamCapabilityProjectionPlan struct {
	Streaming             bool
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
	RequiresCodec         bool
}

// ProjectStreamCapability validates contract stream capabilities and projects them for renderers.
func ProjectStreamCapability(capability StreamCapabilityContractPlan, needsCodec bool) (StreamCapabilityProjectionPlan, error) {
	plan := StreamCapabilityProjectionPlan{
		Streaming:             !capability.IsZero(),
		CanSend:               capability.CanSend,
		CanRecv:               capability.CanRecv,
		CanCloseSend:          capability.CanCloseSend,
		FinishReturnsResponse: capability.FinishReturnsResponse,
		RequiresCodec:         needsCodec,
	}
	if err := validateStreamCapabilities(capability); err != nil {
		return StreamCapabilityProjectionPlan{}, err
	}
	return plan, nil
}

func validateStreamCapabilities(capability StreamCapabilityContractPlan) error {
	switch capability {
	case StreamCapabilityContractPlan{}:
		return nil
	case StreamCapabilityContractPlan{CanSend: true, FinishReturnsResponse: true}:
		return nil
	case StreamCapabilityContractPlan{CanRecv: true}:
		return nil
	case StreamCapabilityContractPlan{CanSend: true, CanRecv: true, CanCloseSend: true}:
		return nil
	default:
		return fmt.Errorf("invalid stream capabilities")
	}
}
