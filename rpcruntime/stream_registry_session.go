package rpcruntime

var streamSessions StreamRegistry

// StreamSession is the runtime-visible record for one active stream.
// Runtime core owns the handle registry and terminal removal; generated service
// code owns method-specific typed dispatch and Native/Message conversion.
type StreamSession struct {
	Kind    ServerKind
	Session any
}

func CreateStreamSession(kind ServerKind, session any) (StreamHandle, error) {
	if kind <= ServerKindInvalid || kind > ServerKindGRPCRemote {
		return 0, ErrInvalidServerKind
	}
	if !hasNonZeroSession(session) {
		return 0, errStreamRegistryZeroSession
	}
	return streamSessions.Create(StreamSession{Kind: kind, Session: session})
}

// LoadStreamSession returns the active stream session without removing its handle.
func LoadStreamSession(handle StreamHandle) (StreamSession, error) {
	value, ok := streamSessions.Load(handle)
	if !ok {
		return StreamSession{}, ErrStreamInvalidHandle
	}
	session, ok := value.(StreamSession)
	if !ok {
		return StreamSession{}, ErrStreamInvalidHandle
	}
	return session, nil
}

// RemoveStreamSession returns the active stream session and removes its handle.
func RemoveStreamSession(handle StreamHandle) (StreamSession, error) {
	value, ok := streamSessions.Take(handle)
	if !ok {
		return StreamSession{}, ErrStreamInvalidHandle
	}
	session, ok := value.(StreamSession)
	if !ok {
		return StreamSession{}, ErrStreamInvalidHandle
	}
	return session, nil
}

func ResetStreamSessionsForTesting() {
	streamSessions = StreamRegistry{}
}
