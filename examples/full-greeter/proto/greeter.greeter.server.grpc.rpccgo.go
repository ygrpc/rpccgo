package greeterv1

import (
	context "context"
	errors "errors"
	io "io"
	rpcruntime "rpccgo/rpcruntime"
	sync "sync"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	proto "google.golang.org/protobuf/proto"
)

// rpccgo message direct generated file for Greeter grpc local server adapter

type GreeterGRPCHandler interface{}

func RegisterGreeterGRPCServer(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return errors.New("rpccgo: grpc registrar is nil")
	}
	registrar.RegisterService(&GreeterGRPCServiceDesc, struct{}{})
	return nil
}

const GreeterSayHelloGRPCFullMethodName = "/examples.full.greeter.v1.Greeter/SayHello"

var GreeterGRPCServiceDesc = grpc.ServiceDesc{
	ServiceName: "examples.full.greeter.v1.Greeter",
	HandlerType: (*GreeterGRPCHandler)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Greeter_SayHello_GRPC_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Collect",
			Handler:       _Greeter_Collect_GRPC_StreamHandler,
			ClientStreams: true,
		},
		{
			StreamName:    "Broadcast",
			Handler:       _Greeter_Broadcast_GRPC_StreamHandler,
			ServerStreams: true,
		},
		{
			StreamName:    "Chat",
			Handler:       _Greeter_Chat_GRPC_StreamHandler,
			ClientStreams: true,
			ServerStreams: true,
		},
	},
	Metadata: "greeter.proto",
}

func _Greeter_SayHello_GRPC_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(SayHelloRequest)
	if err := dec(in); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc request decode failed: %v", err)
	}
	handler := func(ctx context.Context, req any) (any, error) {
		typed, ok := req.(*SayHelloRequest)
		if !ok || typed == nil {
			return nil, status.Error(codes.InvalidArgument, "rpccgo: grpc request type mismatch")
		}
		return greeterSayHelloGRPC(ctx, typed)
	}
	if interceptor == nil {
		return handler(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: GreeterSayHelloGRPCFullMethodName}
	return interceptor(ctx, in, info, handler)
}

func greeterSayHelloGRPC(ctx context.Context, req *SayHelloRequest) (*SayHelloResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "rpccgo: grpc request is nil")
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc request protobuf marshal failed: %v", err)
	}
	respData, err := NewGreeterCGOMessageClientBridge().SayHello(ctx, reqData)
	if err != nil {
		return nil, err
	}
	resp := new(SayHelloResponse)
	if err := proto.Unmarshal(respData, resp); err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc response protobuf unmarshal failed: %v", err)
	}
	return resp, nil
}

func _Greeter_Collect_GRPC_StreamHandler(srv any, stream grpc.ServerStream) error {
	return greeterCollectGRPC(&grpc.GenericServerStream[SayHelloRequest, SayHelloResponse]{ServerStream: stream})
}

func greeterCollectGRPC(stream grpc.ClientStreamingServer[SayHelloRequest, SayHelloResponse]) error {
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartCollect(stream.Context())
	if err != nil {
		return err
	}
	session, ok := rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return errors.New("rpccgo: grpc message stream handle is invalid")
	}
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(stream.Context())
			}
			return err
		}
		reqData, err := proto.Marshal(req)
		if err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(stream.Context())
			}
			return status.Errorf(codes.Internal, "rpccgo: grpc stream request protobuf marshal failed: %v", err)
		}
		if err := session.Send(stream.Context(), reqData); err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(stream.Context())
			}
			return err
		}
	}
	terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return errors.New("rpccgo: grpc message stream handle is invalid")
	}
	respData, err := terminal.Finish(stream.Context())
	if err != nil {
		return err
	}
	resp := new(SayHelloResponse)
	if err := proto.Unmarshal(respData, resp); err != nil {
		return status.Errorf(codes.Internal, "rpccgo: grpc stream response protobuf unmarshal failed: %v", err)
	}
	return stream.SendAndClose(resp)
}

