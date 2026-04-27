package rpcruntime

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

type NativeArrayElem interface {
	~bool |
		~int8 | ~uint8 |
		~int16 | ~uint16 |
		~int32 | ~uint32 |
		~int64 | ~uint64 |
		~float32 | ~float64
}

type releaseEntry struct {
	value  any
	pinner runtime.Pinner
}

// pinnedMap tracks active pin/unpin lifetimes by exported pointer value.
// Entries are short-lived and independently removed, which matches sync.Map well.
var pinnedMap sync.Map

func PinBytes(b []byte) (uintptr, error) {
	if len(b) == 0 {
		return 0, nil
	}

	ptr := uintptr(unsafe.Pointer(&b[0]))
	return registerPinned(ptr, b, unsafe.Pointer(&b[0]))
}

func PinString(s string) ([]byte, uintptr, error) {
	if len(s) == 0 {
		return nil, 0, nil
	}

	// Strings don't provide stable pinnable backing storage for our release registry,
	// so we materialize a byte slice first and pin that copy intentionally.
	b := []byte(s)
	ptr, err := PinBytes(b)
	if err != nil {
		return b, ptr, err
	}
	return b, ptr, nil
}

func PinSlice[T NativeArrayElem](s []T) (uintptr, error) {
	if len(s) == 0 {
		return 0, nil
	}

	ptr := uintptr(unsafe.Pointer(&s[0]))
	return registerPinned(ptr, s, unsafe.Pointer(&s[0]))
}

func Release(ptr uintptr) bool {
	if ptr == 0 {
		return false
	}

	raw, ok := pinnedMap.LoadAndDelete(ptr)
	if !ok {
		return false
	}

	entry := raw.(*releaseEntry)
	entry.pinner.Unpin()
	return true
}

// registerPinned treats the exported pointer value as a unique runtime handle.
// Re-exporting the same backing store returns the existing pointer plus an error;
// the runtime does not add reference counting or copy the backing store for you.
func registerPinned(ptr uintptr, value any, target unsafe.Pointer) (uintptr, error) {
	entry := &releaseEntry{value: value}
	entry.pinner.Pin(target)

	_, loaded := pinnedMap.LoadOrStore(ptr, entry)
	if loaded {
		entry.pinner.Unpin()
		return ptr, fmt.Errorf("pointer %x already pinned", ptr)
	}

	return ptr, nil
}
