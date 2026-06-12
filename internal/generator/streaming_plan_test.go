package generator

import "testing"

func TestBuildStreamingPlanAttachesCapabilityProjectionFromDescriptor(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	methods := methodsByName(t, plan.Services[0].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertStreamCapabilities(t, methods["Unary"], StreamCapabilityContractPlan{})
	assertStreamCapabilities(t, methods["ClientStream"], StreamCapabilityContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	})
	assertStreamCapabilities(t, methods["ServerStream"], StreamCapabilityContractPlan{
		CanRecv: true,
	})
	assertStreamCapabilities(t, methods["BidiStream"], StreamCapabilityContractPlan{
		CanSend:      true,
		CanRecv:      true,
		CanCloseSend: true,
	})
}

func TestValidateMethodRenderPlanRejectsCapabilityMismatch(t *testing.T) {
	method := minimalStreamingMethod(StreamingKindClientStreaming)
	method.Contract.Stream = StreamCapabilityContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	}
	renderPlan, err := BuildMethodRenderPlan(method, "Streamer")
	if err != nil {
		t.Fatalf("BuildMethodRenderPlan() error = %v", err)
	}
	method.RenderPlan = renderPlan
	method.RenderPlan.Stream.FinishReturnsResponse = false

	if err := ValidateMethodRenderPlan(method); err == nil {
		t.Fatal("ValidateMethodRenderPlan() error = nil, want capability mismatch")
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
	method.Contract.Stream = StreamCapabilityContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	}

	renderPlan, err := BuildMethodRenderPlan(method, "Streamer")
	if err != nil {
		t.Fatalf("BuildMethodRenderPlan() error = %v", err)
	}
	if !renderPlan.Stream.RequiresCodec {
		t.Fatal("render capability RequiresCodec = false, want true")
	}
}

func TestValidateMethodContractPlanRejectsStreamCapabilityCapabilityMismatches(t *testing.T) {
	tests := []struct {
		name       string
		streaming  StreamingKind
		capability StreamCapabilityContractPlan
	}{
		{
			name:       "unary capability must be empty",
			streaming:  StreamingKindUnary,
			capability: StreamCapabilityContractPlan{CanRecv: true},
		},
		{
			name:       "client streaming finish returns response",
			streaming:  StreamingKindClientStreaming,
			capability: StreamCapabilityContractPlan{CanSend: true},
		},
		{
			name:       "server streaming cannot send",
			streaming:  StreamingKindServerStreaming,
			capability: StreamCapabilityContractPlan{CanSend: true, CanRecv: true, CanCloseSend: true},
		},
		{
			name:       "bidi streaming can close send",
			streaming:  StreamingKindBidiStreaming,
			capability: StreamCapabilityContractPlan{CanSend: true, CanRecv: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := minimalStreamingMethod(tt.streaming)
			method.Contract.Stream = tt.capability
			if err := ValidateMethodContractPlan(method); err == nil {
				t.Fatal("ValidateMethodContractPlan() error = nil, want capability mismatch")
			}
		})
	}
}

func assertStreamCapabilities(t *testing.T, method MethodPlan, want StreamCapabilityContractPlan) {
	t.Helper()
	if method.Contract.Stream != want {
		t.Fatalf("%s capability = %+v, want %+v", method.Name, method.Contract.Stream, want)
	}
	wantProjection := StreamCapabilityProjectionPlan{
		Streaming:             !want.IsZero(),
		CanSend:               want.CanSend,
		CanRecv:               want.CanRecv,
		CanCloseSend:          want.CanCloseSend,
		FinishReturnsResponse: want.FinishReturnsResponse,
		RequiresCodec:         true,
	}
	if method.RenderPlan.Stream != wantProjection {
		t.Fatalf("%s render capability = %+v, want %+v", method.Name, method.RenderPlan.Stream, wantProjection)
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
