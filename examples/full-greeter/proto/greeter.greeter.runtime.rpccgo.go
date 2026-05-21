package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo service runtime generated file for Greeter

type GreeterNativeAdapter interface {
	SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error)
	StartCollect(ctx context.Context) (GreeterCollectNativeStreamSession, error)
	StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (GreeterBroadcastNativeStreamSession, error)
	StartChat(ctx context.Context) (GreeterChatNativeStreamSession, error)
}

type GreeterMessageAdapter interface {
	SayHelloMessage(ctx context.Context, req []byte) ([]byte, error)
	StartCollectMessage(ctx context.Context) (GreeterCollectMessageStreamSession, error)
	StartBroadcastMessage(ctx context.Context, req []byte) (GreeterBroadcastMessageStreamSession, error)
	StartChatMessage(ctx context.Context) (GreeterChatMessageStreamSession, error)
}

type GreeterActiveAdapter struct {
	Native  GreeterNativeAdapter
	Message GreeterMessageAdapter
}

type GreeterCollectNativeStreamSession interface {
	Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error
	Finish(ctx context.Context) (string, error)
	Cancel(ctx context.Context) error
}

type GreeterCollectMessageStreamSession interface {
	Send(ctx context.Context, req []byte) error
	Finish(ctx context.Context) ([]byte, error)
	Cancel(ctx context.Context) error
}

type GreeterBroadcastNativeStreamSession interface {
	Recv(ctx context.Context) (string, error)
	Done(ctx context.Context) error
	Cancel(ctx context.Context) error
}

type GreeterBroadcastMessageStreamSession interface {
	Recv(ctx context.Context) ([]byte, error)
	Done(ctx context.Context) error
	Cancel(ctx context.Context) error
}

type GreeterChatNativeStreamSession interface {
	Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error
	Recv(ctx context.Context) (string, error)
	CloseSend(ctx context.Context) error
	Done(ctx context.Context) error
	Cancel(ctx context.Context) error
}

type GreeterChatMessageStreamSession interface {
	Send(ctx context.Context, req []byte) error
	Recv(ctx context.Context) ([]byte, error)
	CloseSend(ctx context.Context) error
	Done(ctx context.Context) error
	Cancel(ctx context.Context) error
}

type GreeterCollectNativeStream struct {
	handle rpcruntime.StreamHandle
}

func NewGreeterCollectNativeStream(handle rpcruntime.StreamHandle) GreeterCollectNativeStream {
	return GreeterCollectNativeStream{handle: handle}
}

func (s GreeterCollectNativeStream) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	return rpcruntime.DispatcherStreamSend[GreeterActiveAdapter, GreeterCollectNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterCollectNativeStreamSession) error {
		return session.Send(ctx, name, city)
	})
}

func (s GreeterCollectNativeStream) Finish(ctx context.Context) (string, error) {
	var messageResult string
	err := rpcruntime.DispatcherStreamFinish[GreeterActiveAdapter, GreeterCollectNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterCollectNativeStreamSession) error {
		var callErr error
		messageResult, callErr = session.Finish(ctx)
		return callErr
	})
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (s GreeterCollectNativeStream) Cancel(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCancel[GreeterActiveAdapter, GreeterCollectNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterCollectNativeStreamSession) error {
		return session.Cancel(ctx)
	})
}

type GreeterCollectMessageStream struct {
	handle rpcruntime.StreamHandle
}

func NewGreeterCollectMessageStream(handle rpcruntime.StreamHandle) GreeterCollectMessageStream {
	return GreeterCollectMessageStream{handle: handle}
}

func (s GreeterCollectMessageStream) Send(ctx context.Context, req []byte) error {
	return rpcruntime.DispatcherStreamSend[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterCollectMessageStreamSession) error {
		return session.Send(ctx, req)
	})
}

func (s GreeterCollectMessageStream) Finish(ctx context.Context) ([]byte, error) {
	var resp []byte
	err := rpcruntime.DispatcherStreamFinish[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterCollectMessageStreamSession) error {
		var callErr error
		resp, callErr = session.Finish(ctx)
		return callErr
	})
	return resp, err
}

func (s GreeterCollectMessageStream) Cancel(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCancel[GreeterActiveAdapter, GreeterCollectMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterCollectMessageStreamSession) error {
		return session.Cancel(ctx)
	})
}

type GreeterBroadcastNativeStream struct {
	handle rpcruntime.StreamHandle
}

func NewGreeterBroadcastNativeStream(handle rpcruntime.StreamHandle) GreeterBroadcastNativeStream {
	return GreeterBroadcastNativeStream{handle: handle}
}

