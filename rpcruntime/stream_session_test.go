package rpcruntime

import (
	"errors"
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
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, errStreamSendClosed) {
		t.Fatalf("EnsureCanSend returned %v, want errStreamSendClosed", err)
	}
}

func TestStreamSessionDoubleClose(t *testing.T) {
	var lifecycle StreamLifecycle

	if err := lifecycle.MarkSendClosed(); err != nil {
		t.Fatalf("first MarkSendClosed returned error: %v", err)
	}
	if err := lifecycle.MarkSendClosed(); !errors.Is(err, errStreamSendClosed) {
		t.Fatalf("second MarkSendClosed returned %v, want errStreamSendClosed", err)
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
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, errStreamCanceled) {
		t.Fatalf("EnsureCanSend returned %v, want errStreamCanceled", err)
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
	if err := lifecycle.EnsureCanSend(); !errors.Is(err, errStreamFinalized) {
		t.Fatalf("EnsureCanSend returned %v, want errStreamFinalized", err)
	}
}

func TestStreamSessionOnDoneFinalizes(t *testing.T) {
	var registry StreamRegistry[*StreamLifecycle]
	lifecycle := &StreamLifecycle{}

	handle, err := registry.Create(lifecycle)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	onDoneLifecycle, ok := registry.Take(handle)
	if !ok {
		t.Fatal("Take returned false for onDone")
	}
	if !onDoneLifecycle.Finalize() {
		t.Fatal("Finalize returned false for first onDone")
	}
	if _, ok := registry.Load(handle); ok {
		t.Fatal("Load returned true after onDone Take")
	}
	if _, ok := registry.Take(handle); ok {
		t.Fatal("Take returned true after onDone finalized handle")
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
	}); !errors.Is(err, errStreamFinalized) {
		t.Fatalf("Cancel returned %v, want errStreamFinalized", err)
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
	}); !errors.Is(err, errStreamCanceled) {
		t.Fatalf("second Cancel returned %v, want errStreamCanceled", err)
	}
	if called != 1 {
		t.Fatalf("cancel callback called %d times, want 1", called)
	}
	if lifecycle.Finalize() {
		t.Fatal("Finalize returned true after Cancel")
	}
}
