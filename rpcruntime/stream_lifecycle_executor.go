package rpcruntime

func streamLifecycleEntryFor(registry *StreamRegistry[*StreamEntry], handle StreamHandle) (*StreamEntry, error) {
	if registry == nil {
		return nil, ErrStreamInvalidHandle
	}
	entry, ok := registry.Load(handle)
	if !ok || entry == nil || entry.Lifecycle == nil {
		return nil, ErrStreamInvalidHandle
	}
	return entry, nil
}

func streamLifecycleSessionFor[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle) (*StreamEntry, TSession, error) {
	var zero TSession
	entry, err := streamLifecycleEntryFor(registry, handle)
	if err != nil {
		return nil, zero, err
	}
	typed, ok := any(entry.Session).(TSession)
	if !ok {
		return nil, zero, ErrStreamSessionTypeMismatch
	}
	return entry, typed, nil
}

func streamLifecycleSend[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](registry, handle)
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

func streamLifecycleReceive[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](registry, handle)
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

func streamLifecycleCloseSend[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](registry, handle)
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

func streamLifecycleFinish[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](registry, handle)
	if err != nil {
		return err
	}
	if !entry.Lifecycle.Finalize() {
		if entry.Lifecycle.Canceled() {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	if !registry.Delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}

func streamLifecycleCancel[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	entry, session, err := streamLifecycleSessionFor[TSession](registry, handle)
	if err != nil {
		return err
	}
	if err := entry.Lifecycle.Cancel(nil); err != nil {
		return err
	}
	if !registry.Delete(handle) {
		return ErrStreamInvalidHandle
	}
	if call == nil {
		return nil
	}
	return call(session)
}
