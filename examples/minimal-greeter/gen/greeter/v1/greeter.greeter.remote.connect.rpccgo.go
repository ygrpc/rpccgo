package greeterv1

import (
	context "context"
	errors "errors"
	fmt "fmt"
	proto "google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo message direct generated file for Greeter connect remote server adapter

var _ GreeterMessageAdapter = (*GreeterConnectRemoteServer)(nil)

type GreeterConnectRemoteServer struct {
	client GreeterClient
}

func NewGreeterConnectRemoteServer(client GreeterClient) (*GreeterConnectRemoteServer, error) {
	if client == nil {
		return nil, errors.New("rpccgo: connect remote client is nil")
	}
	return &GreeterConnectRemoteServer{client: client}, nil
}

func RegisterGreeterConnectRemoteServer(client GreeterClient) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	adapter, err := NewGreeterConnectRemoteServer(client)
	if err != nil {
		return rpcruntime.AdapterSnapshot[GreeterMessageAdapter]{}, err
	}
	return RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindConnectRemote, adapter)
}

func (s *GreeterConnectRemoteServer) SayHelloMessage(ctx context.Context, req []byte) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("rpccgo: connect remote client is nil")
	}
	request := new(SayHelloRequest)
	if err := proto.Unmarshal(req, request); err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)
	}
	resp, err := s.client.SayHello(ctx, request)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote response protobuf marshal failed: %w", err)
	}
	return data, nil
}
