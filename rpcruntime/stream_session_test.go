package rpcruntime

import (
	"errors"
	"sync"
	"testing"
)

func TestStreamSessionSendAfterClose(t *testing.T) {
	var lifecycle StreamLifecycle

	if err := lifecycle.EnsureCanSend(); err != nil {
		t.Fatalf("EnsureCanSend returned error before close: %v", err)
	}
	if err := lifecycle.MarkSendClosed(); err != nil {
		t.Fatalf("MarkSendClosed returned error: %v", err)
	}
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("EnsureCanSend returned %v, want ErrStreamSendClosed", err)
	}
}

func TestStreamSessionDoubleClose(t *testing.T) {
	var lifecycle StreamLifecycle

	if err := lifecycle.MarkSendClosed(); err != nil {
		t.Fatalf("first MarkSendClosed returned error: %v", err)
	}
	if err := lifecycle.MarkSendClosed(); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("second MarkSendClosed returned %v, want ErrStreamSendClosed", err)
	}
}

func TestStreamSessionCancelFinalizes(t *testing.T) {
	var lifecycle StreamLifecycle
	called := 0

	if err := lifecycle.Cancel(func() error {
		called++
		return nil
	}); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if called != 1 {
		t.Fatalf("cancel callback called %d times, want 1", called)
	}
	if !lifecycle.Finalized() {
		t.Fatal("Cancel did not finalize lifecycle")
	}
	if !lifecycle.Canceled() {
		t.Fatal("Cancel did not mark lifecycle canceled")
	}
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, ErrStreamCanceled) {
		t.Fatalf("EnsureCanSend returned %v, want ErrStreamCanceled", err)
	}
	if lifecycle.Finalize() {
		t.Fatal("Finalize returned true after Cancel")
	}
}

func TestStreamSessionCancelNilFinalizes(t *testing.T) {
	var lifecycle StreamLifecycle

	if err := lifecycle.Cancel(nil); err != nil {
		t.Fatalf("Cancel(nil) returned error: %v", err)
	}
	if !lifecycle.Finalized() {
		t.Fatal("Cancel(nil) did not finalize lifecycle")
	}
	if !lifecycle.Canceled() {
		t.Fatal("Cancel(nil) did not mark lifecycle canceled")
	}
}

func TestStreamSessionCancelErrorStillFinalizes(t *testing.T) {
	var lifecycle StreamLifecycle
	cancelErr := errors.New("cancel failed")

	if err := lifecycle.Cancel(func() error {
		return cancelErr
	}); !errors.Is(err, cancelErr) {
		t.Fatalf("Cancel returned %v, want cancelErr", err)
	}
	if !lifecycle.Finalized() {
		t.Fatal("Cancel with callback error did not finalize lifecycle")
	}
	if !lifecycle.Canceled() {
		t.Fatal("Cancel with callback error did not mark lifecycle canceled")
	}
}

func TestStreamSessionFinishFinalizes(t *testing.T) {
	var lifecycle StreamLifecycle

	if !lifecycle.Finalize() {
		t.Fatal("Finalize returned false for first finish")
	}
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, ErrStreamFinalized) {
		t.Fatalf("EnsureCanSend returned %v, want ErrStreamFinalized", err)
	}
}

func TestStreamSessionFinishFinalizesRegistryHandle(t *testing.T) {
	var registry StreamRegistry
	lifecycle := &StreamLifecycle{}

	handle, err := registry.Create(lifecycle)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	finishSession, ok := registry.Take(handle)
	if !ok {
		t.Fatal("Take returned false for finish")
	}
	finishLifecycle, ok := finishSession.(*StreamLifecycle)
	if !ok {
		t.Fatalf("Take returned session %#v, want *StreamLifecycle", finishSession)
	}
	if !finishLifecycle.Finalize() {
		t.Fatal("Finalize returned false for first finish")
	}
	if _, ok := registry.Load(handle); ok {
		t.Fatal("Load returned true after finish Take")
	}
	if _, ok := registry.Take(handle); ok {
		t.Fatal("Take returned true after finish finalized handle")
	}
}

