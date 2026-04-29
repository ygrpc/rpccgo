package generator

import "testing"

func TestRenderRuntimeGlueImportsRPCRuntimeOnly(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/greeter.greeter.runtime.rpccgo.go"
	assertGeneratedContentContains(t, plugin, runtimeFile, `rpcruntime "rpccgo/rpcruntime"`)
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc")
}

func TestRenderRuntimeGlueDefinesServiceDispatcherAndRegistration(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/stage1_acceptance.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceNativeAdapter interface {",
		"Unary(ctx context.Context) error",
		"StartClientStream(ctx context.Context) (AllServiceNativeStreamSession, error)",
		"StartServerStream(ctx context.Context) (AllServiceNativeStreamSession, error)",
		"StartBidiStream(ctx context.Context) (AllServiceNativeStreamSession, error)",
		"type AllServiceNativeStreamSession interface {",
		"Send(ctx context.Context) error",
		"Finish(ctx context.Context) error",
		"CloseSend(ctx context.Context) error",
		"Cancel(ctx context.Context) error",
		"var allServiceDispatcher rpcruntime.Dispatcher[AllServiceNativeAdapter]",
		"func registerAllServiceActiveServer(kind rpcruntime.ServerKind, adapter AllServiceNativeAdapter) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"return allServiceDispatcher.Register(kind, rpcruntime.ServerContractNative, adapter)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueUsesRPCRuntimeStreamHandleAndHelpers(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/greeter.greeter.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func loadGreeterNativeStream(handle rpcruntime.StreamHandle) (GreeterNativeStreamSession, bool) {",
		"return rpcruntime.LoadDispatcherStream[GreeterNativeAdapter, GreeterNativeStreamSession](&greeterDispatcher, handle)",
		"func takeGreeterNativeStream(handle rpcruntime.StreamHandle) (GreeterNativeStreamSession, bool) {",
		"return rpcruntime.TakeDispatcherStream[GreeterNativeAdapter, GreeterNativeStreamSession](&greeterDispatcher, handle)",
		"func deleteGreeterNativeStream(handle rpcruntime.StreamHandle) bool {",
		"return rpcruntime.DeleteDispatcherStream[GreeterNativeAdapter](&greeterDispatcher, handle)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "rpcruntime.Handle", " handle Handle", "handle Handle")
}
