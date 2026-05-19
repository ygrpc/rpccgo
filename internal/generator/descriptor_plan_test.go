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

func TestBuildDescriptorPlanBuildsCompleteServicePlans(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", completeServicePlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	if plan.ProtoPath != "test/v1/complete_service_plan.proto" {
		t.Fatalf("ProtoPath = %q, want complete service plan proto", plan.ProtoPath)
	}
	services := servicesByName(t, plan.Services,
		"DefaultService", "ConnectService", "GrpcService", "MessageService",
		"ConnectNativeService", "AllService", "NativeOnlyService",
	)

	wantServices := map[string][]AdapterToken{
		"DefaultService":       {AdapterTokenMessageConnect},
		"ConnectService":       {AdapterTokenMessageConnect},
		"GrpcService":          {AdapterTokenMessageGRPC},
		"MessageService":       {AdapterTokenMessageConnect},
		"ConnectNativeService": {AdapterTokenMessageConnect, AdapterTokenNative},
		"AllService":           {AdapterTokenMessageConnect, AdapterTokenNative},
		"NativeOnlyService":    {AdapterTokenMessageConnect, AdapterTokenNative},
	}
	for name, wantTokens := range wantServices {
		service := services[name]
		assertAdapterTokens(t, service.Adapters, wantTokens)
		if !service.NeedsCodec {
			t.Fatalf("%s NeedsCodec = false, want true", name)
		}
		for _, method := range service.Methods {
			if !method.NeedsCodec {
				t.Fatalf("%s.%s NeedsCodec = false, want true", name, method.Name)
			}
			assertCompleteMethodContracts(t, method)
		}
	}

	methods := methodsByName(t, services["AllService"].Methods, "Unary", "ClientStream", "ServerStream", "BidiStream")
	assertMethodStreaming(t, methods["Unary"], "Unary", StreamingKindUnary)
	assertMethodStreaming(t, methods["ClientStream"], "ClientStream", StreamingKindClientStreaming)
	assertMethodStreaming(t, methods["ServerStream"], "ServerStream", StreamingKindServerStreaming)
	assertMethodStreaming(t, methods["BidiStream"], "BidiStream", StreamingKindBidiStreaming)
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

func TestBuildPackageLevelSymbolPlansIncludesNestedSymbols(t *testing.T) {
	file := simpleTestFile()
	otherFile := nativeClientPackageCollisionNestedFile("test/v1/other.proto", "example.com/test/v1;testv1", "Nested")
	request := newTestCodeGeneratorRequest("paths=source_relative", file, otherFile)
	request.FileToGenerate = []string{file.GetName(), otherFile.GetName()}
	plugin, err := ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}

	symbols := buildPackageLevelSymbolPlans(plugin.Files, "example.com/test/v1")
	if !hasTopLevelSymbol(symbols, "Decode_Nested", "test.v1.Decode.Nested", TopLevelSymbolKindMessage) {
		t.Fatalf("package-level symbols = %#v, want nested message Decode_Nested", symbols)
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

func assertCompleteMethodContracts(t *testing.T, method MethodPlan) {
	t.Helper()

	if method.Request.GoName == "" || method.Response.GoName == "" {
		t.Fatalf("%s request/response descriptor metadata is missing", method.FullName)
	}
	if method.RenderShape.Conversion.MessageToNative.Direction != ConversionDirectionMessageToNative || method.RenderShape.Conversion.NativeToMessage.Direction != ConversionDirectionNativeToMessage {
		t.Fatalf("%s MessageContract = %#v, want request/response IO metadata", method.FullName, method.RenderShape.Conversion)
	}
	if len(method.RenderShape.Conversion.MessageToNative.Native.Request) == 0 || len(method.RenderShape.Conversion.MessageToNative.Native.Response) == 0 {
		t.Fatalf("%s NativeContract missing request or response fields", method.FullName)
	}
	if method.RenderShape.Symbols.NativeAdapterMethod == "" || method.RenderShape.Errors.NativeAdapterUnavailableErr == "" {
		t.Fatalf("%s render symbols/errors are incomplete", method.FullName)
	}
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

func hasTopLevelSymbol(symbols []TopLevelSymbolPlan, goName, fullName string, kind TopLevelSymbolKind) bool {
	for _, symbol := range symbols {
		if symbol.GoName == goName && symbol.FullName == fullName && symbol.Kind == kind {
			return true
		}
	}
	return false
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

func completeServicePlanTestFile() *descriptorpb.FileDescriptorProto {
	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/complete_service_plan.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			completeServicePlanRequestDescriptor("DefaultRequest"),
			completeServicePlanReplyDescriptor("DefaultReply"),
			completeServicePlanRequestDescriptor("ConnectRequest"),
			completeServicePlanReplyDescriptor("ConnectReply"),
			completeServicePlanRequestDescriptor("GrpcRequest"),
			completeServicePlanReplyDescriptor("GrpcReply"),
			completeServicePlanRequestDescriptor("MessageRequest"),
			completeServicePlanReplyDescriptor("MessageReply"),
			completeServicePlanRequestDescriptor("ConnectNativeRequest"),
			completeServicePlanReplyDescriptor("ConnectNativeReply"),
			completeServicePlanRequestDescriptor("AllRequest"),
			completeServicePlanReplyDescriptor("AllReply"),
			completeServicePlanRequestDescriptor("NativeOnlyRequest"),
			completeServicePlanReplyDescriptor("NativeOnlyReply"),
			childMessageDescriptor(),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			completeServicePlanService("DefaultService", "Default", ".test.v1.DefaultRequest", ".test.v1.DefaultReply", false),
			completeServicePlanService("ConnectService", "Connect", ".test.v1.ConnectRequest", ".test.v1.ConnectReply", false),
			completeServicePlanService("GrpcService", "Grpc", ".test.v1.GrpcRequest", ".test.v1.GrpcReply", false),
			completeServicePlanService("MessageService", "Message", ".test.v1.MessageRequest", ".test.v1.MessageReply", false),
			completeServicePlanService("ConnectNativeService", "ConnectNative", ".test.v1.ConnectNativeRequest", ".test.v1.ConnectNativeReply", false),
			completeServicePlanService("AllService", "All", ".test.v1.AllRequest", ".test.v1.AllReply", true),
			completeServicePlanService("NativeOnlyService", "NativeOnly", ".test.v1.NativeOnlyRequest", ".test.v1.NativeOnlyReply", false),
		},
	}
	file.SourceCodeInfo = completeServicePlanServiceComments([]string{
		"",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-grpc\n",
		"@rpccgo: msg-connect\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: msg-connect|native\n",
		"@rpccgo: native\n",
	})
	return file
}

func completeServicePlanRequestDescriptor(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String(name),
		Field: []*descriptorpb.FieldDescriptorProto{
			fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("enabled", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("child", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.Child"),
		},
	}
}

func completeServicePlanReplyDescriptor(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String(name),
		Field: []*descriptorpb.FieldDescriptorProto{
			fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
		},
	}
}

func completeServicePlanService(name, methodPrefix, input, output string, streaming bool) *descriptorpb.ServiceDescriptorProto {
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

func completeServicePlanServiceComments(comments []string) *descriptorpb.SourceCodeInfo {
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
