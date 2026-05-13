package greeterv1

import (
	context "context"
	errors "errors"
	io "io"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	proto "google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo message direct generated file for Greeter grpc remote server adapter

const GreeterCollectGRPCFullMethodName = "/examples.full.greeter.v1.Greeter/Collect"
const GreeterBroadcastGRPCFullMethodName = "/examples.full.greeter.v1.Greeter/Broadcast"
const GreeterChatGRPCFullMethodName = "/examples.full.greeter.v1.Greeter/Chat"

type GreeterGRPCRemoteServer struct {
	conn grpc.ClientConnInterface
}

func NewGreeterGRPCRemoteServer(conn grpc.ClientConnInterface) (*GreeterGRPCRemoteServer, error) {
	if conn == nil {
		return nil, errors.New("rpccgo: grpc remote client connection is nil")
	}
	return &GreeterGRPCRemoteServer{conn: conn}, nil
}

func RegisterGreeterGRPCRemoteServer(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	adapter, err := NewGreeterGRPCRemoteServer(conn)
	if err != nil {
		return rpcruntime.AdapterSnapshot[GreeterMessageAdapter]{}, err
	}
	return RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindGRPCRemote, adapter)
}

func (s *GreeterGRPCRemoteServer) SayHelloMessage(ctx context.Context, req []byte) ([]byte, error) {
	if s == nil || s.conn == nil {
		return nil, errors.New("rpccgo: grpc remote server is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote request protobuf unmarshal failed: %v", err)
	}
	response := new(SayHelloResponse)
	err := s.conn.Invoke(ctx, GreeterSayHelloGRPCFullMethodName, request, response)
	if err != nil {
		return nil, err
	}
	respData, err := proto.Marshal(response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote response protobuf marshal failed: %v", err)
	}
	return respData, nil
}

func (s *GreeterGRPCRemoteServer) StartCollectMessage(ctx context.Context) (GreeterCollectMessageStreamSession, error) {
	if s == nil || s.conn == nil {
		return nil, errors.New("rpccgo: grpc remote server is nil")
	}
	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := s.conn.NewStream(streamCtx, &grpc.StreamDesc{ClientStreams: true}, GreeterCollectGRPCFullMethodName)
	if err != nil {
		cancel()
		return nil, err
	}
	return &GreeterCollectGRPCRemoteClientStreamSession{stream: &grpc.GenericClientStream[SayHelloRequest, SayHelloResponse]{ClientStream: stream}, cancel: cancel}, nil
}

type GreeterCollectGRPCRemoteClientStreamSession struct {
	stream *grpc.GenericClientStream[SayHelloRequest, SayHelloResponse]
	cancel context.CancelFunc
}

func (s *GreeterCollectGRPCRemoteClientStreamSession) Send(ctx context.Context, req []byte) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return errors.New("rpccgo: grpc remote client stream is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote stream request protobuf unmarshal failed: %v", err)
	}
	return s.stream.Send(request)
}

func (s *GreeterCollectGRPCRemoteClientStreamSession) Finish(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil, errors.New("rpccgo: grpc remote client stream is nil")
	}
	defer func() {
		if s.cancel != nil {
			s.cancel()
		}
	}()
	response, err := s.stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	respData, err := proto.Marshal(response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote stream response protobuf marshal failed: %v", err)
	}
	return respData, nil
}

func (s *GreeterCollectGRPCRemoteClientStreamSession) Cancel(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	return s.stream.CloseSend()
}

func (s *GreeterGRPCRemoteServer) StartBroadcastMessage(ctx context.Context, req []byte) (GreeterBroadcastMessageStreamSession, error) {
	if s == nil || s.conn == nil {
		return nil, errors.New("rpccgo: grpc remote server is nil")
	}
	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := s.conn.NewStream(streamCtx, &grpc.StreamDesc{ServerStreams: true}, GreeterBroadcastGRPCFullMethodName)
	if err != nil {
		cancel()
		return nil, err
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		cancel()
		return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote request protobuf unmarshal failed: %v", err)
	}
	client := &grpc.GenericClientStream[SayHelloRequest, SayHelloResponse]{ClientStream: stream}
	if err := client.Send(request); err != nil {
		cancel()
		return nil, err
	}
	if err := client.CloseSend(); err != nil {
		cancel()
		return nil, err
	}
	return &GreeterBroadcastGRPCRemoteServerStreamSession{stream: client, cancel: cancel}, nil
}

type GreeterBroadcastGRPCRemoteServerStreamSession struct {
	stream *grpc.GenericClientStream[SayHelloRequest, SayHelloResponse]
	cancel context.CancelFunc
}

func (s *GreeterBroadcastGRPCRemoteServerStreamSession) Recv(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil, errors.New("rpccgo: grpc remote server stream is nil")
	}
	response, err := s.stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, err
	}
	respData, err := proto.Marshal(response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote stream response protobuf marshal failed: %v", err)
	}
	return respData, nil
}

func (s *GreeterBroadcastGRPCRemoteServerStreamSession) Done(ctx context.Context) error {
	_ = ctx
	if s != nil && s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *GreeterBroadcastGRPCRemoteServerStreamSession) Cancel(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	return s.stream.CloseSend()
}

func (s *GreeterGRPCRemoteServer) StartChatMessage(ctx context.Context) (GreeterChatMessageStreamSession, error) {
	if s == nil || s.conn == nil {
		return nil, errors.New("rpccgo: grpc remote server is nil")
	}
	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := s.conn.NewStream(streamCtx, &grpc.StreamDesc{ClientStreams: true, ServerStreams: true}, GreeterChatGRPCFullMethodName)
	if err != nil {
		cancel()
		return nil, err
	}
	return &GreeterChatGRPCRemoteBidiStreamSession{stream: &grpc.GenericClientStream[SayHelloRequest, SayHelloResponse]{ClientStream: stream}, cancel: cancel}, nil
}

type GreeterChatGRPCRemoteBidiStreamSession struct {
	stream *grpc.GenericClientStream[SayHelloRequest, SayHelloResponse]
	cancel context.CancelFunc
}

func (s *GreeterChatGRPCRemoteBidiStreamSession) Send(ctx context.Context, req []byte) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return errors.New("rpccgo: grpc remote bidi stream is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote bidi request protobuf unmarshal failed: %v", err)
	}
	return s.stream.Send(request)
}

func (s *GreeterChatGRPCRemoteBidiStreamSession) Recv(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil, errors.New("rpccgo: grpc remote bidi stream is nil")
	}
	response, err := s.stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, err
	}
	respData, err := proto.Marshal(response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote bidi response protobuf marshal failed: %v", err)
	}
	return respData, nil
}

func (s *GreeterChatGRPCRemoteBidiStreamSession) CloseSend(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	return s.stream.CloseSend()
}

func (s *GreeterChatGRPCRemoteBidiStreamSession) Done(ctx context.Context) error {
	_ = ctx
	if s != nil && s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *GreeterChatGRPCRemoteBidiStreamSession) Cancel(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	return s.stream.CloseSend()
}
