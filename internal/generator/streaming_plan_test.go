package generator

import (
	"reflect"
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

	if len(plan.Services) != 1 {
		t.Fatalf("Services = %d, want 1", len(plan.Services))
	}
	methods := methodsByName(t, plan.Services[0].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertLifecycle(t, methods["Unary"], LifecyclePlan{})
	assertLifecycle(t, methods["ClientStream"], LifecyclePlan{
		HasStart:        true,
		HasSend:         true,
		HasFinish:       true,
		HasCancel:       true,
		CancelFinalizes: true,
		TerminalKind:    LifecycleTerminalFinishResult,
	})
	assertLifecycle(t, methods["ServerStream"], LifecyclePlan{
		HasStart:        true,
		HasCancel:       true,
		CancelFinalizes: true,
		HasOnRead:       true,
		HasOnDone:       true,
		TerminalKind:    LifecycleTerminalOnDone,
	})
	assertLifecycle(t, methods["BidiStream"], LifecyclePlan{
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

func TestBuildStreamingPlanPreservesMethodPlanMetadata(t *testing.T) {
	method := MethodPlan{
		Name:      "Upload",
		GoName:    "Upload",
		FullName:  "test.v1.Streamer.Upload",
		Streaming: StreamingKindClientStreaming,
		Request: MethodIOPlan{
			GoName:       "UploadRequest",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.UploadRequest",
		},
		Response: MethodIOPlan{
			GoName:       "UploadReply",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.UploadReply",
		},
		NativeContract: NativeContractPlan{
			RequestFields: []FieldPlan{
				{
					Name:     "payload",
					GoName:   "Payload",
					FullName: "test.v1.UploadRequest.payload",
					Kind:     FieldKindBytes,
					Native: NativeFieldPlan{
						Kind:  NativeFieldKindBytes,
						Shape: NativeABIShapeScalar,
					},
				},
			},
			ResponseFields: []FieldPlan{
				{
					Name:     "accepted",
					GoName:   "Accepted",
					FullName: "test.v1.UploadReply.accepted",
					Kind:     FieldKindBool,
					Native: NativeFieldPlan{
						Kind:  NativeFieldKindBool,
						Shape: NativeABIShapeBoolByte,
					},
				},
			},
		},
		MessageContract: MessageContractPlan{
			RequestType: MethodIOPlan{
				GoName:       "UploadRequest",
				GoImportPath: "example.com/test/v1",
				FullName:     "test.v1.UploadRequest",
			},
			ResponseType: MethodIOPlan{
				GoName:       "UploadReply",
				GoImportPath: "example.com/test/v1",
				FullName:     "test.v1.UploadReply",
			},
		},
		NeedsCodec: true,
		RequestBody: []FieldPlan{
			{
				Name:     "payload",
				GoName:   "Payload",
				FullName: "test.v1.UploadRequest.payload",
				Kind:     FieldKindBytes,
				Native: NativeFieldPlan{
					Kind:  NativeFieldKindBytes,
					Shape: NativeABIShapeScalar,
				},
			},
		},
		ResponseBody: []FieldPlan{
			{
				Name:     "accepted",
				GoName:   "Accepted",
				FullName: "test.v1.UploadReply.accepted",
				Kind:     FieldKindBool,
				Native: NativeFieldPlan{
					Kind:  NativeFieldKindBool,
					Shape: NativeABIShapeBoolByte,
				},
			},
		},
	}
	want := method

	got, err := BuildStreamingPlan(method)
	if err != nil {
		t.Fatalf("BuildStreamingPlan() error = %v", err)
	}

	want.Lifecycle = LifecyclePlan{
		HasStart:        true,
		HasSend:         true,
		HasFinish:       true,
		HasCancel:       true,
		CancelFinalizes: true,
		TerminalKind:    LifecycleTerminalFinishResult,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildStreamingPlan() = %#v, want metadata preserved with lifecycle %#v", got, want)
	}
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

func TestBuildStreamingPlanRejectsUnknownStreamingKind(t *testing.T) {
	method := MethodPlan{
		Name:      "Mystery",
		FullName:  "test.v1.Streamer.Mystery",
		Streaming: StreamingKind(99),
	}

	_, err := BuildStreamingPlan(method)
	if err == nil {
		t.Fatal("BuildStreamingPlan() error = nil, want unknown streaming kind error")
	}
	got := err.Error()
	for _, want := range []string{"test.v1.Streamer.Mystery", "unknown streaming kind 99"} {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildStreamingPlan() error = %q, want substring %q", got, want)
		}
	}
}

func assertLifecycle(t *testing.T, got MethodPlan, want LifecyclePlan) {
	t.Helper()

	if got.Lifecycle != want {
		t.Fatalf("%s Lifecycle = %#v, want %#v", got.Name, got.Lifecycle, want)
	}
}

func methodsByName(t *testing.T, methods []MethodPlan, wantNames ...string) map[string]MethodPlan {
	t.Helper()

	if len(methods) != len(wantNames) {
		t.Fatalf("Methods = %d, want %d", len(methods), len(wantNames))
	}
	byName := make(map[string]MethodPlan, len(methods))
	for _, method := range methods {
		if _, exists := byName[method.Name]; exists {
			t.Fatalf("duplicate method name %q", method.Name)
		}
		byName[method.Name] = method
	}
	for _, name := range wantNames {
		if _, exists := byName[name]; !exists {
			t.Fatalf("method %q not found in descriptor plan", name)
		}
	}
	return byName
}
