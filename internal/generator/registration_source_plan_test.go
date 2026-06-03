package generator

import (
	"reflect"
	"testing"
)

func TestRegistrationSourcesForConnectNativeService(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true})

	got := registrationSourcesForService(service)
	want := []RegistrationSourcePlan{
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractNative, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractNative, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractMessage, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportConnect, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportConnect, RegistrationModeRemote),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registrationSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestRegistrationSourcesForGRPCNativeService(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportGRPC, NativeEnabled: true})

	got := registrationSourcesForService(service)
	want := []RegistrationSourcePlan{
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractNative, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractNative, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractMessage, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportGRPC, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportGRPC, RegistrationModeRemote),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registrationSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestRegistrationSourcesForMessageOnlyService(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect})

	got := registrationSourcesForService(service)
	want := []RegistrationSourcePlan{
		registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractMessage, RegistrationTransportNone, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportConnect, RegistrationModeLocal),
		registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportConnect, RegistrationModeRemote),
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
	if got[3].Contract != RegistrationContractMessage || got[3].Transport != RegistrationTransportConnect || got[3].Mode != RegistrationModeLocal {
		t.Fatalf("default local message transport source = %#v, want connect message local", got[3])
	}
	if got[4].Contract != RegistrationContractMessage || got[4].Transport != RegistrationTransportConnect || got[4].Mode != RegistrationModeRemote {
		t.Fatalf("default remote message transport source = %#v, want connect message remote", got[4])
	}
}

func TestRegistrationTransportNoneIsExplicit(t *testing.T) {
	if RegistrationTransportNone == "" {
		t.Fatalf("RegistrationTransportNone must be explicit, got empty string")
	}
	if string(RegistrationTransportNone) != "none" {
		t.Fatalf("RegistrationTransportNone = %q, want none", RegistrationTransportNone)
	}
}

func TestRegistrationSourceValidationRejectsArbitraryAxisCombination(t *testing.T) {
	source := RegistrationSourcePlan{
		Origin:    RegistrationOriginCGO,
		Contract:  RegistrationContractNative,
		Transport: RegistrationTransportConnect,
		Mode:      RegistrationModeRemote,
	}
	if err := ValidateRegistrationSourcePlan(source); err == nil {
		t.Fatalf("ValidateRegistrationSourcePlan(%#v) error = nil, want error", source)
	}
}

func registrationSourceTestService(name string, selection ServiceGenerationSelection) ServicePlan {
	return ServicePlan{
		GoName:     name,
		Generation: selection,
	}
}

func registrationSourceTestPlan(origin RegistrationOrigin, contract RegistrationContract, transport RegistrationTransport, mode RegistrationMode) RegistrationSourcePlan {
	return RegistrationSourcePlan{
		Origin:    origin,
		Contract:  contract,
		Transport: transport,
		Mode:      mode,
	}
}
