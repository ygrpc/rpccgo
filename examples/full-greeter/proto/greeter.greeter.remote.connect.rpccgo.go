package greeterv1

import (
	context "context"
	errors "errors"
	fmt "fmt"
	io "io"
	http "net/http"
	strings "strings"
	connect "connectrpc.com/connect"
	proto "google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo message direct stage file for Greeter connect remote server adapter

var _ *http.Client
var _ GreeterMessageAdapter = (*GreeterConnectRemoteServer)(nil)

type GreeterConnectRemoteServer struct {
	sayHello  *connect.Client[SayHelloRequest, SayHelloResponse]
	collect   *connect.Client[SayHelloRequest, SayHelloResponse]
	broadcast *connect.Client[SayHelloRequest, SayHelloResponse]
	chat      *connect.Client[SayHelloRequest, SayHelloResponse]
}

func NewGreeterConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (*GreeterConnectRemoteServer, error) {
	if httpClient == nil {
		return nil, errors.New("rpccgo: connect remote http client is nil")
	}
	if baseURL == "" {
		return nil, errors.New("rpccgo: connect remote base URL is empty")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &GreeterConnectRemoteServer{
		sayHello:  connect.NewClient[SayHelloRequest, SayHelloResponse](httpClient, baseURL+GreeterSayHelloConnectProcedure, options...),
		collect:   connect.NewClient[SayHelloRequest, SayHelloResponse](httpClient, baseURL+GreeterCollectConnectProcedure, options...),
		broadcast: connect.NewClient[SayHelloRequest, SayHelloResponse](httpClient, baseURL+GreeterBroadcastConnectProcedure, options...),
		chat:      connect.NewClient[SayHelloRequest, SayHelloResponse](httpClient, baseURL+GreeterChatConnectProcedure, options...),
	}, nil
}

func RegisterGreeterConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	adapter, err := NewGreeterConnectRemoteServer(httpClient, baseURL, options...)
	if err != nil {
		return rpcruntime.AdapterSnapshot[GreeterMessageAdapter]{}, err
	}
	return RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindConnectRemote, adapter)
}

func (s *GreeterConnectRemoteServer) SayHelloMessage(ctx context.Context, req []byte) ([]byte, error) {
	if s == nil || s.sayHello == nil {
		return nil, errors.New("rpccgo: connect remote server is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)
	}
	resp, err := s.sayHello.CallUnary(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil {
		return nil, nil
	}
	data, err := proto.Marshal(resp.Msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func (s *GreeterConnectRemoteServer) StartCollectMessage(ctx context.Context) (GreeterCollectMessageStreamSession, error) {
	if s == nil || s.collect == nil {
		return nil, errors.New("rpccgo: connect remote server is nil")
	}
	stream := s.collect.CallClientStream(ctx)
	return &GreeterCollectConnectRemoteClientStreamSession{stream: stream}, nil
}

type GreeterCollectConnectRemoteClientStreamSession struct {
	stream *connect.ClientStreamForClient[SayHelloRequest, SayHelloResponse]
}

func (s *GreeterCollectConnectRemoteClientStreamSession) Send(ctx context.Context, req []byte) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return errors.New("rpccgo: connect remote client stream is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return fmt.Errorf("rpccgo: connect remote stream request protobuf unmarshal failed: %w", err)
	}
	return s.stream.Send(request)
}

func (s *GreeterCollectConnectRemoteClientStreamSession) Finish(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil, errors.New("rpccgo: connect remote client stream is nil")
	}
	resp, err := s.stream.CloseAndReceive()
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil {
		return nil, nil
	}
	data, err := proto.Marshal(resp.Msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote stream response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func (s *GreeterCollectConnectRemoteClientStreamSession) Cancel(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	conn, err := s.stream.Conn()
	if err != nil {
		return nil
	}
	return closeConnectRemoteConn(conn)
}

func (s *GreeterConnectRemoteServer) StartBroadcastMessage(ctx context.Context, req []byte) (GreeterBroadcastMessageStreamSession, error) {
	if s == nil || s.broadcast == nil {
		return nil, errors.New("rpccgo: connect remote server is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)
	}
	stream, err := s.broadcast.CallServerStream(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return &GreeterBroadcastConnectRemoteServerStreamSession{stream: stream}, nil
}

type GreeterBroadcastConnectRemoteServerStreamSession struct {
	stream *connect.ServerStreamForClient[SayHelloResponse]
}

func (s *GreeterBroadcastConnectRemoteServerStreamSession) Recv(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil, errors.New("rpccgo: connect remote server stream is nil")
	}
	if !s.stream.Receive() {
		if err := s.stream.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	msg := s.stream.Msg()
	if msg == nil {
		return nil, nil
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote stream response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func (s *GreeterBroadcastConnectRemoteServerStreamSession) Done(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	return s.stream.Close()
}

func (s *GreeterBroadcastConnectRemoteServerStreamSession) Cancel(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	conn, err := s.stream.Conn()
	if err != nil {
		return s.stream.Close()
	}
	return closeConnectRemoteConn(conn)
}

func (s *GreeterConnectRemoteServer) StartChatMessage(ctx context.Context) (GreeterChatMessageStreamSession, error) {
	if s == nil || s.chat == nil {
		return nil, errors.New("rpccgo: connect remote server is nil")
	}
	stream := s.chat.CallBidiStream(ctx)
	return &GreeterChatConnectRemoteBidiStreamSession{stream: stream}, nil
}

type GreeterChatConnectRemoteBidiStreamSession struct {
	stream *connect.BidiStreamForClient[SayHelloRequest, SayHelloResponse]
}

func (s *GreeterChatConnectRemoteBidiStreamSession) Send(ctx context.Context, req []byte) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return errors.New("rpccgo: connect remote bidi stream is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return fmt.Errorf("rpccgo: connect remote bidi request protobuf unmarshal failed: %w", err)
	}
	return s.stream.Send(request)
}

func (s *GreeterChatConnectRemoteBidiStreamSession) Recv(ctx context.Context) ([]byte, error) {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil, errors.New("rpccgo: connect remote bidi stream is nil")
	}
	resp, err := s.stream.Receive()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote bidi response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func (s *GreeterChatConnectRemoteBidiStreamSession) CloseSend(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	return s.stream.CloseRequest()
}

func (s *GreeterChatConnectRemoteBidiStreamSession) Done(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	return s.stream.CloseResponse()
}

func (s *GreeterChatConnectRemoteBidiStreamSession) Cancel(ctx context.Context) error {
	_ = ctx
	if s == nil || s.stream == nil {
		return nil
	}
	conn, err := s.stream.Conn()
	if err != nil {
		return nil
	}
	return closeConnectRemoteConn(conn)
}

func closeConnectRemoteConn(conn connect.StreamingClientConn) error {
	if conn == nil {
		return nil
	}
	err := conn.CloseRequest()
	if closeErr := conn.CloseResponse(); err == nil {
		err = closeErr
	}
	return err
}
