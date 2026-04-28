package rpcruntime

import "reflect"

type ServerContract string

const (
	ServerContractNative  ServerContract = "native"
	ServerContractMessage ServerContract = "message"
)

func (c ServerContract) String() string {
	return string(c)
}

type ServerKind string

const (
	ServerKindGoNative       ServerKind = "go-native"
	ServerKindCGONative      ServerKind = "cgo-native"
	ServerKindCGOMessage     ServerKind = "cgo-message"
	ServerKindConnectHandler ServerKind = "connect-handler"
	ServerKindGRPCServer     ServerKind = "grpc-server"
	ServerKindConnectRemote  ServerKind = "connect-remote"
	ServerKindGRPCRemote     ServerKind = "grpc-remote"
)

func (k ServerKind) String() string {
	return string(k)
}

type AdapterSnapshot[T any] struct {
	Kind     ServerKind
	Contract ServerContract
	Version  int64
	Adapter  T
}

func (s AdapterSnapshot[T]) HasAdapter() bool {
	return hasNonZeroAdapter(s.Adapter)
}

func hasNonZeroAdapter[T any](adapter T) bool {
	value := reflect.ValueOf(adapter)
	if !value.IsValid() {
		return false
	}
	return !value.IsZero()
}
