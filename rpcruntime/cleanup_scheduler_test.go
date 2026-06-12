package rpcruntime

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestCleanupSchedulerRunsScheduledTaskAfterTickWindow(t *testing.T) {
	scheduler := newCleanupSchedulerForTesting(5*time.Millisecond, 16)
	t.Cleanup(scheduler.stop)

	var fired atomic.Int32
	scheduler.schedule(1, 20*time.Millisecond, func() {
		fired.Add(1)
	})

	waitForCondition(t, 500*time.Millisecond, func() bool {
		return fired.Load() == 1
	}, "expected scheduled cleanup task to run exactly once")
}

func TestCleanupSchedulerAcceptsSignedKeys(t *testing.T) {
	scheduler := newCleanupSchedulerForTesting(5*time.Millisecond, 16)
	t.Cleanup(scheduler.stop)

	var fired atomic.Int32
	scheduler.schedule(-1, 5*time.Millisecond, func() {
		fired.Add(1)
	})

	waitForCondition(t, 500*time.Millisecond, func() bool {
		return fired.Load() == 1
	}, "expected scheduler to accept signed cleanup keys")
}

func TestCleanupSchedulerCancelPreventsExecution(t *testing.T) {
	scheduler := newCleanupSchedulerForTesting(5*time.Millisecond, 16)
	t.Cleanup(scheduler.stop)

	var fired atomic.Int32
	scheduler.schedule(1, 20*time.Millisecond, func() {
		fired.Add(1)
	})
	scheduler.cancel(1)

	time.Sleep(80 * time.Millisecond)
	if got := fired.Load(); got != 0 {
		t.Fatalf("expected canceled cleanup task not to run, got %d executions", got)
	}
}

func TestCleanupSchedulerScheduleReturnsWhenDoneClosesAfterBufferedSend(t *testing.T) {
	scheduler := &cleanupScheduler{
		scheduleCh: make(chan cleanupScheduleRequest, 1),
		doneCh:     make(chan struct{}),
	}

	returned := make(chan struct{})
	go func() {
		scheduler.schedule(1, time.Second, func() {})
		close(returned)
	}()

	waitForCondition(t, 200*time.Millisecond, func() bool {
		return len(scheduler.scheduleCh) == 1
	}, "expected buffered schedule request to be enqueued")

	close(scheduler.doneCh)

	waitForCondition(t, 200*time.Millisecond, func() bool {
		select {
		case <-returned:
			return true
		default:
			return false
		}
	}, "expected schedule to unblock when scheduler is stopped after send")
}

func TestCleanupSchedulerCancelReturnsWhenDoneClosesAfterBufferedSend(t *testing.T) {
	scheduler := &cleanupScheduler{
		cancelCh: make(chan cleanupCancelRequest, 1),
		doneCh:   make(chan struct{}),
	}

	returned := make(chan struct{})
	go func() {
		scheduler.cancel(1)
		close(returned)
	}()

	waitForCondition(t, 200*time.Millisecond, func() bool {
		return len(scheduler.cancelCh) == 1
	}, "expected buffered cancel request to be enqueued")

	close(scheduler.doneCh)

	waitForCondition(t, 200*time.Millisecond, func() bool {
		select {
		case <-returned:
			return true
		default:
			return false
		}
	}, "expected cancel to unblock when scheduler is stopped after send")
}
