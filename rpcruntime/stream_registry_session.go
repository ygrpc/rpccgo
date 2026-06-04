package rpcruntime

// StreamSession is the runtime-visible part of a generated stream session.
// Generated service runtime owns method-specific dispatch; runtime core only
// coordinates registry lookup and final removal.
type StreamSession interface {
	comparable
}

func LoadStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	return loadStreamSession[T](registry, handle)
}

func SendStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	return loadStreamSession[T](registry, handle)
}

func CloseSendStreamSession[T StreamSession](registry *StreamRegistry, handle StreamHandle) (T, error) {
	return loadStreamSession[T](registry, handle)
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
	return session, nil
}
