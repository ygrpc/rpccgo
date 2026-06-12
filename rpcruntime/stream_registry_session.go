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

func LoadStreamSession(handle StreamHandle) (StreamSession, error) {
	return loadStreamSession(handle)
}

func SendStreamSession(handle StreamHandle) (StreamSession, error) {
	return loadStreamSession(handle)
}

func CloseSendStreamSession(handle StreamHandle) (StreamSession, error) {
	return loadStreamSession(handle)
}

func RecvStreamSession(handle StreamHandle) (StreamSession, error) {
	return loadStreamSession(handle)
}

func FinishStreamSession(handle StreamHandle) (StreamSession, error) {
	return takeStreamSession(handle)
}

func CancelStreamSession(handle StreamHandle) (StreamSession, error) {
	return takeStreamSession(handle)
}

func ResetStreamSessionsForTesting() {
	streamSessions = StreamRegistry{}
}

func loadStreamSession(handle StreamHandle) (StreamSession, error) {
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

func takeStreamSession(handle StreamHandle) (StreamSession, error) {
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
