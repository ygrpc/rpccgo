package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestBuildDescriptorPlanBuildsServicesAndMethods(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", descriptorPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	if plan.ProtoPath != "test/v1/planner.proto" {
		t.Fatalf("ProtoPath = %q, want %q", plan.ProtoPath, "test/v1/planner.proto")
	}
	if plan.GoPackageName != "testv1" {
		t.Fatalf("GoPackageName = %q, want %q", plan.GoPackageName, "testv1")
	}
	if plan.GoImportPath != "example.com/test/v1" {
		t.Fatalf("GoImportPath = %q, want %q", plan.GoImportPath, "example.com/test/v1")
	}
	if len(plan.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(plan.Services))
	}

	greeter := plan.Services[0]
	if greeter.Name != "Greeter" || greeter.GoName != "Greeter" || greeter.FullName != "test.v1.Greeter" {
		t.Fatalf("Greeter identity = (%q, %q, %q), want descriptor identity", greeter.Name, greeter.GoName, greeter.FullName)
	}
	assertAdapterTokens(t, greeter.Adapters, []AdapterToken{
		AdapterTokenMessageConnect,
		AdapterTokenNative,
	})
	if len(greeter.Methods) != 2 {
		t.Fatalf("Greeter methods = %d, want 2", len(greeter.Methods))
	}
	assertMethodPlan(t, greeter.Methods[0], MethodPlan{
		Name:      "SayHello",
		GoName:    "SayHello",
		FullName:  "test.v1.Greeter.SayHello",
		Streaming: StreamingKindUnary,
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
	})
	assertMethodPlan(t, greeter.Methods[1], MethodPlan{
		Name:      "Upload",
		GoName:    "Upload",
		FullName:  "test.v1.Greeter.Upload",
		Streaming: StreamingKindClientStreaming,
		Request: MethodIOPlan{
			GoName:       "UploadRequest",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.UploadRequest",
		},
		Response: MethodIOPlan{
			GoName:       "UploadReply",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.UploadReply",
		},
	})

	health := plan.Services[1]
	if health.Name != "Health" || health.GoName != "Health" || health.FullName != "test.v1.Health" {
		t.Fatalf("Health identity = (%q, %q, %q), want descriptor identity", health.Name, health.GoName, health.FullName)
	}
	assertAdapterTokens(t, health.Adapters, []AdapterToken{AdapterTokenMessageConnect})
	if len(health.Methods) != 1 {
		t.Fatalf("Health methods = %d, want 1", len(health.Methods))
	}
	assertMethodPlan(t, health.Methods[0], MethodPlan{
		Name:      "Check",
		GoName:    "Check",
		FullName:  "test.v1.Health.Check",
		Streaming: StreamingKindUnary,
		Request: MethodIOPlan{
			GoName:       "HealthRequest",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.HealthRequest",
		},
		Response: MethodIOPlan{
			GoName:       "HealthReply",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.HealthReply",
		},
	})
}

func TestBuildDescriptorPlanKeepsImportedMethodGoIdent(t *testing.T) {
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/imported.proto",
		commonTypesTestFile(), importedMethodTestFile())

	plan, err := BuildDescriptorPlan(findTestProtoFile(t, plugin, "test/v1/imported.proto"))
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}
	method := plan.Services[0].Methods[0]
	assertMethodPlan(t, method, MethodPlan{
		Name:      "UseCommon",
		GoName:    "UseCommon",
		FullName:  "test.v1.Imported.UseCommon",
		Streaming: StreamingKindUnary,
		Request: MethodIOPlan{
			GoName:       "CommonRequest",
			GoImportPath: "example.com/common/v1",
			FullName:     "common.v1.CommonRequest",
		},
		Response: MethodIOPlan{
			GoName:       "CommonReply",
			GoImportPath: "example.com/common/v1",
			FullName:     "common.v1.CommonReply",
		},
	})
}

func TestBuildDescriptorPlanRejectsInvalidServiceAnnotation(t *testing.T) {
	file := descriptorPlanTestFile()
	file.SourceCodeInfo.Location[0].LeadingComments = proto.String("@rpccgo: msg-conenct\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := BuildDescriptorPlan(plugin.Files[0])
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want invalid annotation error")
	}
	if got, want := err.Error(), `service test.v1.Greeter: unknown @rpccgo token "msg-conenct"`; !strings.Contains(got, want) {
		t.Fatalf("BuildDescriptorPlan() error = %q, want substring %q", got, want)
	}
}

