package generator

import "testing"

func TestProjectStreamLifecycle(t *testing.T) {
	op := func(kind StreamLifecycleOperationKind) StreamLifecycleOperationPlan {
		return StreamLifecycleOperationPlan{Kind: kind}
	}
	tests := []struct {
		name              string
		lifecycle         StreamLifecycleContractPlan
		needsCodec        bool
		wantKind          SessionKind
		wantOperations    []SessionOperationKind
		wantTerminalKind  TerminalKind
		wantTerminalOp    SessionOperationKind
		wantResponseCodec bool
	}{
		{
			name:              "empty lifecycle",
			wantKind:          SessionKindNone,
			wantOperations:    nil,
			wantTerminalKind:  "",
			wantTerminalOp:    "",
			wantResponseCodec: false,
		},
		{
			name: "client streaming",
			lifecycle: StreamLifecycleContractPlan{
				Operations:      []StreamLifecycleOperationPlan{op(StreamLifecycleOperationStart), op(StreamLifecycleOperationSend), op(StreamLifecycleOperationFinish), op(StreamLifecycleOperationCancel)},
				CancelFinalizes: true,
				TerminalKind:    LifecycleTerminalFinishResult,
			},
			needsCodec:        true,
			wantKind:          SessionKindClient,
			wantOperations:    []SessionOperationKind{SessionOperationStart, SessionOperationSend, SessionOperationFinish, SessionOperationCancel},
			wantTerminalKind:  TerminalKindFinish,
			wantTerminalOp:    SessionOperationFinish,
			wantResponseCodec: true,
		},
		{
			name: "server streaming",
			lifecycle: StreamLifecycleContractPlan{
				Operations:      []StreamLifecycleOperationPlan{op(StreamLifecycleOperationStart), op(StreamLifecycleOperationReceive), op(StreamLifecycleOperationDone), op(StreamLifecycleOperationCancel)},
				CancelFinalizes: true,
				TerminalKind:    LifecycleTerminalOnDone,
			},
			wantKind:         SessionKindServer,
			wantOperations:   []SessionOperationKind{SessionOperationStart, SessionOperationReceive, SessionOperationDone, SessionOperationCancel},
			wantTerminalKind: TerminalKindDone,
			wantTerminalOp:   SessionOperationDone,
		},
		{
			name: "bidi streaming",
			lifecycle: StreamLifecycleContractPlan{
				Operations:      []StreamLifecycleOperationPlan{op(StreamLifecycleOperationStart), op(StreamLifecycleOperationSend), op(StreamLifecycleOperationReceive), op(StreamLifecycleOperationCloseSend), op(StreamLifecycleOperationDone), op(StreamLifecycleOperationCancel)},
				CancelFinalizes: true,
				TerminalKind:    LifecycleTerminalOnDone,
			},
			wantKind:         SessionKindBidi,
			wantOperations:   []SessionOperationKind{SessionOperationStart, SessionOperationSend, SessionOperationReceive, SessionOperationCloseSend, SessionOperationDone, SessionOperationCancel},
			wantTerminalKind: TerminalKindDone,
			wantTerminalOp:   SessionOperationDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectStreamLifecycle(tt.lifecycle, tt.needsCodec)
			if err != nil {
				t.Fatalf("ProjectStreamLifecycle() error = %v", err)
			}
			if got.SessionKind != tt.wantKind {
				t.Fatalf("SessionKind = %q, want %q", got.SessionKind, tt.wantKind)
			}
			assertProjectedOperations(t, got.Operations, tt.wantOperations)
			if got.Terminal.Kind != tt.wantTerminalKind || got.Terminal.Operation != tt.wantTerminalOp {
				t.Fatalf("Terminal = %#v, want kind %q operation %q", got.Terminal, tt.wantTerminalKind, tt.wantTerminalOp)
			}
			if got.RequiresCodec != tt.needsCodec {
				t.Fatalf("RequiresCodec = %v, want %v", got.RequiresCodec, tt.needsCodec)
			}
			if got.Terminal.RequiresResponseConvert != tt.wantResponseCodec {
				t.Fatalf("Terminal.RequiresResponseConvert = %v, want %v", got.Terminal.RequiresResponseConvert, tt.wantResponseCodec)
			}
			if got.CancelFinalizes != tt.lifecycle.CancelFinalizes {
				t.Fatalf("CancelFinalizes = %v, want %v", got.CancelFinalizes, tt.lifecycle.CancelFinalizes)
			}
		})
	}
}

func TestProjectStreamLifecycleRejectsInvalidOperationSet(t *testing.T) {
	_, err := ProjectStreamLifecycle(StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationStart}, {Kind: StreamLifecycleOperationFinish}}, TerminalKind: LifecycleTerminalFinishResult}, false)
	if err == nil {
		t.Fatal("ProjectStreamLifecycle() error = nil, want invalid lifecycle plan error")
	}
}

func TestProjectStreamLifecycleRejectsCancelWithoutFinalization(t *testing.T) {
	_, err := ProjectStreamLifecycle(StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{{Kind: StreamLifecycleOperationStart}, {Kind: StreamLifecycleOperationReceive}, {Kind: StreamLifecycleOperationDone}, {Kind: StreamLifecycleOperationCancel}}, TerminalKind: LifecycleTerminalOnDone}, false)
	if err == nil {
		t.Fatal("ProjectStreamLifecycle() error = nil, want cancel finalization error")
	}
}

func assertProjectedOperations(t *testing.T, got []SessionOperationPlan, want []SessionOperationKind) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("operations = %d, want %d: %#v", len(got), len(want), got)
	}
	for i, operation := range got {
		if operation.Kind != want[i] {
			t.Fatalf("operation[%d] = %#v, want %q", i, operation, want[i])
		}
	}
}
