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

	got, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle)
	if !ok {
		t.Fatalf("stream handle %d was not registered", handle)
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
	if _, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle); ok {
		t.Fatalf("zero handle %d should not load a session", handle)
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
	if _, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, 1); ok {
		t.Fatal("failed stream start leaked the first stream handle")
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
		{
			name: "nil session",
			create: func(AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
				return nil, nil
			},
		},
		{
			name: "zero struct session",
			create: func(AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
				return fakeDispatcherStreamSession{}, nil
			},
		},
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
			if _, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, 1); ok {
				t.Fatal("zero session stream start leaked the first stream handle")
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

	got, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle)
	if !ok {
		t.Fatalf("stream handle %d was not registered", handle)
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

func TestDispatcherStreamStartRejectsNilCreateCallback(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	handle, err := dispatcher.StartStream(nil)
	if err == nil {
		t.Fatal("expected nil create callback error")
	}
	if !strings.Contains(err.Error(), "create") {
		t.Fatalf("unexpected nil create callback error %q, want it to mention create", err.Error())
	}
	if handle != 0 {
		t.Fatalf("start stream returned handle %d, want zero", handle)
	}
}

func TestDispatcherStreamHelpersLoadTakeDeleteTypedSessions(t *testing.T) {
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

	if _, ok := LoadDispatcherStream[*fakeDispatcherAdapter, fakeDispatcherStreamSession](&dispatcher, handle); ok {
		t.Fatal("LoadDispatcherStream returned true for mismatched session type")
	}
	if _, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](nil, handle); ok {
		t.Fatal("LoadDispatcherStream returned true for nil dispatcher")
	}

	loaded, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle)
	if !ok {
		t.Fatal("LoadDispatcherStream returned false for created stream")
	}
	if loaded.name != "stream" {
		t.Fatalf("LoadDispatcherStream returned session %q, want stream", loaded.name)
	}

	taken, ok := TakeDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle)
	if !ok {
		t.Fatal("TakeDispatcherStream returned false for created stream")
	}
	if taken != loaded {
		t.Fatalf("TakeDispatcherStream returned %#v, want %#v", taken, loaded)
	}
	if _, ok := LoadDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle); ok {
		t.Fatal("LoadDispatcherStream returned true after TakeDispatcherStream")
	}
	if _, ok := TakeDispatcherStream[*fakeDispatcherAdapter, *fakeDispatcherStreamSession](&dispatcher, handle); ok {
		t.Fatal("TakeDispatcherStream returned true after session was already taken")
	}

	secondHandle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*fakeDispatcherAdapter]) (any, error) {
		return &fakeDispatcherStreamSession{snapshot: snapshot, name: "delete"}, nil
	})
	if err != nil {
		t.Fatalf("start second stream: %v", err)
	}
	if !DeleteDispatcherStream(&dispatcher, secondHandle) {
		t.Fatal("DeleteDispatcherStream returned false for created stream")
	}
	if DeleteDispatcherStream(&dispatcher, secondHandle) {
		t.Fatal("DeleteDispatcherStream returned true for repeated delete")
	}
	if DeleteDispatcherStream[*fakeDispatcherAdapter](nil, secondHandle) {
		t.Fatal("DeleteDispatcherStream returned true for nil dispatcher")
	}
}
