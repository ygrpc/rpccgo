package rpcruntime

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestStoreErrorNilReturnsZero(t *testing.T) {
	if got := StoreError(nil); got != 0 {
		t.Fatalf("expected zero error id, got %d", got)
	}
}

func TestTakeErrorTextConsumesRecord(t *testing.T) {
	id := StoreError(errors.New("boom"))
	if id == 0 {
		t.Fatal("expected non-zero error id")
	}

	data, ptr, ok := TakeErrorText(id)
	if !ok {
		t.Fatal("expected stored error to be found")
	}
	if string(data) != "boom" {
		t.Fatalf("unexpected error text: %q", string(data))
	}
	if ptr == 0 {
		t.Fatal("expected non-zero pointer")
	}

	if _, _, ok := TakeErrorText(id); ok {
		t.Fatal("expected error record to be consumed")
	}
}

func TestTakeErrorTextUnknownIDReturnsEmpty(t *testing.T) {
	data, ptr, ok := TakeErrorText(42)
	if ok {
		t.Fatal("expected unknown error id lookup to fail")
	}
	if len(data) != 0 || ptr != 0 {
		t.Fatal("expected zero-value error text result")
	}
}

func TestStoredErrorExpiresAndGetsRemoved(t *testing.T) {
	oldTTL := errorTTL
	errorTTL = 20 * time.Millisecond
	t.Cleanup(func() {
		errorTTL = oldTTL
	})

	id := StoreError(errors.New("stale"))
	if id == 0 {
		t.Fatal("expected non-zero error id")
	}

	time.Sleep(errorTTL + 40*time.Millisecond)

	if errorRecords.has(id) {
		t.Fatal("expected expired error to be removed from map")
	}

	if data, ptr, ok := TakeErrorText(id); ok || len(data) != 0 || ptr != 0 {
		t.Fatal("expected expired error lookup to return zero values")
	}
}

func TestErrorStoreBackgroundCleanupRemovesExpiredRecordWithoutAccess(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)
	resetErrorCleanupSchedulerForTesting(t, 5*time.Millisecond, 16)

	oldTTL := errorTTL
	errorTTL = 20 * time.Millisecond
	t.Cleanup(func() {
		errorTTL = oldTTL
	})

	id := StoreError(errors.New("stale-without-access"))
	if id == 0 {
		t.Fatal("expected non-zero error id")
	}

	waitForCondition(t, 500*time.Millisecond, func() bool {
		errorRecords.mu.RLock()
		defer errorRecords.mu.RUnlock()
		_, ok := errorRecords.records[id]
		return !ok
	}, "expected background cleanup to remove expired error without any subsequent access")
}

func TestErrorStoreCancelScheduledCleanupOnTake(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)
	resetErrorCleanupSchedulerForTesting(t, 5*time.Millisecond, 16)

	oldTTL := errorTTL
	errorTTL = 40 * time.Millisecond
	t.Cleanup(func() {
		errorTTL = oldTTL
	})

	id := StoreError(errors.New("take-cancels-cleanup"))
	if id == 0 {
		t.Fatal("expected non-zero error id")
	}
	if got := countScheduledErrorCleanupsForTesting(); got != 1 {
		t.Fatalf("expected exactly one scheduled cleanup, got %d", got)
	}

	_, ptr, ok := TakeErrorText(id)
	if !ok {
		t.Fatal("expected stored error to be found")
	}
	if !Release(ptr) {
		t.Fatal("expected pinned error text pointer to be releasable")
	}

	waitForCondition(t, 500*time.Millisecond, func() bool {
		return countScheduledErrorCleanupsForTesting() == 0
	}, "expected take to cancel scheduled cleanup")
}

func TestErrorStoreExpiredTakeDoesNotCancelWhileHoldingStoreLock(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)
	resetErrorCleanupSchedulerForTesting(t, time.Millisecond, 8)

	store := newErrorStore()
	id := ErrorID(1)
	store.store(id, errorRecord{
		text:      "expired",
		expiresAt: time.Now().Add(-time.Second),
	})

	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	errorCleanupScheduler.schedule(uint64(id), time.Millisecond, func() {
		close(callbackStarted)
		<-releaseCallback
		store.delete(id)
	})

	select {
	case <-callbackStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected cleanup callback to start")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		prepared, ok := store.takePrepared(id, func(errorRecord) (preparedErrorText, error) {
			t.Fatal("expired record should not be prepared")
			return preparedErrorText{}, nil
		})
		if ok || len(prepared.data) != 0 || prepared.ptr != 0 || prepared.length != 0 {
			t.Fatalf("expected expired take to return empty result, got %#v ok=%v", prepared, ok)
		}
	}()

	assertStaysBlocked(t, done, 20*time.Millisecond, "expired take should wait for in-flight cleanup before cancel completes")
	close(releaseCallback)
	assertCompletes(t, done, 500*time.Millisecond, "expired take should complete after cleanup callback releases store lock")
}

