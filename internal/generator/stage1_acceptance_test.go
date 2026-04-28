package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestStage1AcceptanceBuildsCompleteServicePlans(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", stage1AcceptanceFile())

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("Generate() returned %d plans, want 1", len(plans))
	}
	if len(plugin.Response().GetFile()) != 0 {
		t.Fatalf("Generate() must not emit renderer, dispatcher, adapter, or example files during Stage 1")
	}

	plan := plans[0]
	if plan.ProtoPath != "test/v1/stage1_acceptance.proto" {
		t.Fatalf("ProtoPath = %q, want stage1 acceptance proto", plan.ProtoPath)
	}
	if len(plan.Services) != 7 {
		t.Fatalf("Services = %d, want 7", len(plan.Services))
	}

	wantServices := map[string]struct {
		tokens     []AdapterToken
		needsCodec bool
	}{
		"DefaultService": {
			tokens: []AdapterToken{AdapterTokenMessageConnect},
		},
		"ConnectService": {
			tokens: []AdapterToken{AdapterTokenMessageConnect},
		},
		"GrpcService": {
			tokens: []AdapterToken{AdapterTokenMessageGRPC},
		},
		"MessageService": {
			tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC},
		},
		"ConnectNativeService": {
			tokens:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative},
			needsCodec: true,
		},
		"AllService": {
			tokens:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC, AdapterTokenNative},
			needsCodec: true,
		},
		"NativeOnlyService": {
			tokens:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative},
			needsCodec: true,
		},
	}

	services := servicesByName(t, plan.Services, "DefaultService", "ConnectService", "GrpcService", "MessageService", "ConnectNativeService", "AllService", "NativeOnlyService")
	for name, want := range wantServices {
		service := services[name]
		assertAdapterTokens(t, service.Adapters, want.tokens)
		if service.NeedsCodec != want.needsCodec {
			t.Fatalf("%s NeedsCodec = %v, want %v", name, service.NeedsCodec, want.needsCodec)
		}
		for _, method := range service.Methods {
			if method.NeedsCodec != want.needsCodec {
				t.Fatalf("%s.%s NeedsCodec = %v, want %v", name, method.Name, method.NeedsCodec, want.needsCodec)
			}
			assertStage1ContractsPresent(t, method)
		}
	}

	methods := methodsByName(t, services["AllService"].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertStage1Method(t, methods["Unary"], StreamingKindUnary, LifecyclePlan{})
	assertStage1Method(t, methods["ClientStream"], StreamingKindClientStreaming, LifecyclePlan{
		HasStart:        true,
		HasSend:         true,
		HasFinish:       true,
		HasCancel:       true,
		CancelFinalizes: true,
		TerminalKind:    LifecycleTerminalFinishResult,
	})
	assertStage1Method(t, methods["ServerStream"], StreamingKindServerStreaming, LifecyclePlan{
		HasStart:        true,
		HasCancel:       true,
		CancelFinalizes: true,
		HasOnRead:       true,
		HasOnDone:       true,
		TerminalKind:    LifecycleTerminalOnDone,
	})
	assertStage1Method(t, methods["BidiStream"], StreamingKindBidiStreaming, LifecyclePlan{
		HasStart:        true,
		HasSend:         true,
		HasCloseSend:    true,
		HasCancel:       true,
		CancelFinalizes: true,
		HasOnRead:       true,
		HasOnDone:       true,
		TerminalKind:    LifecycleTerminalOnDone,
	})
}

