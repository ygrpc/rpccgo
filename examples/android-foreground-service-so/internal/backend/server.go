package backend

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	connect "connectrpc.com/connect"
	foregroundservicev1 "example.com/rpccgo-android-foreground-service-so/proto"
)

const sharedLibraryName = "librpccgo_android_foreground_service.so"

// ForegroundServiceDemoServer implements the Android foreground service demo.
type ForegroundServiceDemoServer struct {
	seq atomic.Int64
}

// NewForegroundServiceDemoServer creates the demo service instance loaded from
// the Android shared library.
func NewForegroundServiceDemoServer() *ForegroundServiceDemoServer {
	return &ForegroundServiceDemoServer{}
}

// ServiceInfo returns process and shared-library markers for the Android app.
func (s *ForegroundServiceDemoServer) ServiceInfo(_ context.Context, _ *foregroundservicev1.ServiceInfoRequest) (*foregroundservicev1.ServiceInfoResponse, error) {
	return &foregroundservicev1.ServiceInfoResponse{
		Library:         sharedLibraryName,
		Pid:             int32(os.Getpid()),
		InstanceAddress: fmt.Sprintf("%p", s),
	}, nil
}

// WatchTicks streams ticks until the client cancels the stream.
func (s *ForegroundServiceDemoServer) WatchTicks(ctx context.Context, req *foregroundservicev1.WatchTicksRequest, stream *connect.ServerStream[foregroundservicev1.Tick]) error {
	if req == nil {
		return fmt.Errorf("watch ticks request is nil")
	}
	if stream == nil {
		return fmt.Errorf("watch ticks stream is nil")
	}
	interval := time.Duration(req.GetIntervalMillis()) * time.Millisecond
	if interval <= 0 {
		interval = time.Second
	}
	caller := strings.TrimSpace(req.GetCaller())
	if caller == "" {
		caller = "android-foreground-service"
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if err := stream.Send(&foregroundservicev1.Tick{
				Seq:             s.seq.Add(1),
				UnixMillis:      now.UnixMilli(),
				Pid:             int32(os.Getpid()),
				InstanceAddress: fmt.Sprintf("%p", s),
				Caller:          caller,
			}); err != nil {
				return err
			}
		}
	}
}
