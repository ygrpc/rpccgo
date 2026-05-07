package greeterv1

import (
	errors "errors"
	fmt "fmt"
	proto "google.golang.org/protobuf/proto"
)

// rpccgo native message codec stage file for Greeter

var greeterNativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")

func convertGreeterSayHelloMessageToNativeRequest(data []byte) (*SayHelloRequest, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterSayHelloNativeToMessageRequest(req *SayHelloRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.New("rpccgo: native request is nil")
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterSayHelloMessageToNativeResponse(data []byte) (*SayHelloResponse, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterSayHelloNativeToMessageResponse(resp *SayHelloResponse) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("rpccgo: native response is nil")
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterCollectMessageToNativeRequest(data []byte) (*SayHelloRequest, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterCollectNativeToMessageRequest(req *SayHelloRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.New("rpccgo: native request is nil")
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterCollectMessageToNativeResponse(data []byte) (*SayHelloResponse, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterCollectNativeToMessageResponse(resp *SayHelloResponse) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("rpccgo: native response is nil")
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterBroadcastMessageToNativeRequest(data []byte) (*SayHelloRequest, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterBroadcastNativeToMessageRequest(req *SayHelloRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.New("rpccgo: native request is nil")
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterBroadcastMessageToNativeResponse(data []byte) (*SayHelloResponse, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterBroadcastNativeToMessageResponse(resp *SayHelloResponse) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("rpccgo: native response is nil")
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterChatMessageToNativeRequest(data []byte) (*SayHelloRequest, error) {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterChatNativeToMessageRequest(req *SayHelloRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.New("rpccgo: native request is nil")
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)
	}
	return data, nil
}

func convertGreeterChatMessageToNativeResponse(data []byte) (*SayHelloResponse, error) {
	var msg SayHelloResponse
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)
	}
	return &msg, nil
}

func convertGreeterChatNativeToMessageResponse(resp *SayHelloResponse) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("rpccgo: native response is nil")
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)
	}
	return data, nil
}
