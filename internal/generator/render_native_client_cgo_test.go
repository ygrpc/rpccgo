package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRenderNativeClientCGODefinesUnaryExportSurface(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/stage1_acceptance.all_service.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		`rpcruntime "rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"type AllServiceUnaryNativeUnaryInput struct {",
		"NamePtr       uintptr",
		"NameLen       int32",
		"NameOwnership int32",
		"Enabled       int8",
		"Child         uintptr",
		"type AllServiceUnaryNativeUnaryOutput struct {",
		"Accepted   int8",
		"PayloadPtr uintptr",
		"PayloadLen int32",
		"func CallAllServiceUnaryNativeUnary(ctx context.Context, input *AllServiceUnaryNativeUnaryInput, output *AllServiceUnaryNativeUnaryOutput) int32 {",
		"err = allServiceDispatcher.Invoke(ctx, func(ctx context.Context, snapshot rpcruntime.AdapterSnapshot[AllServiceNativeAdapter]) error {",
		"resp, callErr = snapshot.Adapter.Unary(ctx, req)",
		"return int32(rpcruntime.StoreError(err))",
		"return int32(rpcruntime.StoreError(errors.New(\"rpccgo: native unary client input is nil\")))",
		"return int32(rpcruntime.StoreError(errors.New(\"rpccgo: native unary client output is nil\")))",
		"return int32(rpcruntime.StoreError(errors.New(\"rpccgo: native unary server returned nil response\")))",
		"if _, err := rpcruntime.LengthFromInt32(input.NameLen); err != nil {",
		"Name := rpcruntime.NewRpcString((*byte)(unsafe.Pointer(input.NamePtr)), input.NameLen, input.NameOwnership > 0)",
		"req.Name = Name.SafeString()",
		"if err := Name.Release(); err != nil {",
		"req.Enabled = input.Enabled != 0",
		"return nil, allServiceNativeClientUnsupportedField",
		"var AcceptedValue int8",
		"AcceptedValue = 1",
		"PayloadLen, err := rpcruntime.LengthToInt32(len(resp.Payload))",
		"PayloadPtr, err := rpcruntime.PinBytes(resp.Payload)",
		"output.Accepted = AcceptedValue",
		"output.PayloadPtr = PayloadPtr",
		"output.PayloadLen = PayloadLen",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf")
}

func TestRenderNativeClientCGOSupportsEnumAsInt32(t *testing.T) {
	file := nativeClientEnumFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/native_enum.enum_service.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		"State int32",
		"req.State = State(input.State)",
		"StateValue := int32(resp.State)",
		"output.State = StateValue",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
}

func TestRenderNativeClientCGORejectsGeneratedHelperCollisions(t *testing.T) {
	tests := []struct {
		name      string
		method    MethodPlan
		wantError string
	}{
		{
			name: "decoder collides with request message",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "decodeAllServiceUnaryNativeUnaryRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.decodeAllServiceUnaryNativeUnaryRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
			},
			wantError: "decodeAllServiceUnaryNativeUnaryRequest",
		},
		{
			name: "encoder collides with response message",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "encodeAllServiceUnaryNativeUnaryResponse", GoImportPath: "example.com/test/v1", FullName: "test.v1.encodeAllServiceUnaryNativeUnaryResponse"},
			},
			wantError: "encodeAllServiceUnaryNativeUnaryResponse",
		},
		{
			name: "string suffix field collision",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
				NativeContract: NativeContractPlan{RequestFields: []FieldPlan{
					{GoName: "Name", FullName: "test.v1.AllRequest.name", Kind: FieldKindString, Native: NativeFieldPlan{Kind: NativeFieldKindString, Shape: NativeABIShapeScalar}},
					{GoName: "NamePtr", FullName: "test.v1.AllRequest.name_ptr", Kind: FieldKindSignedInt32, Native: NativeFieldPlan{Kind: NativeFieldKindSignedNumeric, Shape: NativeABIShapeScalar}},
				}},
			},
			wantError: "NamePtr",
		},
		{
			name: "bytes response suffix field collision",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
				NativeContract: NativeContractPlan{ResponseFields: []FieldPlan{
					{GoName: "Payload", FullName: "test.v1.AllReply.payload", Kind: FieldKindBytes, Native: NativeFieldPlan{Kind: NativeFieldKindBytes, Shape: NativeABIShapeScalar}},
					{GoName: "PayloadLen", FullName: "test.v1.AllReply.payload_len", Kind: FieldKindSignedInt32, Native: NativeFieldPlan{Kind: NativeFieldKindSignedNumeric, Shape: NativeABIShapeScalar}},
				}},
			},
			wantError: "PayloadLen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
			err := RenderNativeStageFiles(plugin, nativeClientCollisionTestFilePlan(tt.method))
			if err == nil {
				t.Fatal("RenderNativeStageFiles() error = nil, want native client cgo symbol collision")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantError) || !strings.Contains(got, "collides") {
				t.Fatalf("RenderNativeStageFiles() error = %q, want collision for %q", got, tt.wantError)
			}
		})
	}
}

func TestRenderNativeClientCGOGeneratedSourceCompiles(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
	})
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated native client go test failed: %v\n%s", err, out)
	}
}

func writeNativeGeneratedModule(t *testing.T, root string, plugin *protogen.Plugin, include func(string) bool) {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !include(name) {
			continue
		}
		target := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir generated dir: %v", err)
		}
		if err := os.WriteFile(target, []byte(generated.GetContent()), 0o644); err != nil {
			t.Fatalf("write generated file %s: %v", name, err)
		}
	}
}

func nativeClientEnumFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_enum.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			stateEnumDescriptor(),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("EnumRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("state", 1, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.State"),
				},
			},
			{
				Name: proto.String("EnumReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("state", 1, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.State"),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("EnumService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Check", ".test.v1.EnumRequest", ".test.v1.EnumReply", false, false),
				},
			},
		},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{
			{
				Path:            []int32{6, 0},
				Span:            []int32{0, 0, 0},
				LeadingComments: proto.String("@rpccgo: native\n"),
			},
		}},
	}
}

func nativeClientCollisionTestFilePlan(method MethodPlan) FilePlan {
	return FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:     "AllService",
			GoName:   "AllService",
			FullName: "test.v1.AllService",
			Methods:  []MethodPlan{method},
			NativeFileFamily: NativeFileFamilyPlan{
				CGONativeClient: GeneratedFilePlan{Filename: "test/v1/collision.client.cgo.rpccgo.go", Enabled: true},
			},
		}},
	}
}
