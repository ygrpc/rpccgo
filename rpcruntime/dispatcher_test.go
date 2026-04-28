package rpcruntime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDispatcherCaptureBeforeRegisterReturnsError(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]

	snapshot, err := dispatcher.Capture()
	if err == nil {
		t.Fatalf("expected capture error, got snapshot %#v", snapshot)
	}
	if !strings.Contains(err.Error(), "active server") {
		t.Fatalf("unexpected capture error %q, want it to mention active server", err.Error())
	}
}

func TestDispatcherInvokeRejectsNilCallback(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	err := dispatcher.Invoke(context.Background(), nil)
	if err == nil {
		t.Fatal("expected nil callback error")
	}
	if !strings.Contains(err.Error(), "invoke") {
		t.Fatalf("unexpected nil callback error %q, want it to mention invoke", err.Error())
	}
}

func TestDispatcherInvokeRejectsNilCallbackBeforeCapture(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]

	err := dispatcher.Invoke(context.Background(), nil)
	if err == nil {
		t.Fatal("expected nil callback error")
	}
	if !strings.Contains(err.Error(), "invoke") {
		t.Fatalf("unexpected nil callback error %q, want it to mention invoke", err.Error())
	}
	if strings.Contains(err.Error(), "active server") {
		t.Fatalf("nil callback error should take precedence over active server capture: %q", err.Error())
	}
}

func TestDispatcherRegisterAndInvoke(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	adapter := &fakeDispatcherAdapter{name: "first"}

	registered, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, adapter)
	if err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	ctx := context.WithValue(context.Background(), fakeDispatcherContextKey{}, "request")
	var invokedSnapshot AdapterSnapshot[*fakeDispatcherAdapter]
	err = dispatcher.Invoke(ctx, func(gotCtx context.Context, snapshot AdapterSnapshot[*fakeDispatcherAdapter]) error {
		if gotCtx != ctx {
			t.Fatalf("invoke received replaced context: got %#v, want %#v", gotCtx, ctx)
		}
		invokedSnapshot = snapshot
		return nil
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}

	if invokedSnapshot.Adapter != adapter {
		t.Fatalf("unexpected invoked adapter: got %#v, want %#v", invokedSnapshot.Adapter, adapter)
	}
	if invokedSnapshot.Kind != ServerKindGoNative || invokedSnapshot.Contract != ServerContractNative {
		t.Fatalf("unexpected invoked metadata: %#v", invokedSnapshot)
	}
	if invokedSnapshot.Version != registered.Version {
		t.Fatalf("unexpected invoked version: got %d, want %d", invokedSnapshot.Version, registered.Version)
	}
}

func TestDispatcherInvokeReturnsCallbackError(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &fakeDispatcherAdapter{name: "adapter"}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	wantErr := errors.New("callback failed")
	err := dispatcher.Invoke(context.Background(), func(context.Context, AdapterSnapshot[*fakeDispatcherAdapter]) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected invoke error: got %v, want %v", err, wantErr)
	}
}

func TestDispatcherLaterRegisterAffectsNewInvoke(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	firstAdapter := &fakeDispatcherAdapter{name: "first"}
	secondAdapter := &fakeDispatcherAdapter{name: "second"}

	first, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("register first adapter: %v", err)
	}

	var firstInvoke AdapterSnapshot[*fakeDispatcherAdapter]
	if err := dispatcher.Invoke(context.Background(), func(_ context.Context, snapshot AdapterSnapshot[*fakeDispatcherAdapter]) error {
		firstInvoke = snapshot
		return nil
	}); err != nil {
		t.Fatalf("first invoke: %v", err)
	}

	second, err := dispatcher.Register(ServerKindConnectHandler, ServerContractMessage, secondAdapter)
	if err != nil {
		t.Fatalf("register second adapter: %v", err)
	}

	var secondInvoke AdapterSnapshot[*fakeDispatcherAdapter]
	if err := dispatcher.Invoke(context.Background(), func(_ context.Context, snapshot AdapterSnapshot[*fakeDispatcherAdapter]) error {
		secondInvoke = snapshot
		return nil
	}); err != nil {
		t.Fatalf("second invoke: %v", err)
	}

	if firstInvoke.Adapter != firstAdapter || firstInvoke.Version != first.Version {
		t.Fatalf("unexpected first invoke snapshot: %#v", firstInvoke)
	}
	if secondInvoke.Adapter != secondAdapter || secondInvoke.Version != second.Version {
		t.Fatalf("unexpected second invoke snapshot: %#v", secondInvoke)
	}
	if secondInvoke.Version == firstInvoke.Version {
		t.Fatalf("expected later invoke to use a newer snapshot, got first %#v and second %#v", firstInvoke, secondInvoke)
	}
}

func TestDispatcherInvokeCapturesOnce(t *testing.T) {
	var dispatcher Dispatcher[*fakeDispatcherAdapter]
	firstAdapter := &fakeDispatcherAdapter{name: "first"}
	secondAdapter := &fakeDispatcherAdapter{name: "second"}

	first, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("register first adapter: %v", err)
	}

	var invokedSnapshot AdapterSnapshot[*fakeDispatcherAdapter]
	err = dispatcher.Invoke(context.Background(), func(ctx context.Context, snapshot AdapterSnapshot[*fakeDispatcherAdapter]) error {
		if _, err := dispatcher.Register(ServerKindConnectHandler, ServerContractMessage, secondAdapter); err != nil {
			t.Fatalf("register second adapter during invoke: %v", err)
		}
		invokedSnapshot = snapshot
		return nil
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}

	if invokedSnapshot.Adapter != firstAdapter || invokedSnapshot.Version != first.Version {
		t.Fatalf("invoke did not keep the start snapshot: got %#v, want adapter %#v version %d", invokedSnapshot, firstAdapter, first.Version)
	}

	latest, err := dispatcher.Capture()
	if err != nil {
		t.Fatalf("capture latest: %v", err)
	}
	if latest.Adapter != secondAdapter {
		t.Fatalf("later capture did not see the second adapter: %#v", latest)
	}
}

type fakeDispatcherContextKey struct{}
