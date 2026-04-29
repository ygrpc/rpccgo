package generator

import (
	"os/exec"
	"strings"
	"testing"
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
		"callbacks *C.AllServiceCGONativeServerCallbacks",
		"func RegisterAllServiceCGONativeServer(callbacks *C.AllServiceCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"return registerAllServiceActiveServer(rpcruntime.ServerKindCGONative, &allServiceCGONativeAdapter{callbacks: callbacks})",
		"type AllServiceGoCGONativeServerCallbacks struct {",
		"func RegisterAllServiceGoCGONativeServerForTesting(callbacks *AllServiceGoCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		`errors.New("rpccgo: AllService cgo native server callbacks are nil")`,
		`errors.New("rpccgo: AllService cgo native server unary callback is missing")`,
		`errors.New("rpccgo: cgo native server streaming is not implemented")`,
		"callback := a.callbacks.Unary",
		"errID := int32(C.callAllServiceUnaryCGONativeUnaryCallback(callback, input, output))",
		"return nil, allServiceCGONativeServerErrorFromID(errID)",
		"_, NamePtr, err := rpcruntime.PinString(req.Name)",
		"pinned = append(pinned, NamePtr)",
		"rpcruntime.Release(pinned[i])",
		"Payload := rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(uintptr(output.PayloadPtr))), int32(output.PayloadLen), output.PayloadOwnership > 0)",
		"resp.Payload = Payload.SafeBytes()",
		"func cleanupAllServiceUnaryCGONativeUnaryResponse(output *C.AllServiceUnaryCGONativeUnaryResponse) {",
		`_ = rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.PayloadPtr)), true, "test.v1.AllReply.payload")`,
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
