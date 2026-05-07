package greeterv1

import (
	context "context"
	errors "errors"
	fmt "fmt"
	http "net/http"
	connect "connectrpc.com/connect"
	proto "google.golang.org/protobuf/proto"
)

// rpccgo message direct stage file for Greeter connect local server adapter

const GreeterConnectServiceName = "examples.minimal.greeter.v1.Greeter"
const GreeterConnectServicePathPrefix = "/examples.minimal.greeter.v1.Greeter/"
const GreeterSayHelloConnectProcedure = "/examples.minimal.greeter.v1.Greeter/SayHello"

func NewGreeterConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {
	greeterSayHelloConnectHandler := connect.NewUnaryHandler(GreeterSayHelloConnectProcedure, greeterConnectSayHello, options...)
	return GreeterConnectServicePathPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case GreeterSayHelloConnectProcedure:
			greeterSayHelloConnectHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func greeterConnectSayHello(ctx context.Context, req *connect.Request[SayHelloRequest]) (*connect.Response[SayHelloResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, errors.New("rpccgo: connect request is nil")
	}
	reqData, err := proto.Marshal(req.Msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect request protobuf marshal failed: %w", err)
	}
	respData, err := NewGreeterCGOMessageClientBridge().SayHello(ctx, reqData)
	if err != nil {
		return nil, err
	}
	resp := new(SayHelloResponse)
	if err := proto.Unmarshal(respData, resp); err != nil {
		return nil, fmt.Errorf("rpccgo: connect response protobuf unmarshal failed: %w", err)
	}
	return connect.NewResponse(resp), nil
}
