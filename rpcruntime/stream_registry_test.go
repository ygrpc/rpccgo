package rpcruntime

import (
	"sync"
	"testing"
)

type testStreamSession struct {
	name string
}

type testStreamSessionInterface interface {
	sessionName() string
}

type testStreamSessionPointer struct {
	name string
}

func (s *testStreamSessionPointer) sessionName() string {
	return s.name
}

func TestStreamRegistryCreateLoadDeleteTake(t *testing.T) {
	var registry StreamRegistry

	handle, err := registry.Create(testStreamSession{name: "stream"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if handle == 0 {
		t.Fatal("Create returned zero handle")
	}

	loaded, ok := registry.Load(handle)
	if !ok {
		t.Fatal("Load returned false for created handle")
	}
	if loaded != (testStreamSession{name: "stream"}) {
		t.Fatalf("Load returned session %#v, want stream", loaded)
	}

	taken, ok := registry.Take(handle)
	if !ok {
		t.Fatal("Take returned false for created handle")
	}
	if taken != (testStreamSession{name: "stream"}) {
		t.Fatalf("Take returned session %#v, want stream", taken)
	}
	if _, ok := registry.Load(handle); ok {
		t.Fatal("Load returned true after Take")
	}
	if _, ok := registry.Take(handle); ok {
		t.Fatal("Take returned true after session was already taken")
	}

	second, err := registry.Create(testStreamSession{name: "delete"})
	if err != nil {
		t.Fatalf("Create returned error for second session: %v", err)
	}
	if !registry.Delete(second) {
		t.Fatal("Delete returned false for created handle")
	}
	if registry.Delete(second) {
		t.Fatal("Delete returned true for repeated delete")
	}
	if _, ok := registry.Load(second); ok {
		t.Fatal("Load returned true after Delete")
	}
}

func TestStreamRegistryRejectsZeroSession(t *testing.T) {
	var registry StreamRegistry

	if handle, err := registry.Create(testStreamSession{}); err == nil {
		t.Fatalf("Create returned nil error for zero struct session with handle %d", handle)
	}

	if handle, err := registry.Create(nil); err == nil {
		t.Fatalf("Create returned nil error for nil pointer session with handle %d", handle)
	}

	var typedNil *testStreamSessionPointer
	var interfaceSession testStreamSessionInterface = typedNil
	if handle, err := registry.Create(interfaceSession); err == nil {
		t.Fatalf("Create returned nil error for typed nil interface session with handle %d", handle)
	}
}

func TestStreamRegistryUnknownHandle(t *testing.T) {
	var registry StreamRegistry

	if _, ok := registry.Load(0); ok {
		t.Fatal("Load returned true for zero handle")
	}
	if registry.Delete(0) {
		t.Fatal("Delete returned true for zero handle")
	}
	if _, ok := registry.Take(0); ok {
		t.Fatal("Take returned true for zero handle")
	}

	const unknown StreamHandle = 99
	if _, ok := registry.Load(unknown); ok {
		t.Fatal("Load returned true for unknown handle")
	}
	if registry.Delete(unknown) {
		t.Fatal("Delete returned true for unknown handle")
	}
	if _, ok := registry.Take(unknown); ok {
		t.Fatal("Take returned true for unknown handle")
	}
}

func TestStreamHandleWrapSkipsZeroAndFindsOpenSlot(t *testing.T) {
	registry := StreamRegistry{
		next:     maxStreamHandle,
		sessions: map[StreamHandle]any{1: testStreamSession{name: "occupied"}},
	}

	handle, err := registry.Create(testStreamSession{name: "wrapped"})
	if err != nil {
		t.Fatalf("Create returned error for max handle: %v", err)
	}
	if handle != maxStreamHandle {
		t.Fatalf("Create returned handle %d, want %d", handle, maxStreamHandle)
	}

	wrapped, err := registry.Create(testStreamSession{name: "after wrap"})
	if err != nil {
		t.Fatalf("Create returned error after wrap: %v", err)
	}
	if wrapped == 0 {
		t.Fatal("Create returned zero handle after wrap")
	}
	if wrapped != 2 {
		t.Fatalf("Create returned handle %d after wrap, want 2", wrapped)
	}
}

func TestStreamHandleWrapReportsExhaustion(t *testing.T) {
	registry := StreamRegistry{
		next:     maxStreamHandle,
		sessions: map[StreamHandle]any{1: testStreamSession{name: "occupied"}},
	}
	registry.maxHandleForTesting = 1

	if handle, err := registry.Create(testStreamSession{name: "wrapped"}); err == nil {
		t.Fatalf("Create returned nil error after exhaustion with handle %d", handle)
	}
}

func TestStreamHandleInclusiveMaxIsAllocatable(t *testing.T) {
	registry := StreamRegistry{
		next: 1,
		sessions: map[StreamHandle]any{
			1: testStreamSession{name: "first"},
			2: testStreamSession{name: "second"},
		},
		maxHandleForTesting: 3,
	}

	handle, err := registry.Create(testStreamSession{name: "third"})
	if err != nil {
		t.Fatalf("Create returned error before handle space was exhausted: %v", err)
	}
	if handle != 3 {
		t.Fatalf("Create returned handle %d, want 3", handle)
	}
}

func TestStreamHandleInclusiveMaxReportsExhaustionOnlyWhenFull(t *testing.T) {
	registry := StreamRegistry{
		next: 1,
		sessions: map[StreamHandle]any{
			1: testStreamSession{name: "first"},
			2: testStreamSession{name: "second"},
			3: testStreamSession{name: "third"},
		},
		maxHandleForTesting: 3,
	}

	if handle, err := registry.Create(testStreamSession{name: "fourth"}); err == nil {
		t.Fatalf("Create returned nil error after handle space was exhausted with handle %d", handle)
	}
}

func TestStreamRegistryConcurrentCreateReturnsUniqueNonZeroInt32Handles(t *testing.T) {
	var registry StreamRegistry
	const workers = 8
	const perWorker = 128

	handles := make(chan StreamHandle, workers*perWorker)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				handle, err := registry.Create(testStreamSession{name: "stream"})
				if err != nil {
					t.Errorf("Create returned error: %v", err)
					return
				}
				handles <- handle
			}
		}()
	}
	wg.Wait()
	close(handles)

	seen := make(map[StreamHandle]struct{}, workers*perWorker)
	for handle := range handles {
		if handle <= 0 {
			t.Fatalf("Create returned non-positive handle %d", handle)
		}
		if _, ok := seen[handle]; ok {
			t.Fatalf("Create returned duplicate handle %d", handle)
		}
		seen[handle] = struct{}{}
	}
	if got, want := len(seen), workers*perWorker; got != want {
		t.Fatalf("Create returned %d handles, want %d", got, want)
	}
}