func TestErrorStoreExpiredHasDoesNotCancelWhileHoldingStoreLock(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)
	resetErrorCleanupSchedulerForTesting(t, time.Millisecond, 8)

	store := newErrorStore()
	id := ErrorID(1)
	store.store(id, errorRecord{
		text:      "expired",
		expiresAt: time.Now().Add(-time.Second),
	})

	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	errorCleanupScheduler.schedule(uint64(id), time.Millisecond, func() {
		close(callbackStarted)
		<-releaseCallback
		store.delete(id)
	})

	select {
	case <-callbackStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected cleanup callback to start")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if store.has(id) {
			t.Fatal("expected expired record lookup to fail")
		}
	}()

	assertStaysBlocked(t, done, 20*time.Millisecond, "expired has should wait for in-flight cleanup before cancel completes")
	close(releaseCallback)
	assertCompletes(t, done, 500*time.Millisecond, "expired has should complete after cleanup callback releases store lock")
}

func TestTakeErrorTextKeepsRecordWhenPinFails(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)

	id := StoreError(errors.New("keep-me"))
	pinErrorText = func(string) ([]byte, uintptr, error) {
		return nil, 0, fmt.Errorf("pin failed")
	}

	if data, ptr, ok := TakeErrorText(id); ok || len(data) != 0 || ptr != 0 {
		t.Fatal("expected failed pin to return zero values")
	}

	pinErrorText = PinString
	data, ptr, ok := TakeErrorText(id)
	if !ok {
		t.Fatal("expected error record to remain available after pin failure")
	}
	if got, want := string(data), "keep-me"; got != want {
		t.Fatalf("unexpected error text after retry: got %q want %q", got, want)
	}
	if ptr == 0 {
		t.Fatal("expected retry to return pinned pointer")
	}
	if !Release(ptr) {
		t.Fatal("expected retry pointer to be releasable")
	}
}

func TestTakeErrorTextForExportKeepsRecordAndReleasesPointerWhenLengthFails(t *testing.T) {
	resetErrorRuntimeStateForTesting(t)

	id := StoreError(errors.New("retry-export"))
	var pinnedPtr uintptr
	pinErrorText = func(text string) ([]byte, uintptr, error) {
		data, ptr, err := PinString(text)
		pinnedPtr = ptr
		return data, ptr, err
	}
	errorTextLengthToInt32ForExport = func(int) (int32, error) {
		return 0, fmt.Errorf("length conversion failed")
	}

	if prepared, ok := takeErrorTextForExport(id); ok || len(prepared.data) != 0 || prepared.ptr != 0 || prepared.length != 0 {
		t.Fatalf("expected failed length conversion to return empty result, got %#v ok=%v", prepared, ok)
	}
	if pinnedPtr == 0 {
		t.Fatal("expected failed attempt to pin error text before length conversion")
	}
	if Release(pinnedPtr) {
		t.Fatal("expected failed length conversion path to release pinned pointer")
	}

	pinErrorText = PinString
	errorTextLengthToInt32ForExport = LengthToInt32
	prepared, ok := takeErrorTextForExport(id)
	if !ok {
		t.Fatal("expected error record to remain available after length conversion failure")
	}
	if got, want := string(prepared.data), "retry-export"; got != want {
		t.Fatalf("unexpected error text after retry: got %q want %q", got, want)
	}
	if prepared.ptr == 0 {
		t.Fatal("expected retry to return pinned pointer")
	}
	if got, want := prepared.length, int32(len(prepared.data)); got != want {
		t.Fatalf("unexpected prepared length: got %d want %d", got, want)
	}
	if !Release(prepared.ptr) {
		t.Fatal("expected retry pointer to be releasable")
	}
	if prepared, ok := takeErrorTextForExport(id); ok || len(prepared.data) != 0 || prepared.ptr != 0 || prepared.length != 0 {
		t.Fatalf("expected successful retry to consume record, got %#v ok=%v", prepared, ok)
	}
}

func resetErrorRuntimeStateForTesting(t *testing.T) {
	t.Helper()

	originalStore := errorRecords
	originalScheduler := errorCleanupScheduler
	originalPin := pinErrorText
	originalLength := errorTextLengthToInt32ForExport

	errorRecords = newErrorStore()
	errorCleanupScheduler = newCleanupSchedulerForTesting(100*time.Millisecond, 256)
	pinErrorText = PinString
	errorTextLengthToInt32ForExport = LengthToInt32
	releaseAllPinnedForTesting()

	t.Cleanup(func() {
		errorRecords = originalStore
		errorCleanupScheduler.stop()
		errorCleanupScheduler = originalScheduler
		pinErrorText = originalPin
		errorTextLengthToInt32ForExport = originalLength
		releaseAllPinnedForTesting()
	})
}

func resetErrorCleanupSchedulerForTesting(t *testing.T, tick time.Duration, wheelLen int) {
	t.Helper()

	originalScheduler := errorCleanupScheduler
	errorCleanupScheduler = newCleanupSchedulerForTesting(tick, wheelLen)
	t.Cleanup(func() {
		errorCleanupScheduler.stop()
		errorCleanupScheduler = originalScheduler
	})
}

func countScheduledErrorCleanupsForTesting() int {
	return errorCleanupScheduler.pendingCount()
}

func releaseAllPinnedForTesting() {
	pinnedMap.Range(func(key, value any) bool {
		Release(key.(uintptr))
		return true
	})
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(message)
}

func assertStaysBlocked(t *testing.T, done <-chan struct{}, duration time.Duration, message string) {
	t.Helper()

	select {
	case <-done:
		t.Fatal(message)
	case <-time.After(duration):
	}
}

func assertCompletes(t *testing.T, done <-chan struct{}, timeout time.Duration, message string) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal(message)
	}
}
