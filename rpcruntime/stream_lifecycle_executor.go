package rpcruntime

type streamLifecycleExecutor[TAdapter any] struct {
	dispatcher *Dispatcher[TAdapter]
}

func (e streamLifecycleExecutor[TAdapter]) entryFor(handle StreamHandle) (*dispatcherStreamEntry, error) {
	if e.dispatcher == nil {
		return nil, ErrStreamInvalidHandle
	}

	entry, ok := e.dispatcher.streams.Load(handle)
	if !ok || entry == nil {
		return nil, ErrStreamInvalidHandle
	}
	return entry, nil
}

func streamLifecycleSessionFor[TAdapter any, TSession any](e streamLifecycleExecutor[TAdapter], handle StreamHandle) (*dispatcherStreamEntry, TSession, error) {
	var zero TSession
	entry, err := e.entryFor(handle)
	if err != nil {
		return nil, zero, err
	}
	typed, ok := entry.session.(TSession)
	if !ok {
		return nil, zero, ErrStreamInvalidHandle
	}
	return entry, typed, nil
}

func streamLifecycleSend[TAdapter any, TSession any](e streamLifecycleExecutor[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TAdapter, TSession](e, handle)
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

func streamLifecycleReceive[TAdapter any, TSession any](e streamLifecycleExecutor[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TAdapter, TSession](e, handle)
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

func streamLifecycleCloseSend[TAdapter any, TSession any](e streamLifecycleExecutor[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TAdapter, TSession](e, handle)
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

func streamLifecycleFinish[TAdapter any, TSession any](e streamLifecycleExecutor[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TAdapter, TSession](e, handle)
	if err != nil {
		return err
	}
	if !entry.lifecycle.Finalize() {
		if entry.lifecycle.Canceled() {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	if !e.dispatcher.streams.Delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func streamLifecycleCancel[TAdapter any, TSession any](e streamLifecycleExecutor[TAdapter], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TAdapter, TSession](e, handle)
	if err != nil {
		return err
	}
	if err := entry.lifecycle.Cancel(nil); err != nil {
		return err
	}
	if !e.dispatcher.streams.Delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}
