package generator

import "testing"

func TestBuildStreamingPlanAttachesRenderPlanToDescriptorPlan(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	methods := methodsByName(t, plan.Services[0].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertLifecycleOperations(t, methods["Unary"])
	assertRenderSession(t, methods["Unary"], SessionKindNone)
	assertRenderOperations(t, methods["Unary"])
	assertLifecycleOperations(t, methods["ClientStream"], StreamLifecycleOperationStart, StreamLifecycleOperationSend, StreamLifecycleOperationFinish, StreamLifecycleOperationCancel)
	assertLifecycleTerminal(t, methods["ClientStream"], LifecycleTerminalFinishResult)
	assertRenderSession(t, methods["ClientStream"], SessionKindClient)
	assertRenderOperations(t, methods["ClientStream"], SessionOperationStart, SessionOperationSend, SessionOperationFinish, SessionOperationCancel)
	assertRenderTerminal(t, methods["ClientStream"], TerminalKindFinish, SessionOperationFinish)
	assertLifecycleOperations(t, methods["ServerStream"], StreamLifecycleOperationStart, StreamLifecycleOperationReceive, StreamLifecycleOperationDone, StreamLifecycleOperationCancel)
	assertLifecycleTerminal(t, methods["ServerStream"], LifecycleTerminalOnDone)
	assertRenderSession(t, methods["ServerStream"], SessionKindServer)
	assertRenderOperations(t, methods["ServerStream"], SessionOperationStart, SessionOperationReceive, SessionOperationDone, SessionOperationCancel)
	assertRenderTerminal(t, methods["ServerStream"], TerminalKindDone, SessionOperationDone)
	assertLifecycleOperations(t, methods["BidiStream"], StreamLifecycleOperationStart, StreamLifecycleOperationSend, StreamLifecycleOperationReceive, StreamLifecycleOperationCloseSend, StreamLifecycleOperationDone, StreamLifecycleOperationCancel)
	assertLifecycleTerminal(t, methods["BidiStream"], LifecycleTerminalOnDone)
	assertRenderSession(t, methods["BidiStream"], SessionKindBidi)
	assertRenderOperations(t, methods["BidiStream"], SessionOperationStart, SessionOperationSend, SessionOperationReceive, SessionOperationCloseSend, SessionOperationDone, SessionOperationCancel)
	assertRenderTerminal(t, methods["BidiStream"], TerminalKindDone, SessionOperationDone)
}

func TestValidateMethodRenderPlanRejectsInvalidMatrix(t *testing.T) {
	method := MethodPlan{
		Name:      "Unary",
		FullName:  "test.v1.Streamer.Unary",
		Streaming: StreamingKindUnary,
		RenderPlan: MethodRenderPlan{
			Lifecycle: StreamLifecycleProjectionPlan{SessionKind: SessionKindClient, Operations: []SessionOperationPlan{{Kind: SessionOperationStart}}},
			Symbols:   RenderSymbolsPlan{NativeAdapterMethod: "Unary", MessageAdapterMethod: "Unary"},
			Errors:    RenderErrorsPlan{NativeServerUnavailableErr: "Native", MessageServerUnavailableErr: "Message", UnknownActiveContractErr: "Unknown", NativeMessageConverterErr: "Converter"},
		},
	}
	if err := ValidateMethodRenderPlan(method); err == nil {
		t.Fatal("ValidateMethodRenderPlan() error = nil, want invalid render matrix error")
	}
}

func TestBuildStreamingPlanRejectsUnknownStreamingKind(t *testing.T) {
	method := MethodPlan{Name: "Mystery", FullName: "test.v1.Streamer.Mystery", Streaming: StreamingKind(99)}
	_, err := BuildStreamingPlan(method, "Streamer")
	if err == nil {
		t.Fatal("BuildStreamingPlan() error = nil, want unknown streaming kind error")
	}
}

func TestValidateMethodRenderPlanKeepsCodecInRenderInputs(t *testing.T) {
	method := minimalStreamingMethod(StreamingKindClientStreaming)
	method.NeedsCodec = true
	method.Contract.RenderInputs.NeedsCodec = true
	method.Contract.Lifecycle = StreamLifecycleContractPlan{
		Operations: []StreamLifecycleOperationPlan{
			{Kind: StreamLifecycleOperationStart},
			{Kind: StreamLifecycleOperationSend},
			{Kind: StreamLifecycleOperationFinish},
		},
		TerminalKind: LifecycleTerminalFinishResult,
	}

	renderPlan, err := BuildMethodRenderPlan(method, "Streamer")
	if err != nil {
		t.Fatalf("BuildMethodRenderPlan() error = %v", err)
	}
	if !renderPlan.Lifecycle.RequiresCodec {
		t.Fatal("render lifecycle RequiresCodec = false, want true from render input")
	}

	method.Contract.RenderInputs.NeedsCodec = false
	if err := ValidateMethodContractPlan(method); err == nil {
		t.Fatal("ValidateMethodContractPlan() error = nil, want codec render input mismatch")
	}
}

func TestValidateMethodContractPlanRejectsLifecyclePolicyMismatches(t *testing.T) {
	tests := []struct {
		name      string
		streaming StreamingKind
		lifecycle StreamLifecycleContractPlan
	}{
		{
			name:      "unary lifecycle must be empty",
			streaming: StreamingKindUnary,
			lifecycle: StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationStart}}},
		},
		{
			name:      "streaming lifecycle must have start",
			streaming: StreamingKindClientStreaming,
			lifecycle: StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationFinish}}, TerminalKind: LifecycleTerminalFinishResult},
		},
		{
			name:      "client streaming must finish with result",
			streaming: StreamingKindClientStreaming,
			lifecycle: StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationStart}, {Kind: StreamLifecycleOperationDone}}, TerminalKind: LifecycleTerminalOnDone},
		},
		{
			name:      "server streaming must terminate on done",
			streaming: StreamingKindServerStreaming,
			lifecycle: StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationStart}, {Kind: StreamLifecycleOperationFinish}}, TerminalKind: LifecycleTerminalFinishResult},
		},
		{
			name:      "cancel operation must finalize",
			streaming: StreamingKindServerStreaming,
			lifecycle: StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationStart}, {Kind: StreamLifecycleOperationReceive}, {Kind: StreamLifecycleOperationDone}, {Kind: StreamLifecycleOperationCancel}}, TerminalKind: LifecycleTerminalOnDone},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := minimalStreamingMethod(tt.streaming)
			method.Contract.Lifecycle = tt.lifecycle
			if err := ValidateMethodContractPlan(method); err == nil {
				t.Fatal("ValidateMethodContractPlan() error = nil, want lifecycle policy error")
			}
		})
	}
}

