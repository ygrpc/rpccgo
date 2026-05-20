package rpcruntime

import (
	"errors"
	"strings"
	"testing"
)

type fakeDispatcherStreamSession struct {
	snapshot AdapterSnapshot[*fakeDispatcherAdapter]
	name     string
}

func TestDispatcherStreamStartCapturesActiveServerSnapshot(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	firstAdapter := &fakeDispatcherAdapter{name: "first"}
	secondAdapter := &fakeDispatcherAdapter{name: "second"}

	first, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("register first adapter: %v", err)
	}

	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}
	if handle == 0 {
		t.Fatal("start stream returned zero handle")
	}

	second, err := dispatcher.Register(ServerKindConnectHandler, ServerContractMessage, secondAdapter)
	if err != nil {
		t.Fatalf("register second adapter: %v", err)
	}
	if second.Version == first.Version {
		t.Fatalf("expected second registration to use a new version, got %d", second.Version)
	}

	var got *fakeDispatcherStreamSession
	if err := DispatcherStreamReceive(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		got = session
		return nil
	}); err != nil {
		t.Fatalf("receive stream: %v", err)
	}
	if got.snapshot.Adapter != firstAdapter || got.snapshot.Version != first.Version {
		t.Fatalf("stream did not keep first snapshot: got %#v, want adapter %#v version %d", got.snapshot, firstAdapter, first.Version)
	}
	if got.snapshot.Adapter == secondAdapter || got.snapshot.Version == second.Version {
		t.Fatalf("stream snapshot was replaced by later registration: got %#v, later %#v", got.snapshot, second)
	}
}

func TestDispatcherStreamStartWithoutActiveServerReturnsErrorAndNoHandle(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]

	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot}, nil
	})
	if err == nil {
		t.Fatal("expected start stream error")
	}
	if !strings.Contains(err.Error(), "active server") {
		t.Fatalf("unexpected start stream error %q, want it to mention active server", err.Error())
	}
	if handle != 0 {
		t.Fatalf("start stream returned handle %d, want zero", handle)
	}
	if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("zero handle returned %v, want invalid handle", err)
	}
}

func TestDispatcherStreamStartNilCreateErrorTakesPrecedenceBeforeCapture(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]

	handle, err := dispatcher.StartStream(nil)
	if err == nil {
		t.Fatal("expected nil create callback error")
	}
	if !strings.Contains(err.Error(), "create") {
		t.Fatalf("unexpected nil create callback error %q, want it to mention create", err.Error())
	}
	if strings.Contains(err.Error(), "active server") {
		t.Fatalf("nil create error should take precedence over active server capture: %q", err.Error())
	}
	if handle != 0 {
		t.Fatalf("start stream returned handle %d, want zero", handle)
	}
}

func TestDispatcherStreamStartCreateFailureDoesNotLeakHandle(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	wantErr := errors.New("create stream failed")
	handle, err := dispatcher.StartStream(func(AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected start stream error: got %v, want %v", err, wantErr)
	}
	if handle != 0 {
		t.Fatalf("start stream returned handle %d, want zero", handle)
	}
	if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, 1, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("failed stream start leaked handle: %v", err)
	}

	nextHandle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "next"}, nil
	})
	if err != nil {
		t.Fatalf("start stream after create failure: %v", err)
	}
	if nextHandle != 1 {
		t.Fatalf("start stream after create failure returned handle %d, want 1", nextHandle)
	}
}

func TestDispatcherStreamStartZeroSessionDoesNotLeakHandle(t *testing.T) {
	tests := []struct {
		name   string
		create func(AdapterSnapshot[*fakeDispatcherAdapter]) (any, error)
	}{
		{name: "nil session", create: func(AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) { return nil, nil }},
		{name: "zero struct session", create: func(AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) { return fakeDispatcherStreamSession{}, nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dispatcher Dispatcher[*fakeDispatcherAdapter]
			if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
				t.Fatalf("register adapter: %v", err)
			}

			handle, err := dispatcher.StartStream(tt.create)
			if !errors.Is(err, errStreamRegistryZeroSession) {
				t.Fatalf("unexpected start stream error: got %v, want %v", err, errStreamRegistryZeroSession)
			}
			if handle != 0 {
				t.Fatalf("start stream returned handle %d, want zero", handle)
			}
			if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, 1, nil); !errors.Is(err, ErrStreamInvalidHandle) {
				t.Fatalf("zero session stream start leaked handle: %v", err)
			}

			nextHandle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
				return &fakeDispatcherStreamSession{snapshot: snapshot, name: "next"}, nil
			})
			if err != nil {
				t.Fatalf("start stream after zero session: %v", err)
			}
			if nextHandle != 1 {
				t.Fatalf("start stream after zero session returned handle %d, want 1", nextHandle)
			}
		})
	}
}

