package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo service runtime stage file for Greeter

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
	Cancel(ctx context.Context) error
}

type GreeterChatMessageStreamSession interface {
	Send(ctx context.Context, req []byte) error
	Recv(ctx context.Context) ([]byte, error)
	CloseSend(ctx context.Context) error
	Done(ctx context.Context) error
	Cancel(ctx context.Context) error
}

var greeterDispatcher rpcruntime.Dispatcher[GreeterActiveAdapter]
var greeterNativeContractMismatchErr = errors.New("rpccgo: native contract mismatch: active server is message and native/message converter is not enabled")
var greeterMessageContractMismatchErr = errors.New("rpccgo: message contract mismatch: active server is native and native/message converter is not enabled")

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

type GreeterCGONativeClientBridge struct{}

func (GreeterCGONativeClientBridge) SayHello(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (string, error) {
	var messageResult string
	err := greeterDispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) error {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return greeterNativeContractMismatchErr
			}
			var callErr error
			messageResult, callErr = snapshot.Adapter.Native.SayHello(ctx, name, city)
			return callErr
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return greeterNativeContractMismatchErr
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
			return greeterNativeContractMismatchErr
		}
	})
	if err != nil {
		return "", err
	}
	return messageResult, nil
}

func (GreeterCGONativeClientBridge) StartCollect(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterNativeContractMismatchErr
			}
			return snapshot.Adapter.Native.StartCollect(ctx)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterNativeContractMismatchErr
			}
			messageSession, err := snapshot.Adapter.Message.StartCollectMessage(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterCollectMessageToNativeStreamSession{message: messageSession}, nil
		default:
			return nil, greeterNativeContractMismatchErr
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

func (GreeterCGONativeClientBridge) StartBroadcast(ctx context.Context, name *rpcruntime.RpcString, city *rpcruntime.RpcString) (rpcruntime.StreamHandle, error) {
	return greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterNativeContractMismatchErr
			}
			return snapshot.Adapter.Native.StartBroadcast(ctx, name, city)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterNativeContractMismatchErr
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
			return nil, greeterNativeContractMismatchErr
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

func (GreeterCGONativeClientBridge) StartChat(ctx context.Context) (rpcruntime.StreamHandle, error) {
	return greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterNativeContractMismatchErr
			}
			return snapshot.Adapter.Native.StartChat(ctx)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterNativeContractMismatchErr
			}
			messageSession, err := snapshot.Adapter.Message.StartChatMessage(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterChatMessageToNativeStreamSession{message: messageSession}, nil
		default:
			return nil, greeterNativeContractMismatchErr
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

func NewGreeterCGONativeClientBridge() GreeterCGONativeClientBridge {
	return GreeterCGONativeClientBridge{}
}

func RegisterGreeterCGONativeActiveServer(kind rpcruntime.ServerKind, adapter GreeterNativeAdapter) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {
	return registerGreeterActiveServer(kind, adapter)
}

type GreeterCGOMessageClientBridge struct{}

func (GreeterCGOMessageClientBridge) SayHello(ctx context.Context, req []byte) ([]byte, error) {
	var resp []byte
	err := greeterDispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) error {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return greeterMessageContractMismatchErr
			}
			var callErr error
			resp, callErr = snapshot.Adapter.Message.SayHelloMessage(ctx, req)
			return callErr
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return greeterMessageContractMismatchErr
			}
			name, city, err := convertGreeterSayHelloMessageToNativeRequest(req)
			if err != nil {
				return err
			}
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
		default:
			return greeterMessageContractMismatchErr
		}
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (GreeterCGOMessageClientBridge) StartCollect(ctx context.Context) (rpcruntime.StreamHandle, error) {
	handle, err := greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterMessageContractMismatchErr
			}
			return snapshot.Adapter.Message.StartCollectMessage(ctx)
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterMessageContractMismatchErr
			}
			nativeSession, err := snapshot.Adapter.Native.StartCollect(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterCollectNativeToMessageStreamSession{native: nativeSession}, nil
		default:
			return nil, greeterMessageContractMismatchErr
		}
	})
	if err != nil {
		return 0, err
	}
	return handle, nil
}

type greeterCollectNativeToMessageStreamSession struct {
	native GreeterCollectNativeStreamSession
}

func (s *greeterCollectNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {
	name, city, err := convertGreeterCollectMessageToNativeRequest(req)
	if err != nil {
		return err
	}
	return s.native.Send(ctx, name, city)
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

func (GreeterCGOMessageClientBridge) StartBroadcast(ctx context.Context, req []byte) (rpcruntime.StreamHandle, error) {
	handle, err := greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterMessageContractMismatchErr
			}
			return snapshot.Adapter.Message.StartBroadcastMessage(ctx, req)
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterMessageContractMismatchErr
			}
			name, city, err := convertGreeterBroadcastMessageToNativeRequest(req)
			if err != nil {
				return nil, err
			}
			nativeSession, err := snapshot.Adapter.Native.StartBroadcast(ctx, name, city)
			if err != nil {
				return nil, err
			}
			return &greeterBroadcastNativeToMessageStreamSession{native: nativeSession}, nil
		default:
			return nil, greeterMessageContractMismatchErr
		}
	})
	if err != nil {
		return 0, err
	}
	return handle, nil
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
	if done, ok := s.native.(interface{ Done(context.Context) error }); ok {
		return done.Done(ctx)
	}
	return nil
}

func (s *greeterBroadcastNativeToMessageStreamSession) Cancel(ctx context.Context) error {
	return s.native.Cancel(ctx)
}

func (GreeterCGOMessageClientBridge) StartChat(ctx context.Context) (rpcruntime.StreamHandle, error) {
	handle, err := greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterMessageContractMismatchErr
			}
			return snapshot.Adapter.Message.StartChatMessage(ctx)
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterMessageContractMismatchErr
			}
			nativeSession, err := snapshot.Adapter.Native.StartChat(ctx)
			if err != nil {
				return nil, err
			}
			return &greeterChatNativeToMessageStreamSession{native: nativeSession}, nil
		default:
			return nil, greeterMessageContractMismatchErr
		}
	})
	if err != nil {
		return 0, err
	}
	return handle, nil
}

type greeterChatNativeToMessageStreamSession struct {
	native GreeterChatNativeStreamSession
}

func (s *greeterChatNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {
	name, city, err := convertGreeterChatMessageToNativeRequest(req)
	if err != nil {
		return err
	}
	return s.native.Send(ctx, name, city)
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
	if done, ok := s.native.(interface{ Done(context.Context) error }); ok {
		return done.Done(ctx)
	}
	return nil
}

func (s *greeterChatNativeToMessageStreamSession) Cancel(ctx context.Context) error {
	return s.native.Cancel(ctx)
}

func NewGreeterCGOMessageClientBridge() GreeterCGOMessageClientBridge {
	return GreeterCGOMessageClientBridge{}
}

func RegisterGreeterCGOMessageActiveServer(kind rpcruntime.ServerKind, adapter GreeterMessageAdapter) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	return registerGreeterMessageActiveServer(kind, adapter)
}
