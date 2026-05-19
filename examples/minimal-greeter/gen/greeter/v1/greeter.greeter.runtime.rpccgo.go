package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo service runtime generated file for Greeter

type GreeterNativeAdapter interface {
	SayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error)
}

type GreeterMessageAdapter interface {
	SayHelloMessage(ctx context.Context, req []byte) ([]byte, error)
}

type GreeterActiveAdapter struct {
	Native  GreeterNativeAdapter
	Message GreeterMessageAdapter
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

func (r greeterActiveRouter) invokeNativeSayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error) {
	var messageResult string
	err := r.dispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[GreeterActiveAdapter]) error {
		switch snapshot.Contract {
		case rpcruntime.ServerContractNative:
			if snapshot.Adapter.Native == nil {
				return GreeterNativeAdapterUnavailableErr
			}
			var callErr error
			messageResult, callErr = snapshot.Adapter.Native.SayHello(ctx, name)
			return callErr
		case rpcruntime.ServerContractMessage:
			if snapshot.Adapter.Message == nil {
				return GreeterMessageAdapterUnavailableErr
			}
			messageReq, err := convertGreeterSayHelloNativeToMessageRequest(name)
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
			return withGreeterSayHelloMessageToNativeRequest(req, func(name *rpcruntime.RpcString) error {
				messageResult, err := snapshot.Adapter.Native.SayHello(ctx, name)
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

type GreeterCGONativeClientBridge struct{}

func (GreeterCGONativeClientBridge) SayHello(ctx context.Context, name *rpcruntime.RpcString) (string, error) {
	return greeterRouter.invokeNativeSayHello(ctx, name)
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

func NewGreeterCGOMessageClientBridge() GreeterCGOMessageClientBridge {
	return GreeterCGOMessageClientBridge{}
}

func RegisterGreeterCGOMessageActiveServer(kind rpcruntime.ServerKind, adapter GreeterMessageAdapter) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	return registerGreeterMessageActiveServer(kind, adapter)
}
