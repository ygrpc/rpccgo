package rpcruntime

import (
	"errors"
	"io"
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

func TestStreamSessionCancelAfterRegistryTakeFinalizesHandle(t *testing.T) {
	var registry StreamRegistry[*StreamLifecycle]
	lifecycle := &StreamLifecycle{}
	called := 0

	handle, err := registry.Create(lifecycle)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	cancelLifecycle, ok := registry.Take(handle)
	if !ok {
		t.Fatal("Take returned false for cancel")
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
			if !errors.Is(cancelErr, errStreamFinalized) {
				t.Fatalf("attempt %d: Cancel returned %v, want nil or errStreamFinalized", attempt, cancelErr)
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

func TestRunServerStreamCallsDoneAfterEOF(t *testing.T) {
	recvCalls := 0
	sendCalls := 0
	doneCalls := 0
	cancelCalls := 0

	err := RunServerStream(
		func() (string, error) {
			recvCalls++
			return "", io.EOF
		},
		func(string) error {
			sendCalls++
			return nil
		},
		func() error {
			doneCalls++
			return nil
		},
		func() error {
			cancelCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("RunServerStream returned error: %v", err)
	}
	if recvCalls != 1 || sendCalls != 0 || doneCalls != 1 || cancelCalls != 0 {
		t.Fatalf("calls recv=%d send=%d done=%d cancel=%d, want 1/0/1/0", recvCalls, sendCalls, doneCalls, cancelCalls)
	}
}

func TestRunServerStreamCancelsAfterRecvError(t *testing.T) {
	recvErr := errors.New("recv failed")
	cancelCalls := 0

	err := RunServerStream(
		func() (string, error) {
			return "", recvErr
		},
		func(string) error {
			t.Fatal("send should not be called after recv error")
			return nil
		},
		func() error {
			t.Fatal("done should not be called after recv error")
			return nil
		},
		func() error {
			cancelCalls++
			return nil
		},
	)
	if !errors.Is(err, recvErr) {
		t.Fatalf("RunServerStream returned %v, want recvErr", err)
	}
	if cancelCalls != 1 {
		t.Fatalf("cancel called %d times, want 1", cancelCalls)
	}
}

func TestRunBidiStreamCloseSendAfterReceiveEOF(t *testing.T) {
	receiveCalls := 0
	closeSendCalls := 0
	doneCalls := 0
	cancelCalls := 0

	err := RunBidiStream(
		func() (string, error) {
			receiveCalls++
			return "", io.EOF
		},
		func(string) error {
			t.Fatal("send to session should not be called after receive EOF")
			return nil
		},
		func() error {
			closeSendCalls++
			return nil
		},
		func() (string, error) {
			return "", io.EOF
		},
		func(string) error {
			t.Fatal("send to peer should not be called after response EOF")
			return nil
		},
		func() error {
			doneCalls++
			return nil
		},
		func() error {
			cancelCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("RunBidiStream returned error: %v", err)
	}
	if receiveCalls != 1 || closeSendCalls != 1 || doneCalls != 1 || cancelCalls != 0 {
		t.Fatalf("calls receive=%d closeSend=%d done=%d cancel=%d, want 1/1/1/0", receiveCalls, closeSendCalls, doneCalls, cancelCalls)
	}
}
