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

func dispatcherStreamEntryFor[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle) (*dispatcherStreamEntry, TSession, error) {
	var zero TSession
	if dispatcher == nil {
		return nil, zero, ErrStreamInvalidHandle
	}

	entry, ok := dispatcher.streams.Load(handle)
	if !ok || entry == nil {
		return nil, zero, ErrStreamInvalidHandle
	}
	typed, ok := entry.session.(TSession)
	if !ok {
		return nil, zero, ErrStreamInvalidHandle
	}
	return entry, typed, nil
}

func DispatcherStreamSend[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := dispatcherStreamEntryFor[TAdapter, TSession](dispatcher, handle)
	if err != nil {
		return err
	}
	if err := entry.lifecycle.EnsureCanSend(); err != nil {
		return err
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func DispatcherStreamReceive[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := dispatcherStreamEntryFor[TAdapter, TSession](dispatcher, handle)
	if err != nil {
		return err
	}
	if entry.lifecycle.Finalized() {
		if entry.lifecycle.Canceled() {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func DispatcherStreamCloseSend[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := dispatcherStreamEntryFor[TAdapter, TSession](dispatcher, handle)
	if err != nil {
		return err
	}
	if err := entry.lifecycle.MarkSendClosed(); err != nil {
		return err
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func DispatcherStreamFinish[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := dispatcherStreamEntryFor[TAdapter, TSession](dispatcher, handle)
	if err != nil {
		return err
	}
	if !entry.lifecycle.Finalize() {
		if entry.lifecycle.Canceled() {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	if !dispatcher.streams.Delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func DispatcherStreamDone[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	return DispatcherStreamFinish(dispatcher, handle, call)
}

func DispatcherStreamCancel[TAdapter any, TSession any](dispatcher *Dispatcher[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := dispatcherStreamEntryFor[TAdapter, TSession](dispatcher, handle)
	if err != nil {
		return err
	}
	if err := entry.lifecycle.Cancel(nil); err != nil {
		return err
	}
	if !dispatcher.streams.Delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}
