package rpcruntime

import (
	"errors"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

const rpcInputCleanupDeadline = 2 * time.Second

func TestTakeErrorTextUsesUnifiedReleasePath(t *testing.T) {
	id := StoreError(errors.New("stored failure"))

	_, ptr, ok := TakeErrorText(id)
	if !ok {
		t.Fatal("expected stored error to be found")
	}
	if !Release(ptr) {
		t.Fatal("expected Release(ptr) to release error text")
	}
	if Release(ptr) {
		t.Fatal("expected pointer to be removed after release")
	}
}

func TestTakeErrorTextForExportPopulatesInt32LengthSlot(t *testing.T) {
	id := StoreError(errors.New("stored failure"))

	var textPtr uintptr
	var textLen int32 = -1
	status := TakeErrorTextForExport(int32(id), &textPtr, &textLen)
	if status != 0 {
		t.Fatalf("expected TakeErrorTextForExport to succeed, got status %d", status)
	}
	if textPtr == 0 {
		t.Fatal("expected TakeErrorTextForExport to populate the text pointer")
	}
	if textLen != int32(len("stored failure")) {
		t.Fatalf("unexpected error text length: got %d want %d", textLen, len("stored failure"))
	}
	if !Release(textPtr) {
		t.Fatal("expected exported error text pointer to be releasable")
	}
}

func TestTakeErrorTextForExportKeepsRecordAndReleasesPointerWhenLengthConversionFails(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)

	id := StoreError(errors.New("stored failure"))
	errorTextLengthToInt32ForExport = func(int) (int32, error) {
		return 0, errors.New("length overflow")
	}

	beforePinned := countPinnedEntriesForTesting()
	var textPtr uintptr = 99
	var textLen int32 = 99
	status := TakeErrorTextForExport(int32(id), &textPtr, &textLen)
	if status != -1 {
		t.Fatalf("expected TakeErrorTextForExport to fail, got status %d", status)
	}
	if textPtr != 99 || textLen != 99 {
		t.Fatalf("expected failed export to leave outputs unchanged, got ptr=%#x len=%d", textPtr, textLen)
	}
	if got := countPinnedEntriesForTesting(); got != beforePinned {
		t.Fatalf("expected failed export not to leak pinned entries: got %d want %d", got, beforePinned)
	}

	errorTextLengthToInt32ForExport = LengthToInt32
	status = TakeErrorTextForExport(int32(id), &textPtr, &textLen)
	if status != 0 {
		t.Fatalf("expected retry after restoring length conversion to succeed, got status %d", status)
	}
	if textPtr == 0 {
		t.Fatal("expected retry to populate text pointer")
	}
	if textLen != int32(len("stored failure")) {
		t.Fatalf("unexpected retry text length: got %d want %d", textLen, len("stored failure"))
	}
	if !Release(textPtr) {
		t.Fatal("expected retry pointer to be releasable")
	}
}

func countPinnedEntriesForTesting() int {
	count := 0
	pinnedMap.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func TestResetFreeCallbackForTestingClearsRegisteredState(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	called := false
	RegisterFreeCallback(func(ptr unsafe.Pointer) {
		called = true
	})
	if !freeFuncRegistered() {
		t.Fatal("expected registered free callback to be visible before reset")
	}

	ResetFreeCallbackForTesting()

	if freeFuncRegistered() {
		t.Fatal("expected reset to clear registered free callback state")
	}
	callRegisteredFree(unsafe.Pointer(new(byte)))
	if called {
		t.Fatal("expected reset callback state to suppress free callback invocation")
	}
}

func TestResetFreeCallbackForTestingAllowsReRegister(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	firstCalls := 0
	RegisterFreeCallback(func(ptr unsafe.Pointer) {
		firstCalls++
	})
	callRegisteredFree(unsafe.Pointer(new(byte)))
	if got, want := firstCalls, 1; got != want {
		t.Fatalf("first callback calls mismatch: got %d want %d", got, want)
	}

	ResetFreeCallbackForTesting()
	callRegisteredFree(unsafe.Pointer(new(byte)))
	if got, want := firstCalls, 1; got != want {
		t.Fatalf("expected reset to suppress first callback after clear: got %d want %d", got, want)
	}

	secondCalls := 0
	RegisterFreeCallback(func(ptr unsafe.Pointer) {
		secondCalls++
	})
	callRegisteredFree(unsafe.Pointer(new(byte)))
	if got, want := firstCalls, 1; got != want {
		t.Fatalf("expected second registration not to reuse first callback: got %d want %d", got, want)
	}
	if got, want := secondCalls, 1; got != want {
		t.Fatalf("second callback calls mismatch: got %d want %d", got, want)
	}
}

func TestRpcBytesCleanupReleasesOwnedPointerAfterGC(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var wantPtr uintptr

	func() {
		src := []byte("cleanup-bytes")
		wantPtr = uintptr(unsafe.Pointer(&src[0]))
		_ = NewRpcBytes(&src[0], int32(len(src)), true)
	}()

	waitForRpcInputCleanupOnce(t, recorder, wantPtr)
}

func TestRpcStringCleanupReleasesOwnedPointerAfterGC(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var wantPtr uintptr

	func() {
		src := []byte("cleanup-string")
		wantPtr = uintptr(unsafe.Pointer(&src[0]))
		_ = NewRpcString(&src[0], int32(len(src)), true)
	}()

	waitForRpcInputCleanupOnce(t, recorder, wantPtr)
}

func TestRpcBytesCleanupAfterSafeBytesStillFreesOriginalPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var wantPtr uintptr

	func() {
		src := []byte("cleanup-safe-bytes")
		wantPtr = uintptr(unsafe.Pointer(&src[0]))
		rpc := NewRpcBytes(&src[0], int32(len(src)), true)
		safe := rpc.SafeBytes()
		if len(safe) == 0 {
			t.Fatal("expected SafeBytes to produce copied content before GC")
		}
		if unsafe.SliceData(safe) == &src[0] {
			t.Fatal("expected SafeBytes copy to live at a different pointer than the original input")
		}
	}()

	waitForRpcInputCleanupOnce(t, recorder, wantPtr)
}

