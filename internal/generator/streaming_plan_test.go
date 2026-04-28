package generator

import (
	"strings"
	"testing"
)

func TestBuildStreamingPlanBuildsLifecycleByKind(t *testing.T) {
	tests := []struct {
		name      string
		streaming StreamingKind
		want      LifecyclePlan
	}{
		{
			name:      "Unary",
			streaming: StreamingKindUnary,
			want:      LifecyclePlan{},
		},
		{
			name:      "ClientStream",
			streaming: StreamingKindClientStreaming,
			want: LifecyclePlan{
				HasStart:        true,
				HasSend:         true,
				HasFinish:       true,
				HasCancel:       true,
				CancelFinalizes: true,
				TerminalKind:    LifecycleTerminalFinishResult,
			},
		},
		{
			name:      "ServerStream",
			streaming: StreamingKindServerStreaming,
			want: LifecyclePlan{
				HasStart:        true,
				HasCancel:       true,
				CancelFinalizes: true,
				HasOnRead:       true,
				HasOnDone:       true,
				TerminalKind:    LifecycleTerminalOnDone,
			},
		},
		{
			name:      "BidiStream",
			streaming: StreamingKindBidiStreaming,
			want: LifecyclePlan{
				HasStart:        true,
				HasSend:         true,
				HasCloseSend:    true,
				HasCancel:       true,
				CancelFinalizes: true,
				HasOnRead:       true,
				HasOnDone:       true,
				TerminalKind:    LifecycleTerminalOnDone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := MethodPlan{
				Name:      tt.name,
				FullName:  "test.v1.Streamer." + tt.name,
				Streaming: tt.streaming,
			}

			got, err := BuildStreamingPlan(method)
			if err != nil {
				t.Fatalf("BuildStreamingPlan() error = %v", err)
			}
			if got.Lifecycle != tt.want {
				t.Fatalf("Lifecycle = %#v, want %#v", got.Lifecycle, tt.want)
			}
		})
	}
}

func TestBuildStreamingPlanAttachesLifecycleToDescriptorPlan(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	methods := plan.Services[0].Methods
	assertLifecycle(t, methods[0], LifecyclePlan{})
	assertLifecycle(t, methods[1], LifecyclePlan{
		HasStart:        true,
		HasSend:         true,
		HasFinish:       true,
		HasCancel:       true,
		CancelFinalizes: true,
		TerminalKind:    LifecycleTerminalFinishResult,
	})
	assertLifecycle(t, methods[2], LifecyclePlan{
		HasStart:        true,
		HasCancel:       true,
		CancelFinalizes: true,
		HasOnRead:       true,
		HasOnDone:       true,
		TerminalKind:    LifecycleTerminalOnDone,
	})
	assertLifecycle(t, methods[3], LifecyclePlan{
		HasStart:        true,
		HasSend:         true,
		HasCloseSend:    true,
		HasCancel:       true,
		CancelFinalizes: true,
		HasOnRead:       true,
		HasOnDone:       true,
		TerminalKind:    LifecycleTerminalOnDone,
	})
}

func TestValidateStreamingLifecyclePlanRejectsInvalidMatrix(t *testing.T) {
	tests := []struct {
		name      string
		method    MethodPlan
		wantParts []string
	}{
		{
			name: "unary operation",
			method: MethodPlan{
				Name:      "Unary",
				FullName:  "test.v1.Streamer.Unary",
				Streaming: StreamingKindUnary,
				Lifecycle: LifecyclePlan{
					HasStart: true,
				},
			},
			wantParts: []string{"test.v1.Streamer.Unary", "unary", "invalid lifecycle matrix"},
		},
		{
			name: "client stream cancel does not finalize",
			method: MethodPlan{
				Name:      "ClientStream",
				FullName:  "test.v1.Streamer.ClientStream",
				Streaming: StreamingKindClientStreaming,
				Lifecycle: LifecyclePlan{
					HasStart:     true,
					HasSend:      true,
					HasFinish:    true,
					HasCancel:    true,
					TerminalKind: LifecycleTerminalFinishResult,
				},
			},
			wantParts: []string{"test.v1.Streamer.ClientStream", "client_streaming", "invalid lifecycle matrix"},
		},
		{
			name: "server stream has send",
			method: MethodPlan{
				Name:      "ServerStream",
				FullName:  "test.v1.Streamer.ServerStream",
				Streaming: StreamingKindServerStreaming,
				Lifecycle: LifecyclePlan{
					HasStart:        true,
					HasSend:         true,
					HasCancel:       true,
					CancelFinalizes: true,
					HasOnRead:       true,
					HasOnDone:       true,
					TerminalKind:    LifecycleTerminalOnDone,
				},
			},
			wantParts: []string{"test.v1.Streamer.ServerStream", "server_streaming", "invalid lifecycle matrix"},
		},
		{
			name: "bidi stream wrong terminal",
			method: MethodPlan{
				Name:      "BidiStream",
				FullName:  "test.v1.Streamer.BidiStream",
				Streaming: StreamingKindBidiStreaming,
				Lifecycle: LifecyclePlan{
					HasStart:        true,
					HasSend:         true,
					HasCloseSend:    true,
					HasCancel:       true,
					CancelFinalizes: true,
					HasOnRead:       true,
					HasOnDone:       true,
					TerminalKind:    LifecycleTerminalFinishResult,
				},
			},
			wantParts: []string{"test.v1.Streamer.BidiStream", "bidi_streaming", "invalid lifecycle matrix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStreamingLifecyclePlan(tt.method)
			if err == nil {
				t.Fatal("ValidateStreamingLifecyclePlan() error = nil, want invalid matrix error")
			}
			got := err.Error()
			for _, want := range tt.wantParts {
				if !strings.Contains(got, want) {
					t.Fatalf("ValidateStreamingLifecyclePlan() error = %q, want substring %q", got, want)
				}
			}
		})
	}
}

func assertLifecycle(t *testing.T, got MethodPlan, want LifecyclePlan) {
	t.Helper()

	if got.Lifecycle != want {
		t.Fatalf("%s Lifecycle = %#v, want %#v", got.Name, got.Lifecycle, want)
	}
}
