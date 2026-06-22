package rpcruntime

import (
	"sync/atomic"
)

var streamSessions StreamRegistry

// StreamSession is the runtime-visible record for one active stream.
// Runtime core owns the handle registry and terminal removal; generated service
// code owns method-specific typed dispatch and Native/Message conversion.
type StreamSession struct {
	Kind    ServerKind
	Session any

	CallbackReceiveEnabled atomic.Bool
	canceled               atomic.Bool
	done                   atomic.Bool
	doneCallbackStarted    atomic.Bool
	activeCallbacks        atomic.Int32
	stateChanged           chan struct{}
}

func newStreamSession(kind ServerKind, session any) *StreamSession {
	return &StreamSession{
		Kind:         kind,
		Session:      session,
		stateChanged: make(chan struct{}, 1),
	}
}

func CreateStreamSession(kind ServerKind, session any) (StreamHandle, error) {
	if kind <= ServerKindInvalid || kind > ServerKindGRPCRemote {
		return 0, ErrInvalidServerKind
	}
	if !hasNonZeroSession(session) {
		return 0, errStreamRegistryZeroSession
	}
	return streamSessions.Create(newStreamSession(kind, session))
}

// LoadStreamSession returns the active stream session without removing its handle.
func LoadStreamSession(handle StreamHandle) (*StreamSession, error) {
	value, ok := streamSessions.Load(handle)
	if !ok {
		return nil, ErrStreamInvalidHandle
	}
	session, ok := value.(*StreamSession)
	if !ok {
		return nil, ErrStreamInvalidHandle
	}
	return session, nil
}

// EnableStreamCallbackReceive marks an active stream as callback-receive owned.
func EnableStreamCallbackReceive(handle StreamHandle) (*StreamSession, error) {
	if handle == 0 {
		return nil, ErrStreamInvalidHandle
	}

	streamSessions.mu.Lock()
	defer streamSessions.mu.Unlock()

	value, ok := streamSessions.entries[handle]
	if !ok {
		return nil, ErrStreamInvalidHandle
	}
	session, ok := value.(*StreamSession)
	if !ok {
		return nil, ErrStreamInvalidHandle
	}
	if session.CallbackReceiveEnabled.Load() {
		return nil, ErrStreamInvalidHandle
	}
	session.CallbackReceiveEnabled.Store(true)
	return session, nil
}

// StreamCallbackReceiveEnabled reports whether Recv is owned by callback mode.
func StreamCallbackReceiveEnabled(handle StreamHandle) bool {
	session, err := LoadStreamSession(handle)
	return err == nil && session.CallbackReceiveEnabled.Load()
}

// StreamCallbackReceiveState returns callback receive state for an active stream.
func StreamCallbackReceiveState(handle StreamHandle) (*StreamSession, error) {
	session, err := LoadStreamSession(handle)
	if err != nil {
		return nil, err
	}
	if !session.CallbackReceiveEnabled.Load() {
		return nil, ErrStreamInvalidHandle
	}
	return session, nil
}

// BeginCallback enters an external callback if the stream has not been canceled.
func (s *StreamSession) BeginCallback() bool {
	if s.canceled.Load() || s.done.Load() {
		return false
	}
	s.activeCallbacks.Add(1)
	if s.canceled.Load() || s.done.Load() {
		if s.activeCallbacks.Add(-1) == 0 {
			s.signalStateChange()
		}
		return false
	}
	return true
}

// EndCallback leaves an external callback.
func (s *StreamSession) EndCallback() {
	if s.activeCallbacks.Add(-1) == 0 {
		s.signalStateChange()
	}
}

// MarkCanceled prevents future callbacks and waits for an in-flight callback.
func (s *StreamSession) MarkCanceled() {
	s.canceled.Store(true)
	for s.activeCallbacks.Load() > 0 {
		<-s.stateChanged
	}
}

// WaitDone waits until the callback receive loop has delivered onDone.
func (s *StreamSession) WaitDone() {
	for !s.done.Load() {
		<-s.stateChanged
	}
}

// BeginDoneCallback enters the terminal callback if it has not run yet.
func (s *StreamSession) BeginDoneCallback() bool {
	if s.done.Load() {
		return false
	}
	if !s.doneCallbackStarted.CompareAndSwap(false, true) {
		return false
	}
	s.activeCallbacks.Add(1)
	if s.done.Load() {
		if s.activeCallbacks.Add(-1) == 0 {
			s.signalStateChange()
		}
		return false
	}
	return true
}

// EndDoneCallback records that the terminal callback has completed.
func (s *StreamSession) EndDoneCallback() {
	s.done.Store(true)
	if s.activeCallbacks.Add(-1) == 0 {
		s.signalStateChange()
		return
	}
	s.signalStateChange()
}

// RemoveStreamSession returns the active stream session and removes its handle.
func RemoveStreamSession(handle StreamHandle) (*StreamSession, error) {
	value, ok := streamSessions.Take(handle)
	if !ok {
		return nil, ErrStreamInvalidHandle
	}
	session, ok := value.(*StreamSession)
	if !ok {
		return nil, ErrStreamInvalidHandle
	}
	return session, nil
}

func ResetStreamSessionsForTesting() {
	streamSessions = StreamRegistry{}
}

func (s *StreamSession) signalStateChange() {
	select {
	case s.stateChanged <- struct{}{}:
	default:
	}
}
