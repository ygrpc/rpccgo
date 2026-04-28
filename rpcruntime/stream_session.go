package rpcruntime

import (
	"errors"
	"sync"
)

var (
	errStreamSendClosed = errors.New("stream send side is closed")
	errStreamFinalized  = errors.New("stream is finalized")
	errStreamCanceled   = errors.New("stream is canceled")
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
		return errStreamCanceled
	}
	if l.finalized {
		return errStreamFinalized
	}
	if l.sendClosed {
		return errStreamSendClosed
	}
	l.sendClosed = true
	return nil
}

func (l *StreamLifecycle) EnsureCanSend() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.canceled {
		return errStreamCanceled
	}
	if l.finalized {
		return errStreamFinalized
	}
	if l.sendClosed {
		return errStreamSendClosed
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

func (l *StreamLifecycle) Cancel(cancel func() error) error {
	l.mu.Lock()
	if l.finalized {
		err := errStreamFinalized
		if l.canceled {
			err = errStreamCanceled
		}
		l.mu.Unlock()
		return err
	}
	l.canceled = true
	l.finalized = true
	l.mu.Unlock()

	if cancel == nil {
		return nil
	}
	return cancel()
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