func assertLifecycleOperations(t *testing.T, method MethodPlan, want ...StreamLifecycleOperationKind) {
	t.Helper()
	got := method.Contract.Lifecycle.Operations
	if len(got) != len(want) {
		t.Fatalf("%s lifecycle operations = %d, want %d: %#v", method.Name, len(got), len(want), got)
	}
	for i, op := range got {
		if op.Kind != want[i] {
			t.Fatalf("%s lifecycle operation[%d] = %#v, want %q", method.Name, i, op, want[i])
		}
		if !method.Contract.Lifecycle.HasOperation(want[i]) {
			t.Fatalf("%s lifecycle HasOperation(%q) = false", method.Name, want[i])
		}
	}
}

func assertLifecycleTerminal(t *testing.T, method MethodPlan, want LifecycleTerminalKind) {
	t.Helper()
	if method.Contract.Lifecycle.TerminalKind != want || !method.Contract.Lifecycle.CancelFinalizes {
		t.Fatalf("%s lifecycle terminal = %#v, want terminal %q with cancel finalizes", method.Name, method.Contract.Lifecycle, want)
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
	method.Contract.RenderInputs.NeedsCodec = method.NeedsCodec
	return method
}

func assertRenderSession(t *testing.T, method MethodPlan, want SessionKind) {
	t.Helper()
	if method.RenderPlan.Lifecycle.SessionKind != want {
		t.Fatalf("%s Session.Kind = %q, want %q", method.Name, method.RenderPlan.Lifecycle.SessionKind, want)
	}
	if err := ValidateMethodRenderPlan(method); err != nil {
		t.Fatalf("%s ValidateMethodRenderPlan() error = %v", method.Name, err)
	}
}

func assertRenderOperations(t *testing.T, method MethodPlan, want ...SessionOperationKind) {
	t.Helper()
	got := method.RenderPlan.Lifecycle.Operations
	if len(got) != len(want) {
		t.Fatalf("%s operations = %d, want %d: %#v", method.Name, len(got), len(want), got)
	}
	for i, op := range got {
		if op.Kind != want[i] {
			t.Fatalf("%s operation[%d] = %#v, want %q", method.Name, i, op, want[i])
		}
	}
}

func assertRenderTerminal(t *testing.T, method MethodPlan, wantKind TerminalKind, wantOperation SessionOperationKind) {
	t.Helper()
	terminal := method.RenderPlan.Lifecycle.Terminal
	if terminal.Kind != wantKind || terminal.Operation != wantOperation || !terminal.ReleasesHandle {
		t.Fatalf("%s terminal = %#v, want kind %q operation %q releasing handle", method.Name, terminal, wantKind, wantOperation)
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
