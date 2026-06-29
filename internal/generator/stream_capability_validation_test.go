package generator

import "testing"

func TestValidateStreamCapabilitiesAcceptsValidCapabilities(t *testing.T) {
	tests := []struct {
		name       string
		capability StreamCapabilityContractPlan
	}{
		{name: "unary"},
		{
			name: "client streaming",
			capability: StreamCapabilityContractPlan{
				CanSend:               true,
				FinishReturnsResponse: true,
			},
		},
		{
			name:       "server streaming",
			capability: StreamCapabilityContractPlan{CanRecv: true},
		},
		{
			name: "bidi streaming",
			capability: StreamCapabilityContractPlan{
				CanSend:      true,
				CanRecv:      true,
				CanCloseSend: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateStreamCapabilities(tt.capability); err != nil {
				t.Fatalf("validateStreamCapabilities() error = %v", err)
			}
		})
	}
}

func TestValidateStreamCapabilitiesRejectsInvalidCapabilities(t *testing.T) {
	tests := []struct {
		name       string
		capability StreamCapabilityContractPlan
	}{
		{
			name:       "send without response",
			capability: StreamCapabilityContractPlan{CanSend: true},
		},
		{
			name:       "close send without bidi",
			capability: StreamCapabilityContractPlan{CanCloseSend: true},
		},
		{
			name: "bidi returns finish response",
			capability: StreamCapabilityContractPlan{
				CanSend:               true,
				CanRecv:               true,
				CanCloseSend:          true,
				FinishReturnsResponse: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateStreamCapabilities(tt.capability); err == nil {
				t.Fatal("validateStreamCapabilities() error = nil, want invalid capabilities error")
			}
		})
	}
}
