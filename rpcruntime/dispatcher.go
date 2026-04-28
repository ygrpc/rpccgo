package rpcruntime

import (
	"context"
	"errors"
)

var (
	errDispatcherNoActiveServer  = errors.New("dispatcher has no active server")
	errDispatcherNilInvoke       = errors.New("dispatcher invoke callback is nil")
	errDispatcherNilStreamCreate = errors.New("dispatcher stream create callback is nil")
)

type Dispatcher[T any] struct {
	slot    ActiveServerSlot[T]
	streams StreamRegistry[any]
}

func (d *Dispatcher[T]) Register(kind ServerKind, contract ServerContract, adapter T) (AdapterSnapshot[T], error) {
	return d.slot.Store(kind, contract, adapter)
}

func (d *Dispatcher[T]) Capture() (AdapterSnapshot[T], error) {
	snapshot, ok := d.slot.Load()
	if !ok {
		return AdapterSnapshot[T]{}, errDispatcherNoActiveServer
	}
	return snapshot, nil
}

func (d *Dispatcher[T]) Invoke(ctx context.Context, invoke func(context.Context, AdapterSnapshot[T]) error) error {
	if invoke == nil {
		return errDispatcherNilInvoke
	}

	snapshot, err := d.Capture()
	if err != nil {
		return err
	}
	return invoke(ctx, snapshot)
}

func (d *Dispatcher[T]) StartStream(create func(AdapterSnapshot[T]) (session any, err error)) (StreamHandle, error) {
	if create == nil {
		return 0, errDispatcherNilStreamCreate
	}

	snapshot, err := d.Capture()
	if err != nil {
		return 0, err
	}

	session, err := create(snapshot)
	if err != nil {
		return 0, err
	}
	return d.streams.Create(session)
}

func LoadDispatcherStream[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle) (TSession, bool) {
	var zero TSession
	if dispatcher == nil {
		return zero, false
	}

	session, ok := dispatcher.streams.Load(handle)
	if !ok {
		return zero, false
	}
	typed, ok := session.(TSession)
	if !ok {
		return zero, false
	}
	return typed, true
}

func TakeDispatcherStream[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle) (TSession, bool) {
	var zero TSession
	if dispatcher == nil {
		return zero, false
	}

	session, ok := dispatcher.streams.Load(handle)
	if !ok {
		return zero, false
	}
	if _, ok := session.(TSession); !ok {
		return zero, false
	}

	taken, ok := dispatcher.streams.Take(handle)
	if !ok {
		return zero, false
	}
	typed, ok := taken.(TSession)
	if !ok {
		return zero, false
	}
	return typed, true
}

func DeleteDispatcherStream[TAdapter any](dispatcher *Dispatcher[TAdapter], handle StreamHandle) bool {
	if dispatcher == nil {
		return false
	}
	return dispatcher.streams.Delete(handle)
}
