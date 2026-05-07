package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo service runtime stage file for Greeter

type GreeterNativeAdapter interface {
	SayHello(ctx context.Context, req *SayHelloRequest) (*SayHelloResponse, error)
	StartCollect(ctx context.Context) (GreeterCollectNativeStreamSession, error)
	StartBroadcast(ctx context.Context, req *SayHelloRequest) (GreeterBroadcastNativeStreamSession, error)
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
	Send(ctx context.Context, req *SayHelloRequest) error
	Finish(ctx context.Context) (*SayHelloResponse, error)
	Cancel(ctx context.Context) error
}

type GreeterCollectMessageStreamSession interface {
	Send(ctx context.Context, req []byte) error
	Finish(ctx context.Context) ([]byte, error)
	Cancel(ctx context.Context) error
}

type GreeterBroadcastNativeStreamSession interface {
	Recv(ctx context.Context) (*SayHelloResponse, error)
	Cancel(ctx context.Context) error
}

type GreeterBroadcastMessageStreamSession interface {
	Recv(ctx context.Context) ([]byte, error)
	Done(ctx context.Context) error
	Cancel(ctx context.Context) error
}

type GreeterChatNativeStreamSession interface {
	Send(ctx context.Context, req *SayHelloRequest) error
	Recv(ctx context.Context) (*SayHelloResponse, error)
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

func loadGreeterCollectNativeStream(handle rpcruntime.StreamHandle) (GreeterCollectNativeStreamSession, bool) {
	return rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterCollectNativeStreamSession](&greeterDispatcher, handle)
}

func takeGreeterCollectNativeStream(handle rpcruntime.StreamHandle) (GreeterCollectNativeStreamSession, bool) {
	return rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectNativeStreamSession](&greeterDispatcher, handle)
}

func deleteGreeterCollectNativeStream(handle rpcruntime.StreamHandle) bool {
	return rpcruntime.DeleteDispatcherStream[GreeterActiveAdapter](&greeterDispatcher, handle)
}

func loadGreeterCollectMessageStream(handle rpcruntime.StreamHandle) (GreeterCollectMessageStreamSession, bool) {
	return rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](&greeterDispatcher, handle)
}

func takeGreeterCollectMessageStream(handle rpcruntime.StreamHandle) (GreeterCollectMessageStreamSession, bool) {
	return rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterCollectMessageStreamSession](&greeterDispatcher, handle)
}

func deleteGreeterCollectMessageStream(handle rpcruntime.StreamHandle) bool {
	return rpcruntime.DeleteDispatcherStream[GreeterActiveAdapter](&greeterDispatcher, handle)
}

func loadGreeterBroadcastNativeStream(handle rpcruntime.StreamHandle) (GreeterBroadcastNativeStreamSession, bool) {
	return rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterBroadcastNativeStreamSession](&greeterDispatcher, handle)
}

func takeGreeterBroadcastNativeStream(handle rpcruntime.StreamHandle) (GreeterBroadcastNativeStreamSession, bool) {
	return rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastNativeStreamSession](&greeterDispatcher, handle)
}

func deleteGreeterBroadcastNativeStream(handle rpcruntime.StreamHandle) bool {
	return rpcruntime.DeleteDispatcherStream[GreeterActiveAdapter](&greeterDispatcher, handle)
}

func loadGreeterBroadcastMessageStream(handle rpcruntime.StreamHandle) (GreeterBroadcastMessageStreamSession, bool) {
	return rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](&greeterDispatcher, handle)
}

func takeGreeterBroadcastMessageStream(handle rpcruntime.StreamHandle) (GreeterBroadcastMessageStreamSession, bool) {
	return rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterBroadcastMessageStreamSession](&greeterDispatcher, handle)
}

func deleteGreeterBroadcastMessageStream(handle rpcruntime.StreamHandle) bool {
	return rpcruntime.DeleteDispatcherStream[GreeterActiveAdapter](&greeterDispatcher, handle)
}

func loadGreeterChatNativeStream(handle rpcruntime.StreamHandle) (GreeterChatNativeStreamSession, bool) {
	return rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterChatNativeStreamSession](&greeterDispatcher, handle)
}

func takeGreeterChatNativeStream(handle rpcruntime.StreamHandle) (GreeterChatNativeStreamSession, bool) {
	return rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterChatNativeStreamSession](&greeterDispatcher, handle)
}

