package generator

import "testing"

func TestBuildStreamingPlanAttachesRenderPlanToDescriptorPlan(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	methods := methodsByName(t, plan.Services[0].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertRenderSession(t, methods["Unary"], SessionKindNone)
	assertRenderOperations(t, methods["Unary"])
	assertRenderSession(t, methods["ClientStream"], SessionKindClient)
	assertRenderOperations(t, methods["ClientStream"], SessionOperationStart, SessionOperationSend, SessionOperationFinish, SessionOperationCancel)
	assertRenderTerminal(t, methods["ClientStream"], TerminalKindFinish, SessionOperationFinish)
	assertRenderSession(t, methods["ServerStream"], SessionKindServer)
	assertRenderOperations(t, methods["ServerStream"], SessionOperationStart, SessionOperationReceive, SessionOperationDone, SessionOperationCancel)
	assertRenderTerminal(t, methods["ServerStream"], TerminalKindDone, SessionOperationDone)
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
			Session: SessionRenderPlan{Kind: SessionKindClient, Operations: []SessionOperationPlan{{Kind: SessionOperationStart, Enabled: true}}},
			Symbols: RenderSymbolsPlan{NativeAdapterMethod: "Unary", MessageAdapterMethod: "UnaryMessage"},
			Errors:  RenderErrorsPlan{NativeAdapterUnavailableErr: "Native", MessageAdapterUnavailableErr: "Message", UnknownActiveContractErr: "Unknown", NativeMessageConverterErr: "Converter"},
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

func assertRenderSession(t *testing.T, method MethodPlan, want SessionKind) {
	t.Helper()
	if method.RenderPlan.Session.Kind != want {
		t.Fatalf("%s Session.Kind = %q, want %q", method.Name, method.RenderPlan.Session.Kind, want)
	}
	if err := ValidateMethodRenderPlan(method); err != nil {
		t.Fatalf("%s ValidateMethodRenderPlan() error = %v", method.Name, err)
	}
}

func assertRenderOperations(t *testing.T, method MethodPlan, want ...SessionOperationKind) {
	t.Helper()
	got := method.RenderPlan.Session.Operations
	if len(got) != len(want) {
		t.Fatalf("%s operations = %d, want %d: %#v", method.Name, len(got), len(want), got)
	}
	for i, op := range got {
		if op.Kind != want[i] || !op.Enabled {
			t.Fatalf("%s operation[%d] = %#v, want enabled %q", method.Name, i, op, want[i])
		}
	}
}

func assertRenderTerminal(t *testing.T, method MethodPlan, wantKind TerminalKind, wantOperation SessionOperationKind) {
	t.Helper()
	terminal := method.RenderPlan.Terminal
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
