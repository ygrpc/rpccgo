package greeterv1

import (
	errors "errors"
	fmt "fmt"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
	proto "google.golang.org/protobuf/proto"
)

// rpccgo native message codec stage file for Greeter

var greeterNativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")

func convertGreeterSayHelloMessageToNativeRequest(data []byte) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, nil, err
	}
	name := rpcruntime.NewRpcString(nil, 0, false)
	if msg.Name != "" {
		data, ptr, err := rpcruntime.PinString(msg.Name)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		name = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.Name)), false)
	}
	city := rpcruntime.NewRpcString(nil, 0, false)
	if msg.City != "" {
		data, ptr, err := rpcruntime.PinString(msg.City)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		city = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.City)), false)
	}
	return name, city, nil
}

func convertGreeterSayHelloNativeToMessageRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) ([]byte, error) {
	msg := &SayHelloRequest{}
	msg.Name = name.SafeString()
	msg.City = city.SafeString()
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterSayHelloMessageToNativeResponse(data []byte) (string, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	message := msg.Message
	return message, nil
}

func convertGreeterSayHelloNativeToMessageResponse(message string) ([]byte, error) {
	msg := &SayHelloResponse{}
	msg.Message = message
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterCollectMessageToNativeRequest(data []byte) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, nil, err
	}
	name := rpcruntime.NewRpcString(nil, 0, false)
	if msg.Name != "" {
		data, ptr, err := rpcruntime.PinString(msg.Name)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		name = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.Name)), false)
	}
	city := rpcruntime.NewRpcString(nil, 0, false)
	if msg.City != "" {
		data, ptr, err := rpcruntime.PinString(msg.City)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		city = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.City)), false)
	}
	return name, city, nil
}

func convertGreeterCollectNativeToMessageRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) ([]byte, error) {
	msg := &SayHelloRequest{}
	msg.Name = name.SafeString()
	msg.City = city.SafeString()
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterCollectMessageToNativeResponse(data []byte) (string, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	message := msg.Message
	return message, nil
}

func convertGreeterCollectNativeToMessageResponse(message string) ([]byte, error) {
	msg := &SayHelloResponse{}
	msg.Message = message
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterBroadcastMessageToNativeRequest(data []byte) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, nil, err
	}
	name := rpcruntime.NewRpcString(nil, 0, false)
	if msg.Name != "" {
		data, ptr, err := rpcruntime.PinString(msg.Name)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		name = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.Name)), false)
	}
	city := rpcruntime.NewRpcString(nil, 0, false)
	if msg.City != "" {
		data, ptr, err := rpcruntime.PinString(msg.City)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		city = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.City)), false)
	}
	return name, city, nil
}

func convertGreeterBroadcastNativeToMessageRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) ([]byte, error) {
	msg := &SayHelloRequest{}
	msg.Name = name.SafeString()
	msg.City = city.SafeString()
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterBroadcastMessageToNativeResponse(data []byte) (string, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	message := msg.Message
	return message, nil
}

func convertGreeterBroadcastNativeToMessageResponse(message string) ([]byte, error) {
	msg := &SayHelloResponse{}
	msg.Message = message
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterChatMessageToNativeRequest(data []byte) (*rpcruntime.RpcString, *rpcruntime.RpcString, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, nil, err
	}
	name := rpcruntime.NewRpcString(nil, 0, false)
	if msg.Name != "" {
		data, ptr, err := rpcruntime.PinString(msg.Name)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		name = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.Name)), false)
	}
	city := rpcruntime.NewRpcString(nil, 0, false)
	if msg.City != "" {
		data, ptr, err := rpcruntime.PinString(msg.City)
		_ = data
		if err != nil {
			return nil, nil, err
		}
		defer rpcruntime.Release(ptr)
		city = rpcruntime.NewRpcString((*byte)(unsafe.Pointer(ptr)), int32(len(msg.City)), false)
	}
	return name, city, nil
}

func convertGreeterChatNativeToMessageRequest(name *rpcruntime.RpcString, city *rpcruntime.RpcString) ([]byte, error) {
	msg := &SayHelloRequest{}
	msg.Name = name.SafeString()
	msg.City = city.SafeString()
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterChatMessageToNativeResponse(data []byte) (string, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	message := msg.Message
	return message, nil
}

func convertGreeterChatNativeToMessageResponse(message string) ([]byte, error) {
	msg := &SayHelloResponse{}
	msg.Message = message
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}