func TestDispatcherStreamStartReentrantRegisterKeepsStartSnapshot(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	firstAdapter := &fakeDispatcherAdapter{name: "first"}
	secondAdapter := &fakeDispatcherAdapter{name: "second"}

	first, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("register first adapter: %v", err)
	}

	var createSnapshot AdapterSnapshot[*fakeDispatcherAdapter]
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		createSnapshot = snapshot
		if _, err := dispatcher.Register(ServerKindConnectHandler, ServerContractMessage, secondAdapter); err != nil {
			t.Fatalf("register second adapter during stream start: %v", err)
		}
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}
	if createSnapshot.Adapter != firstAdapter || createSnapshot.Version != first.Version {
		t.Fatalf("create received snapshot %#v, want adapter %#v version %d", createSnapshot, firstAdapter, first.Version)
	}

	var got *fakeDispatcherStreamSession
	if err := DispatcherStreamReceive(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		got = session
		return nil
	}); err != nil {
		t.Fatalf("receive stream: %v", err)
	}
	if got.snapshot.Adapter != firstAdapter || got.snapshot.Version != first.Version {
		t.Fatalf("stream did not keep first snapshot: got %#v, want adapter %#v version %d", got.snapshot, firstAdapter, first.Version)
	}

	latest, err := dispatcher.Capture()
	if err != nil {
		t.Fatalf("capture latest: %v", err)
	}
	if latest.Adapter != secondAdapter {
		t.Fatalf("later capture did not see the second adapter: %#v", latest)
	}
}

func TestDispatcherStreamExecutorOperations(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	calls := []string{}
	if err := DispatcherStreamSend(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		calls = append(calls, "send:"+session.name)
		return nil
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := DispatcherStreamReceive(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		calls = append(calls, "receive:"+session.name)
		return nil
	}); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got := strings.Join(calls, ","); got != "send:stream,receive:stream" {
		t.Fatalf("calls = %q", got)
	}
}

func TestDispatcherStreamCloseSendBlocksFurtherSend(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	closed := 0
	if err := DispatcherStreamCloseSend(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		closed++
		return nil
	}); err != nil {
		t.Fatalf("close send: %v", err)
	}
	if closed != 1 {
		t.Fatalf("close callback called %d times, want 1", closed)
	}
	if err := DispatcherStreamSend[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("send after close returned %v, want %v", err, ErrStreamSendClosed)
	}
	if err := DispatcherStreamCloseSend[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("second close send returned %v, want %v", err, ErrStreamSendClosed)
	}
}

func TestDispatcherStreamFinishConsumesHandleOnce(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	calls := 0
	if err := DispatcherStreamFinish(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if calls != 1 {
		t.Fatalf("finish callback called %d times, want 1", calls)
	}
	if err := DispatcherStreamFinish[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("second finish returned %v, want invalid handle", err)
	}
	if calls != 1 {
		t.Fatalf("finish callback called after invalid handle; calls=%d", calls)
	}
}

func TestDispatcherStreamCancelConsumesHandleOnce(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	calls := 0
	if err := DispatcherStreamCancel(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if calls != 1 {
		t.Fatalf("cancel callback called %d times, want 1", calls)
	}
	if err := DispatcherStreamCancel[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("second cancel returned %v, want invalid handle", err)
	}
	if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive after cancel returned %v, want invalid handle", err)
	}
}

func TestDispatcherStreamTerminalCallbackErrorConsumesHandle(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	wantErr := errors.New("finish failed")
	if err := DispatcherStreamFinish(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("finish returned %v, want %v", err, wantErr)
	}
	if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive after failed finish returned %v, want invalid handle", err)
	}
}

func TestDispatcherStreamCancelCallbackErrorConsumesHandle(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	wantErr := errors.New("cancel failed")
	if err := DispatcherStreamCancel(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("cancel returned %v, want %v", err, wantErr)
	}
	if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive after failed cancel returned %v, want invalid handle", err)
	}
}

func TestDispatcherStreamCloseSendCallbackErrorKeepsHandleSendClosed(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	wantErr := errors.New("close send failed")
	if err := DispatcherStreamCloseSend(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("close send returned %v, want %v", err, wantErr)
	}
	if err := DispatcherStreamSend[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("send after failed close send returned %v, want send closed", err)
	}
	if err := DispatcherStreamReceive[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle, nil); err != nil {
		t.Fatalf("receive after failed close send: %v", err)
	}
}

func TestDispatcherStreamTypeMismatchDoesNotConsumeHandle(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "stream"}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	if err := DispatcherStreamFinish[*fakeDispatcherAdapter, fakeDispatcherStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("finish with mismatched type returned %v, want invalid handle", err)
	}
	called := false
	if err := DispatcherStreamReceive(&dispatcher, handle, func(session *fakeDispatcherStreamSession) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("receive after type mismatch: %v", err)
	}
	if !called {
		t.Fatal("receive did not observe live stream after type mismatch")
	}
}