func TestStage1AcceptanceRejectsBadServiceTokens(t *testing.T) {
	tests := []struct {
		name      string
		comment   string
		wantError string
	}{
		{
			name:      "unknown token",
			comment:   "@rpccgo: msg-connect|bogus\n",
			wantError: `unknown @rpccgo token "bogus"`,
		},
		{
			name:      "spelling error",
			comment:   "@rpccgo: msg-conenct\n",
			wantError: `unknown @rpccgo token "msg-conenct"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := stage1AcceptanceFile()
			file.SourceCodeInfo.Location[1].LeadingComments = proto.String(tt.comment)
			plugin := newTestPlugin(t, "paths=source_relative", file)

			_, err := Generate(plugin)
			if err == nil {
				t.Fatal("Generate() error = nil, want invalid token error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Generate() error = %q, want substring %q", err.Error(), tt.wantError)
			}
		})
	}
}

func assertStage1ContractsPresent(t *testing.T, method MethodPlan) {
	t.Helper()

	if method.Request.GoName == "" || method.Response.GoName == "" {
		t.Fatalf("%s request/response descriptor metadata is missing", method.FullName)
	}
	if method.MessageContract.RequestType != method.Request || method.MessageContract.ResponseType != method.Response {
		t.Fatalf("%s MessageContract = %#v, want request/response IO metadata", method.FullName, method.MessageContract)
	}
	if len(method.NativeContract.RequestFields) == 0 || len(method.NativeContract.ResponseFields) == 0 {
		t.Fatalf("%s NativeContract missing request or response fields", method.FullName)
	}
	if len(method.RequestBody) != len(method.NativeContract.RequestFields) || len(method.ResponseBody) != len(method.NativeContract.ResponseFields) {
		t.Fatalf("%s request/response bodies do not match native contract fields", method.FullName)
	}
}

func assertStage1Method(t *testing.T, method MethodPlan, streaming StreamingKind, lifecycle LifecyclePlan) {
	t.Helper()

	if method.Streaming != streaming {
		t.Fatalf("%s Streaming = %v, want %v", method.FullName, method.Streaming, streaming)
	}
	if method.Lifecycle != lifecycle {
		t.Fatalf("%s Lifecycle = %#v, want %#v", method.FullName, method.Lifecycle, lifecycle)
	}
	assertStage1ContractsPresent(t, method)
}

func servicesByName(t *testing.T, services []ServicePlan, wantNames ...string) map[string]ServicePlan {
	t.Helper()

	if len(services) != len(wantNames) {
		t.Fatalf("Services = %d, want %d", len(services), len(wantNames))
	}
	byName := make(map[string]ServicePlan, len(services))
	for _, service := range services {
		if _, exists := byName[service.Name]; exists {
			t.Fatalf("duplicate service name %q", service.Name)
		}
		byName[service.Name] = service
	}
	for _, name := range wantNames {
		if _, exists := byName[name]; !exists {
			t.Fatalf("service %q not found in file plan", name)
		}
	}
	return byName
}

func stage1AcceptanceFile() *descriptorpb.FileDescriptorProto {
	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/stage1_acceptance.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			stage1RequestDescriptor("DefaultRequest"),
			stage1ReplyDescriptor("DefaultReply"),
			stage1RequestDescriptor("ConnectRequest"),
			stage1ReplyDescriptor("ConnectReply"),
			stage1RequestDescriptor("GrpcRequest"),
			stage1ReplyDescriptor("GrpcReply"),
			stage1RequestDescriptor("MessageRequest"),
			stage1ReplyDescriptor("MessageReply"),
			stage1RequestDescriptor("ConnectNativeRequest"),
			stage1ReplyDescriptor("ConnectNativeReply"),
			stage1RequestDescriptor("AllRequest"),
			stage1ReplyDescriptor("AllReply"),
			stage1RequestDescriptor("NativeOnlyRequest"),
			stage1ReplyDescriptor("NativeOnlyReply"),
			childMessageDescriptor(),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			stage1Service("DefaultService", "Default", ".test.v1.DefaultRequest", ".test.v1.DefaultReply", false),
			stage1Service("ConnectService", "Connect", ".test.v1.ConnectRequest", ".test.v1.ConnectReply", false),
			stage1Service("GrpcService", "Grpc", ".test.v1.GrpcRequest", ".test.v1.GrpcReply", false),
			stage1Service("MessageService", "Message", ".test.v1.MessageRequest", ".test.v1.MessageReply", false),
			stage1Service("ConnectNativeService", "ConnectNative", ".test.v1.ConnectNativeRequest", ".test.v1.ConnectNativeReply", false),
			stage1Service("AllService", "All", ".test.v1.AllRequest", ".test.v1.AllReply", true),
			stage1Service("NativeOnlyService", "NativeOnly", ".test.v1.NativeOnlyRequest", ".test.v1.NativeOnlyReply", false),
		},
	}
	file.SourceCodeInfo = stage1ServiceComments([]string{
		"",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-grpc\n",
		"@rpccgo: msg-connect|msg-grpc\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: msg-connect|msg-grpc|native\n",
		"@rpccgo: native\n",
	})
	return file
}

func stage1RequestDescriptor(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String(name),
		Field: []*descriptorpb.FieldDescriptorProto{
			fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("enabled", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("child", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.Child"),
		},
	}
}

func stage1ReplyDescriptor(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String(name),
		Field: []*descriptorpb.FieldDescriptorProto{
			fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
		},
	}
}

func stage1Service(name, methodPrefix, input, output string, streaming bool) *descriptorpb.ServiceDescriptorProto {
	if !streaming {
		return &descriptorpb.ServiceDescriptorProto{
			Name: proto.String(name),
			Method: []*descriptorpb.MethodDescriptorProto{
				methodDescriptor(methodPrefix+"Unary", input, output, false, false),
			},
		}
	}
	return &descriptorpb.ServiceDescriptorProto{
		Name: proto.String(name),
		Method: []*descriptorpb.MethodDescriptorProto{
			methodDescriptor("Unary", input, output, false, false),
			methodDescriptor("ClientStream", input, output, true, false),
			methodDescriptor("ServerStream", input, output, false, true),
			methodDescriptor("BidiStream", input, output, true, true),
		},
	}
}

func stage1ServiceComments(comments []string) *descriptorpb.SourceCodeInfo {
	locations := make([]*descriptorpb.SourceCodeInfo_Location, 0, len(comments))
	for index, comment := range comments {
		if comment == "" {
			continue
		}
		locations = append(locations, &descriptorpb.SourceCodeInfo_Location{
			Path:            []int32{6, int32(index)},
			Span:            []int32{int32(index), 0, int32(index), 1},
			LeadingComments: proto.String(comment),
		})
	}
	return &descriptorpb.SourceCodeInfo{Location: locations}
}
