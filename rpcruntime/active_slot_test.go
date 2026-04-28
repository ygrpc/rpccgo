package rpcruntime

import (
	"strings"
	"sync"
	"testing"
)

func TestActiveServerSlotLoadBeforeStore(t *testing.T) {
	var slot ActiveServerSlot[*fakeDispatcherAdapter]

	if snapshot, ok := slot.Load(); ok {
		t.Fatalf("expected empty slot, got snapshot %#v", snapshot)
	}
}

func TestActiveServerSlotStoreAndLoad(t *testing.T) {
	var slot ActiveServerSlot[*fakeDispatcherAdapter]
	firstAdapter := &fakeDispatcherAdapter{name: "first"}
	secondAdapter := &fakeDispatcherAdapter{name: "second"}

	first, err := slot.Store(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("store first adapter: %v", err)
	}
	if first.Version != 1 {
		t.Fatalf("unexpected first version: got %d, want 1", first.Version)
	}

	loadedFirst, ok := slot.Load()
	if !ok {
		t.Fatal("expected loaded first snapshot")
	}
	if loadedFirst.Adapter != firstAdapter {
		t.Fatalf("unexpected loaded first adapter: got %#v, want %#v", loadedFirst.Adapter, firstAdapter)
	}
	if loadedFirst.Kind != ServerKindGoNative || loadedFirst.Contract != ServerContractNative || loadedFirst.Version != first.Version {
		t.Fatalf("loaded first snapshot mismatch: %#v", loadedFirst)
	}

	second, err := slot.Store(ServerKindConnectHandler, ServerContractMessage, secondAdapter)
	if err != nil {
		t.Fatalf("store second adapter: %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("unexpected second version: got %d, want 2", second.Version)
	}

	loadedSecond, ok := slot.Load()
	if !ok {
		t.Fatal("expected loaded second snapshot")
	}
	if loadedSecond.Adapter != secondAdapter {
		t.Fatalf("unexpected loaded second adapter: got %#v, want %#v", loadedSecond.Adapter, secondAdapter)
	}
	if loadedSecond.Version != second.Version {
		t.Fatalf("unexpected loaded second version: got %d, want %d", loadedSecond.Version, second.Version)
	}
	if loadedFirst.Adapter != firstAdapter || loadedFirst.Version != 1 {
		t.Fatalf("old snapshot was not stable after later store: %#v", loadedFirst)
	}
}

func TestActiveServerSlotStoreRejectsMissingKindContractOrAdapter(t *testing.T) {
	tests := []struct {
		name     string
		kind     ServerKind
		contract ServerContract
		adapter  *fakeDispatcherAdapter
		wantErr  string
	}{
		{
			name:     "zero kind",
			contract: ServerContractNative,
			adapter:  &fakeDispatcherAdapter{name: "adapter"},
			wantErr:  "server kind",
		},
		{
			name:    "zero contract",
			kind:    ServerKindGoNative,
			adapter: &fakeDispatcherAdapter{name: "adapter"},
			wantErr: "server contract",
		},
		{
			name:     "nil adapter",
			kind:     ServerKindGoNative,
			contract: ServerContractNative,
			adapter:  nil,
			wantErr:  "adapter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var slot ActiveServerSlot[*fakeDispatcherAdapter]

			if snapshot, err := slot.Store(tt.kind, tt.contract, tt.adapter); err == nil {
				t.Fatalf("expected error, got snapshot %#v", snapshot)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error %q, want it to mention %q", err.Error(), tt.wantErr)
			}

			if snapshot, ok := slot.Load(); ok {
				t.Fatalf("invalid store changed slot: %#v", snapshot)
			}
		})
	}
}

func TestActiveServerSlotStoreRejectsTypedNilInterfaceAdapter(t *testing.T) {
	var impl *fakeDispatcherAdapterWithMethod
	var adapter fakeDispatcherInterface = impl
	var slot ActiveServerSlot[fakeDispatcherInterface]

	if snapshot, err := slot.Store(ServerKindGoNative, ServerContractNative, adapter); err == nil {
		t.Fatalf("expected typed nil adapter error, got snapshot %#v", snapshot)
	} else if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("unexpected error %q, want it to mention adapter", err.Error())
	}

	if snapshot, ok := slot.Load(); ok {
		t.Fatalf("typed nil store changed slot: %#v", snapshot)
	}
}

func TestActiveServerSlotStoreRejectsZeroStructAdapter(t *testing.T) {
	var slot ActiveServerSlot[fakeValueAdapter]

	if snapshot, err := slot.Store(ServerKindGoNative, ServerContractNative, fakeValueAdapter{}); err == nil {
		t.Fatalf("expected zero adapter error, got snapshot %#v", snapshot)
	} else if !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("unexpected error %q, want it to mention adapter", err.Error())
	}
}

func TestActiveServerSlotConcurrentStoreLoad(t *testing.T) {
	var slot ActiveServerSlot[*fakeDispatcherAdapter]
	const workers = 16
	const iterations = 128

	var (
		mu         sync.Mutex
		successes  = make(map[int64]*fakeDispatcherAdapter)
		storeGroup sync.WaitGroup
		loadGroup  sync.WaitGroup
	)

	for worker := 0; worker < workers; worker++ {
		storeGroup.Add(1)
		go func() {
			defer storeGroup.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				adapter := &fakeDispatcherAdapter{name: "adapter"}
				snapshot, err := slot.Store(ServerKindGoNative, ServerContractNative, adapter)
				if err != nil {
					t.Errorf("store failed: %v", err)
					return
				}
				if snapshot.Version == 0 || !snapshot.HasAdapter() {
					t.Errorf("store returned zero snapshot: %#v", snapshot)
					return
				}

				mu.Lock()
				successes[snapshot.Version] = adapter
				mu.Unlock()
			}
		}()
	}

	for worker := 0; worker < workers; worker++ {
		loadGroup.Add(1)
		go func() {
			defer loadGroup.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				snapshot, ok := slot.Load()
				if !ok {
					continue
				}
				if snapshot.Version == 0 || !snapshot.HasAdapter() {
					t.Errorf("load returned zero snapshot: %#v", snapshot)
					return
				}
			}
		}()
	}

	storeGroup.Wait()
	loadGroup.Wait()

	final, ok := slot.Load()
	if !ok {
		t.Fatal("expected final snapshot after stores")
	}
	if final.Version == 0 || !final.HasAdapter() {
		t.Fatalf("final snapshot is zero: %#v", final)
	}

	mu.Lock()
	registeredAdapter := successes[final.Version]
	mu.Unlock()
	if registeredAdapter == nil {
		t.Fatalf("final snapshot version %d was not recorded as a successful store", final.Version)
	}
	if final.Adapter != registeredAdapter {
		t.Fatalf("final snapshot adapter mismatch: got %#v, want %#v", final.Adapter, registeredAdapter)
	}
}

type fakeValueAdapter struct {
	name string
}
