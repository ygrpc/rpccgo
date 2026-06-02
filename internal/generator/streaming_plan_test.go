package generator

import "testing"

func TestBuildStreamingPlanAttachesCapabilityProjectionFromDescriptor(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	methods := methodsByName(t, plan.Services[0].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertLifecycleCapabilities(t, methods["Unary"], StreamLifecycleContractPlan{})
	assertLifecycleCapabilities(t, methods["ClientStream"], StreamLifecycleContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	})
	assertLifecycleCapabilities(t, methods["ServerStream"], StreamLifecycleContractPlan{
		CanRecv: true,
	})
	assertLifecycleCapabilities(t, methods["BidiStream"], StreamLifecycleContractPlan{
		CanSend:      true,
		CanRecv:      true,
		CanCloseSend: true,
	})
}

func TestValidateMethodRenderPlanRejectsCapabilityMismatch(t *testing.T) {
	method := minimalStreamingMethod(StreamingKindClientStreaming)
	method.Contract.Lifecycle = StreamLifecycleContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	}
	renderPlan, err := BuildMethodRenderPlan(method, "Streamer")
	if err != nil {
		t.Fatalf("BuildMethodRenderPlan() error = %v", err)
	}
	method.RenderPlan = renderPlan
	method.RenderPlan.Lifecycle.FinishReturnsResponse = false

	if err := ValidateMethodRenderPlan(method); err == nil {
		t.Fatal("ValidateMethodRenderPlan() error = nil, want lifecycle capability mismatch")
	}
}

func TestBuildStreamingPlanRejectsUnknownStreamingKind(t *testing.T) {
	method := MethodPlan{Name: "Mystery", FullName: "test.v1.Streamer.Mystery", Streaming: StreamingKind(99)}
	_, err := BuildStreamingPlan(method, "Streamer")
	if err == nil {
		t.Fatal("BuildStreamingPlan() error = nil, want unknown streaming kind error")
	}
}

func TestValidateMethodRenderPlanRequiresCodec(t *testing.T) {
	method := minimalStreamingMethod(StreamingKindClientStreaming)
	method.Contract.Lifecycle = StreamLifecycleContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	}

	renderPlan, err := BuildMethodRenderPlan(method, "Streamer")
	if err != nil {
		t.Fatalf("BuildMethodRenderPlan() error = %v", err)
	}
	if !renderPlan.Lifecycle.RequiresCodec {
		t.Fatal("render lifecycle RequiresCodec = false, want true")
	}
}

func TestValidateMethodContractPlanRejectsLifecycleCapabilityMismatches(t *testing.T) {
	tests := []struct {
		name      string
		streaming StreamingKind
		lifecycle StreamLifecycleContractPlan
	}{
		{
			name:      "unary lifecycle must be empty",
			streaming: StreamingKindUnary,
			lifecycle: StreamLifecycleContractPlan{CanRecv: true},
		},
		{
			name:      "client streaming finish returns response",
			streaming: StreamingKindClientStreaming,
			lifecycle: StreamLifecycleContractPlan{CanSend: true},
		},
		{
			name:      "server streaming cannot send",
			streaming: StreamingKindServerStreaming,
			lifecycle: StreamLifecycleContractPlan{CanSend: true, CanRecv: true, CanCloseSend: true},
		},
		{
			name:      "bidi streaming can close send",
			streaming: StreamingKindBidiStreaming,
			lifecycle: StreamLifecycleContractPlan{CanSend: true, CanRecv: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := minimalStreamingMethod(tt.streaming)
			method.Contract.Lifecycle = tt.lifecycle
			if err := ValidateMethodContractPlan(method); err == nil {
				t.Fatal("ValidateMethodContractPlan() error = nil, want lifecycle capability mismatch")
			}
		})
	}
}

func assertLifecycleCapabilities(t *testing.T, method MethodPlan, want StreamLifecycleContractPlan) {
	t.Helper()
	if method.Contract.Lifecycle != want {
		t.Fatalf("%s lifecycle = %+v, want %+v", method.Name, method.Contract.Lifecycle, want)
	}
	wantProjection := StreamLifecycleProjectionPlan{
		Streaming:             !want.IsZero(),
		CanSend:               want.CanSend,
		CanRecv:               want.CanRecv,
		CanCloseSend:          want.CanCloseSend,
		FinishReturnsResponse: want.FinishReturnsResponse,
		RequiresCodec:         true,
	}
	if method.RenderPlan.Lifecycle != wantProjection {
		t.Fatalf("%s render lifecycle = %+v, want %+v", method.Name, method.RenderPlan.Lifecycle, wantProjection)
	}
	if err := ValidateMethodRenderPlan(method); err != nil {
		t.Fatalf("%s ValidateMethodRenderPlan() error = %v", method.Name, err)
	}
}

func minimalStreamingMethod(streaming StreamingKind) MethodPlan {
	method := MethodPlan{
		Name:      "Method",
		GoName:    "Method",
		FullName:  "test.v1.Streamer.Method",
		Streaming: streaming,
		Request:   MethodIOPlan{GoName: "Request", FullName: "test.v1.Request"},
		Response:  MethodIOPlan{GoName: "Response", FullName: "test.v1.Response"},
	}
	method.Contract.Message.RequestType = method.Request
	method.Contract.Message.ResponseType = method.Response
	return method
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
