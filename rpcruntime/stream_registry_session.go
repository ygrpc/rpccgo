package rpcruntime

// StreamSession is the runtime-visible part of a generated stream session.
// Generated service runtime owns the method-specific send/recv/finish/cancel
// closures; runtime core only coordinates registry lookup and lifecycle state.
type StreamSession interface {
	comparable
	StreamLifecycle() *StreamLifecycle
}

func LoadStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	return loadStreamSession[T](registry, handle)
}

func SendStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	session, err := loadStreamSession[T](registry, handle)
	if err != nil {
		return session, err
	}
	if err := session.StreamLifecycle().EnsureCanSend(); err != nil {
		var zero T
		return zero, err
	}
	return session, nil
}

func CloseSendStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	session, err := loadStreamSession[T](registry, handle)
	if err != nil {
		return session, err
	}
	if err := session.StreamLifecycle().MarkSendClosed(); err != nil {
		var zero T
		return zero, err
	}
	return session, nil
}

func RecvStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	return loadStreamSession[T](registry, handle)
}

func FinishStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	session, err := loadStreamSession[T](registry, handle)
	if err != nil {
		return session, err
	}
	taken, ok := registry.Take(handle)
	if !ok {
		var zero T
		return zero, ErrStreamInvalidHandle
	}
	takenSession, ok := taken.(T)
	if !ok || takenSession != session {
		var zero T
		return zero, ErrStreamInvalidHandle
	}
	if !session.StreamLifecycle().Finalize() {
		var zero T
		return zero, ErrStreamInvalidHandle
	}
	return session, nil
}

func CancelStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	session, err := loadStreamSession[T](registry, handle)
	if err != nil {
		return session, err
	}
	taken, ok := registry.Take(handle)
	if !ok {
		var zero T
		return zero, ErrStreamInvalidHandle
	}
	takenSession, ok := taken.(T)
	if !ok || takenSession != session {
		var zero T
		return zero, ErrStreamInvalidHandle
	}
	if err := session.StreamLifecycle().MarkCanceled(); err != nil {
		var zero T
		return zero, err
	}
	return session, nil
}

func loadStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	var zero T
	if registry == nil {
		return zero, ErrStreamInvalidHandle
	}
	value, ok := registry.Load(handle)
	if !ok {
		return zero, ErrStreamInvalidHandle
	}
	session, ok := value.(T)
	if !ok {
		return zero, ErrStreamInvalidHandle
	}
	if session.StreamLifecycle() == nil {
		return zero, ErrStreamInvalidHandle
	}
	return session, nil
}
