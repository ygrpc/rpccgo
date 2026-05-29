package rpcruntime

import (
	"errors"
	"strings"
	"testing"
)

type fakeRegistryStreamSession struct {
	name string
}

func createRegistryStream(t *testing.T, registry *StreamRegistry[*StreamEntry]) StreamHandle {
	t.Helper()
	handle, err := registry.Create(NewStreamEntry(&fakeRegistryStreamSession{name: "stream"}))
	if err != nil {
		t.Fatalf("create stream entry: %v", err)
	}
	return handle
}

func TestStreamRegistryLifecycleOperations(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	handle := createRegistryStream(t, &registry)

	calls := []string{}
	if err := StreamRegistrySend(&registry, handle, func(session *fakeRegistryStreamSession) error {
		calls = append(calls, "send:"+session.name)
		return nil
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := StreamRegistryReceive(&registry, handle, func(session *fakeRegistryStreamSession) error {
		calls = append(calls, "receive:"+session.name)
		return nil
	}); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got := strings.Join(calls, ","); got != "send:stream,receive:stream" {
		t.Fatalf("calls = %q", got)
	}
}

func TestStreamRegistryLifecycleInvalidHandleAndNilRegistry(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]

	if err := StreamRegistryReceive[*fakeRegistryStreamSession](&registry, 0, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("zero handle returned %v, want invalid handle", err)
	}
	if err := StreamRegistryReceive[*fakeRegistryStreamSession](nil, 1, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("nil registry returned %v, want invalid handle", err)
	}
}

func TestStreamRegistryLifecycleCloseSendBlocksFurtherSend(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	handle := createRegistryStream(t, &registry)

	closed := 0
	if err := StreamRegistryCloseSend(&registry, handle, func(session *fakeRegistryStreamSession) error {
		closed++
		return nil
	}); err != nil {
		t.Fatalf("close send: %v", err)
	}
	if closed != 1 {
		t.Fatalf("close callback called %d times, want 1", closed)
	}
	if err := StreamRegistrySend[*fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("send after close returned %v, want send closed", err)
	}
	if err := StreamRegistryCloseSend[*fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("second close send returned %v, want send closed", err)
	}
	if err := StreamRegistryReceive[*fakeRegistryStreamSession](&registry, handle, nil); err != nil {
		t.Fatalf("receive after close send: %v", err)
	}
}

func TestStreamRegistryLifecycleFinishAndDoneConsumeHandle(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	finishHandle := createRegistryStream(t, &registry)
	doneHandle := createRegistryStream(t, &registry)

	if err := StreamRegistryFinish[*fakeRegistryStreamSession](&registry, finishHandle, nil); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if err := StreamRegistryFinish[*fakeRegistryStreamSession](&registry, finishHandle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("second finish returned %v, want invalid handle", err)
	}
	if err := StreamRegistryDone[*fakeRegistryStreamSession](&registry, doneHandle, nil); err != nil {
		t.Fatalf("done: %v", err)
	}
	if err := StreamRegistryReceive[*fakeRegistryStreamSession](&registry, doneHandle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive after done returned %v, want invalid handle", err)
	}
}

func TestStreamRegistryLifecycleCancelConsumesHandle(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	handle := createRegistryStream(t, &registry)

	calls := 0
	if err := StreamRegistryCancel(&registry, handle, func(session *fakeRegistryStreamSession) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if calls != 1 {
		t.Fatalf("cancel callback called %d times, want 1", calls)
	}
	if err := StreamRegistryCancel[*fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("second cancel returned %v, want invalid handle", err)
	}
	if err := StreamRegistryReceive[*fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive after cancel returned %v, want invalid handle", err)
	}
}

func TestStreamRegistryLifecycleTerminalCallbackErrorConsumesHandle(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	handle := createRegistryStream(t, &registry)
	wantErr := errors.New("finish failed")

	if err := StreamRegistryFinish(&registry, handle, func(session *fakeRegistryStreamSession) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("finish returned %v, want %v", err, wantErr)
	}
	if err := StreamRegistryReceive[*fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive after failed finish returned %v, want invalid handle", err)
	}
}

func TestStreamRegistryLifecycleCloseSendCallbackErrorKeepsHandleSendClosed(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	handle := createRegistryStream(t, &registry)
	wantErr := errors.New("close send failed")

	if err := StreamRegistryCloseSend(&registry, handle, func(session *fakeRegistryStreamSession) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("close send returned %v, want %v", err, wantErr)
	}
	if err := StreamRegistrySend[*fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("send after failed close send returned %v, want send closed", err)
	}
	if err := StreamRegistryReceive[*fakeRegistryStreamSession](&registry, handle, nil); err != nil {
		t.Fatalf("receive after failed close send: %v", err)
	}
}

func TestStreamRegistryLifecycleWrongSessionTypeDoesNotConsumeHandle(t *testing.T) {
	var registry StreamRegistry[*StreamEntry]
	handle := createRegistryStream(t, &registry)

	if err := StreamRegistryFinish[fakeRegistryStreamSession](&registry, handle, nil); !errors.Is(err, ErrStreamSessionTypeMismatch) {
		t.Fatalf("wrong typed registry returned %v, want session type mismatch", err)
	}
	called := false
	if err := StreamRegistryReceive(&registry, handle, func(session *fakeRegistryStreamSession) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("receive after wrong registry: %v", err)
	}
	if !called {
		t.Fatal("receive did not observe live stream after wrong registry operation")
	}
}
