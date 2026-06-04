package generator

import (
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
)

func TestBuildRuntimeMethodProjectionsProjectUnaryMethod(t *testing.T) {
	service, g := runtimeProjectionTestContext(t)
	method, err := runtimeProjectionTestMethod(service.GoName, "Unary", StreamingKindUnary)
	if err != nil {
		t.Fatalf("runtimeProjectionTestMethod() error = %v", err)
	}
	service.Methods = []MethodPlan{method}

	projections, err := buildRuntimeMethodProjections(g, service)
	if err != nil {
		t.Fatalf("buildRuntimeMethodProjections() error = %v", err)
	}
	if len(projections) != 1 {
		t.Fatalf("buildRuntimeMethodProjections() len = %d, want 1", len(projections))
	}

	got := projections[0]
	if got.Identity.GoName != "Unary" {
		t.Fatalf("projection identity go name = %q, want Unary", got.Identity.GoName)
	}
	if got.Stream.Shape != runtimeStreamUnary {
		t.Fatalf("projection stream shape = %v, want unary", got.Stream.Shape)
	}
	if got.Stream.Streaming {
		t.Fatal("projection streaming = true, want false")
	}
	if got.Native.Args != ", name *rpcruntime.RpcString" {
		t.Fatalf("projection native args = %q, want request params", got.Native.Args)
	}
	if got.Native.Returns != "string, error" {
		t.Fatalf("projection native returns = %q, want response returns", got.Native.Returns)
	}
	if got.Native.NoRegisteredZero != "\"\", rpcruntime.ErrNoRegisteredServer" {
		t.Fatalf("projection no-registered zero = %q, want rpcruntime no registered zero", got.Native.NoRegisteredZero)
	}
	if got.Codec.NativeRequestToMessage != "convertGreeterUnaryNativeToMessageRequest" {
		t.Fatalf("projection native request codec = %q", got.Codec.NativeRequestToMessage)
	}
	if got.Codec.MessageToNativeResponse != "convertGreeterUnaryMessageToNativeResponse" {
		t.Fatalf("projection message response codec = %q", got.Codec.MessageToNativeResponse)
	}
	if got.Message.RequestType != "HelloRequest" {
		t.Fatalf("projection request type = %q, want same-package request type", got.Message.RequestType)
	}
	if got.Message.ResponseType != "HelloReply" {
		t.Fatalf("projection response type = %q, want same-package response type", got.Message.ResponseType)
	}
}

func TestBuildRuntimeMethodProjectionsProjectStreamingShapes(t *testing.T) {
	tests := []struct {
		name               string
		streaming          StreamingKind
		wantShape          runtimeStreamShape
		wantStartReq       bool
		wantCanSend        bool
		wantCanRecv        bool
		wantCanCloseSend   bool
		wantFinishResponse bool
	}{
		{
			name:               "Upload",
			streaming:          StreamingKindClientStreaming,
			wantShape:          runtimeStreamClient,
			wantCanSend:        true,
			wantFinishResponse: true,
		},
		{
			name:         "List",
			streaming:    StreamingKindServerStreaming,
			wantShape:    runtimeStreamServer,
			wantStartReq: true,
			wantCanRecv:  true,
		},
		{
			name:             "Chat",
			streaming:        StreamingKindBidiStreaming,
			wantShape:        runtimeStreamBidi,
			wantCanSend:      true,
			wantCanRecv:      true,
			wantCanCloseSend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, g := runtimeProjectionTestContext(t)
			method, err := runtimeProjectionTestMethod(service.GoName, tt.name, tt.streaming)
			if err != nil {
				t.Fatalf("runtimeProjectionTestMethod() error = %v", err)
			}
			service.Methods = []MethodPlan{method}

			projections, err := buildRuntimeMethodProjections(g, service)
			if err != nil {
				t.Fatalf("buildRuntimeMethodProjections() error = %v", err)
			}
			got := projections[0]

			if got.Stream.Shape != tt.wantShape {
				t.Fatalf("projection stream shape = %v, want %v", got.Stream.Shape, tt.wantShape)
			}
			if got.Stream.StartAcceptsRequest != tt.wantStartReq {
				t.Fatalf("projection start accepts request = %v, want %v", got.Stream.StartAcceptsRequest, tt.wantStartReq)
			}
			if got.Stream.CanSend != tt.wantCanSend {
				t.Fatalf("projection can send = %v, want %v", got.Stream.CanSend, tt.wantCanSend)
			}
			if got.Stream.CanRecv != tt.wantCanRecv {
				t.Fatalf("projection can recv = %v, want %v", got.Stream.CanRecv, tt.wantCanRecv)
			}
			if got.Stream.CanCloseSend != tt.wantCanCloseSend {
				t.Fatalf("projection can close send = %v, want %v", got.Stream.CanCloseSend, tt.wantCanCloseSend)
			}
			if got.Stream.FinishReturnsResponse != tt.wantFinishResponse {
				t.Fatalf("projection finish returns response = %v, want %v", got.Stream.FinishReturnsResponse, tt.wantFinishResponse)
			}
			if !got.Stream.Streaming {
				t.Fatal("projection streaming = false, want true")
			}
			if got.Symbols.NativeSourceSessionType == "" || got.Symbols.MessageSourceSessionType == "" {
				t.Fatalf("projection source session types = %#v, want both names", got.Symbols)
			}
		})
	}
}

