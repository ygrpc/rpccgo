package greeterv1

import (
	context "context"
	errors "errors"
	fmt "fmt"
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
	sayHello *connect.Client[SayHelloRequest, SayHelloResponse]
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
		sayHello: connect.NewClient[SayHelloRequest, SayHelloResponse](httpClient, baseURL+GreeterSayHelloConnectProcedure, options...),
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