func deleteGreeterChatNativeStream(handle rpcruntime.StreamHandle) bool {
	return rpcruntime.DeleteDispatcherStream[GreeterActiveAdapter](&greeterDispatcher, handle)
}

func loadGreeterChatMessageStream(handle rpcruntime.StreamHandle) (GreeterChatMessageStreamSession, bool) {
	return rpcruntime.LoadDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](&greeterDispatcher, handle)
}

func takeGreeterChatMessageStream(handle rpcruntime.StreamHandle) (GreeterChatMessageStreamSession, bool) {
	return rpcruntime.TakeDispatcherStream[GreeterActiveAdapter, GreeterChatMessageStreamSession](&greeterDispatcher, handle)
}

func deleteGreeterChatMessageStream(handle rpcruntime.StreamHandle) bool {
	return rpcruntime.DeleteDispatcherStream[GreeterActiveAdapter](&greeterDispatcher, handle)
}

type GreeterCGONativeClientBridge struct{}

func (GreeterCGONativeClientBridge) SayHello(ctx context.Context, req *SayHelloRequest) (*SayHelloResponse, error) {
	var resp *SayHelloResponse
	err := greeterDispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) error {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return greeterNativeContractMismatchErr
			}
			var callErr error
			resp, callErr = snapshot.Adapter.Native.SayHello(ctx, req)
			return callErr
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return greeterNativeContractMismatchErr
			}
			messageReq, err := convertGreeterSayHelloNativeToMessageRequest(req)
			if err != nil {
				return err
			}
			messageResp, err := snapshot.Adapter.Message.SayHelloMessage(ctx, messageReq)
			if err != nil {
				return err
			}
			nativeResp, err := convertGreeterSayHelloMessageToNativeResponse(messageResp)
			if err != nil {
				return err
			}
			resp = nativeResp
			return nil
		default:
			return greeterNativeContractMismatchErr
		}
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
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

func (GreeterCGONativeClientBridge) LoadCollectNativeStream(handle rpcruntime.StreamHandle) (GreeterCollectNativeStreamSession, bool) {
	return loadGreeterCollectNativeStream(handle)
}

func (GreeterCGONativeClientBridge) TakeCollectNativeStream(handle rpcruntime.StreamHandle) (GreeterCollectNativeStreamSession, bool) {
	return takeGreeterCollectNativeStream(handle)
}

type greeterCollectMessageToNativeStreamSession struct {
	message GreeterCollectMessageStreamSession
}

func (s *greeterCollectMessageToNativeStreamSession) Send(ctx context.Context, req *SayHelloRequest) error {
	messageReq, err := convertGreeterCollectNativeToMessageRequest(req)
	if err != nil {
		return err
	}
	return s.message.Send(ctx, messageReq)
}