func TestStreamSessionCancelAfterRegistryTakeFinalizesHandle(t *testing.T) {
	var registry StreamRegistry
	lifecycle := &StreamLifecycle{}
	called := 0

	handle, err := registry.Create(lifecycle)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	cancelSession, ok := registry.Take(handle)
	if !ok {
		t.Fatal("Take returned false for cancel")
	}
	cancelLifecycle, ok := cancelSession.(*StreamLifecycle)
	if !ok {
		t.Fatalf("Take returned session %#v, want *StreamLifecycle", cancelSession)
	}
	if err := cancelLifecycle.Cancel(func() error {
		called++
		return nil
	}); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if called != 1 {
		t.Fatalf("cancel callback called %d times, want 1", called)
	}
	if _, ok := registry.Load(handle); ok {
		t.Fatal("Load returned true after cancel Take")
	}
	if _, ok := registry.Take(handle); ok {
		t.Fatal("Take returned true after cancel finalized handle")
	}
}

func TestStreamSessionDoubleTerminalOperationOnlySucceedsOnce(t *testing.T) {
	var lifecycle StreamLifecycle
	called := 0

	if !lifecycle.Finalize() {
		t.Fatal("Finalize returned false for first terminal operation")
	}
	if err := lifecycle.Cancel(func() error {
		called++
		return nil
	}); !errors.Is(err, ErrStreamFinalized) {
		t.Fatalf("Cancel returned %v, want ErrStreamFinalized", err)
	}
	if called != 0 {
		t.Fatalf("cancel callback called after finalization %d times, want 0", called)
	}
	if lifecycle.Finalize() {
		t.Fatal("second Finalize returned true")
	}
}

func TestStreamSessionCancelTerminalOperationOnlySucceedsOnce(t *testing.T) {
	var lifecycle StreamLifecycle
	called := 0

	if err := lifecycle.Cancel(func() error {
		called++
		return nil
	}); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if err := lifecycle.Cancel(func() error {
		called++
		return nil
	}); !errors.Is(err, ErrStreamCanceled) {
		t.Fatalf("second Cancel returned %v, want ErrStreamCanceled", err)
	}
	if called != 1 {
		t.Fatalf("cancel callback called %d times, want 1", called)
	}
	if lifecycle.Finalize() {
		t.Fatal("Finalize returned true after Cancel")
	}
}

func TestStreamSessionConcurrentCancelAndFinalizeOnlyOneTerminalWins(t *testing.T) {
	const attempts = 200

	for attempt := 0; attempt < attempts; attempt++ {
		var lifecycle StreamLifecycle
		start := make(chan struct{})
		var wg sync.WaitGroup
		var callbackMu sync.Mutex
		callbackCalls := 0
		var finalizeOK bool
		var cancelErr error

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			finalizeOK = lifecycle.Finalize()
		}()
		go func() {
			defer wg.Done()
			<-start
			cancelErr = lifecycle.Cancel(func() error {
				callbackMu.Lock()
				defer callbackMu.Unlock()
				callbackCalls++
				return nil
			})
		}()

		close(start)
		wg.Wait()

		cancelOK := cancelErr == nil
		if finalizeOK && cancelOK {
			t.Fatalf("attempt %d: Finalize and Cancel both succeeded", attempt)
		}
		if !finalizeOK && !cancelOK {
			if !errors.Is(cancelErr, ErrStreamFinalized) {
				t.Fatalf("attempt %d: Cancel returned %v, want nil or ErrStreamFinalized", attempt, cancelErr)
			}
		}
		callbackMu.Lock()
		calls := callbackCalls
		callbackMu.Unlock()
		if calls > 1 {
			t.Fatalf("attempt %d: cancel callback called %d times, want at most 1", attempt, calls)
		}
		if cancelOK && calls != 1 {
			t.Fatalf("attempt %d: successful Cancel called callback %d times, want 1", attempt, calls)
		}
		if !cancelOK && calls != 0 {
			t.Fatalf("attempt %d: failed Cancel called callback %d times, want 0", attempt, calls)
		}
	}
}
