package rpcruntime

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestClientStreamingEndpointsSendRecvFinish(t *testing.T) {
	streamClosed := errors.New("stream closed")
	client, server, streamCtx := NewClientStreaming[int, string](context.Background(), LocalStreamOptions{
		RequestBuffer: 16,
		StreamClosed:  streamClosed,
	})
	handlerDone := make(chan error, 1)
	go func() {
		req, err := server.Recv(streamCtx)
		if err != nil {
			handlerDone <- err
			return
		}
		if req != 7 {
			handlerDone <- errors.New("unexpected request")
			return
		}
		if _, err := server.Recv(streamCtx); !errors.Is(err, io.EOF) {
			handlerDone <- errors.New("expected EOF after finish")
			return
		}
		server.Complete("done", nil)
		handlerDone <- nil
	}()

	if err := client.Send(context.Background(), 7); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	resp, err := client.Finish(context.Background())
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if resp != "done" {
		t.Fatalf("Finish() response = %q, want done", resp)
	}
	assertLocalStreamCompletes(t, handlerDone)
}

func TestClientStreamingFinishReturnsWhenParentContextIsCanceled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	client, _, _ := NewClientStreaming[int, string](parent, LocalStreamOptions{})
	cancel()

	finishDone := make(chan error, 1)
	go func() {
		_, err := client.Finish(context.Background())
		finishDone <- err
	}()
	select {
	case err := <-finishDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Finish() error = %v, want context canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Finish() did not observe parent context cancellation")
	}
}

func TestClientStreamingFinishTimeoutCancelsStreamContext(t *testing.T) {
	client, _, streamCtx := NewClientStreaming[int, string](context.Background(), LocalStreamOptions{})
	finishCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	if _, err := client.Finish(finishCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Finish() error = %v, want deadline exceeded", err)
	}
	select {
	case <-streamCtx.Done():
		if !errors.Is(streamCtx.Err(), context.Canceled) {
			t.Fatalf("stream context error = %v, want context canceled", streamCtx.Err())
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Finish() timeout did not cancel stream context")
	}
}

func TestServerStreamingEndpointsFinishMakesServerSendEOF(t *testing.T) {
	client, server, streamCtx := NewServerStreaming[string](context.Background(), LocalStreamOptions{
		ResponseBuffer: 1,
		StreamClosed:   errors.New("stream closed"),
	})
	sendDone := make(chan error, 1)
	go func() {
		err := server.Send(streamCtx, "one")
		server.Complete(err)
		sendDone <- err
	}()

	if err := client.Finish(context.Background()); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if err := <-sendDone; !errors.Is(err, io.EOF) {
		t.Fatalf("server Send() error = %v, want EOF after Finish", err)
	}
}

func TestServerStreamingFinishReturnsWhenParentContextIsCanceled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	client, _, _ := NewServerStreaming[string](parent, LocalStreamOptions{})
	cancel()

	finishDone := make(chan error, 1)
	go func() {
		finishDone <- client.Finish(context.Background())
	}()
	select {
	case err := <-finishDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Finish() error = %v, want context canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Finish() did not observe parent context cancellation")
	}
}

func TestBidiStreamingEndpointsRoundTripAndCloseSend(t *testing.T) {
	client, server, streamCtx := NewBidiStreaming[int, string](context.Background(), LocalStreamOptions{
		RequestBuffer:  16,
		ResponseBuffer: 1,
		StreamClosed:   errors.New("stream closed"),
	})
	handlerDone := make(chan error, 1)
	go func() {
		req, err := server.Recv(streamCtx)
		if err != nil {
			handlerDone <- err
			return
		}
		if req != 9 {
			handlerDone <- errors.New("unexpected request")
			return
		}
		if err := server.Send(streamCtx, "nine"); err != nil {
			handlerDone <- err
			return
		}
		if _, err := server.Recv(streamCtx); !errors.Is(err, io.EOF) {
			handlerDone <- errors.New("expected EOF after close send")
			return
		}
		server.Complete(nil)
		handlerDone <- nil
	}()

	if err := client.Send(context.Background(), 9); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	resp, err := client.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	if resp != "nine" {
		t.Fatalf("Recv() response = %q, want nine", resp)
	}
	if err := client.CloseSend(context.Background()); err != nil {
		t.Fatalf("CloseSend() error = %v", err)
	}
	assertLocalStreamCompletes(t, handlerDone)
}

