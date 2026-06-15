package rpcruntime

import (
	"errors"
	"fmt"
	"unsafe"

	protobuf "google.golang.org/protobuf/proto"
)

// DecodeMessage unmarshals a borrowed protobuf ptr/len payload into message.
// Callers provide the concrete protobuf pointer instance to decode into.
func DecodeMessage(ptr uintptr, length int32, message protobuf.Message) error {
	if length < 0 {
		return errors.New("message length is negative")
	}
	if isNilMessage(message) {
		return errors.New("message is nil")
	}
	protobuf.Reset(message)
	if length == 0 {
		return nil
	}
	if ptr == 0 {
		return errors.New("message pointer is nil")
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))
	if err := protobuf.Unmarshal(data, message); err != nil {
		return fmt.Errorf("protobuf unmarshal failed: %w", err)
	}
	return nil
}

// EncodeMessage marshals message into a pinned protobuf ptr/len payload.
// Callers must release the returned pointer with Release after the ABI consumer is done with it.
func EncodeMessage(message protobuf.Message) (uintptr, int32, error) {
	if isNilMessage(message) {
		return 0, 0, errors.New("message is nil")
	}
	data, err := protobuf.Marshal(message)
	if err != nil {
		return 0, 0, fmt.Errorf("protobuf marshal failed: %w", err)
	}
	length, err := LengthToInt32(len(data))
	if err != nil {
		return 0, 0, err
	}
	ptr, err := PinBytes(data)
	if err != nil {
		return 0, 0, err
	}
	return ptr, length, nil
}

func isNilMessage(message protobuf.Message) bool {
	if message == nil {
		return true
	}
	return !message.ProtoReflect().IsValid()
}
