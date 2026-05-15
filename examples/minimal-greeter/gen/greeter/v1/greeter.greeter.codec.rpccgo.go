package greeterv1

import (
	errors "errors"
	fmt "fmt"
	goruntime "runtime"
	rpcruntime "rpccgo/rpcruntime"
	unsafe "unsafe"
	proto "google.golang.org/protobuf/proto"
)

// rpccgo native message codec generated file for Greeter

var greeterNativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")

func withGreeterSayHelloMessageToNativeRequest(data []byte, fn func(name *rpcruntime.RpcString) error) error {
	var msg SayHelloRequest
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}
	msgOwner := &msg
	var name *rpcruntime.RpcString
	if msg.Name != "" {
		name = rpcruntime.NewRpcStringView(unsafe.StringData(msg.Name), int32(len(msg.Name)), msgOwner)
	} else {
		name = rpcruntime.EmptyRpcString()
	}
	err := fn(name)
	goruntime.KeepAlive(&msg)
	return err
}

func convertGreeterSayHelloNativeToMessageRequest(name *rpcruntime.RpcString) ([]byte, error) {
	msg := &SayHelloRequest{}
	msg.Name = name.SafeString()
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