func (s GreeterBroadcastNativeStream) Recv(ctx context.Context) (string, error) {
	var messageResult string
	err := rpcruntime.DispatcherStreamReceive[GreeterActiveAdapter, GreeterBroadcastNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterBroadcastNativeStreamSession) error {
		var callErr error
		messageResult, callErr = session.Recv(ctx)
		return callErr
	})
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (s GreeterBroadcastNativeStream) Done(ctx context.Context) error {
	return rpcruntime.DispatcherStreamDone[GreeterActiveAdapter, GreeterBroadcastNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterBroadcastNativeStreamSession) error {
		return session.Done(ctx)
	})
}

func (s GreeterBroadcastNativeStream) Cancel(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCancel[GreeterActiveAdapter, GreeterBroadcastNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterBroadcastNativeStreamSession) error {
		return session.Cancel(ctx)
	})
}

type GreeterBroadcastMessageStream struct {
	handle rpcruntime.StreamHandle
}

func NewGreeterBroadcastMessageStream(handle rpcruntime.StreamHandle) GreeterBroadcastMessageStream {
	return GreeterBroadcastMessageStream{handle: handle}
}

func (s GreeterBroadcastMessageStream) Recv(ctx context.Context) ([]byte, error) {
	var resp []byte
	err := rpcruntime.DispatcherStreamReceive[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterBroadcastMessageStreamSession) error {
		var callErr error
		resp, callErr = session.Recv(ctx)
		return callErr
	})
	return resp, err
}

func (s GreeterBroadcastMessageStream) Done(ctx context.Context) error {
	return rpcruntime.DispatcherStreamDone[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterBroadcastMessageStreamSession) error {
		return session.Done(ctx)
	})
}

func (s GreeterBroadcastMessageStream) Cancel(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCancel[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterBroadcastMessageStreamSession) error {
		return session.Cancel(ctx)
	})
}

type GreeterChatNativeStream struct {
	handle rpcruntime.StreamHandle
}

func NewGreeterChatNativeStream(handle rpcruntime.StreamHandle) GreeterChatNativeStream {
	return GreeterChatNativeStream{handle: handle}
}

func (s GreeterChatNativeStream) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	return rpcruntime.DispatcherStreamSend[GreeterActiveAdapter, GreeterChatNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatNativeStreamSession) error {
		return session.Send(ctx, name, city)
	})
}

func (s GreeterChatNativeStream) Recv(ctx context.Context) (string, error) {
	var messageResult string
	err := rpcruntime.DispatcherStreamReceive[GreeterActiveAdapter, GreeterChatNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatNativeStreamSession) error {
		var callErr error
		messageResult, callErr = session.Recv(ctx)
		return callErr
	})
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (s GreeterChatNativeStream) CloseSend(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCloseSend[GreeterActiveAdapter, GreeterChatNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatNativeStreamSession) error {
		return session.CloseSend(ctx)
	})
}

func (s GreeterChatNativeStream) Done(ctx context.Context) error {
	return rpcruntime.DispatcherStreamDone[GreeterActiveAdapter, GreeterChatNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatNativeStreamSession) error {
		return session.Done(ctx)
	})
}

func (s GreeterChatNativeStream) Cancel(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCancel[GreeterActiveAdapter, GreeterChatNativeStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatNativeStreamSession) error {
		return session.Cancel(ctx)
	})
}

type GreeterChatMessageStream struct {
	handle rpcruntime.StreamHandle
}

func NewGreeterChatMessageStream(handle rpcruntime.StreamHandle) GreeterChatMessageStream {
	return GreeterChatMessageStream{handle: handle}
}

func (s GreeterChatMessageStream) Send(ctx context.Context, req []byte) error {
	return rpcruntime.DispatcherStreamSend[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatMessageStreamSession) error {
		return session.Send(ctx, req)
	})
}

func (s GreeterChatMessageStream) Recv(ctx context.Context) ([]byte, error) {
	var resp []byte
	err := rpcruntime.DispatcherStreamReceive[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatMessageStreamSession) error {
		var callErr error
		resp, callErr = session.Recv(ctx)
		return callErr
	})
	return resp, err
}

func (s GreeterChatMessageStream) CloseSend(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCloseSend[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatMessageStreamSession) error {
		return session.CloseSend(ctx)
	})
}

func (s GreeterChatMessageStream) Done(ctx context.Context) error {
	return rpcruntime.DispatcherStreamDone[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatMessageStreamSession) error {
		return session.Done(ctx)
	})
}

