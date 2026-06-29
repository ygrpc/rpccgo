package generator

import "fmt"

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
