package generator

import "testing"

func TestProjectRegistrationSourceCoversValidSources(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true})

	tests := []struct {
		name                               string
		source                             RegistrationSourcePlan
		wantRegistrationKind               runtimeRegistrationKind
		wantRegisterName                   string
		wantInputName                      string
		wantInputType                      string
		wantNilErr                         string
		wantSourceExpr                     string
		wantLabel                          string
		wantTransportConstructorShape      transportMessageStreamConstructorShape
		wantTransportConstructorReturnsErr bool
	}{
		{
			name:                 "go native",
			source:               registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractNative, RegistrationTransportNone, RegistrationModeLocal),
			wantRegistrationKind: runtimeRegistrationKindNative,
			wantRegisterName:     "registerGreeterGoNativeServer",
			wantInputName:        "server",
			wantInputType:        "GreeterNativeServer",
			wantNilErr:           "GreeterNativeServerUnavailableErr",
			wantSourceExpr:       "server",
			wantLabel:            "go native",
		},
		{
			name:                 "cgo native",
			source:               registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractNative, RegistrationTransportNone, RegistrationModeLocal),
			wantRegistrationKind: runtimeRegistrationKindCGONativeForward,
			wantRegisterName:     "RegisterGreeterCGONativeServer",
			wantInputName:        "server",
			wantInputType:        "GreeterNativeServer",
			wantNilErr:           "GreeterNativeServerUnavailableErr",
			wantSourceExpr:       "server",
			wantLabel:            "cgo native",
		},
		{
			name:                 "cgo message",
			source:               registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractMessage, RegistrationTransportNone, RegistrationModeLocal),
			wantRegistrationKind: runtimeRegistrationKindMessage,
			wantRegisterName:     "registerGreeterCGOMessageServer",
			wantInputName:        "server",
			wantInputType:        "GreeterCGOMessageServer",
			wantNilErr:           "GreeterMessageServerUnavailableErr",
			wantSourceExpr:       "server",
			wantLabel:            "cgo message",
		},
		{
			name:                          "connect local",
			source:                        registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportConnect, RegistrationModeLocal),
			wantRegistrationKind:          runtimeRegistrationKindTransportMessage,
			wantRegisterName:              "RegisterGreeterConnectHandler",
			wantInputName:                 "handler",
			wantInputType:                 "GreeterHandler",
			wantNilErr:                    "GreeterMessageServerUnavailableErr",
			wantSourceExpr:                "handler",
			wantLabel:                     "connect handler",
			wantTransportConstructorShape: transportMessageStreamConstructorShapeConnectLocal,
		},
		{
			name:                               "connect remote",
			source:                             registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportConnect, RegistrationModeRemote),
			wantRegistrationKind:               runtimeRegistrationKindTransportMessage,
			wantRegisterName:                   "RegisterGreeterConnectRemoteServer",
			wantInputName:                      "client",
			wantInputType:                      "GreeterClient",
			wantNilErr:                         "GreeterMessageServerUnavailableErr",
			wantSourceExpr:                     "client",
			wantLabel:                          "connect remote",
			wantTransportConstructorShape:      transportMessageStreamConstructorShapeConnectRemote,
			wantTransportConstructorReturnsErr: true,
		},
		{
			name:                          "grpc local",
			source:                        registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportGRPC, RegistrationModeLocal),
			wantRegistrationKind:          runtimeRegistrationKindTransportMessage,
			wantRegisterName:              "RegisterGreeterGRPCServer",
			wantInputName:                 "server",
			wantInputType:                 "GreeterServer",
			wantNilErr:                    "GreeterMessageServerUnavailableErr",
			wantSourceExpr:                "server",
			wantLabel:                     "grpc server",
			wantTransportConstructorShape: transportMessageStreamConstructorShapeGRPCLocal,
		},
		{
			name:                               "grpc remote",
			source:                             registrationSourceTestPlan(RegistrationOriginGo, RegistrationContractMessage, RegistrationTransportGRPC, RegistrationModeRemote),
			wantRegistrationKind:               runtimeRegistrationKindTransportMessage,
			wantRegisterName:                   "RegisterGreeterGRPCRemoteServer",
			wantInputName:                      "client",
			wantInputType:                      "GreeterClient",
			wantNilErr:                         "GreeterMessageServerUnavailableErr",
			wantSourceExpr:                     "client",
			wantLabel:                          "grpc remote",
			wantTransportConstructorShape:      transportMessageStreamConstructorShapeGRPCRemote,
			wantTransportConstructorReturnsErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectRegistrationSource(service, tt.source)
			if err != nil {
				t.Fatalf("ProjectRegistrationSource() error = %v", err)
			}
			if got.registrationKind != tt.wantRegistrationKind {
				t.Fatalf("registrationKind = %q, want %q", got.registrationKind, tt.wantRegistrationKind)
			}
			if got.registerName != tt.wantRegisterName {
				t.Fatalf("registerName = %q, want %q", got.registerName, tt.wantRegisterName)
			}
			if got.inputName != tt.wantInputName {
				t.Fatalf("inputName = %q, want %q", got.inputName, tt.wantInputName)
			}
			if got.inputType != tt.wantInputType {
				t.Fatalf("inputType = %q, want %q", got.inputType, tt.wantInputType)
			}
			if got.nilErr != tt.wantNilErr {
				t.Fatalf("nilErr = %q, want %q", got.nilErr, tt.wantNilErr)
			}
			if got.sourceExpr != tt.wantSourceExpr {
				t.Fatalf("sourceExpr = %q, want %q", got.sourceExpr, tt.wantSourceExpr)
			}
			if got.label != tt.wantLabel {
				t.Fatalf("label = %q, want %q", got.label, tt.wantLabel)
			}
			if got.transportStreamConstructorShape != tt.wantTransportConstructorShape {
				t.Fatalf("transportStreamConstructorShape = %q, want %q", got.transportStreamConstructorShape, tt.wantTransportConstructorShape)
			}
			if got.transportStreamConstructorReturnsErr != tt.wantTransportConstructorReturnsErr {
				t.Fatalf("transportStreamConstructorReturnsErr = %v, want %v", got.transportStreamConstructorReturnsErr, tt.wantTransportConstructorReturnsErr)
			}
		})
	}
}

func TestProjectRegistrationSourceRejectsInvalidSource(t *testing.T) {
	service := registrationSourceTestService("Greeter", ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true})
	source := registrationSourceTestPlan(RegistrationOriginCGO, RegistrationContractNative, RegistrationTransportConnect, RegistrationModeRemote)

	if _, err := ProjectRegistrationSource(service, source); err == nil {
		t.Fatalf("ProjectRegistrationSource(%#v) error = nil, want error", source)
	}
}
