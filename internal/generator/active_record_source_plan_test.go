package generator

import (
	"reflect"
	"testing"
)

func TestActiveRecordSourcesForConnectNativeService(t *testing.T) {
	service := activeRecordSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true})

	got := activeRecordSourcesForService(service)
	want := []ActiveRecordSourcePlan{
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractMessage, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractMessage, ActiveRecordTransportConnect, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractMessage, ActiveRecordTransportConnect, ActiveRecordModeRemote),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeRecordSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestActiveRecordSourcesForGRPCNativeService(t *testing.T) {
	service := activeRecordSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportGRPC, NativeEnabled: true})

	got := activeRecordSourcesForService(service)
	want := []ActiveRecordSourcePlan{
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractMessage, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractMessage, ActiveRecordTransportGRPC, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractMessage, ActiveRecordTransportGRPC, ActiveRecordModeRemote),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeRecordSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestActiveRecordSourcesForMessageOnlyService(t *testing.T) {
	service := activeRecordSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect})

	got := activeRecordSourcesForService(service)
	want := []ActiveRecordSourcePlan{
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractMessage, ActiveRecordTransportNone, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractMessage, ActiveRecordTransportConnect, ActiveRecordModeLocal),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractMessage, ActiveRecordTransportConnect, ActiveRecordModeRemote),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeRecordSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestActiveRecordSourcesForNativeDirectiveIncludesDefaultConnectSources(t *testing.T) {
	selection, err := ParseServiceRPCCGOOptions("// @rpccgo:native")
	if err != nil {
		t.Fatalf("ParseServiceRPCCGOOptions() error = %v", err)
	}
	service := activeRecordSourceTestService("Greeter", selection)

	got := activeRecordSourcesForService(service)
	if len(got) != 5 {
		t.Fatalf("activeRecordSourcesForService() source count = %d, want 5", len(got))
	}
	if got[3].Contract != ActiveRecordContractMessage || got[3].Transport != ActiveRecordTransportConnect || got[3].Mode != ActiveRecordModeLocal {
		t.Fatalf("default local message transport source = %#v, want connect message local", got[3])
	}
	if got[4].Contract != ActiveRecordContractMessage || got[4].Transport != ActiveRecordTransportConnect || got[4].Mode != ActiveRecordModeRemote {
		t.Fatalf("default remote message transport source = %#v, want connect message remote", got[4])
	}
}

func TestActiveRecordTransportNoneIsExplicit(t *testing.T) {
	if ActiveRecordTransportNone == "" {
		t.Fatalf("ActiveRecordTransportNone must be explicit, got empty string")
	}
	if string(ActiveRecordTransportNone) != "none" {
		t.Fatalf("ActiveRecordTransportNone = %q, want none", ActiveRecordTransportNone)
	}
}

func TestActiveRecordSourceValidationRejectsArbitraryAxisCombination(t *testing.T) {
	source := ActiveRecordSourcePlan{
		Origin:    ActiveRecordOriginCGO,
		Contract:  ActiveRecordContractNative,
		Transport: ActiveRecordTransportConnect,
		Mode:      ActiveRecordModeRemote,
	}
	if err := ValidateActiveRecordSourcePlan(source); err == nil {
		t.Fatalf("ValidateActiveRecordSourcePlan(%#v) error = nil, want error", source)
	}
}

func activeRecordSourceTestService(name string, selection ServiceGenerationSelection) ServicePlan {
	return ServicePlan{
		GoName:     name,
		Generation: selection,
	}
}

func activeRecordSourceTestPlan(origin ActiveRecordOrigin, contract ActiveRecordContract, transport ActiveRecordTransport, mode ActiveRecordMode) ActiveRecordSourcePlan {
	return ActiveRecordSourcePlan{
		Origin:    origin,
		Contract:  contract,
		Transport: transport,
		Mode:      mode,
	}
}