func (s GreeterChatMessageStream) Cancel(ctx context.Context) error {
	return rpcruntime.DispatcherStreamCancel[GreeterActiveAdapter, GreeterChatMessageStreamSession](GreeterDispatcherForRuntime(), s.handle, func(session GreeterChatMessageStreamSession) error {
		return session.Cancel(ctx)
	})
}

var greeterDispatcher rpcruntime.Dispatcher[GreeterActiveAdapter]
var greeterRouter = greeterActiveRouter{dispatcher: &greeterDispatcher}
var GreeterNativeMessageConverterUnavailableErr = errors.New("rpccgo: native/message converter is not enabled")
var GreeterNativeAdapterUnavailableErr = errors.New("rpccgo: native adapter is unavailable")
var GreeterMessageAdapterUnavailableErr = errors.New("rpccgo: message adapter is unavailable")
var GreeterUnknownActiveContractErr = errors.New("rpccgo: unknown active server contract")

func GreeterDispatcherForRuntime() *rpcruntime.Dispatcher[GreeterActiveAdapter] {
	return &greeterDispatcher
}

func registerGreeterActiveServer(kind rpcruntime.ServerKind, adapter GreeterNativeAdapter) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	snapshot, err := greeterDispatcher.Register(kind, rpcruntime.ServerContractNative, GreeterActiveAdapter{Native: adapter})
	if err != nil {
		return rpcruntime.AdapterSnapshot[GreeterNativeAdapter]{}, err
	}
	return rpcruntime.AdapterSnapshot[GreeterNativeAdapter]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil
}

func registerGreeterMessageActiveServer(kind rpcruntime.ServerKind, adapter GreeterMessageAdapter) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	snapshot, err := greeterDispatcher.Register(kind, rpcruntime.ServerContractMessage, GreeterActiveAdapter{Message: adapter})
	if err != nil {
		return rpcruntime.AdapterSnapshot[GreeterMessageAdapter]{}, err
	}
	return rpcruntime.AdapterSnapshot[GreeterMessageAdapter]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: adapter}, nil
}

type greeterActiveRouter struct {
	dispatcher *rpcruntime.Dispatcher[GreeterActiveAdapter]
}

func (r greeterActiveRouter) invokeNativeSayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	var messageResult string
	err := r.dispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) error {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return GreeterNativeAdapterUnavailableErr
			}
			var callErr error
			messageResult, callErr = snapshot.Adapter.Native.SayHello(ctx, name, city)
			return callErr
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return GreeterMessageAdapterUnavailableErr
			}
			messageReq, err := convertGreeterSayHelloNativeToMessageRequest(name, city)
			if err != nil {
				return err
			}
			messageResp, err := snapshot.Adapter.Message.SayHelloMessage(ctx, messageReq)
			if err != nil {
				return err
			}
			var callErr error
			messageResult, callErr = convertGreeterSayHelloMessageToNativeResponse(messageResp)
			return callErr
		default:
			return GreeterUnknownActiveContractErr
		}
	})
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (r greeterActiveRouter) invokeMessageSayHello(ctx context.Context, req []byte) ([]byte, error) {
	var resp []byte
	err := r.dispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) error {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return GreeterMessageAdapterUnavailableErr
			}
			var callErr error
			resp, callErr = snapshot.Adapter.Message.SayHelloMessage(ctx, req)
			return callErr
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return GreeterNativeAdapterUnavailableErr
			}
			return withGreeterSayHelloMessageToNativeRequest(req, func(name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
				messageResult, err := snapshot.Adapter.Native.SayHello(ctx, name, city)
				if err != nil {
					return err
				}
				messageResp, err := convertGreeterSayHelloNativeToMessageResponse(messageResult)
				if err != nil {
					return err
				}
				resp = messageResp
				return nil
			})
		default:
			return GreeterUnknownActiveContractErr
		}
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (r greeterActiveRouter) startNativeCollect(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, GreeterNativeAdapterUnavailableErr
			}
			return snapshot.Adapter.Native.StartCollect(ctx)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, GreeterMessageAdapterUnavailableErr
			}
			messageSession, err := snapshot.Adapter.Message.StartCollectMessage(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterCollectMessageToNativeStreamSession{message: messageSession}, nil
		default:
			return nil, GreeterUnknownActiveContractErr
		}
	})
}

type greeterCollectMessageToNativeStreamSession struct {
	message GreeterCollectMessageStreamSession
}

func (s *greeterCollectMessageToNativeStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	messageReq, err := convertGreeterCollectNativeToMessageRequest(name, city)
	if err != nil {
		return err
	}
	return s.message.Send(ctx, messageReq)
}