func _Greeter_Broadcast_GRPC_StreamHandler(srv any, stream grpc.ServerStream) error {
	m := new(SayHelloRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return greeterBroadcastGRPC(m, &grpc.GenericServerStream[SayHelloRequest, SayHelloResponse]{ServerStream: stream})
}

func greeterBroadcastGRPC(req *SayHelloRequest, stream grpc.ServerStreamingServer[SayHelloResponse]) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "rpccgo: grpc request is nil")
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return status.Errorf(codes.Internal, "rpccgo: grpc stream request protobuf marshal failed: %v", err)
	}
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartBroadcast(stream.Context(), reqData)
	if err != nil {
		return err
	}
	session, ok := rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return errors.New("rpccgo: grpc message stream handle is invalid")
	}
	for {
		respData, err := session.Recv(stream.Context())
		if err != nil {
			terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
			if !ok {
				return errors.New("rpccgo: grpc message stream handle is invalid")
			}
			if errors.Is(err, io.EOF) {
				return terminal.Done(stream.Context())
			}
			_ = terminal.Cancel(stream.Context())
			return err
		}
		resp := new(SayHelloResponse)
		if err := proto.Unmarshal(respData, resp); err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(stream.Context())
			}
			return status.Errorf(codes.Internal, "rpccgo: grpc stream response protobuf unmarshal failed: %v", err)
		}
		if err := stream.Send(resp); err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(stream.Context())
			}
			return err
		}
	}
}

func _Greeter_Chat_GRPC_StreamHandler(srv any, stream grpc.ServerStream) error {
	return greeterChatGRPC(&grpc.GenericServerStream[SayHelloRequest, SayHelloResponse]{ServerStream: stream})
}

func greeterChatGRPC(stream grpc.BidiStreamingServer[SayHelloRequest, SayHelloResponse]) error {
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartChat(stream.Context())
	if err != nil {
		return err
	}
	session, ok := rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return errors.New("rpccgo: grpc message stream handle is invalid")
	}
	var terminalOnce sync.Once
	finish := func(done bool) error {
		var finishErr error
		terminalOnce.Do(func() {
			terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
			if !ok {
				finishErr = errors.New("rpccgo: grpc message stream handle is invalid")
				return
			}
			if done {
				finishErr = terminal.Done(stream.Context())
				return
			}
			finishErr = terminal.Cancel(stream.Context())
		})
		return finishErr
	}
	receiveErrCh := make(chan error, 1)
	sendErrCh := make(chan error, 1)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					receiveErrCh <- session.CloseSend(stream.Context())
					return
				}
				receiveErrCh <- err
				return
			}
			reqData, err := proto.Marshal(req)
			if err != nil {
				receiveErrCh <- status.Errorf(codes.Internal, "rpccgo: grpc bidi request protobuf marshal failed: %v", err)
				return
			}
			if err := session.Send(stream.Context(), reqData); err != nil {
				receiveErrCh <- err
				return
			}
		}
	}()
	go func() {
		for {
			respData, err := session.Recv(stream.Context())
			if err != nil {
				if errors.Is(err, io.EOF) {
					sendErrCh <- finish(true)
					return
				}
				sendErrCh <- err
				return
			}
			resp := new(SayHelloResponse)
			if err := proto.Unmarshal(respData, resp); err != nil {
				sendErrCh <- status.Errorf(codes.Internal, "rpccgo: grpc bidi response protobuf unmarshal failed: %v", err)
				return
			}
			if err := stream.Send(resp); err != nil {
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
				_ = finish(false)
				return err
			}
		case err := <-sendErrCh:
			sendErrCh = nil
			if err != nil {
				_ = finish(false)
				return err
			}
			if receiveErrCh == nil {
				return nil
			}
		}
	}
	return nil
}
