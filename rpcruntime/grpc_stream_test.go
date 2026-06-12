package rpcruntime

import (
	"context"
	"errors"
	"io"
	"testing"

	"google.golang.org/grpc"
)

type grpcStreamTestRequest struct{ Value string }
type grpcStreamTestResponse struct{ Value string }

var (
	_ grpc.ClientStreamingServer[grpcStreamTestRequest, grpcStreamTestResponse] = (*GRPCClientStreamingServer[grpcStreamTestRequest, grpcStreamTestResponse])(nil)
	_ grpc.ServerStreamingServer[grpcStreamTestResponse]                        = (*GRPCServerStreamingServer[grpcStreamTestResponse])(nil)
	_ grpc.BidiStreamingServer[grpcStreamTestRequest, grpcStreamTestResponse]   = (*GRPCBidiStreamingServer[grpcStreamTestRequest, grpcStreamTestResponse])(nil)
)

func TestGRPCClientStreamingServerRoundTrip(t *testing.T) {
	client, server, streamCtx := NewClientStreaming[*grpcStreamTestRequest, *grpcStreamTestResponse](context.Background(), LocalStreamOptions{
		StreamClosed: errors.New("stream closed"),
		NilRequest:   errors.New("nil request"),
	})
	adapter := NewGRPCClientStreamingServer[grpcStreamTestRequest, grpcStreamTestResponse](streamCtx, server)

	go func() {
		req, err := adapter.Recv()
		if err != nil {
			adapter.Complete(err)
			return
		}
		if err := adapter.SendAndClose(&grpcStreamTestResponse{Value: req.Value}); err != nil {
			adapter.Complete(err)
			return
		}
		adapter.Complete(nil)
	}()

	if err := client.Send(context.Background(), &grpcStreamTestRequest{Value: "ok"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	resp, err := client.Finish(context.Background())
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if resp == nil || resp.Value != "ok" {
		t.Fatalf("Finish() response = %#v, want ok", resp)
	}
}

func TestGRPCClientStreamingServerRequiresSendAndClose(t *testing.T) {
	client, server, streamCtx := NewClientStreaming[*grpcStreamTestRequest, *grpcStreamTestResponse](context.Background(), LocalStreamOptions{})
	adapter := NewGRPCClientStreamingServer[grpcStreamTestRequest, grpcStreamTestResponse](streamCtx, server)
	adapter.Complete(nil)

	if _, err := client.Finish(context.Background()); !errors.Is(err, errGRPCStreamNoResponse) {
		t.Fatalf("Finish() error = %v, want %v", err, errGRPCStreamNoResponse)
	}
}

func TestGRPCServerStreamingServerFinishIsGraceful(t *testing.T) {
	client, server, streamCtx := NewServerStreaming[*grpcStreamTestResponse](context.Background(), LocalStreamOptions{
		StreamClosed: errors.New("stream closed"),
		NilResponse:  errors.New("nil response"),
	})
	adapter := NewGRPCServerStreamingServer[grpcStreamTestResponse](streamCtx, server)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if err := adapter.Send(&grpcStreamTestResponse{Value: "next"}); err != nil {
				if !errors.Is(err, io.EOF) {
					server.Complete(err)
					return
				}
				server.Complete(nil)
				return
			}
		}
	}()

	if _, err := client.Recv(context.Background()); err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	if err := client.Finish(context.Background()); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	<-done
}
