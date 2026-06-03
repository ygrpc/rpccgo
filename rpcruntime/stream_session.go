package rpcruntime

import (
	"errors"
	"sync"
)

var (
	ErrStreamInvalidHandle = errors.New("stream handle is invalid")
	ErrStreamSendClosed    = errors.New("stream send side is closed")
	ErrStreamFinalized     = errors.New("stream is finalized")
	ErrStreamCanceled      = errors.New("stream is canceled")
)

type StreamLifecycle struct {
	mu         sync.Mutex
	sendClosed bool
	finalized  bool
	canceled   bool
}

func (l *StreamLifecycle) MarkSendClosed() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.canceled {
		return ErrStreamCanceled
	}
	if l.finalized {
		return ErrStreamFinalized
	}
	if l.sendClosed {
		return ErrStreamSendClosed
	}
	l.sendClosed = true
	return nil
}

func (l *StreamLifecycle) EnsureCanSend() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.canceled {
		return ErrStreamCanceled
	}
	if l.finalized {
		return ErrStreamFinalized
	}
	if l.sendClosed {
		return ErrStreamSendClosed
	}
	return nil
}

func (l *StreamLifecycle) Finalize() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.finalized {
		return false
	}
	l.finalized = true
	return true
}

func (l *StreamLifecycle) MarkCanceled() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.finalized {
		if l.canceled {
			return ErrStreamCanceled
		}
		return ErrStreamFinalized
	}
	l.canceled = true
	l.finalized = true
	return nil
}

func (l *StreamLifecycle) Finalized() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.finalized
}

func (l *StreamLifecycle) Canceled() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.canceled
}
