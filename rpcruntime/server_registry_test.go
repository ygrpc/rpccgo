package rpcruntime

import (
	"errors"
	"sync"
	"testing"
)

type testRegisteredServer struct {
	name string
}

type testRegisteredServerInterface interface {
	serverName() string
}

type testRegisteredServerPointer struct {
	name string
}

func (s *testRegisteredServerPointer) serverName() string {
	return s.name
}

func TestServerKindZeroValueIsInvalid(t *testing.T) {
	var kind ServerKind
	if kind != ServerKindInvalid {
		t.Fatalf("zero ServerKind = %d, want ServerKindInvalid", kind)
	}

	kinds := []ServerKind{
		ServerKindGoNative,
		ServerKindCGONative,
		ServerKindCGOMessage,
		ServerKindConnect,
		ServerKindGRPC,
		ServerKindConnectRemote,
		ServerKindGRPCRemote,
	}
	for _, kind := range kinds {
		if kind == ServerKindInvalid {
			t.Fatalf("valid server kind %d equals ServerKindInvalid", kind)
		}
	}
}

func TestServerRegistryRegisterLoadReplaceAndClear(t *testing.T) {
	var registry ServerRegistry
	const serviceID ServiceID = "rpccgo.test.v1.Greeter"

	first := RegisteredServer{
		Kind:   ServerKindGoNative,
		Server: testRegisteredServer{name: "first"},
	}
	if err := registry.Register(serviceID, first); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	loaded, err := registry.Load(serviceID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded != first {
		t.Fatalf("Load returned %#v, want %#v", loaded, first)
	}

	second := RegisteredServer{
		Kind:   ServerKindCGOMessage,
		Server: testRegisteredServer{name: "second"},
	}
	if err := registry.Register(serviceID, second); err != nil {
		t.Fatalf("replacement Register returned error: %v", err)
	}

	loaded, err = registry.Load(serviceID)
	if err != nil {
		t.Fatalf("Load after replacement returned error: %v", err)
	}
	if loaded != second {
		t.Fatalf("Load after replacement returned %#v, want %#v", loaded, second)
	}

	if err := registry.Clear(serviceID); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if _, err := registry.Load(serviceID); !errors.Is(err, ErrNoRegisteredServer) {
		t.Fatalf("Load after Clear returned %v, want ErrNoRegisteredServer", err)
	}
}

func TestServerRegistryKeepsServiceIDsIndependent(t *testing.T) {
	var registry ServerRegistry
	const greeter ServiceID = "rpccgo.test.v1.Greeter"
	const echo ServiceID = "rpccgo.test.v1.Echo"

	greeterServer := RegisteredServer{
		Kind:   ServerKindConnect,
		Server: testRegisteredServer{name: "greeter"},
	}
	echoServer := RegisteredServer{
		Kind:   ServerKindGRPC,
		Server: testRegisteredServer{name: "echo"},
	}

	if err := registry.Register(greeter, greeterServer); err != nil {
		t.Fatalf("Register greeter returned error: %v", err)
	}
	if err := registry.Register(echo, echoServer); err != nil {
		t.Fatalf("Register echo returned error: %v", err)
	}
	if err := registry.Clear(greeter); err != nil {
		t.Fatalf("Clear greeter returned error: %v", err)
	}

	if _, err := registry.Load(greeter); !errors.Is(err, ErrNoRegisteredServer) {
		t.Fatalf("Load greeter after Clear returned %v, want ErrNoRegisteredServer", err)
	}
	loaded, err := registry.Load(echo)
	if err != nil {
		t.Fatalf("Load echo returned error: %v", err)
	}
	if loaded != echoServer {
		t.Fatalf("Load echo returned %#v, want %#v", loaded, echoServer)
	}
}

func TestServerRegistryRejectsInvalidRegistration(t *testing.T) {
	tests := []struct {
		name      string
		serviceID ServiceID
		server    RegisteredServer
		want      error
	}{
		{
			name:      "empty service id",
			serviceID: "",
			server: RegisteredServer{
				Kind:   ServerKindGoNative,
				Server: testRegisteredServer{name: "server"},
			},
			want: ErrEmptyServiceID,
		},
		{
			name:      "zero server kind",
			serviceID: "rpccgo.test.v1.Greeter",
			server: RegisteredServer{
				Kind:   ServerKindInvalid,
				Server: testRegisteredServer{name: "server"},
			},
			want: ErrInvalidServerKind,
		},
		{
			name:      "unknown server kind",
			serviceID: "rpccgo.test.v1.Greeter",
			server: RegisteredServer{
				Kind:   ServerKindGRPCRemote + 1,
				Server: testRegisteredServer{name: "server"},
			},
			want: ErrInvalidServerKind,
		},
		{
			name:      "nil server",
			serviceID: "rpccgo.test.v1.Greeter",
			server: RegisteredServer{
				Kind:   ServerKindGoNative,
				Server: nil,
			},
			want: ErrNilRegisteredServer,
		},
		{
			name:      "typed nil server",
			serviceID: "rpccgo.test.v1.Greeter",
			server: RegisteredServer{
				Kind:   ServerKindGoNative,
				Server: (*testRegisteredServerPointer)(nil),
			},
			want: ErrNilRegisteredServer,
		},
		{
			name:      "typed nil interface server",
			serviceID: "rpccgo.test.v1.Greeter",
			server: RegisteredServer{
				Kind:   ServerKindGoNative,
				Server: testRegisteredServerInterface((*testRegisteredServerPointer)(nil)),
			},
			want: ErrNilRegisteredServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var registry ServerRegistry
			if err := registry.Register(tt.serviceID, tt.server); !errors.Is(err, tt.want) {
				t.Fatalf("Register returned %v, want %v", err, tt.want)
			}
		})
	}
}