func runtimeProjectionTestContext(t *testing.T) (ServicePlan, *protogen.GeneratedFile) {
	t.Helper()

	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		ProtoPath:     "test/v1/greeter.proto",
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services:      nil,
		TopLevelSymbols: []TopLevelSymbolPlan{
			{GoName: "HelloRequest", FullName: "test.v1.HelloRequest", Kind: TopLevelSymbolKindMessage},
			{GoName: "HelloReply", FullName: "test.v1.HelloReply", Kind: TopLevelSymbolKindMessage},
		},
	}
	file := GeneratedArtifactPlan{Kind: GeneratedArtifactKindRuntime, Filename: "test/v1/greeter.greeter.runtime.rpccgo.go"}
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))
	service := ServicePlan{
		Name:       "Greeter",
		GoName:     "Greeter",
		FullName:   "test.v1.Greeter",
		Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
	}
	return service, g
}

func runtimeProjectionTestMethod(serviceName, methodName string, streaming StreamingKind) (MethodPlan, error) {
	method := MethodPlan{
		Name:       methodName,
		GoName:     methodName,
		FullName:   "test.v1.Greeter." + methodName,
		DocComment: "// " + methodName + " docs",
		Streaming:  streaming,
		Request: MethodIOPlan{
			GoName:       "HelloRequest",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.HelloRequest",
		},
		Response: MethodIOPlan{
			GoName:       "HelloReply",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.HelloReply",
		},
		Contract: MethodContractPlan{
			Native: NativeContractPlan{
				RequestFields: []FieldPlan{{
					Name:     "name",
					GoName:   "Name",
					FullName: "test.v1.HelloRequest.name",
					Kind:     FieldKindString,
					Native: NativeFieldPlan{
						Kind:  NativeFieldKindString,
						Shape: NativeABIShapeScalar,
					},
				}},
				ResponseFields: []FieldPlan{{
					Name:     "message",
					GoName:   "Message",
					FullName: "test.v1.HelloReply.message",
					Kind:     FieldKindString,
					Native: NativeFieldPlan{
						Kind:  NativeFieldKindString,
						Shape: NativeABIShapeScalar,
					},
				}},
			},
			Message: MessageContractPlan{
				RequestType: MethodIOPlan{
					GoName:       "HelloRequest",
					GoImportPath: "example.com/test/v1",
					FullName:     "test.v1.HelloRequest",
				},
				ResponseType: MethodIOPlan{
					GoName:       "HelloReply",
					GoImportPath: "example.com/test/v1",
					FullName:     "test.v1.HelloReply",
				},
			},
		},
	}
	return BuildStreamingPlan(method, serviceName)
}
