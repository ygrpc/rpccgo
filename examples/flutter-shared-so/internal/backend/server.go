package backend

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	connect "connectrpc.com/connect"
	fluttersharedv1 "example.com/rpccgo-flutter-shared-so/proto"
)

const sharedLibraryName = "librpccgo_flutter_shared.so"

// SharedSoDemoServer implements the message-contract server used by both
// Flutter FFI and Kotlin/JNI in this example.
type SharedSoDemoServer struct {
	mu       sync.Mutex
	value    int64
	revision int64
}

// NewSharedSoDemoServer creates one mutable service instance whose state is
// used to prove that Flutter FFI and Kotlin/JNI enter the same Go runtime.
func NewSharedSoDemoServer() *SharedSoDemoServer {
	return &SharedSoDemoServer{}
}

// ComposeGreeting formats a greeting that identifies which caller path reached
// the shared rpccgo runtime.
func (*SharedSoDemoServer) ComposeGreeting(_ context.Context, req *fluttersharedv1.ComposeGreetingRequest) (*fluttersharedv1.ComposeGreetingResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("compose greeting request is nil")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		name = "flutter-kotlin"
	}
	caller := strings.TrimSpace(req.GetCaller())
	if caller == "" {
		caller = "unknown-caller"
	}

	return &fluttersharedv1.ComposeGreetingResponse{
		Message:  fmt.Sprintf("hello %s via %s", name, caller),
		ServedBy: "go-connect-handler",
		Library:  sharedLibraryName,
	}, nil
}

// IncrementRuntimeState changes the state owned by this Go service instance.
func (s *SharedSoDemoServer) IncrementRuntimeState(_ context.Context, req *fluttersharedv1.IncrementRuntimeStateRequest) (*fluttersharedv1.RuntimeStateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("increment runtime state request is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.value += int64(req.GetDelta())
	s.revision++
	resp := s.runtimeStateResponse(req.GetCaller())
	log.Printf("rpccgo shared runtime write: instance_address=%s pid=%d caller=%s value=%d revision=%d", resp.GetInstanceAddress(), resp.GetPid(), resp.GetCaller(), resp.GetValue(), resp.GetRevision())
	return resp, nil
}

// ReadRuntimeState returns the state currently owned by this Go service instance.
func (s *SharedSoDemoServer) ReadRuntimeState(_ context.Context, req *fluttersharedv1.ReadRuntimeStateRequest) (*fluttersharedv1.RuntimeStateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read runtime state request is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	resp := s.runtimeStateResponse(req.GetCaller())
	log.Printf("rpccgo shared runtime read: instance_address=%s pid=%d caller=%s value=%d revision=%d", resp.GetInstanceAddress(), resp.GetPid(), resp.GetCaller(), resp.GetValue(), resp.GetRevision())
	return resp, nil
}

// WatchRuntimeState streams snapshots of the current runtime state.
func (s *SharedSoDemoServer) WatchRuntimeState(ctx context.Context, req *fluttersharedv1.ReadRuntimeStateRequest, stream *connect.ServerStream[fluttersharedv1.RuntimeStateResponse]) error {
	if req == nil {
		return fmt.Errorf("watch runtime state request is nil")
	}
	if stream == nil {
		return fmt.Errorf("watch runtime state stream is nil")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		s.mu.Lock()
		s.value++
		s.revision++
		resp := s.runtimeStateResponse(req.GetCaller())
		s.mu.Unlock()
		if err := stream.Send(resp); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// CollectRuntimeState receives a client stream of state changes and returns the
// final state snapshot.
func (s *SharedSoDemoServer) CollectRuntimeState(ctx context.Context, stream *connect.ClientStream[fluttersharedv1.IncrementRuntimeStateRequest]) (*fluttersharedv1.RuntimeStateResponse, error) {
	if stream == nil {
		return nil, fmt.Errorf("collect runtime state stream is nil")
	}

	caller := "unknown-caller"
	for stream.Receive() {
		req := stream.Msg()
		if req == nil {
			return nil, fmt.Errorf("collect runtime state request is nil")
		}
		if trimmed := strings.TrimSpace(req.GetCaller()); trimmed != "" {
			caller = trimmed
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		s.mu.Lock()
		s.value += int64(req.GetDelta())
		s.revision++
		s.mu.Unlock()
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	resp := s.runtimeStateResponse(caller)
	log.Printf("rpccgo shared runtime collect: instance_address=%s pid=%d caller=%s value=%d revision=%d", resp.GetInstanceAddress(), resp.GetPid(), resp.GetCaller(), resp.GetValue(), resp.GetRevision())
	return resp, nil
}

// StreamRuntimeState streams three snapshots of the current runtime state.
func (s *SharedSoDemoServer) StreamRuntimeState(ctx context.Context, req *fluttersharedv1.ReadRuntimeStateRequest, stream *connect.ServerStream[fluttersharedv1.RuntimeStateResponse]) error {
	if req == nil {
		return fmt.Errorf("stream runtime state request is nil")
	}
	if stream == nil {
		return fmt.Errorf("stream runtime state stream is nil")
	}
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		s.mu.Lock()
		resp := s.runtimeStateResponse(req.GetCaller())
		s.mu.Unlock()
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

// ChatRuntimeState applies each streamed state change and returns a snapshot for
// every request.
func (s *SharedSoDemoServer) ChatRuntimeState(ctx context.Context, stream *connect.BidiStream[fluttersharedv1.IncrementRuntimeStateRequest, fluttersharedv1.RuntimeStateResponse]) error {
	if stream == nil {
		return fmt.Errorf("chat runtime state stream is nil")
	}

	for {
		req, err := stream.Receive()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if req == nil {
			return fmt.Errorf("chat runtime state request is nil")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		s.mu.Lock()
		s.value += int64(req.GetDelta())
		s.revision++
		resp := s.runtimeStateResponse(req.GetCaller())
		s.mu.Unlock()
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

func (s *SharedSoDemoServer) runtimeStateResponse(caller string) *fluttersharedv1.RuntimeStateResponse {
	caller = strings.TrimSpace(caller)
	if caller == "" {
		caller = "unknown-caller"
	}
	return &fluttersharedv1.RuntimeStateResponse{
		Value:           s.value,
		Revision:        s.revision,
		InstanceAddress: fmt.Sprintf("%p", s),
		Caller:          caller,
		Pid:             int32(os.Getpid()),
	}
}
