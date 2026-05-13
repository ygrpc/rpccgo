package greeterv1

import (
	context "context"
	errors "errors"
	fmt "fmt"
	io "io"
	rpcruntime "rpccgo/rpcruntime"
	sync "sync"
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
	session, ok := rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return nil, errors.New("rpccgo: connect message stream handle is invalid")
	}
	for stream.Receive() {
		reqData, err := proto.Marshal(stream.Msg())
		if err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(ctx)
			}
			return nil, fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)
		}
		if err := session.Send(ctx, reqData); err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(ctx)
			}
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
			_ = terminal.Cancel(ctx)
		}
		return nil, err
	}
	terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
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
	session, ok := rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return errors.New("rpccgo: connect message stream handle is invalid")
	}
	for {
		respData, err := session.Recv(ctx)
		if err != nil {
			terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
			if !ok {
				return errors.New("rpccgo: connect message stream handle is invalid")
			}
			if errors.Is(err, io.EOF) {
				return terminal.Done(ctx)
			}
			_ = terminal.Cancel(ctx)
			return err
		}
		resp := new(SayHelloResponse)
		if err := proto.Unmarshal(respData, resp); err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(ctx)
			}
			return fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)
		}
		if err := stream.Send(resp); err != nil {
			if terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle)); ok {
				_ = terminal.Cancel(ctx)
			}
			return err
		}
	}
}

func greeterConnectChat(ctx context.Context, stream *connect.BidiStream[SayHelloRequest, SayHelloResponse]) error {
	bridge := NewGreeterCGOMessageClientBridge()
	handle, err := bridge.StartChat(ctx)
	if err != nil {
		return err
	}
	session, ok := rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
	if !ok {
		return errors.New("rpccgo: connect message stream handle is invalid")
	}
	var terminalOnce sync.Once
	finish := func(done bool) error {
		var finishErr error
		terminalOnce.Do(func() {
			terminal, ok := rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), rpcruntime.StreamHandle(handle))
			if !ok {
				finishErr = errors.New("rpccgo: connect message stream handle is invalid")
				return
			}
			if done {
				finishErr = terminal.Done(ctx)
				return
			}
			finishErr = terminal.Cancel(ctx)
		})
		return finishErr
	}
	receiveErrCh := make(chan error, 1)
	sendErrCh := make(chan error, 1)
	go func() {
		for {
			req, err := stream.Receive()
			if err != nil {
				if errors.Is(err, io.EOF) {
					receiveErrCh <- session.CloseSend(ctx)
					return
				}
				receiveErrCh <- err
				return
			}
			reqData, err := proto.Marshal(req)
			if err != nil {
				receiveErrCh <- fmt.Errorf("rpccgo: connect bidi request protobuf marshal failed: %w", err)
				return
			}
			if err := session.Send(ctx, reqData); err != nil {
				receiveErrCh <- err
				return
			}
		}
	}()
	go func() {
		for {
			respData, err := session.Recv(ctx)
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
				sendErrCh <- fmt.Errorf("rpccgo: connect bidi response protobuf unmarshal failed: %w", err)
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