func (s *greeterCollectMessageToNativeStreamSession) Finish(ctx context.Context) (*SayHelloResponse, error) {
	messageResp, err := s.message.Finish(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterCollectMessageToNativeResponse(messageResp)
}

func (s *greeterCollectMessageToNativeStreamSession) Cancel(ctx context.Context) error {
	return s.message.Cancel(ctx)
}

func (GreeterCGONativeClientBridge) StartBroadcast(ctx context.Context, req *SayHelloRequest) (rpcruntime.StreamHandle, error) {
	return greeterDispatcher.StartStream(func(snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) (any, error) {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return nil, greeterNativeContractMismatchErr
			}
			return snapshot.Adapter.Native.StartBroadcast(ctx, req)
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return nil, greeterNativeContractMismatchErr
			}
			messageReq, err := convertGreeterBroadcastNativeToMessageRequest(req)
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

func (GreeterCGONativeClientBridge) LoadBroadcastNativeStream(handle rpcruntime.StreamHandle) (GreeterBroadcastNativeStreamSession, bool) {
	return loadGreeterBroadcastNativeStream(handle)
}

func (GreeterCGONativeClientBridge) TakeBroadcastNativeStream(handle rpcruntime.StreamHandle) (GreeterBroadcastNativeStreamSession, bool) {
	return takeGreeterBroadcastNativeStream(handle)
}

type greeterBroadcastMessageToNativeStreamSession struct {
	message GreeterBroadcastMessageStreamSession
}

func (s *greeterBroadcastMessageToNativeStreamSession) Recv(ctx context.Context) (*SayHelloResponse, error) {
	messageResp, err := s.message.Recv(ctx)
	if err != nil {
		return nil, err
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

func (GreeterCGONativeClientBridge) LoadChatNativeStream(handle rpcruntime.StreamHandle) (GreeterChatNativeStreamSession, bool) {
	return loadGreeterChatNativeStream(handle)
}

func (GreeterCGONativeClientBridge) TakeChatNativeStream(handle rpcruntime.StreamHandle) (GreeterChatNativeStreamSession, bool) {
	return takeGreeterChatNativeStream(handle)
}

type greeterChatMessageToNativeStreamSession struct {
	message GreeterChatMessageStreamSession
}

func (s *greeterChatMessageToNativeStreamSession) Send(ctx context.Context, req *SayHelloRequest) error {
	messageReq, err := convertGreeterChatNativeToMessageRequest(req)
	if err != nil {
		return err
	}
	return s.message.Send(ctx, messageReq)
}

func (s *greeterChatMessageToNativeStreamSession) Recv(ctx context.Context) (*SayHelloResponse, error) {
	messageResp, err := s.message.Recv(ctx)
	if err != nil {
		return nil, err
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
			nativeReq, err := convertGreeterSayHelloMessageToNativeRequest(req)
			if err != nil {
				return err
			}
			nativeResp, err := snapshot.Adapter.Native.SayHello(ctx, nativeReq)
			if err != nil {
				return err
			}
			messageResp, err := convertGreeterSayHelloNativeToMessageResponse(nativeResp)
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

func (GreeterCGOMessageClientBridge) LoadCollectMessageStream(handle rpcruntime.StreamHandle) (GreeterCollectMessageStreamSession, bool) {
	return loadGreeterCollectMessageStream(handle)
}

func (GreeterCGOMessageClientBridge) TakeCollectMessageStream(handle rpcruntime.StreamHandle) (GreeterCollectMessageStreamSession, bool) {
	return takeGreeterCollectMessageStream(handle)
}

type greeterCollectNativeToMessageStreamSession struct {
	native GreeterCollectNativeStreamSession
}

func (s *greeterCollectNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {
	nativeReq, err := convertGreeterCollectMessageToNativeRequest(req)
	if err != nil {
		return err
	}
	return s.native.Send(ctx, nativeReq)
}

func (s *greeterCollectNativeToMessageStreamSession) Finish(ctx context.Context) ([]byte, error) {
	nativeResp, err := s.native.Finish(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterCollectNativeToMessageResponse(nativeResp)
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
			nativeReq, err := convertGreeterBroadcastMessageToNativeRequest(req)
			if err != nil {
				return nil, err
			}
			nativeSession, err := snapshot.Adapter.Native.StartBroadcast(ctx, nativeReq)
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

func (GreeterCGOMessageClientBridge) LoadBroadcastMessageStream(handle rpcruntime.StreamHandle) (GreeterBroadcastMessageStreamSession, bool) {
	return loadGreeterBroadcastMessageStream(handle)
}

func (GreeterCGOMessageClientBridge) TakeBroadcastMessageStream(handle rpcruntime.StreamHandle) (GreeterBroadcastMessageStreamSession, bool) {
	return takeGreeterBroadcastMessageStream(handle)
}

type greeterBroadcastNativeToMessageStreamSession struct {
	native GreeterBroadcastNativeStreamSession
}

func (s *greeterBroadcastNativeToMessageStreamSession) Recv(ctx context.Context) ([]byte, error) {
	nativeResp, err := s.native.Recv(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterBroadcastNativeToMessageResponse(nativeResp)
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

func (GreeterCGOMessageClientBridge) LoadChatMessageStream(handle rpcruntime.StreamHandle) (GreeterChatMessageStreamSession, bool) {
	return loadGreeterChatMessageStream(handle)
}

func (GreeterCGOMessageClientBridge) TakeChatMessageStream(handle rpcruntime.StreamHandle) (GreeterChatMessageStreamSession, bool) {
	return takeGreeterChatMessageStream(handle)
}

type greeterChatNativeToMessageStreamSession struct {
	native GreeterChatNativeStreamSession
}

func (s *greeterChatNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {
	nativeReq, err := convertGreeterChatMessageToNativeRequest(req)
	if err != nil {
		return err
	}
	return s.native.Send(ctx, nativeReq)
}

func (s *greeterChatNativeToMessageStreamSession) Recv(ctx context.Context) ([]byte, error) {
	nativeResp, err := s.native.Recv(ctx)
	if err != nil {
		return nil, err
	}
	return convertGreeterChatNativeToMessageResponse(nativeResp)
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
