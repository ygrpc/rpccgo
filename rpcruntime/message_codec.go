package rpcruntime

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	protobuf "google.golang.org/protobuf/proto"
)

// DecodeMessage unmarshals a borrowed protobuf ptr/len payload into message.
// Callers provide the concrete protobuf pointer instance to decode into.
func DecodeMessage[T protobuf.Message](ptr uintptr, length int32, message T) error {
	if length < 0 {
		return errors.New("message length is negative")
	}
	if isNilMessage(message) {
		return errors.New("message is nil")
	}
	if length == 0 {
		protobuf.Reset(message)
		return nil
	}
	if ptr == 0 {
		return errors.New("message pointer is nil")
	}
	protobuf.Reset(message)
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))
	if err := protobuf.Unmarshal(data, message); err != nil {
		return fmt.Errorf("protobuf unmarshal failed: %w", err)
	}
	return nil
}

// EncodeMessage marshals message into a pinned protobuf ptr/len payload.
// Callers must release the returned pointer with Release after the ABI consumer is done with it.
func EncodeMessage[T protobuf.Message](message T) (uintptr, int32, error) {
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

func isNilMessage[T protobuf.Message](message T) bool {
	value := reflect.ValueOf(message)
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
