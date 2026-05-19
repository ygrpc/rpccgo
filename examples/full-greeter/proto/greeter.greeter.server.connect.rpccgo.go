package greeterv1

import (
	context "context"
	errors "errors"
	fmt "fmt"
	rpcruntime "rpccgo/rpcruntime"
	http "net/http"
	connect "connectrpc.com/connect"
	proto "google.golang.org/protobuf/proto"
)

// rpccgo message direct generated file for Greeter connect local server adapter

const GreeterConnectServiceName = "examples.full.greeter.v1.Greeter"
const GreeterConnectServicePathPrefix = "/examples.full.greeter.v1.Greeter/"
const GreeterSayHelloConnectProcedure = "/examples.full.greeter.v1.Greeter/SayHello"
const GreeterCollectConnectProcedure = "/examples.full.greeter.v1.Greeter/Collect"
const GreeterBroadcastConnectProcedure = "/examples.full.greeter.v1.Greeter/Broadcast"
const GreeterChatConnectProcedure = "/examples.full.greeter.v1.Greeter/Chat"

func NewGreeterConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {
	greeterSayHelloConnectHandler := connect.NewUnaryHandler(GreeterSayHelloConnectProcedure, greeterConnectSayHello, options...)
	greeterCollectConnectHandler := connect.NewClientStreamHandler(GreeterCollectConnectProcedure, greeterConnectCollect, options...)
	greeterBroadcastConnectHandler := connect.NewServerStreamHandler(GreeterBroadcastConnectProcedure, greeterConnectBroadcast, options...)
	greeterChatConnectHandler := connect.NewBidiStreamHandler(GreeterChatConnectProcedure, greeterConnectChat, options...)
	return GreeterConnectServicePathPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case GreeterSayHelloConnectProcedure:
			greeterSayHelloConnectHandler.ServeHTTP(w, r)
		case GreeterCollectConnectProcedure:
			greeterCollectConnectHandler.ServeHTTP(w, r)
		case GreeterBroadcastConnectProcedure:
			greeterBroadcastConnectHandler.ServeHTTP(w, r)
		case GreeterChatConnectProcedure:
			greeterChatConnectHandler.ServeHTTP(w, r)
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

func greeterConnectCollect(ctx context.Context, stream *connect.ClientStream[SayHelloRequest]) (*connect.Response[SayHelloResponse], error) {
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartCollect(ctx)
	if err != nil {
		return nil, err
	}
	session, err := rpcruntime.RequireDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"))
	if err != nil {
		return nil, errors.New("rpccgo: connect message stream handle is invalid")
	}
	for stream.Receive() {
		reqData, err := proto.Marshal(stream.Msg())
		if err != nil {
			_ = rpcruntime.EndDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"), func(terminal GreeterCollectMessageStreamSession) error {
				return terminal.Cancel(ctx)
			})
			return nil, fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)
		}
		if err := session.Send(ctx, reqData); err != nil {
			_ = rpcruntime.EndDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"), func(terminal GreeterCollectMessageStreamSession) error {
				return terminal.Cancel(ctx)
			})
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		_ = rpcruntime.EndDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"), func(terminal GreeterCollectMessageStreamSession) error {
			return terminal.Cancel(ctx)
		})
		return nil, err
	}
	terminal, err := rpcruntime.TakeRequiredDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"))
	if err != nil {
		return nil, errors.New("rpccgo: connect message stream handle is invalid")
	}
	respData, err := terminal.Finish(ctx)
	if err != nil {
		return nil, err
	}
	resp := new(SayHelloResponse)
	if err := proto.Unmarshal(respData, resp); err != nil {
		return nil, fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)
	}
	return connect.NewResponse(resp), nil
}

func greeterConnectBroadcast(ctx context.Context, req *connect.Request[SayHelloRequest], stream *connect.ServerStream[SayHelloResponse]) error {
	if req == nil || req.Msg == nil {
		return errors.New("rpccgo: connect request is nil")
	}
	reqData, err := proto.Marshal(req.Msg)
	if err != nil {
		return fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)
	}
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartBroadcast(ctx, reqData)
	if err != nil {
		return err
	}
	session, err := rpcruntime.RequireDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"))
	if err != nil {
		return errors.New("rpccgo: connect message stream handle is invalid")
	}
	terminal := rpcruntime.NewDispatcherStreamTerminal[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: connect message stream handle is invalid"))
	return rpcruntime.RunServerStream(
		func() ([]byte, error) {
			return session.Recv(ctx)
		},
		func(respData []byte) error {
			resp := new(SayHelloResponse)
			if err := proto.Unmarshal(respData, resp); err != nil {
				return fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)
			}
			return stream.Send(resp)
		},
		func() error {
			return terminal.End(func(session GreeterBroadcastMessageStreamSession) error { return session.Done(ctx) })
		},
		func() error {
			return terminal.End(func(session GreeterBroadcastMessageStreamSession) error { return session.Cancel(ctx) })
		},
	)
}

func greeterConnectChat(ctx context.Context, stream *connect.BidiStream[SayHelloRequest, SayHelloResponse]) error {
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartChat(ctx)
	if err != nil {
		return err
	}
	session, err := rpcruntime.RequireDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: stream handle is invalid"))
	if err != nil {
		return errors.New("rpccgo: connect message stream handle is invalid")
	}
	terminal := rpcruntime.NewDispatcherStreamTerminal[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle), errors.New("rpccgo: connect message stream handle is invalid"))
	return rpcruntime.RunBidiStream(
		func() (*SayHelloRequest, error) {
			return stream.Receive()
		},
		func(req *SayHelloRequest) error {
			reqData, err := proto.Marshal(req)
			if err != nil {
				return fmt.Errorf("rpccgo: connect bidi request protobuf marshal failed: %w", err)
			}
			return session.Send(ctx, reqData)
		},
		func() error {
			return session.CloseSend(ctx)
		},
		func() ([]byte, error) {
			return session.Recv(ctx)
		},
		func(respData []byte) error {
			resp := new(SayHelloResponse)
			if err := proto.Unmarshal(respData, resp); err != nil {
				return fmt.Errorf("rpccgo: connect bidi response protobuf unmarshal failed: %w", err)
			}
			return stream.Send(resp)
		},
		func() error {
			return terminal.End(func(session GreeterChatMessageStreamSession) error { return session.Done(ctx) })
		},
		func() error {
			return terminal.End(func(session GreeterChatMessageStreamSession) error { return session.Cancel(ctx) })
		},
	)
}
