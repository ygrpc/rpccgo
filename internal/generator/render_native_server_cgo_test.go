package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRenderNativeServerCGODefinesUnaryCallbackTableAdapterAndRegistration(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/stage1_acceptance.all_service.server.cgo.rpccgo.go"
	for _, fragment := range []string{
		`import "C"`,
		"typedef struct AllServiceUnaryCGONativeUnaryRequest {",
		"uintptr_t NamePtr;",
		"int32_t NameLen;",
		"int8_t Enabled;",
		"typedef struct AllServiceCGONativeServerCallbacks {",
		"AllServiceUnaryCGONativeUnaryCallback Unary;",
		"static inline int32_t callAllServiceUnaryCGONativeUnaryCallback",
		`rpcruntime "rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"type allServiceCGONativeAdapter struct {",
		"callbacks C.AllServiceCGONativeServerCallbacks",
		"func RegisterAllServiceCGONativeServer(callbacks *C.AllServiceCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"callbacksCopy := *callbacks",
		"return registerAllServiceActiveServer(rpcruntime.ServerKindCGONative, &allServiceCGONativeAdapter{callbacks: callbacksCopy})",
		"type AllServiceGoCGONativeServerCallbacks struct {",
		"func RegisterAllServiceGoCGONativeServerForTesting(callbacks *AllServiceGoCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		`errors.New("rpccgo: AllService cgo native server callbacks are nil")`,
		`errors.New("rpccgo: AllService cgo native server unary callback is missing")`,
		`errors.New("rpccgo: cgo native server streaming is not implemented")`,
		"callback := a.callbacks.Unary",
		"errID := int32(C.callAllServiceUnaryCGONativeUnaryCallback(callback, input, output))",
		"callbackErr := allServiceCGONativeServerErrorFromID(errID)",
		"return nil, errors.Join(callbackErr, cleanupErr)",
		"_, NamePtr, err := rpcruntime.PinString(req.Name)",
		"pinned = append(pinned, NamePtr)",
		"rpcruntime.Release(pinned[i])",
		"Payload := rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(uintptr(output.PayloadPtr))), int32(output.PayloadLen), false)",
		"resp.Payload = Payload.SafeBytes()",
		"func cleanupAllServiceUnaryCGONativeUnaryResponse(output *C.AllServiceUnaryCGONativeUnaryResponse) error {",
		`if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.PayloadPtr)), true, "test.v1.AllReply.payload"); err != nil {`,
		"cleanupErr = errors.Join(cleanupErr, err)",
		"func allServiceCGONativeServerErrorFromID(errID int32) error {",
		"rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))",
		"//export StoreAllServiceCGONativeServerErrorTextForExport",
		"func StoreAllServiceCGONativeServerErrorTextForExport(text *C.char, textLen C.int32_t) C.int32_t {",
		"return C.int32_t(rpcruntime.StoreError(errors.New(string(data))))",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf")
}

func TestRenderNativeServerCGOScalarOnlyGeneratedSourceCompiles(t *testing.T) {
	file := nativeServerScalarOnlyFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/native_scalar.scalar.server.cgo.rpccgo.go"
	assertGeneratedContentContains(t, plugin, cgoServerFile, `unsafe "unsafe"`)
	assertGeneratedContentContains(t, plugin, cgoServerFile, "func StoreScalarCGONativeServerErrorTextForExport(text *C.char, textLen C.int32_t) C.int32_t {")

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".server.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
	})
	writeNativeServerCGOTestFile(t, filepath.Join(tmp, "test/v1/native_scalar_stubs.go"), `package testv1

type ScalarRequest struct {
	Enabled bool
	Count int32
}

type ScalarReply struct {
	Accepted bool
	Count int32
}
`)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scalar-only generated cgo native server go test failed: %v\n%s", err, out)
	}
}

func TestRenderNativeServerCGORejectsGeneratedSymbolCollisions(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := nativeServerCGOCollisionTestFilePlan("Greeter", []MethodPlan{{
		Name:      "SayHello",
		GoName:    "SayHello",
		FullName:  "test.v1.Greeter.SayHello",
		Streaming: StreamingKindUnary,
		Request:   MethodIOPlan{GoName: "GreeterSayHelloCGONativeUnaryRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.GreeterSayHelloCGONativeUnaryRequest"},
		Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
	}})

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want cgo native server symbol collision")
	}
	if got := err.Error(); !strings.Contains(got, "GreeterSayHelloCGONativeUnaryRequest") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want collision for cgo request type", got)
	}
}

func TestRenderNativeServerCGORejectsPackageAndSiblingSymbolCollisions(t *testing.T) {
	t.Run("package enum collides with callback table", func(t *testing.T) {
		plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
		plan := nativeServerCGOCollisionTestFilePlan("Greeter", []MethodPlan{{
			Name:      "SayHello",
			GoName:    "SayHello",
			FullName:  "test.v1.Greeter.SayHello",
			Streaming: StreamingKindUnary,
			Request:   MethodIOPlan{GoName: "HelloRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloRequest"},
			Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
		}})
		plan.TopLevelSymbols = []TopLevelSymbolPlan{{
			GoName:   "GreeterCGONativeServerCallbacks",
			FullName: "test.v1.GreeterCGONativeServerCallbacks",
			Kind:     TopLevelSymbolKindEnum,
		}}

		err := RenderNativeStageFiles(plugin, plan)
		if err == nil {
			t.Fatal("RenderNativeStageFiles() error = nil, want package symbol collision")
		}
		if got := err.Error(); !strings.Contains(got, "GreeterCGONativeServerCallbacks") || !strings.Contains(got, "collides") {
			t.Fatalf("RenderNativeStageFiles() error = %q, want package collision", got)
		}
	})

	t.Run("sibling service collides with generated helper", func(t *testing.T) {
		plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
		plan := nativeServerCGOCollisionTestFilePlan("AllService", []MethodPlan{{
			Name:      "Unary",
			GoName:    "Unary",
			FullName:  "test.v1.AllService.Unary",
			Streaming: StreamingKindUnary,
			Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
			Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
		}})
		plan.Services = append(plan.Services, ServicePlan{
			Name:     "All",
			GoName:   "All",
			FullName: "test.v1.All",
			Methods: []MethodPlan{{
				Name:      "ServiceUnary",
				GoName:    "ServiceUnary",
				FullName:  "test.v1.All.ServiceUnary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "OtherRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.OtherRequest"},
				Response:  MethodIOPlan{GoName: "OtherReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.OtherReply"},
			}},
			NativeFileFamily: NativeFileFamilyPlan{
				CGONativeServer: GeneratedFilePlan{Filename: "test/v1/collision_sibling.server.cgo.rpccgo.go", Enabled: true},
			},
		})

		err := RenderNativeStageFiles(plugin, plan)
		if err == nil {
			t.Fatal("RenderNativeStageFiles() error = nil, want sibling symbol collision")
		}
		if got := err.Error(); !strings.Contains(got, "AllServiceUnaryCGONativeUnaryRequest") || !strings.Contains(got, "collides") {
			t.Fatalf("RenderNativeStageFiles() error = %q, want sibling collision", got)
		}
	})
}

func TestRenderNativeServerCGOGeneratedSourceCompiles(t *testing.T) {
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
			strings.Contains(name, ".server.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
	})
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated cgo native server go test failed: %v\n%s", err, out)
	}
}

func nativeServerScalarOnlyFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_scalar.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("ScalarRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("enabled", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("count", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
			{
				Name: proto.String("ScalarReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("count", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Scalar"),
			Method: []*descriptorpb.MethodDescriptorProto{
				methodDescriptor("Unary", ".test.v1.ScalarRequest", ".test.v1.ScalarReply", false, false),
			},
		}},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
			Path:            []int32{6, 0},
			Span:            []int32{0, 0, 0},
			LeadingComments: proto.String("@rpccgo: native\n"),
		}}},
	}
}

func writeNativeServerCGOTestFile(t *testing.T, target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

func nativeServerCGOCollisionTestFilePlan(serviceName string, methods []MethodPlan) FilePlan {
	return FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:     serviceName,
			GoName:   serviceName,
			FullName: "test.v1." + serviceName,
			Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenNative}},
			Methods:  methods,
			NativeFileFamily: NativeFileFamilyPlan{
				CGONativeServer: GeneratedFilePlan{Filename: "test/v1/collision.server.cgo.rpccgo.go", Enabled: true},
			},
		}},
	}
}