func TestBidiStreamingCloseSendDoesNotWaitForServerRecv(t *testing.T) {
	client, server, streamCtx := NewBidiStreaming[int, string](context.Background(), LocalStreamOptions{
		RequestBuffer:  1,
		ResponseBuffer: 1,
		StreamClosed:   errors.New("stream closed"),
	})
	handlerDone := make(chan error, 1)
	go func() {
		if err := server.Send(streamCtx, "ready"); err != nil {
			handlerDone <- err
			return
		}
		if _, err := server.Recv(streamCtx); !errors.Is(err, io.EOF) {
			handlerDone <- errors.New("expected EOF after close send")
			return
		}
		server.Complete(nil)
		handlerDone <- nil
	}()

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- client.CloseSend(context.Background())
	}()
	assertLocalStreamCompletes(t, closeDone)

	resp, err := client.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	if resp != "ready" {
		t.Fatalf("Recv() response = %q, want ready", resp)
	}
	assertLocalStreamCompletes(t, handlerDone)
}

func TestBidiStreamingAllowsConcurrentSendAndRecv(t *testing.T) {
	client, server, streamCtx := NewBidiStreaming[int, string](context.Background(), LocalStreamOptions{
		RequestBuffer:  1,
		ResponseBuffer: 1,
		StreamClosed:   errors.New("stream closed"),
	})
	handlerDone := make(chan error, 1)
	go func() {
		if err := server.Send(streamCtx, "ready"); err != nil {
			handlerDone <- err
			return
		}
		req, err := server.Recv(streamCtx)
		if err != nil {
			handlerDone <- err
			return
		}
		if req != 7 {
			handlerDone <- errors.New("unexpected request")
			return
		}
		server.Complete(nil)
		handlerDone <- nil
	}()

	recvDone := make(chan error, 1)
	go func() {
		resp, err := client.Recv(context.Background())
		if err == nil && resp != "ready" {
			err = errors.New("unexpected response")
		}
		recvDone <- err
	}()
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- client.Send(context.Background(), 7)
	}()

	assertLocalStreamCompletes(t, recvDone)
	assertLocalStreamCompletes(t, sendDone)
	assertLocalStreamCompletes(t, handlerDone)
}

func TestBidiStreamingFinishReturnsWhenParentContextIsCanceled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	client, _, _ := NewBidiStreaming[int, string](parent, LocalStreamOptions{})
	cancel()

	finishDone := make(chan error, 1)
	go func() {
		finishDone <- client.Finish(context.Background())
	}()
	select {
	case err := <-finishDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Finish() error = %v, want context canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Finish() did not observe parent context cancellation")
	}
}

func TestLocalStreamingRejectsConfiguredNilValues(t *testing.T) {
	nilRequest := errors.New("request is nil")
	client, _, _ := NewClientStreaming[*int, string](context.Background(), LocalStreamOptions{NilRequest: nilRequest})
	if err := client.Send(context.Background(), nil); !errors.Is(err, nilRequest) {
		t.Fatalf("Send() error = %v, want %v", err, nilRequest)
	}

	nilResponse := errors.New("response is nil")
	_, server, _ := NewServerStreaming[*string](context.Background(), LocalStreamOptions{NilResponse: nilResponse})
	if err := server.Send(context.Background(), nil); !errors.Is(err, nilResponse) {
		t.Fatalf("Send() error = %v, want %v", err, nilResponse)
	}
}

func assertLocalStreamCompletes(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("local stream operation timed out")
	}
}
