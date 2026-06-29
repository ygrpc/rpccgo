package generator

import (
	"reflect"
	"testing"
)

func TestRegistrationSourcesForConnectNativeService(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true})

	got := registrationSourcesForService(service)
	want := []RegistrationSourceKind{
		RegistrationSourceGoNative,
		RegistrationSourceCGONative,
		RegistrationSourceCGOMessage,
		RegistrationSourceConnectHandler,
		RegistrationSourceConnectRemote,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registrationSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestRegistrationSourcesForGRPCNativeService(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportGRPC, NativeEnabled: true})

	got := registrationSourcesForService(service)
	want := []RegistrationSourceKind{
		RegistrationSourceGoNative,
		RegistrationSourceCGONative,
		RegistrationSourceCGOMessage,
		RegistrationSourceGRPCServer,
		RegistrationSourceGRPCRemote,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registrationSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestRegistrationSourcesForMessageOnlyService(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect})

	got := registrationSourcesForService(service)
	want := []RegistrationSourceKind{
		RegistrationSourceCGOMessage,
		RegistrationSourceConnectHandler,
		RegistrationSourceConnectRemote,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registrationSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestRegistrationSourcesForNativeDirectiveIncludesDefaultConnectSources(t *testing.T) {
	selection, err := ParseServiceRPCCGOOptions("// @rpccgo:native")
	if err != nil {
		t.Fatalf("ParseServiceRPCCGOOptions() error = %v", err)
	}
	service := registrationSourceTestService("Greeter", selection)

	got := registrationSourcesForService(service)
	if len(got) != 5 {
		t.Fatalf("registrationSourcesForService() source count = %d, want 5", len(got))
	}
	if got[3] != RegistrationSourceConnectHandler {
		t.Fatalf("default local message transport source = %#v, want connect handler", got[3])
	}
	if got[4] != RegistrationSourceConnectRemote {
		t.Fatalf("default remote message transport source = %#v, want connect remote", got[4])
	}
}

func TestRegistrationSourceKindsAreExplicit(t *testing.T) {
	for _, source := range []RegistrationSourceKind{
		RegistrationSourceGoNative,
		RegistrationSourceCGONative,
		RegistrationSourceCGOMessage,
		RegistrationSourceConnectHandler,
		RegistrationSourceConnectRemote,
		RegistrationSourceGRPCServer,
		RegistrationSourceGRPCRemote,
	} {
		if source == "" {
			t.Fatalf("registration source kind must be explicit")
		}
	}
}

func TestRegistrationSourceValidationRejectsUnknownKind(t *testing.T) {
	source := RegistrationSourceKind("bogus")
	if err := validateRegistrationSourceKind(source); err == nil {
		t.Fatalf("validateRegistrationSourceKind(%#v) error = nil, want error", source)
	}
}

func registrationSourceTestService(name string, selection ServiceGenerationSelection) ServicePlan {
	return ServicePlan{
		GoName:     name,
		Generation: selection,
	}
}
