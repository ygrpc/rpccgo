package rpcruntime

import (
	"context"
	"errors"
)

var (
	ErrNoActiveServer            = errors.New("dispatcher has no active server")
	errDispatcherNilInvoke       = errors.New("dispatcher invoke callback is nil")
	errDispatcherNilStreamCreate = errors.New("dispatcher stream create callback is nil")
)

type Dispatcher[T any] struct {
	slot    ActiveServerSlot[T]
	streams StreamRegistry[*dispatcherStreamEntry]
}

type dispatcherStreamEntry struct {
	session   any
	lifecycle StreamLifecycle
}

func (d *Dispatcher[T]) Register(kind ServerKind, contract ServerContract, adapter T) (AdapterSnapshot[T], error) {
	return d.slot.Store(kind, contract, adapter)
}

func (d *Dispatcher[T]) Capture() (AdapterSnapshot[T], error) {
	snapshot, ok := d.slot.Load()
	if !ok {
		return AdapterSnapshot[T]{}, ErrNoActiveServer
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
	if !hasNonZeroSession(session) {
		return 0, errStreamRegistryZeroSession
	}
	return d.streams.Create(&dispatcherStreamEntry{session: session})
}

func DispatcherStreamSend[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleSend(streamLifecycleExecutor[TAdapter]{dispatcher: dispatcher}, handle, call)
}

func DispatcherStreamReceive[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleReceive(streamLifecycleExecutor[TAdapter]{dispatcher: dispatcher}, handle, call)
}

func DispatcherStreamCloseSend[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleCloseSend(streamLifecycleExecutor[TAdapter]{dispatcher: dispatcher}, handle, call)
}

func DispatcherStreamFinish[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleFinish(streamLifecycleExecutor[TAdapter]{dispatcher: dispatcher}, handle, call)
}

func DispatcherStreamDone[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return DispatcherStreamFinish(dispatcher, handle, call)
}

func DispatcherStreamCancel[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleCancel(streamLifecycleExecutor[TAdapter]{dispatcher: dispatcher}, handle, call)
}