func TestRpcStringCleanupAfterSafeStringStillFreesOriginalPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var wantPtr uintptr

	func() {
		src := []byte("cleanup-safe-string")
		wantPtr = uintptr(unsafe.Pointer(&src[0]))
		rpc := NewRpcString(&src[0], int32(len(src)), true)
		safe := rpc.SafeString()
		if safe == "" {
			t.Fatal("expected SafeString to produce copied content before GC")
		}
		if unsafe.StringData(safe) == &src[0] {
			t.Fatal("expected SafeString copy to live at a different pointer than the original input")
		}
	}()

	waitForRpcInputCleanupOnce(t, recorder, wantPtr)
}

func TestRpcBytesReleasePreventsCleanupDoubleFree(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var rpc *RpcBytes
	var wantPtr uintptr

	func() {
		src := []byte("release-then-gc-bytes")
		wantPtr = uintptr(unsafe.Pointer(&src[0]))
		rpc = NewRpcBytes(&src[0], int32(len(src)), true)
	}()

	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected explicit release to free exactly once before GC, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected explicit release to free original pointer %#x, got %#x", wantPtr, got)
	}

	rpc = nil
	assertRpcInputCleanupCallCountStays(t, recorder, 1)
}

func TestRpcStringReleasePreventsCleanupDoubleFree(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var rpc *RpcString
	var wantPtr uintptr

	func() {
		src := []byte("release-then-gc-string")
		wantPtr = uintptr(unsafe.Pointer(&src[0]))
		rpc = NewRpcString(&src[0], int32(len(src)), true)
	}()

	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected explicit release to free exactly once before GC, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected explicit release to free original pointer %#x, got %#x", wantPtr, got)
	}

	rpc = nil
	assertRpcInputCleanupCallCountStays(t, recorder, 1)
}

func TestRpcBytesCleanupDoesNotFreeBorrowedOrEmptyInputs(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	t.Run("borrowed", func(t *testing.T) {
		recorder := registerFreeCallbackRecorder()
		func() {
			src := []byte("borrowed-bytes")
			_ = NewRpcBytes(&src[0], int32(len(src)), false)
		}()
		assertRpcInputCleanupCallCountStays(t, recorder, 0)
	})

	t.Run("nil", func(t *testing.T) {
		recorder := registerFreeCallbackRecorder()
		func() {
			_ = NewRpcBytes(nil, 0, true)
		}()
		assertRpcInputCleanupCallCountStays(t, recorder, 0)
	})

	t.Run("zero-length-owned", func(t *testing.T) {
		recorder := registerFreeCallbackRecorder()
		var wantPtr uintptr
		func() {
			src := []byte{0}
			wantPtr = uintptr(unsafe.Pointer(&src[0]))
			_ = NewRpcBytes(&src[0], 0, true)
		}()
		waitForRpcInputCleanupOnce(t, recorder, wantPtr)
	})
}

func TestRpcStringCleanupDoesNotFreeBorrowedOrEmptyInputs(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	t.Run("borrowed", func(t *testing.T) {
		recorder := registerFreeCallbackRecorder()
		func() {
			src := []byte("borrowed-string")
			_ = NewRpcString(&src[0], int32(len(src)), false)
		}()
		assertRpcInputCleanupCallCountStays(t, recorder, 0)
	})

	t.Run("nil", func(t *testing.T) {
		recorder := registerFreeCallbackRecorder()
		func() {
			_ = NewRpcString(nil, 0, true)
		}()
		assertRpcInputCleanupCallCountStays(t, recorder, 0)
	})

	t.Run("zero-length-owned", func(t *testing.T) {
		recorder := registerFreeCallbackRecorder()
		var wantPtr uintptr
		func() {
			src := []byte{0}
			wantPtr = uintptr(unsafe.Pointer(&src[0]))
			_ = NewRpcString(&src[0], 0, true)
		}()
		waitForRpcInputCleanupOnce(t, recorder, wantPtr)
	})
}

