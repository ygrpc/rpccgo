package rpcruntime

import "testing"

func TestAdapterSnapshotZeroValueHasNoAdapter(t *testing.T) {
	var snapshot AdapterSnapshot[*fakeDispatcherAdapter]

	if snapshot.HasAdapter() {
		t.Fatal("expected zero snapshot to have no adapter")
	}
}

func TestAdapterSnapshotNilInterfaceHasNoAdapter(t *testing.T) {
	var adapter fakeDispatcherInterface
	snapshot := AdapterSnapshot[fakeDispatcherInterface]{
		Kind:     ServerKindGoNative,
		Contract: ServerContractNative,
		Version:  1,
		Adapter:  adapter,
	}

	if snapshot.HasAdapter() {
		t.Fatal("expected nil interface adapter to be treated as missing")
	}
}

func TestAdapterSnapshotTypedNilInterfaceHasNoAdapter(t *testing.T) {
	var impl *fakeDispatcherAdapterWithMethod
	var adapter fakeDispatcherInterface = impl
	snapshot := AdapterSnapshot[fakeDispatcherInterface]{
		Kind:     ServerKindGoNative,
		Contract: ServerContractNative,
		Version:  1,
		Adapter:  adapter,
	}

	if snapshot.HasAdapter() {
		t.Fatal("expected typed nil interface adapter to be treated as missing")
	}
}

func TestAdapterSnapshotNonZeroAdapterIsPresent(t *testing.T) {
	snapshot := AdapterSnapshot[*fakeDispatcherAdapter]{
		Kind:     ServerKindGoNative,
		Contract: ServerContractNative,
		Version:  1,
		Adapter:  &fakeDispatcherAdapter{name: "go"},
	}

	if !snapshot.HasAdapter() {
		t.Fatal("expected non-zero adapter to be present")
	}
}

func TestServerContractNativeAndMessageAreDistinct(t *testing.T) {
	if ServerContractNative == ServerContractMessage {
		t.Fatal("expected native and message contracts to be distinct")
	}
	if ServerContractNative.String() != "native" {
		t.Fatalf("unexpected native contract string: %q", ServerContractNative.String())
	}
	if ServerContractMessage.String() != "message" {
		t.Fatalf("unexpected message contract string: %q", ServerContractMessage.String())
	}
}

func TestServerKindStableStringRepresentations(t *testing.T) {
	tests := []struct {
		name string
		kind ServerKind
		want string
	}{
		{name: "go native", kind: ServerKindGoNative, want: "go-native"},
		{name: "cgo native", kind: ServerKindCGONative, want: "cgo-native"},
		{name: "cgo message", kind: ServerKindCGOMessage, want: "cgo-message"},
		{name: "connect handler", kind: ServerKindConnectHandler, want: "connect-handler"},
		{name: "grpc server", kind: ServerKindGRPCServer, want: "grpc-server"},
		{name: "connect remote", kind: ServerKindConnectRemote, want: "connect-remote"},
		{name: "grpc remote", kind: ServerKindGRPCRemote, want: "grpc-remote"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Fatalf("unexpected server kind string: got %q, want %q", got, tt.want)
			}
		})
	}
}

type fakeDispatcherAdapter struct {
	name string
}

type fakeDispatcherAdapterWithMethod struct{}

func (*fakeDispatcherAdapterWithMethod) dispatch() {}

type fakeDispatcherInterface interface {
	dispatch()
}
