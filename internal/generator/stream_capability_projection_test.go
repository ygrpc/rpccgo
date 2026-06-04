package generator

import "testing"

func TestProjectStreamCapability(t *testing.T) {
	tests := []struct {
		name       string
		capability StreamCapabilityContractPlan
		needsCodec bool
	}{
		{name: "unary"},
		{
			name: "client streaming",
			capability: StreamCapabilityContractPlan{
				CanSend:               true,
				FinishReturnsResponse: true,
			},
			needsCodec: true,
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
			got, err := ProjectStreamCapability(tt.capability, tt.needsCodec)
			if err != nil {
				t.Fatalf("ProjectStreamCapability() error = %v", err)
			}
			want := StreamCapabilityProjectionPlan{
				Streaming:             !tt.capability.IsZero(),
				CanSend:               tt.capability.CanSend,
				CanRecv:               tt.capability.CanRecv,
				CanCloseSend:          tt.capability.CanCloseSend,
				FinishReturnsResponse: tt.capability.FinishReturnsResponse,
				RequiresCodec:         tt.needsCodec,
			}
			if got != want {
				t.Fatalf("ProjectStreamCapability() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestProjectStreamCapabilityRejectsInvalidCapabilities(t *testing.T) {
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
			if _, err := ProjectStreamCapability(tt.capability, false); err == nil {
				t.Fatal("ProjectStreamCapability() error = nil, want invalid capabilities error")
			}
		})
	}
}
