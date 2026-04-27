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
