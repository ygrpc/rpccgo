package rpcruntime

// StreamEntry stores the typed session plus its lifecycle state for registry-backed
// generated stream helpers. Generated service runtime owns the registry and keeps
// method-specific session data in Session.
type StreamEntry struct {
	Session   any
	Lifecycle *StreamLifecycle
}

// NewStreamEntry creates a stream entry with fresh lifecycle state.
func NewStreamEntry(session any) *StreamEntry {
	return &StreamEntry{Session: session, Lifecycle: &StreamLifecycle{}}
}

type streamRegistryExecutor struct {
	registry *StreamRegistry[*StreamEntry]
}

func (e streamRegistryExecutor) entryFor(handle StreamHandle) (*StreamEntry, error) {
	if e.registry == nil {
		return nil, ErrStreamInvalidHandle
	}
	entry, ok := e.registry.Load(handle)
	if !ok || entry == nil || entry.Lifecycle == nil {
		return nil, ErrStreamInvalidHandle
	}
	return entry, nil
}

func (e streamRegistryExecutor) delete(handle StreamHandle) bool {
	if e.registry == nil {
		return false
	}
	return e.registry.Delete(handle)
}

// StreamRegistrySend loads a session from registry, checks send lifecycle state,
// and invokes call when the handle and session type are valid.
func StreamRegistrySend[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleSend(streamRegistryExecutor{registry: registry}, handle, call)
}

// StreamRegistryReceive loads a session from registry, checks receive lifecycle
// state, and invokes call when the handle and session type are valid.
func StreamRegistryReceive[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleReceive(streamRegistryExecutor{registry: registry}, handle, call)
}

// StreamRegistryCloseSend marks the send side closed and invokes call. The handle
// remains valid for receive or terminal operations.
func StreamRegistryCloseSend[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleCloseSend(streamRegistryExecutor{registry: registry}, handle, call)
}

// StreamRegistryFinish finalizes the stream, removes the handle from registry,
// and invokes call once.
func StreamRegistryFinish[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleFinish(streamRegistryExecutor{registry: registry}, handle, call)
}

// StreamRegistryDone is an alias for StreamRegistryFinish for peer-done paths.
func StreamRegistryDone[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	return StreamRegistryFinish(registry, handle, call)
}

// StreamRegistryCancel cancels the stream, removes the handle from registry,
// and invokes call once.
func StreamRegistryCancel[TSession any](registry *StreamRegistry[*StreamEntry], handle StreamHandle, call func(TSession) error) error {
	return streamLifecycleCancel(streamRegistryExecutor{registry: registry}, handle, call)
}
