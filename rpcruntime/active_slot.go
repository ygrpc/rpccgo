package rpcruntime

import (
	"errors"
	"sync"
)

var (
	ErrNoActiveServer                  = errors.New("dispatcher has no active server")
	errActiveServerSlotMissingKind     = errors.New("active server slot requires server kind")
	errActiveServerSlotMissingContract = errors.New("active server slot requires server contract")
	errActiveServerSlotMissingAdapter  = errors.New("active server slot requires adapter")
)

type ActiveServerSlot[T any] struct {
	mu sync.Mutex
	// version 从 1 开始递增，0 表示未初始化,用于区分当前 slot 所持有的 adapter 是否有效，以及是否发生过更新。
	version  int64
	snapshot AdapterSnapshot[T]
}

func (s *ActiveServerSlot[T]) Store(kind ServerKind, contract ServerContract, adapter T) (AdapterSnapshot[T], error) {
	if kind == "" {
		return AdapterSnapshot[T]{}, errActiveServerSlotMissingKind
	}
	if contract == "" {
		return AdapterSnapshot[T]{}, errActiveServerSlotMissingContract
	}

	next := AdapterSnapshot[T]{
		Kind:     kind,
		Contract: contract,
		Adapter:  adapter,
	}
	if !next.HasAdapter() {
		return AdapterSnapshot[T]{}, errActiveServerSlotMissingAdapter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.version++
	next.Version = s.version
	s.snapshot = next
	return next, nil
}

func (s *ActiveServerSlot[T]) Load() (AdapterSnapshot[T], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.snapshot.Version == 0 || s.snapshot.Kind == "" || s.snapshot.Contract == "" || !s.snapshot.HasAdapter() {
		return AdapterSnapshot[T]{}, false
	}
	return s.snapshot, true
}