func TestMethodStreamingPlan(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", streamingPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}
	if len(plan.Services) != 1 {
		t.Fatalf("Services = %d, want 1", len(plan.Services))
	}

	methods := plan.Services[0].Methods
	if len(methods) != 4 {
		t.Fatalf("Methods = %d, want 4", len(methods))
	}
	assertMethodStreaming(t, methods[0], "Unary", StreamingKindUnary)
	assertMethodStreaming(t, methods[1], "ClientStream", StreamingKindClientStreaming)
	assertMethodStreaming(t, methods[2], "ServerStream", StreamingKindServerStreaming)
	assertMethodStreaming(t, methods[3], "BidiStream", StreamingKindBidiStreaming)
}

func assertMethodPlan(t *testing.T, got MethodPlan, want MethodPlan) {
	t.Helper()

	if got.Name != want.Name || got.GoName != want.GoName || got.FullName != want.FullName {
		t.Fatalf("Method identity = (%q, %q, %q), want (%q, %q, %q)",
			got.Name, got.GoName, got.FullName, want.Name, want.GoName, want.FullName)
	}
	if got.Streaming != want.Streaming {
		t.Fatalf("%s Streaming = %v, want %v", got.Name, got.Streaming, want.Streaming)
	}
	if got.Request != want.Request {
		t.Fatalf("%s Request = %#v, want %#v", got.Name, got.Request, want.Request)
	}
	if got.Response != want.Response {
		t.Fatalf("%s Response = %#v, want %#v", got.Name, got.Response, want.Response)
	}
}

func assertMethodStreaming(t *testing.T, got MethodPlan, name string, want StreamingKind) {
	t.Helper()

	if got.Name != name {
		t.Fatalf("Method name = %q, want %q", got.Name, name)
	}
	if got.Streaming != want {
		t.Fatalf("%s Streaming = %v, want %v", got.Name, got.Streaming, want)
	}
}

func findTestProtoFile(t *testing.T, plugin *protogen.Plugin, path string) *protogen.File {
	t.Helper()

	for _, file := range plugin.Files {
		if file.Desc.Path() == path {
			return file
		}
	}
	t.Fatalf("file %q not found in plugin", path)
	return nil
}

func descriptorPlanTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/planner.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("HelloRequest")},
			{Name: proto.String("HelloReply")},
			{Name: proto.String("UploadRequest")},
			{Name: proto.String("UploadReply")},
			{Name: proto.String("HealthRequest")},
			{Name: proto.String("HealthReply")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("SayHello", ".test.v1.HelloRequest", ".test.v1.HelloReply", false, false),
					methodDescriptor("Upload", ".test.v1.UploadRequest", ".test.v1.UploadReply", true, false),
				},
			},
			{
				Name: proto.String("Health"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Check", ".test.v1.HealthRequest", ".test.v1.HealthReply", false, false),
				},
			},
		},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{
			Location: []*descriptorpb.SourceCodeInfo_Location{
				{
					Path:            []int32{6, 0},
					Span:            []int32{11, 0, 19},
					LeadingComments: proto.String("@rpccgo: native\n"),
				},
			},
		},
	}
}

func streamingPlanTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/streaming.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("StreamRequest")},
			{Name: proto.String("StreamReply")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Streamer"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Unary", ".test.v1.StreamRequest", ".test.v1.StreamReply", false, false),
					methodDescriptor("ClientStream", ".test.v1.StreamRequest", ".test.v1.StreamReply", true, false),
					methodDescriptor("ServerStream", ".test.v1.StreamRequest", ".test.v1.StreamReply", false, true),
					methodDescriptor("BidiStream", ".test.v1.StreamRequest", ".test.v1.StreamReply", true, true),
				},
			},
		},
	}
}

func commonTypesTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("common/v1/types.proto"),
		Package: proto.String("common.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/common/v1;commonv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("CommonRequest")},
			{Name: proto.String("CommonReply")},
		},
	}
}

func importedMethodTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/v1/imported.proto"),
		Package:    proto.String("test.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"common/v1/types.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Imported"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("UseCommon", ".common.v1.CommonRequest", ".common.v1.CommonReply", false, false),
				},
			},
		},
	}
}

func methodDescriptor(name, input, output string, clientStreaming, serverStreaming bool) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:            proto.String(name),
		InputType:       proto.String(input),
		OutputType:      proto.String(output),
		ClientStreaming: proto.Bool(clientStreaming),
		ServerStreaming: proto.Bool(serverStreaming),
	}
}
