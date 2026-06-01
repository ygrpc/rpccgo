package generator

import (
	"reflect"
	"testing"
)

func TestActiveRecordSourcesForConnectNativeService(t *testing.T) {
	service := activeRecordSourceTestService("Greeter", AdapterTokenMessageConnect, AdapterTokenNative)

	got := activeRecordSourcesForService(service)
	want := []ActiveRecordSourcePlan{
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal, ActiveRecordRendererNative, false, "server", "go native", "registerGreeterGoNativeServer", "server", "GreeterNativeServer", "GreeterNativeServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal, ActiveRecordRendererNative, true, "server", "cgo native", "RegisterGreeterCGONativeServer", "server", "GreeterNativeServer", "GreeterNativeServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractMessage, ActiveRecordTransportNone, ActiveRecordModeLocal, ActiveRecordRendererMessage, false, "server", "cgo message", "registerGreeterCGOMessageServer", "server", "GreeterCGOMessageServer", "GreeterMessageServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractTransportMessage, ActiveRecordTransportConnect, ActiveRecordModeLocal, ActiveRecordRendererTransportMessage, false, "handler", "connect handler", "RegisterGreeterConnectHandler", "handler", "GreeterHandler", "GreeterMessageServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractTransportMessage, ActiveRecordTransportConnect, ActiveRecordModeRemote, ActiveRecordRendererTransportMessage, false, "client", "connect remote", "RegisterGreeterConnectRemoteServer", "client", "GreeterClient", "GreeterMessageServerUnavailableErr"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeRecordSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestActiveRecordSourcesForGRPCNativeService(t *testing.T) {
	service := activeRecordSourceTestService("Greeter", AdapterTokenMessageGRPC, AdapterTokenNative)

	got := activeRecordSourcesForService(service)
	want := []ActiveRecordSourcePlan{
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal, ActiveRecordRendererNative, false, "server", "go native", "registerGreeterGoNativeServer", "server", "GreeterNativeServer", "GreeterNativeServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractNative, ActiveRecordTransportNone, ActiveRecordModeLocal, ActiveRecordRendererNative, true, "server", "cgo native", "RegisterGreeterCGONativeServer", "server", "GreeterNativeServer", "GreeterNativeServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginCGO, ActiveRecordContractMessage, ActiveRecordTransportNone, ActiveRecordModeLocal, ActiveRecordRendererMessage, false, "server", "cgo message", "registerGreeterCGOMessageServer", "server", "GreeterCGOMessageServer", "GreeterMessageServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractTransportMessage, ActiveRecordTransportGRPC, ActiveRecordModeLocal, ActiveRecordRendererTransportMessage, false, "server", "grpc server", "RegisterGreeterGRPCServer", "server", "GreeterServer", "GreeterMessageServerUnavailableErr"),
		activeRecordSourceTestPlan(ActiveRecordOriginGo, ActiveRecordContractTransportMessage, ActiveRecordTransportGRPC, ActiveRecordModeRemote, ActiveRecordRendererTransportMessage, false, "client", "grpc remote", "RegisterGreeterGRPCRemoteServer", "client", "GreeterClient", "GreeterMessageServerUnavailableErr"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activeRecordSourcesForService() = %#v, want %#v", got, want)
	}
}

func TestActiveRecordSourcesForNativeDirectiveIncludesDefaultConnectSources(t *testing.T) {
	adapters, err := ParseServiceRPCCGOOptions("// @rpccgo:native")
	if err != nil {
		t.Fatalf("ParseServiceRPCCGOOptions() error = %v", err)
	}
	service := activeRecordSourceTestService("Greeter", adapters.Tokens...)

	got := activeRecordSourcesForService(service)
	if len(got) != 5 {
		t.Fatalf("activeRecordSourcesForService() source count = %d, want 5", len(got))
	}
	if got[3].Transport != ActiveRecordTransportConnect || got[3].Mode != ActiveRecordModeLocal || got[3].Label != "connect handler" {
		t.Fatalf("default local message transport source = %#v, want connect handler", got[3])
	}
	if got[4].Transport != ActiveRecordTransportConnect || got[4].Mode != ActiveRecordModeRemote || got[4].Label != "connect remote" {
		t.Fatalf("default remote message transport source = %#v, want connect remote", got[4])
	}
}

func TestActiveRecordSourcesExposeRecordRenderer(t *testing.T) {
	service := activeRecordSourceTestService("Greeter", AdapterTokenMessageConnect, AdapterTokenNative)

	got := activeRecordSourcesForService(service)
	assertRecordRenderer(t, got[0], ActiveRecordRendererNative, false)
	assertRecordRenderer(t, got[1], ActiveRecordRendererNative, true)
	assertRecordRenderer(t, got[2], ActiveRecordRendererMessage, false)
	assertRecordRenderer(t, got[3], ActiveRecordRendererTransportMessage, false)
	assertRecordRenderer(t, got[4], ActiveRecordRendererTransportMessage, false)
}

func activeRecordSourceTestService(name string, tokens ...AdapterToken) ServicePlan {
	return ServicePlan{
		GoName:   name,
		Adapters: AdapterSelection{Tokens: tokens},
	}
}

func activeRecordSourceTestPlan(origin ActiveRecordOrigin, contract ActiveRecordContract, transport ActiveRecordTransport, mode ActiveRecordMode, renderer ActiveRecordRenderer, aliasGoNative bool, sourceExpr, label, registerName, inputName, inputType, nilErr string) ActiveRecordSourcePlan {
	return ActiveRecordSourcePlan{
		Origin:                    origin,
		Contract:                  contract,
		Transport:                 transport,
		Mode:                      mode,
		RecordRenderer:            renderer,
		AliasGoNativeRegistration: aliasGoNative,
		SourceExpr:                sourceExpr,
		Label:                     label,
		RegisterName:              registerName,
		InputName:                 inputName,
		InputType:                 inputType,
		NilErr:                    nilErr,
	}
}

func assertRecordRenderer(t *testing.T, source ActiveRecordSourcePlan, renderer ActiveRecordRenderer, aliasGoNative bool) {
	t.Helper()
	if source.RecordRenderer != renderer || source.AliasGoNativeRegistration != aliasGoNative {
		t.Fatalf("%s renderer = %q aliasGoNative = %t, want %q/%t", source.Label, source.RecordRenderer, source.AliasGoNativeRegistration, renderer, aliasGoNative)
	}
}
