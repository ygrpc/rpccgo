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

	gotAny, ok := dispatcher.streams.Load(handle)
	if !ok {
		t.Fatalf("stream handle %d was not registered", handle)
	}
	got, ok := gotAny.(*fakeDispatcherStreamSession)
	if !ok {
		t.Fatalf("unexpected stream session type %T", gotAny)
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
	if _, ok := dispatcher.streams.Load(handle); ok {
		t.Fatalf("zero handle %d should not load a session", handle)
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
	if _, ok := dispatcher.streams.Load(1); ok {
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
