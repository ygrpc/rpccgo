package generator

import "testing"

func TestProjectStreamLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		lifecycle  StreamLifecycleContractPlan
		needsCodec bool
	}{
		{name: "unary"},
		{
			name: "client streaming",
			lifecycle: StreamLifecycleContractPlan{
				CanSend:               true,
				FinishReturnsResponse: true,
			},
			needsCodec: true,
		},
		{
			name:      "server streaming",
			lifecycle: StreamLifecycleContractPlan{CanRecv: true},
		},
		{
			name: "bidi streaming",
			lifecycle: StreamLifecycleContractPlan{
				CanSend:      true,
				CanRecv:      true,
				CanCloseSend: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectStreamLifecycle(tt.lifecycle, tt.needsCodec)
			if err != nil {
				t.Fatalf("ProjectStreamLifecycle() error = %v", err)
			}
			want := StreamLifecycleProjectionPlan{
				Streaming:             !tt.lifecycle.IsZero(),
				CanSend:               tt.lifecycle.CanSend,
				CanRecv:               tt.lifecycle.CanRecv,
				CanCloseSend:          tt.lifecycle.CanCloseSend,
				FinishReturnsResponse: tt.lifecycle.FinishReturnsResponse,
				RequiresCodec:         tt.needsCodec,
			}
			if got != want {
				t.Fatalf("ProjectStreamLifecycle() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestProjectStreamLifecycleRejectsInvalidCapabilities(t *testing.T) {
	tests := []struct {
		name      string
		lifecycle StreamLifecycleContractPlan
	}{
		{
			name:      "send without response",
			lifecycle: StreamLifecycleContractPlan{CanSend: true},
		},
		{
			name:      "close send without bidi",
			lifecycle: StreamLifecycleContractPlan{CanCloseSend: true},
		},
		{
			name: "bidi returns finish response",
			lifecycle: StreamLifecycleContractPlan{
				CanSend:               true,
				CanRecv:               true,
				CanCloseSend:          true,
				FinishReturnsResponse: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ProjectStreamLifecycle(tt.lifecycle, false); err == nil {
				t.Fatal("ProjectStreamLifecycle() error = nil, want invalid capabilities error")
			}
		})
	}
}
