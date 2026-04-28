package rpcruntime

import (
	"errors"
	"reflect"
	"sync"
)

const maxStreamHandle = StreamHandle(1<<31 - 1)

type StreamHandle int32

var (
	errStreamRegistryZeroSession = errors.New("stream registry requires non-zero session")
	errStreamRegistryExhausted   = errors.New("stream registry handle space exhausted")
)

type StreamRegistry[T any] struct {
	mu       sync.Mutex
	next     StreamHandle
	sessions map[StreamHandle]T

	maxHandleForTesting StreamHandle
}

func (r *StreamRegistry[T]) Create(session T) (StreamHandle, error) {
	if !hasNonZeroSession(session) {
		return 0, errStreamRegistryZeroSession
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	handle, err := r.allocateLocked()
	if err != nil {
		return 0, err
	}
	if r.sessions == nil {
		r.sessions = make(map[StreamHandle]T)
	}
	r.sessions[handle] = session
	return handle, nil
}

func (r *StreamRegistry[T]) Load(handle StreamHandle) (T, bool) {
	var zero T
	if handle == 0 {
		return zero, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[handle]
	if !ok {
		return zero, false
	}
	return session, true
}

func (r *StreamRegistry[T]) Delete(handle StreamHandle) bool {
	if handle == 0 {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sessions[handle]; !ok {
		return false
	}
	delete(r.sessions, handle)
	return true
}

func (r *StreamRegistry[T]) Take(handle StreamHandle) (T, bool) {
	var zero T
	if handle == 0 {
		return zero, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[handle]
	if !ok {
		return zero, false
	}
	delete(r.sessions, handle)
	return session, true
}

func (r *StreamRegistry[T]) allocateLocked() (StreamHandle, error) {
	limit := r.maxHandle()
	if limit <= 0 {
		return 0, errStreamRegistryExhausted
	}

	next := r.next
	if next <= 0 || next > limit {
		next = 1
	}

	for scanned := StreamHandle(0); scanned < limit; scanned++ {
		handle := next
		next++
		if next <= 0 || next > limit {
			next = 1
		}
		if handle == 0 {
			continue
		}
		if _, exists := r.sessions[handle]; exists {
			continue
		}
		r.next = next
		return handle, nil
	}
	return 0, errStreamRegistryExhausted
}

func (r *StreamRegistry[T]) maxHandle() StreamHandle {
	if r.maxHandleForTesting > 0 {
		return r.maxHandleForTesting
	}
	return maxStreamHandle
}

func hasNonZeroSession[T any](session T) bool {
	value := reflect.ValueOf(session)
	if !value.IsValid() {
		return false
	}
	return !value.IsZero()
}
