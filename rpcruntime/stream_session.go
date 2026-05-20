package rpcruntime

import (
	"errors"
	"io"
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

func (l *StreamLifecycle) Cancel(cancel func() error) error {
	l.mu.Lock()
	if l.finalized {
		err := ErrStreamFinalized
		if l.canceled {
			err = ErrStreamCanceled
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

func RunServerStream[T any](recv func() (T, error), send func(T) error, done func() error, cancel func() error) error {
	for {
		item, err := recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if done == nil {
					return nil
				}
				return done()
			}
			return errors.Join(err, callStreamCancel(cancel))
		}
		if send == nil {
			continue
		}
		if err := send(item); err != nil {
			return errors.Join(err, callStreamCancel(cancel))
		}
	}
}

func RunBidiStream[TReq any, TResp any](receive func() (TReq, error), sendToSession func(TReq) error, closeSend func() error, recvFromSession func() (TResp, error), sendToPeer func(TResp) error, done func() error, cancel func() error) error {
	receiveErrCh := make(chan error, 1)
	sendErrCh := make(chan error, 1)
	var terminalOnce sync.Once
	var terminalMu sync.Mutex
	terminalFinished := false
	markTerminalFinished := func() {
		terminalMu.Lock()
		terminalFinished = true
		terminalMu.Unlock()
	}
	isTerminalFinished := func() bool {
		terminalMu.Lock()
		defer terminalMu.Unlock()
		return terminalFinished
	}
	finish := func(donePath bool) error {
		var finishErr error
		terminalOnce.Do(func() {
			defer markTerminalFinished()
			if donePath {
				if done != nil {
					finishErr = done()
				}
				return
			}
			finishErr = callStreamCancel(cancel)
		})
		return finishErr
	}

	go func() {
		for {
			req, err := receive()
			if err != nil {
				if errors.Is(err, io.EOF) {
					if closeSend == nil {
						receiveErrCh <- nil
						return
					}
					err := closeSend()
					if errors.Is(err, ErrStreamInvalidHandle) && isTerminalFinished() {
						err = nil
					}
					receiveErrCh <- err
					return
				}
				receiveErrCh <- err
				return
			}
			if sendToSession == nil {
				continue
			}
			if err := sendToSession(req); err != nil {
				receiveErrCh <- err
				return
			}
		}
	}()
	go func() {
		for {
			resp, err := recvFromSession()
			if err != nil {
				if errors.Is(err, io.EOF) {
					sendErrCh <- finish(true)
					return
				}
				sendErrCh <- err
				return
			}
			if sendToPeer == nil {
				continue
			}
			if err := sendToPeer(resp); err != nil {
				sendErrCh <- err
				return
			}
		}
	}()

	for receiveErrCh != nil || sendErrCh != nil {
		select {
		case err := <-receiveErrCh:
			receiveErrCh = nil
			if err != nil {
				return errors.Join(err, finish(false))
			}
		case err := <-sendErrCh:
			sendErrCh = nil
			if err != nil {
				return errors.Join(err, finish(false))
			}
			if receiveErrCh == nil {
				return nil
			}
		}
	}
	return nil
}

func callStreamCancel(cancel func() error) error {
	if cancel == nil {
		return nil
	}
	return cancel()
}
