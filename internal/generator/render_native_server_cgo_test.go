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
		`rpcruntime "rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"type AllServiceCGONativeServerCallbacks struct {",
		"Unary func(ctx context.Context, input *AllServiceUnaryCGONativeUnaryRequest, output *AllServiceUnaryCGONativeUnaryResponse) int32",
		"type allServiceCGONativeAdapter struct {",
		"callbacks *AllServiceCGONativeServerCallbacks",
		"func RegisterAllServiceCGONativeServer(callbacks *AllServiceCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"return registerAllServiceActiveServer(rpcruntime.ServerKindCGONative, &allServiceCGONativeAdapter{callbacks: callbacks})",
		`errors.New("rpccgo: AllService cgo native server callbacks are nil")`,
		`errors.New("rpccgo: AllService cgo native server unary callback is missing")`,
		`errors.New("rpccgo: cgo native server streaming is not implemented")`,
		"callback := a.callbacks.Unary",
		"errID := callback(ctx, input, output)",
		"return nil, allServiceCGONativeServerErrorFromID(errID)",
		"type AllServiceUnaryCGONativeUnaryRequest struct {",
		"NamePtr uintptr",
		"NameLen int32",
		"Enabled int8",
		"type AllServiceUnaryCGONativeUnaryResponse struct {",
		"PayloadPtr       uintptr",
		"PayloadLen       int32",
		"PayloadOwnership int32",
		"_, NamePtr, err := rpcruntime.PinString(req.Name)",
		"pinned = append(pinned, NamePtr)",
		"rpcruntime.Release(pinned[i])",
		"Payload := rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen, output.PayloadOwnership > 0)",
		"resp.Payload = Payload.SafeBytes()",
		"if err := Payload.Release(); err != nil {",
		"func allServiceCGONativeServerErrorFromID(errID int32) error {",
		"rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))",
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