func (s *greeterCollectMessageToNativeStreamSession) Finish(ctx context.Context) (string, error) {
	messageResp, err := s.message.Finish(ctx)
	if err != nil {
		return "", err
	}
	return convertGreeterCollectMessageToNativeResponse(messageResp)
}

func (s *greeterCollectMessageToNativeStreamSession) Cancel(ctx context.Context) error {
	return s.message.Cancel(ctx)
}

func (r greeterActiveRouter) startMessageCollect(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, GreeterMessageAdapterUnavailableErr
			}
			return snapshot.Adapter.Message.StartCollectMessage(ctx)
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, GreeterNativeAdapterUnavailableErr
			}
			nativeSession, err := snapshot.Adapter.Native.StartCollect(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterCollectNativeToMessageStreamSession{native: nativeSession}, nil
		default:
			return nil, GreeterUnknownActiveContractErr
		}
	})
}

type greeterCollectNativeToMessageStreamSession struct {
	native GreeterCollectNativeStreamSession
}

func (s *greeterCollectNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {
	return withGreeterCollectMessageToNativeRequest(req, func(name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
		return s.native.Send(ctx, name, city)
	})
}

func (s *greeterCollectNativeToMessageStreamSession) Finish(ctx context.Context) ([]byte, error) {
	messageResult, err := s.native.Finish(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterCollectNativeToMessageResponse(messageResult)
}

func (s *greeterCollectNativeToMessageStreamSession) Cancel(ctx context.Context) error {
	return s.native.Cancel(ctx)
}

func (r greeterActiveRouter) startNativeBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (rpcruntime.StreamHandle, error) {
	return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, GreeterNativeAdapterUnavailableErr
			}
			return snapshot.Adapter.Native.StartBroadcast(ctx, name, city)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, GreeterMessageAdapterUnavailableErr
			}
			messageReq, err := convertGreeterBroadcastNativeToMessageRequest(name, city)
			if err != nil {
				return nil, err
			}
			messageSession, err := snapshot.Adapter.Message.StartBroadcastMessage(ctx, messageReq)
			if err != nil {
				return nil, err
			}
			return &greeterBroadcastMessageToNativeStreamSession{message: messageSession}, nil
		default:
			return nil, GreeterUnknownActiveContractErr
		}
	})
}

type greeterBroadcastMessageToNativeStreamSession struct {
	message GreeterBroadcastMessageStreamSession
}

func (s *greeterBroadcastMessageToNativeStreamSession) Recv(ctx context.Context) (string, error) {
	messageResp, err := s.message.Recv(ctx)
	if err != nil {
		return "", err
	}
	return convertGreeterBroadcastMessageToNativeResponse(messageResp)
}

func (s *greeterBroadcastMessageToNativeStreamSession) Done(ctx context.Context) error {
	return s.message.Done(ctx)
}

func (s *greeterBroadcastMessageToNativeStreamSession) Cancel(ctx context.Context) error {
	return s.message.Cancel(ctx)
}

func (r greeterActiveRouter) startMessageBroadcast(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {
	return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, GreeterMessageAdapterUnavailableErr
			}
			return snapshot.Adapter.Message.StartBroadcastMessage(ctx, req)
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, GreeterNativeAdapterUnavailableErr
			}
			var session any
			err := withGreeterBroadcastMessageToNativeRequest(req, func(name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
				nativeSession, err := snapshot.Adapter.Native.StartBroadcast(ctx, name, city)
				if err != nil {
					return err
				}
				session = &greeterBroadcastNativeToMessageStreamSession{native: nativeSession}
				return nil
			})
			if err != nil {
				return nil, err
			}
			return session, nil
		default:
			return nil, GreeterUnknownActiveContractErr
		}
	})
}

type greeterBroadcastNativeToMessageStreamSession struct {
	native GreeterBroadcastNativeStreamSession
}

func (s *greeterBroadcastNativeToMessageStreamSession) Recv(ctx context.Context) ([]byte, error) {
	messageResult, err := s.native.Recv(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterBroadcastNativeToMessageResponse(messageResult)
}

func (s *greeterBroadcastNativeToMessageStreamSession) Done(ctx context.Context) error {
	return s.native.Done(ctx)
}

func (s *greeterBroadcastNativeToMessageStreamSession) Cancel(ctx context.Context) error {
	return s.native.Cancel(ctx)
}

func (r greeterActiveRouter) startNativeChat(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, GreeterNativeAdapterUnavailableErr
			}
			return snapshot.Adapter.Native.StartChat(ctx)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, GreeterMessageAdapterUnavailableErr
			}
			messageSession, err := snapshot.Adapter.Message.StartChatMessage(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterChatMessageToNativeStreamSession{message: messageSession}, nil
		default:
			return nil, GreeterUnknownActiveContractErr
		}
	})
}

