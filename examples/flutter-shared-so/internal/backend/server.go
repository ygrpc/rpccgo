package backend

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

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
