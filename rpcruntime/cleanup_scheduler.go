package rpcruntime

import (
	"sync"
	"sync/atomic"
	"time"
)

type cleanupCallback func()

type cleanupScheduleRequest struct {
	key   uint64
	delay time.Duration
	fn    cleanupCallback
	done  chan struct{}
}

type cleanupCancelRequest struct {
	key  uint64
	done chan struct{}
}

type scheduledCleanup struct {
	key    uint64
	slot   int
	rounds int
	fn     cleanupCallback
}

type cleanupScheduler struct {
	tick     time.Duration
	wheelLen int

	scheduleCh chan cleanupScheduleRequest
	cancelCh   chan cleanupCancelRequest
	stopCh     chan struct{}
	doneCh     chan struct{}

	stopOnce sync.Once
	pending  atomic.Int64
}

func newCleanupScheduler(tick time.Duration, wheelLen int) *cleanupScheduler {
	if tick <= 0 {
		tick = 100 * time.Millisecond
	}
	if wheelLen < 1 {
		wheelLen = 256
	}

	s := &cleanupScheduler{
		tick:       tick,
		wheelLen:   wheelLen,
		scheduleCh: make(chan cleanupScheduleRequest, 64),
		cancelCh:   make(chan cleanupCancelRequest, 64),
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
	go s.run()
	return s
}

func newCleanupSchedulerForTesting(tick time.Duration, wheelLen int) *cleanupScheduler {
	return newCleanupScheduler(tick, wheelLen)
}

func (s *cleanupScheduler) schedule(key uint64, delay time.Duration, fn cleanupCallback) {
	if s == nil || fn == nil {
		return
	}

	req := cleanupScheduleRequest{
		key:   key,
		delay: delay,
		fn:    fn,
		done:  make(chan struct{}),
	}
	select {
	case s.scheduleCh <- req:
		select {
		case <-req.done:
		case <-s.doneCh:
		}
	case <-s.doneCh:
	}
}

func (s *cleanupScheduler) cancel(key uint64) {
	if s == nil {
		return
	}
	req := cleanupCancelRequest{
		key:  key,
		done: make(chan struct{}),
	}
	select {
	case s.cancelCh <- req:
		select {
		case <-req.done:
		case <-s.doneCh:
		}
	case <-s.doneCh:
	}
}

func (s *cleanupScheduler) stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		<-s.doneCh
	})
}

func (s *cleanupScheduler) pendingCount() int {
	if s == nil {
		return 0
	}
	return int(s.pending.Load())
}

func (s *cleanupScheduler) run() {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()
	defer close(s.doneCh)

	currentSlot := 0
	buckets := make([]map[uint64]*scheduledCleanup, s.wheelLen)
	entries := make(map[uint64]*scheduledCleanup)

	remove := func(key uint64) {
		entry, ok := entries[key]
		if !ok {
			return
		}
		delete(entries, key)
		if bucket := buckets[entry.slot]; bucket != nil {
			delete(bucket, key)
		}
		s.pending.Add(-1)
	}

	for {
		select {
		case req := <-s.scheduleCh:
			remove(req.key)

			ticks := s.delayToTicks(req.delay)
			slot := (currentSlot + ticks) % s.wheelLen
			rounds := (ticks - 1) / s.wheelLen
			entry := &scheduledCleanup{
				key:    req.key,
				slot:   slot,
				rounds: rounds,
				fn:     req.fn,
			}
			if buckets[slot] == nil {
				buckets[slot] = make(map[uint64]*scheduledCleanup)
			}
			buckets[slot][req.key] = entry
			entries[req.key] = entry
			s.pending.Add(1)
			close(req.done)

		case req := <-s.cancelCh:
			remove(req.key)
			close(req.done)

		case <-ticker.C:
			currentSlot = (currentSlot + 1) % s.wheelLen
			bucket := buckets[currentSlot]
			if len(bucket) == 0 {
				continue
			}

			var callbacks []cleanupCallback
			for key, entry := range bucket {
				if entry.rounds > 0 {
					entry.rounds--
					continue
				}
				delete(bucket, key)
				delete(entries, key)
				s.pending.Add(-1)
				callbacks = append(callbacks, entry.fn)
			}
			for _, callback := range callbacks {
				runCleanupCallback(callback)
			}

		case <-s.stopCh:
			return
		}
	}
}

func (s *cleanupScheduler) delayToTicks(delay time.Duration) int {
	if delay <= 0 {
		return 1
	}
	ticks := int(delay / s.tick)
	if delay%s.tick != 0 {
		ticks++
	}
	if ticks < 1 {
		return 1
	}
	return ticks
}

func runCleanupCallback(fn cleanupCallback) {
	defer func() {
		_ = recover()
	}()
	fn()
}