func TestServerRegistryRejectsNilRegistry(t *testing.T) {
	var registry *ServerRegistry
	server := RegisteredServer{
		Kind:   ServerKindGoNative,
		Server: testRegisteredServer{name: "server"},
	}

	if err := registry.Register("rpccgo.test.v1.Greeter", server); !errors.Is(err, errNilServerRegistry) {
		t.Fatalf("Register returned %v, want errNilServerRegistry", err)
	}
	if _, err := registry.Load("rpccgo.test.v1.Greeter"); !errors.Is(err, errNilServerRegistry) {
		t.Fatalf("Load returned %v, want errNilServerRegistry", err)
	}
	if err := registry.Clear("rpccgo.test.v1.Greeter"); !errors.Is(err, errNilServerRegistry) {
		t.Fatalf("Clear returned %v, want errNilServerRegistry", err)
	}
}

func TestServerRegistryRejectsEmptyServiceIDForLoadAndClear(t *testing.T) {
	var registry ServerRegistry

	if _, err := registry.Load(""); !errors.Is(err, ErrEmptyServiceID) {
		t.Fatalf("Load returned %v, want ErrEmptyServiceID", err)
	}
	if err := registry.Clear(""); !errors.Is(err, ErrEmptyServiceID) {
		t.Fatalf("Clear returned %v, want ErrEmptyServiceID", err)
	}
}

func TestServerRegistryPackageLevelAPI(t *testing.T) {
	const serviceID ServiceID = "rpccgo.test.v1.PackageLevel"
	if err := ClearServer(serviceID); err != nil {
		t.Fatalf("initial ClearServer returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := ClearServer(serviceID); err != nil {
			t.Fatalf("cleanup ClearServer returned error: %v", err)
		}
	})

	server := RegisteredServer{
		Kind:   ServerKindConnectRemote,
		Server: testRegisteredServer{name: "remote"},
	}
	if err := RegisterServer(serviceID, server); err != nil {
		t.Fatalf("RegisterServer returned error: %v", err)
	}
	loaded, err := LoadServer(serviceID)
	if err != nil {
		t.Fatalf("LoadServer returned error: %v", err)
	}
	if loaded != server {
		t.Fatalf("LoadServer returned %#v, want %#v", loaded, server)
	}
}

func TestServerRegistryConcurrentReplaceAndLoad(t *testing.T) {
	var registry ServerRegistry
	const serviceID ServiceID = "rpccgo.test.v1.Concurrent"
	records := []RegisteredServer{
		{Kind: ServerKindGoNative, Server: testRegisteredServer{name: "go"}},
		{Kind: ServerKindCGONative, Server: testRegisteredServer{name: "cgo-native"}},
		{Kind: ServerKindCGOMessage, Server: testRegisteredServer{name: "cgo-message"}},
		{Kind: ServerKindConnect, Server: testRegisteredServer{name: "connect"}},
		{Kind: ServerKindGRPC, Server: testRegisteredServer{name: "grpc"}},
		{Kind: ServerKindConnectRemote, Server: testRegisteredServer{name: "connect-remote"}},
		{Kind: ServerKindGRPCRemote, Server: testRegisteredServer{name: "grpc-remote"}},
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				record := records[(worker+n)%len(records)]
				if err := registry.Register(serviceID, record); err != nil {
					t.Errorf("Register returned error: %v", err)
					return
				}
			}
		}(i)
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				loaded, err := registry.Load(serviceID)
				if errors.Is(err, ErrNoRegisteredServer) {
					continue
				}
				if err != nil {
					t.Errorf("Load returned error: %v", err)
					return
				}
				if !registeredServerInList(loaded, records) {
					t.Errorf("Load returned unexpected record %#v", loaded)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func registeredServerInList(server RegisteredServer, servers []RegisteredServer) bool {
	for _, candidate := range servers {
		if server == candidate {
			return true
		}
	}
	return false
}
