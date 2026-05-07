package greeterv1

import (
	context "context"
	errors "errors"
	rpcruntime "rpccgo/rpcruntime"
)

// rpccgo service runtime stage file for Greeter

type GreeterNativeAdapter interface {
	SayHello(ctx context.Context, req *SayHelloRequest) (*SayHelloResponse, error)
}

type GreeterMessageAdapter interface {
	SayHelloMessage(ctx context.Context, req []byte) ([]byte, error)
}

type GreeterActiveAdapter struct {
	Native  GreeterNativeAdapter
	Message GreeterMessageAdapter
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

func NewGreeterCGOMessageClientBridge() GreeterCGOMessageClientBridge {
	return GreeterCGOMessageClientBridge{}
}

func RegisterGreeterCGOMessageActiveServer(kind rpcruntime.ServerKind, adapter GreeterMessageAdapter) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	return registerGreeterMessageActiveServer(kind, adapter)
}
