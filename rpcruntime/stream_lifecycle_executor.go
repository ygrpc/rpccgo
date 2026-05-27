package rpcruntime

type streamLifecycleExecutor interface {
	entryFor(handle StreamHandle) (*StreamEntry, error)
	delete(handle StreamHandle) bool
}

type dispatcherStreamExecutor[TAdapter any] struct {
	dispatcher *Dispatcher[TAdapter]
}

func (e dispatcherStreamExecutor[TAdapter]) entryFor(handle StreamHandle) (*StreamEntry, error) {
	if e.dispatcher == nil {
		return nil, ErrStreamInvalidHandle
	}

	entry, ok := e.dispatcher.streams.Load(handle)
	if !ok || entry == nil {
		return nil, ErrStreamInvalidHandle
	}
	return &StreamEntry{Session: entry.session, Lifecycle: &entry.lifecycle}, nil
}

func (e dispatcherStreamExecutor[TAdapter]) delete(handle StreamHandle) bool {
	if e.dispatcher == nil {
		return false
	}
	return e.dispatcher.streams.Delete(handle)
}

func streamLifecycleSessionFor[TSession any](e streamLifecycleExecutor, handle StreamHandle) (*StreamEntry, TSession, error) {
	var zero TSession
	entry, err := e.entryFor(handle)
	if err != nil {
		return nil, zero, err
	}
	typed, ok := any(entry.Session).(TSession)
	if !ok {
		return nil, zero, ErrStreamInvalidHandle
	}
	return entry, typed, nil
}

func streamLifecycleSend[TSession any](e streamLifecycleExecutor, handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](e, handle)
	if err != nil {
		return err
	}
	if err := entry.Lifecycle.EnsureCanSend(); err != nil {
		return err
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func streamLifecycleReceive[TSession any](e streamLifecycleExecutor, handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](e, handle)
	if err != nil {
		return err
	}
	if entry.Lifecycle.Finalized() {
		if entry.Lifecycle.Canceled() {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func streamLifecycleCloseSend[TSession any](e streamLifecycleExecutor, handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](e, handle)
	if err != nil {
		return err
	}
	if err := entry.Lifecycle.MarkSendClosed(); err != nil {
		return err
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func streamLifecycleFinish[TSession any](e streamLifecycleExecutor, handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](e, handle)
	if err != nil {
		return err
	}
	if !entry.Lifecycle.Finalize() {
		if entry.Lifecycle.Canceled() {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	if !e.delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func streamLifecycleCancel[TSession any](e streamLifecycleExecutor, handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](e, handle)
	if err != nil {
		return err
	}
	if err := entry.Lifecycle.Cancel(nil); err != nil {
		return err
	}
	if !e.delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}
