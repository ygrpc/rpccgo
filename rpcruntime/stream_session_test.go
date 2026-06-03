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

func TestStreamSessionMarkCanceledFinalizes(t *testing.T) {
	var lifecycle StreamLifecycle

	if err := lifecycle.MarkCanceled(); err != nil {
		t.Fatalf("MarkCanceled returned error: %v", err)
	}
	if !lifecycle.Finalized() {
		t.Fatal("MarkCanceled did not finalize lifecycle")
	}
	if !lifecycle.Canceled() {
		t.Fatal("MarkCanceled did not mark lifecycle canceled")
	}
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, ErrStreamCanceled) {
		t.Fatalf("EnsureCanSend returned %v, want ErrStreamCanceled", err)
	}
	if lifecycle.Finalize() {
		t.Fatal("Finalize returned true after Cancel")
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
	if err := cancelLifecycle.MarkCanceled(); err != nil {
		t.Fatalf("MarkCanceled returned error: %v", err)
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

	if !lifecycle.Finalize() {
		t.Fatal("Finalize returned false for first terminal operation")
	}
	if err := lifecycle.MarkCanceled(); !errors.Is(err, ErrStreamFinalized) {
		t.Fatalf("MarkCanceled returned %v, want ErrStreamFinalized", err)
	}
	if lifecycle.Finalize() {
		t.Fatal("second Finalize returned true")
	}
}

func TestStreamSessionCancelTerminalOperationOnlySucceedsOnce(t *testing.T) {
	var lifecycle StreamLifecycle

	if err := lifecycle.MarkCanceled(); err != nil {
		t.Fatalf("MarkCanceled returned error: %v", err)
	}
	if err := lifecycle.MarkCanceled(); !errors.Is(err, ErrStreamCanceled) {
		t.Fatalf("second MarkCanceled returned %v, want ErrStreamCanceled", err)
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
			cancelErr = lifecycle.MarkCanceled()
		}()

		close(start)
		wg.Wait()

		cancelOK := cancelErr == nil
		if finalizeOK && cancelOK {
			t.Fatalf("attempt %d: Finalize and Cancel both succeeded", attempt)
		}
		if !finalizeOK && !cancelOK {
			if !errors.Is(cancelErr, ErrStreamFinalized) {
				t.Fatalf("attempt %d: MarkCanceled returned %v, want nil or ErrStreamFinalized", attempt, cancelErr)
			}
		}
	}
}
