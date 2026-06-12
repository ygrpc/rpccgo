package rpcruntime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	registeredFreeCallback func(unsafe.Pointer)
	registeredFreeMu       sync.RWMutex
	// rpcInputReleaseStates is keyed by raw C pointer and supports concurrent
	// cleanup/release/retry flows plus whole-map Range scans during retry/reset.
	rpcInputReleaseStates sync.Map

	rpcInputBeforeRetryPendingHookForTesting func()
)

type rpcInputReleaseState struct {
	ptr          unsafe.Pointer
	label        string
	mu           sync.Mutex
	released     atomic.Bool
	retryPending atomic.Bool
}

// callRegisteredFree calls the previously registered C free callback.
// It is a no-op if ptr is nil or no callback has been registered.
func callRegisteredFree(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	if fn := loadRegisteredFreeCallback(); fn != nil {
		fn(ptr)
	}
}

// RegisterFreeCallback stores a Go closure that wraps the C free function.
// Typically called once during initialization from the cgo export layer.
func RegisterFreeCallback(fn func(unsafe.Pointer)) {
	registeredFreeMu.Lock()
	registeredFreeCallback = fn
	registeredFreeMu.Unlock()

	retryPendingRpcInputs()
}

// ResetFreeCallbackForTesting clears the registered C free callback.
// It is intended for tests that need to simulate a fresh export/runtime state.
func ResetFreeCallbackForTesting() {
	registeredFreeMu.Lock()
	registeredFreeCallback = nil
	registeredFreeMu.Unlock()
	clearRpcInputReleaseStates()
}

// freeFuncRegistered reports whether a C free callback has been registered.
func freeFuncRegistered() bool {
	return loadRegisteredFreeCallback() != nil
}

// ReleaseC frees C-owned memory through the registered free callback.
// If owned is false or ptr is nil, it is a no-op.
func ReleaseC(ptr unsafe.Pointer, owned bool, label string) error {
	if !owned || ptr == nil {
		return nil
	}
	fn := loadRegisteredFreeCallback()
	if fn == nil {
		return fmt.Errorf("%s ownership requires registered free func", label)
	}
	fn(ptr)
	return nil
}

func registerRpcInputRelease(ptr unsafe.Pointer, owned bool, label string) {
	if !owned || ptr == nil {
		return
	}
	rpcInputReleaseStates.LoadOrStore(ptr, &rpcInputReleaseState{
		ptr:   ptr,
		label: label,
	})
}

// releaseRpcInput routes cleanup and explicit release through the same
// once-only state machine.
func releaseRpcInput(ptr unsafe.Pointer, owned bool, label string) error {
	if !owned || ptr == nil {
		return nil
	}

	raw, ok := rpcInputReleaseStates.Load(ptr)
	if !ok {
		return nil
	}

	state := raw.(*rpcInputReleaseState)
	if state.released.Load() {
		return nil
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.released.Load() {
		return nil
	}
	if err := ReleaseC(ptr, true, label); err != nil {
		if rpcInputBeforeRetryPendingHookForTesting != nil {
			rpcInputBeforeRetryPendingHookForTesting()
		}
		state.retryPending.Store(true)
		if loadRegisteredFreeCallback() != nil {
			if retryErr := ReleaseC(ptr, true, label); retryErr == nil {
				state.retryPending.Store(false)
				state.released.Store(true)
				rpcInputReleaseStates.Delete(ptr)
				return nil
			}
		}
		return err
	}

	state.retryPending.Store(false)
	state.released.Store(true)
	rpcInputReleaseStates.Delete(ptr)
	return nil
}

func loadRegisteredFreeCallback() func(unsafe.Pointer) {
	registeredFreeMu.RLock()
	defer registeredFreeMu.RUnlock()
	return registeredFreeCallback
}

func retryPendingRpcInputs() {
	rpcInputReleaseStates.Range(func(key, value any) bool {
		state := value.(*rpcInputReleaseState)
		if !state.retryPending.Load() {
			return true
		}
		_ = releaseRpcInput(state.ptr, true, state.label)
		return true
	})
}

func clearRpcInputReleaseStates() {
	rpcInputReleaseStates.Range(func(key, value any) bool {
		rpcInputReleaseStates.Delete(key)
		return true
	})
}

// TakeErrorTextForExport wraps TakeErrorText with output-pointer semantics
// suitable for the cgo export ABI. Returns 0 on success, -1 on failure.
func TakeErrorTextForExport(errID int32, textPtr *uintptr, textLen *int32) int32 {
	prepared, ok := takeErrorTextForExport(ErrorID(errID))
	if !ok {
		return -1
	}
	if textPtr != nil {
		*textPtr = prepared.ptr
	}
	if textLen != nil {
		*textLen = prepared.length
	}
	_ = prepared.data
	return 0
}