func TestRpcBytesCleanupFailureRetriesAfterRegisteringCallback(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	src := []byte("cleanup-before-register")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], int32(len(src)), true)

	rpcInputCleanup(rpcInputCleanupArg{
		ptr:   unsafe.Pointer(rpc.ptr),
		label: rpcBytesLabel,
	})
	if _, ok := rpcInputReleaseStates.Load(unsafe.Pointer(rpc.ptr)); !ok {
		t.Fatal("expected failed cleanup to keep release state for later retry")
	}

	recorder := registerFreeCallbackRecorder()
	waitForRpcInputRelease(t, recorder, wantPtr)
	if _, ok := rpcInputReleaseStates.Load(unsafe.Pointer(rpc.ptr)); ok {
		t.Fatal("expected retry after callback registration to clear release state")
	}
}

func TestRpcBytesCleanupFailureLateRegisterDoesNotMissRetry(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	reachedWindow := make(chan struct{})
	releaseWindow := make(chan struct{})
	rpcInputBeforeRetryPendingHookForTesting = func() {
		close(reachedWindow)
		<-releaseWindow
	}
	t.Cleanup(func() {
		rpcInputBeforeRetryPendingHookForTesting = nil
	})

	src := []byte("cleanup-late-register-bytes")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], int32(len(src)), true)

	done := make(chan struct{})
	go func() {
		defer close(done)
		rpcInputCleanup(rpcInputCleanupArg{
			ptr:   unsafe.Pointer(rpc.ptr),
			label: rpcBytesLabel,
		})
	}()

	<-reachedWindow
	recorder := registerFreeCallbackRecorder()
	close(releaseWindow)
	<-done

	waitForRpcInputRelease(t, recorder, wantPtr)
	if _, ok := rpcInputReleaseStates.Load(unsafe.Pointer(rpc.ptr)); ok {
		t.Fatal("expected late-register retry to clear bytes release state")
	}
}

func TestRpcStringCleanupFailureLateRegisterDoesNotMissRetry(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	reachedWindow := make(chan struct{})
	releaseWindow := make(chan struct{})
	rpcInputBeforeRetryPendingHookForTesting = func() {
		close(reachedWindow)
		<-releaseWindow
	}
	t.Cleanup(func() {
		rpcInputBeforeRetryPendingHookForTesting = nil
	})

	src := []byte("cleanup-late-register-string")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcString(&src[0], int32(len(src)), true)

	done := make(chan struct{})
	go func() {
		defer close(done)
		rpcInputCleanup(rpcInputCleanupArg{
			ptr:   unsafe.Pointer(rpc.ptr),
			label: rpcStringLabel,
		})
	}()

	<-reachedWindow
	recorder := registerFreeCallbackRecorder()
	close(releaseWindow)
	<-done

	waitForRpcInputRelease(t, recorder, wantPtr)
	if _, ok := rpcInputReleaseStates.Load(unsafe.Pointer(rpc.ptr)); ok {
		t.Fatal("expected late-register retry to clear string release state")
	}
}

func waitForRpcInputCleanupOnce(t *testing.T, recorder *freeCallbackRecorder, wantPtr uintptr) {
	t.Helper()

	deadline := time.Now().Add(rpcInputCleanupDeadline)
	seenCleanup := false
	for time.Now().Before(deadline) {
		runtime.GC()
		runtime.Gosched()
		if got := recorder.calls.Load(); got != 1 {
			if seenCleanup {
				t.Fatalf("expected cleanup call count to stay at 1 after first cleanup, got %d", got)
			}
		} else {
			seenCleanup = true
			if got := recorder.ptr.Load(); got != wantPtr {
				t.Fatalf("expected cleanup to free original pointer %#x, got %#x", wantPtr, got)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !seenCleanup {
		t.Fatalf("expected cleanup callback to run exactly once before deadline, got %d calls", recorder.calls.Load())
	}
}

func waitForRpcInputRelease(t *testing.T, recorder *freeCallbackRecorder, wantPtr uintptr) {
	t.Helper()

	deadline := time.Now().Add(rpcInputCleanupDeadline)
	for time.Now().Before(deadline) {
		if got := recorder.calls.Load(); got == 1 {
			if ptr := recorder.ptr.Load(); ptr != wantPtr {
				t.Fatalf("expected release to free original pointer %#x, got %#x", wantPtr, ptr)
			}
			return
		}
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected compensating release to run before deadline, got %d calls", recorder.calls.Load())
}

func assertRpcInputCleanupCallCountStays(t *testing.T, recorder *freeCallbackRecorder, wantCalls int32) {
	t.Helper()

	deadline := time.Now().Add(rpcInputCleanupDeadline)
	for time.Now().Before(deadline) {
		runtime.GC()
		runtime.Gosched()
		if got := recorder.calls.Load(); got != wantCalls {
			t.Fatalf("expected cleanup call count to stay at %d, got %d", wantCalls, got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