type greeterChatMessageToNativeStreamSession struct {
	message GreeterChatMessageStreamSession
}

func (s *greeterChatMessageToNativeStreamSession) Send(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
	messageReq, err := convertGreeterChatNativeToMessageRequest(name, city)
	if err != nil {
		return err
	}
	return s.message.Send(ctx, messageReq)
}

func (s *greeterChatMessageToNativeStreamSession) Recv(ctx context.Context) (string, error) {
	messageResp, err := s.message.Recv(ctx)
	if err != nil {
		return "", err
	}
	return convertGreeterChatMessageToNativeResponse(messageResp)
}

func (s *greeterChatMessageToNativeStreamSession) CloseSend(ctx context.Context) error {
	return s.message.CloseSend(ctx)
}

func (s *greeterChatMessageToNativeStreamSession) Done(ctx context.Context) error {
	return s.message.Done(ctx)
}

func (s *greeterChatMessageToNativeStreamSession) Cancel(ctx context.Context) error {
	return s.message.Cancel(ctx)
}

func (r greeterActiveRouter) startMessageChat(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return r.dispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, GreeterMessageAdapterUnavailableErr
			}
			return snapshot.Adapter.Message.StartChatMessage(ctx)
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, GreeterNativeAdapterUnavailableErr
			}
			nativeSession, err := snapshot.Adapter.Native.StartChat(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterChatNativeToMessageStreamSession{native: nativeSession}, nil
		default:
			return nil, GreeterUnknownActiveContractErr
		}
	})
}

type greeterChatNativeToMessageStreamSession struct {
	native GreeterChatNativeStreamSession
}

func (s *greeterChatNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {
	return withGreeterChatMessageToNativeRequest(req, func(name *rpcruntime.RpcString, city *rpcruntime.RpcString) error {
		return s.native.Send(ctx, name, city)
	})
}

func (s *greeterChatNativeToMessageStreamSession) Recv(ctx context.Context) ([]byte, error) {
	messageResult, err := s.native.Recv(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterChatNativeToMessageResponse(messageResult)
}

func (s *greeterChatNativeToMessageStreamSession) CloseSend(ctx context.Context) error {
	return s.native.CloseSend(ctx)
}

func (s *greeterChatNativeToMessageStreamSession) Done(ctx context.Context) error {
	return s.native.Done(ctx)
}

func (s *greeterChatNativeToMessageStreamSession) Cancel(ctx context.Context) error {
	return s.native.Cancel(ctx)
}

type GreeterCGONativeClientBridge struct{}

func (GreeterCGONativeClientBridge) SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	return greeterRouter.invokeNativeSayHello(ctx, name, city)
}

func (GreeterCGONativeClientBridge) StartCollect(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return greeterRouter.startNativeCollect(ctx)
}

func (GreeterCGONativeClientBridge) StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (rpcruntime.StreamHandle, error) {
	return greeterRouter.startNativeBroadcast(ctx, name, city)
}

func (GreeterCGONativeClientBridge) StartChat(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return greeterRouter.startNativeChat(ctx)
}

func NewGreeterCGONativeClientBridge() GreeterCGONativeClientBridge {
	return GreeterCGONativeClientBridge{}
}

func RegisterGreeterCGONativeActiveServer(kind rpcruntime.ServerKind, adapter GreeterNativeAdapter) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	return registerGreeterActiveServer(kind, adapter)
}

type GreeterCGOMessageClientBridge struct{}

func (GreeterCGOMessageClientBridge) SayHello(ctx context.Context, req []byte) ([]byte, error) {
	return greeterRouter.invokeMessageSayHello(ctx, req)
}

func (GreeterCGOMessageClientBridge) StartCollect(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return greeterRouter.startMessageCollect(ctx)
}

func (GreeterCGOMessageClientBridge) StartBroadcast(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {
	return greeterRouter.startMessageBroadcast(ctx, req)
}

func (GreeterCGOMessageClientBridge) StartChat(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return greeterRouter.startMessageChat(ctx)
}

func NewGreeterCGOMessageClientBridge() GreeterCGOMessageClientBridge {
	return GreeterCGOMessageClientBridge{}
}

func RegisterGreeterCGOMessageActiveServer(kind rpcruntime.ServerKind, adapter GreeterMessageAdapter) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	return registerGreeterMessageActiveServer(kind, adapter)
}
