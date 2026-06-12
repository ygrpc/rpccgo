package rpcruntime

import (
	"errors"
	"reflect"
	"sync"
)

type ServiceID string

type ServerKind int

const (
	ServerKindInvalid ServerKind = iota
	ServerKindGoNative
	ServerKindCGONative
	ServerKindCGOMessage
	ServerKindConnect
	ServerKindGRPC
	ServerKindConnectRemote
	ServerKindGRPCRemote
)

var (
	ErrEmptyServiceID      = errors.New("server registry requires non-empty service id")
	ErrInvalidServerKind   = errors.New("server registry requires valid server kind")
	ErrNilRegisteredServer = errors.New("server registry requires non-nil server")
	errNilServerRegistry   = errors.New("server registry is nil")
	defaultServerRegistry  ServerRegistry
)

type RegisteredServer struct {
	Kind   ServerKind
	Server any
}

type ServerRegistry struct {
	mu      sync.RWMutex
	servers map[ServiceID]RegisteredServer
}

func RegisterServer(serviceID ServiceID, server RegisteredServer) error {
	return defaultServerRegistry.Register(serviceID, server)
}

func LoadServer(serviceID ServiceID) (RegisteredServer, error) {
	return defaultServerRegistry.Load(serviceID)
}

func ClearServer(serviceID ServiceID) error {
	return defaultServerRegistry.Clear(serviceID)
}

func (r *ServerRegistry) Register(serviceID ServiceID, server RegisteredServer) error {
	if r == nil {
		return errNilServerRegistry
	}
	if err := validateServiceID(serviceID); err != nil {
		return err
	}
	if err := validateRegisteredServer(server); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.servers == nil {
		r.servers = make(map[ServiceID]RegisteredServer)
	}
	r.servers[serviceID] = server
	return nil
}

func (r *ServerRegistry) Load(serviceID ServiceID) (RegisteredServer, error) {
	if r == nil {
		return RegisteredServer{}, errNilServerRegistry
	}
	if err := validateServiceID(serviceID); err != nil {
		return RegisteredServer{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	server, ok := r.servers[serviceID]
	if !ok {
		return RegisteredServer{}, ErrNoRegisteredServer
	}
	return server, nil
}

func (r *ServerRegistry) Clear(serviceID ServiceID) error {
	if r == nil {
		return errNilServerRegistry
	}
	if err := validateServiceID(serviceID); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.servers, serviceID)
	return nil
}

func validateServiceID(serviceID ServiceID) error {
	if serviceID == "" {
		return ErrEmptyServiceID
	}
	return nil
}

func validateRegisteredServer(server RegisteredServer) error {
	if server.Kind <= ServerKindInvalid || server.Kind > ServerKindGRPCRemote {
		return ErrInvalidServerKind
	}
	if isNilServer(server.Server) {
		return ErrNilRegisteredServer
	}
	return nil
}

func isNilServer(server any) bool {
	value := reflect.ValueOf(server)
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
