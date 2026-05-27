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
	assertGeneratedContentContains(t, plugin, runtimeFile, `errors "errors"`)
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc")
}

func TestRenderRuntimeGlueDefinesServiceActiveSlotAndRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderNativeStageFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderNativeStageFiles() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"CGONativeClientBridge",
		"CGOMessageClientBridge",
	)
	for _, fragment := range []string{
		"type AllServiceNativeAdapter interface {",
		"Unary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error)",
		"StartClientStream(ctx context.Context) (AllServiceClientStreamNativeStreamSession, error)",
		"StartServerStream(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (AllServiceServerStreamNativeStreamSession, error)",
		"StartBidiStream(ctx context.Context) (AllServiceBidiStreamNativeStreamSession, error)",
		"type AllServiceClientStreamNativeStreamSession interface {",
		"Send(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error",
		"Finish(ctx context.Context) (bool, []byte, error)",
		"Cancel(ctx context.Context) error",
		"type AllServiceServerStreamNativeStreamSession interface {",
		"Recv(ctx context.Context) (bool, []byte, error)",
		"Done(ctx context.Context) error",
		"type AllServiceBidiStreamNativeStreamSession interface {",
		"CloseSend(ctx context.Context) error",
		"Done(ctx context.Context) error",
		"var allServiceStreamRegistry rpcruntime.StreamRegistry[*rpcruntime.StreamEntry]",
		"func registerAllServiceActiveServer(kind rpcruntime.ServerKind, adapter AllServiceNativeAdapter) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"snapshot, err := allServiceActiveSlot.Store(kind, rpcruntime.ServerContractNative, adapter)",
		"type allServiceRuntimeBridge struct {",
		"func (r allServiceRuntimeBridge) invokeNativeUnary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error) {",
		"func InvokeAllServiceNativeUnary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error) {",
		"return allServiceBridge.invokeNativeUnary(ctx, name, enabled, child)",
		"var acceptedResult bool",
		"var payloadResult []byte",
		"acceptedResult, payloadResult, err = adapter.Unary(ctx, name, enabled, child)",
		"func RegisterAllServiceCGONativeActiveServer(kind rpcruntime.ServerKind, adapter AllServiceNativeAdapter) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesMessageContractActiveSlotAndRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceMessageAdapter interface {",
		"UnaryMessage(ctx context.Context, req []byte) ([]byte, error)",
		"StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error)",
		"StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error)",
		"StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error)",
		"type AllServiceClientStreamMessageStreamSession interface {",
		"Send(ctx context.Context, req []byte) error",
		"Finish(ctx context.Context) ([]byte, error)",
		"Cancel(ctx context.Context) error",
		"type AllServiceServerStreamMessageStreamSession interface {",
		"Recv(ctx context.Context) ([]byte, error)",
		"Done(ctx context.Context) error",
		"type AllServiceBidiStreamMessageStreamSession interface {",
		"CloseSend(ctx context.Context) error",
		"func registerAllServiceMessageActiveServer(kind rpcruntime.ServerKind, adapter AllServiceMessageAdapter) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"snapshot, err := allServiceActiveSlot.Store(kind, rpcruntime.ServerContractMessage, adapter)",
		"func (r allServiceRuntimeBridge) invokeMessageUnary(ctx context.Context, req []byte) ([]byte, error) {",
		"func InvokeAllServiceMessageUnary(ctx context.Context, req []byte) ([]byte, error) {",
		"return allServiceBridge.invokeMessageUnary(ctx, req)",
		"case rpcruntime.ServerContractMessage:",
		"return nil, AllServiceMessageAdapterUnavailableErr",
		"return adapter.UnaryMessage(ctx, req)",
		"AllServiceNativeMessageConverterUnavailableErr",
		"func RegisterAllServiceCGOMessageActiveServer(kind rpcruntime.ServerKind, adapter AllServiceMessageAdapter) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"allServiceMessageDispatcher",
		"rpcruntime.Dispatcher[AllServiceNativeAdapter]",
		"rpcruntime.Dispatcher[AllServiceMessageAdapter]",
		"renderRuntimeMessageContractMismatchCheck",
		"if _, nativeErr :=",
	)
}

func TestRenderRuntimeGlueDefinesConnectDirectRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.default_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func RegisterDefaultServiceConnectHandler(handler DefaultServiceHandler) (rpcruntime.AdapterSnapshot[DefaultServiceHandler], error) {",
		"snapshot, err := defaultServiceActiveSlot.Store(rpcruntime.ServerKindConnectHandler, rpcruntime.ServerContractMessage, handler)",
		"return rpcruntime.AdapterSnapshot[DefaultServiceHandler]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: handler}, nil",
		"case rpcruntime.ServerKindConnectHandler:",
		"handler, ok := snapshot.Adapter.(DefaultServiceHandler)",
		"messageResp, err := handler.DefaultUnary(ctx, messageReq)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueRoutesNativeUnaryToConnectHandler(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.connect_native_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"case rpcruntime.ServerKindConnectHandler:",
		"handler, ok := snapshot.Adapter.(ConnectNativeServiceHandler)",
		"new(ConnectNativeRequest)",
		"proto.Unmarshal(messageReq",
		"handler.ConnectNativeUnary(ctx",
		"messageResp, err = proto.Marshal(directResp)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesGRPCDirectRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.grpc_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func RegisterGrpcServiceGRPCServer(server GrpcServiceServer) (rpcruntime.AdapterSnapshot[GrpcServiceServer], error) {",
		"snapshot, err := grpcServiceActiveSlot.Store(rpcruntime.ServerKindGRPCServer, rpcruntime.ServerContractMessage, server)",
		"return rpcruntime.AdapterSnapshot[GrpcServiceServer]{Kind: snapshot.Kind, Contract: snapshot.Contract, Version: snapshot.Version, Adapter: server}, nil",
		"case rpcruntime.ServerKindGRPCServer:",
		"server, ok := snapshot.Adapter.(GrpcServiceServer)",
		"messageResp, err := server.GrpcUnary(ctx, messageReq)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesMethodSpecificStreamFacades(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceClientStreamNativeStream struct {",
		"handle rpcruntime.StreamHandle",
		"func NewAllServiceClientStreamNativeStream(handle rpcruntime.StreamHandle) AllServiceClientStreamNativeStream {",
		"func (s AllServiceClientStreamNativeStream) Send(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error {",
		"rpcruntime.StreamRegistrySend[AllServiceClientStreamNativeStreamSession](&allServiceStreamRegistry, s.handle, func(session AllServiceClientStreamNativeStreamSession) error {",
		"func (s AllServiceClientStreamNativeStream) Finish(ctx context.Context) (bool, []byte, error) {",
		"rpcruntime.StreamRegistryFinish[AllServiceClientStreamNativeStreamSession](&allServiceStreamRegistry, s.handle, func(session AllServiceClientStreamNativeStreamSession) error {",
		"func (s AllServiceServerStreamNativeStream) Recv(ctx context.Context) (bool, []byte, error) {",
		"func (s AllServiceServerStreamNativeStream) Done(ctx context.Context) error {",
		"rpcruntime.StreamRegistryDone[AllServiceServerStreamNativeStreamSession](&allServiceStreamRegistry, s.handle, func(session AllServiceServerStreamNativeStreamSession) error {",
		"func (s AllServiceBidiStreamNativeStream) CloseSend(ctx context.Context) error {",
		"type AllServiceClientStreamMessageStream struct {",
		"func NewAllServiceClientStreamMessageStream(handle rpcruntime.StreamHandle) AllServiceClientStreamMessageStream {",
		"func (s AllServiceClientStreamMessageStream) Send(ctx context.Context, req []byte) error {",
		"rpcruntime.StreamRegistrySend[AllServiceClientStreamMessageStreamSession](&allServiceStreamRegistry, s.handle, func(session AllServiceClientStreamMessageStreamSession) error {",
		"func (s AllServiceClientStreamMessageStream) Finish(ctx context.Context) ([]byte, error) {",
		"func (s AllServiceServerStreamMessageStream) Recv(ctx context.Context) ([]byte, error) {",
		"func (s AllServiceBidiStreamMessageStream) CloseSend(ctx context.Context) error {",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile, "type AllServiceUnaryNativeStream", "type AllServiceUnaryMessageStream")
}

func TestRenderRuntimeGlueUsesRPCRuntimeStreamHandleAndCoreStreamRegistry(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func StartAllServiceNativeClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"var allServiceStreamRegistry rpcruntime.StreamRegistry[*rpcruntime.StreamEntry]",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"rpcruntime.Handle",
		" handle Handle",
		"handle Handle",
		"LoadClientStreamNativeStream",
		"TakeClientStreamNativeStream",
		"func loadAllServiceClientStreamNativeStream",
		"func takeAllServiceClientStreamNativeStream",
		"func deleteAllServiceClientStreamNativeStream",
	)

}

func TestRenderRuntimeGlueUsesRPCRuntimeStreamHandleForMessageHelpers(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func StartAllServiceMessageClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"var allServiceStreamRegistry rpcruntime.StreamRegistry[*rpcruntime.StreamEntry]",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"LoadClientStreamMessageStream",
		"TakeClientStreamMessageStream",
		"func loadAllServiceClientStreamMessageStream",
		"func takeAllServiceClientStreamMessageStream",
		"func deleteAllServiceClientStreamMessageStream",
	)
}

func TestRenderRuntimeStageFilesWrapsNativeStreamsForMessageClientCodec(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"case rpcruntime.ServerContractNative:",
		"nativeSession, err := adapter.StartClientStream(ctx)",
		"return r.streams.Create(rpcruntime.NewStreamEntry(&allServiceClientStreamNativeToMessageStreamSession{native: nativeSession}))",
		"type allServiceClientStreamNativeToMessageStreamSession struct {",
		"native AllServiceClientStreamNativeStreamSession",
		"func (s *allServiceClientStreamNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {",
		"return s.native.Send(ctx, name, enabled, child)",
		"acceptedResult, payloadResult, err := s.native.Finish(ctx)",
		"return convertAllServiceClientStreamNativeToMessageResponse(acceptedResult, payloadResult)",
		"err := withAllServiceServerStreamMessageToNativeRequest(req, func(name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error {",
		"nativeSession, err := adapter.StartServerStream(ctx, name, enabled, child)",
		"session = &allServiceServerStreamNativeToMessageStreamSession{native: nativeSession}",
		"acceptedResult, payloadResult, err := s.native.Recv(ctx)",
		"return convertAllServiceServerStreamNativeToMessageResponse(acceptedResult, payloadResult)",
		"nativeSession, err := adapter.StartBidiStream(ctx)",
		"return r.streams.Create(rpcruntime.NewStreamEntry(&allServiceBidiStreamNativeToMessageStreamSession{native: nativeSession}))",
		"func (s *allServiceBidiStreamNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {",
		"return convertAllServiceBidiStreamNativeToMessageResponse(acceptedResult, payloadResult)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeStageFilesWrapsMessageStreamsForNativeClientCodec(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"messageSession, err := adapter.StartClientStreamMessage(ctx)",
		"return r.streams.Create(rpcruntime.NewStreamEntry(&allServiceClientStreamMessageToNativeStreamSession{message: messageSession}))",
		"type allServiceClientStreamMessageToNativeStreamSession struct {",
		"message AllServiceClientStreamMessageStreamSession",
		"messageReq, err := convertAllServiceClientStreamNativeToMessageRequest(name, enabled, child)",
		"return s.message.Send(ctx, messageReq)",
		"messageResp, err := s.message.Finish(ctx)",
		"return convertAllServiceClientStreamMessageToNativeResponse(messageResp)",
		"messageReq, err := convertAllServiceServerStreamNativeToMessageRequest(name, enabled, child)",
		"messageSession, err := adapter.StartServerStreamMessage(ctx, messageReq)",
		"return r.streams.Create(rpcruntime.NewStreamEntry(&allServiceServerStreamMessageToNativeStreamSession{message: messageSession}))",
		"messageResp, err := s.message.Recv(ctx)",
		"return convertAllServiceServerStreamMessageToNativeResponse(messageResp)",
		"return s.message.Done(ctx)",
		"messageSession, err := adapter.StartBidiStreamMessage(ctx)",
		"return r.streams.Create(rpcruntime.NewStreamEntry(&allServiceBidiStreamMessageToNativeStreamSession{message: messageSession}))",
		"messageReq, err := convertAllServiceBidiStreamNativeToMessageRequest(name, enabled, child)",
		"return convertAllServiceBidiStreamMessageToNativeResponse(messageResp)",
		"return s.message.CloseSend(ctx)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeRejectsUnknownStreamingKind(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:   "Greeter",
			GoName: "Greeter",
			Methods: []MethodPlan{{
				Name:      "Mystery",
				GoName:    "Mystery",
				FullName:  "test.v1.Greeter.Mystery",
				Streaming: StreamingKind(99),
			}},
			NativeFileFamily: NativeFileFamilyPlan{
				Runtime: GeneratedFilePlan{Filename: "test/v1/greeter.greeter.runtime.rpccgo.go", Enabled: true},
			},
		}},
	}

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want unknown streaming kind error")
	}
	if got := err.Error(); !strings.Contains(got, "Mystery") || !strings.Contains(got, "render session") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want method name and render-shape validation error", got)
	}
}

func TestRenderRuntimeRejectsAdapterMethodSymbolCollision(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:   "Greeter",
			GoName: "Greeter",
			Methods: []MethodPlan{
				runtimeTestMethod("StartFoo", StreamingKindUnary, "StartFoo", "StartFooMessage", SessionKindNone),
				runtimeTestMethod("Foo", StreamingKindClientStreaming, "StartFoo", "StartFooMessage", SessionKindClient),
			},
			NativeFileFamily: NativeFileFamilyPlan{
				Runtime: GeneratedFilePlan{Filename: "test/v1/greeter.greeter.runtime.rpccgo.go", Enabled: true},
			},
		}},
	}

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want adapter method collision error")
	}
	if got := err.Error(); !strings.Contains(got, "StartFoo") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want colliding adapter method name", got)
	}
}

func runtimeTestMethod(name string, streaming StreamingKind, nativeAdapterMethod string, messageAdapterMethod string, sessionKind SessionKind) MethodPlan {
	method := MethodPlan{Name: name, GoName: name, FullName: "test.v1.Greeter." + name, Streaming: streaming}
	method.RenderPlan = MethodRenderPlan{
		Lifecycle: StreamLifecycleProjectionPlan{SessionKind: sessionKind},
		Symbols:   RenderSymbolsPlan{NativeAdapterMethod: nativeAdapterMethod, MessageAdapterMethod: messageAdapterMethod},
		Errors:    RenderErrorsPlan{NativeAdapterUnavailableErr: "GreeterNativeAdapterUnavailableErr", MessageAdapterUnavailableErr: "GreeterMessageAdapterUnavailableErr", UnknownActiveContractErr: "GreeterUnknownActiveContractErr", NativeMessageConverterErr: "GreeterNativeMessageConverterUnavailableErr"},
	}
	if sessionKind != SessionKindNone {
		method.Contract.Lifecycle = runtimeTestLifecycle(sessionKind)
		method.RenderPlan.Lifecycle.Operations = runtimeTestSessionOperations(sessionKind)
		method.RenderPlan.Lifecycle.Terminal = runtimeTestTerminalPolicy(sessionKind)
		method.RenderPlan.Symbols.NativeSessionType = "Greeter" + name + "NativeStreamSession"
		method.RenderPlan.Symbols.MessageSessionType = "Greeter" + name + "MessageStreamSession"
	}
	return method
}

func runtimeTestLifecycle(sessionKind SessionKind) StreamLifecycleContractPlan {
	op := func(kind StreamLifecycleOperationKind) StreamLifecycleOperationPlan {
		return StreamLifecycleOperationPlan{Kind: kind}
	}
	switch sessionKind {
	case SessionKindClient:
		return StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{op(StreamLifecycleOperationStart), op(StreamLifecycleOperationSend), op(StreamLifecycleOperationFinish), op(StreamLifecycleOperationCancel)}, CancelFinalizes: true, TerminalKind: LifecycleTerminalFinishResult}
	case SessionKindServer:
		return StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{op(StreamLifecycleOperationStart), op(StreamLifecycleOperationReceive), op(StreamLifecycleOperationDone), op(StreamLifecycleOperationCancel)}, CancelFinalizes: true, TerminalKind: LifecycleTerminalOnDone}
	case SessionKindBidi:
		return StreamLifecycleContractPlan{Operations: []StreamLifecycleOperationPlan{op(StreamLifecycleOperationStart), op(StreamLifecycleOperationSend), op(StreamLifecycleOperationReceive), op(StreamLifecycleOperationCloseSend), op(StreamLifecycleOperationDone), op(StreamLifecycleOperationCancel)}, CancelFinalizes: true, TerminalKind: LifecycleTerminalOnDone}
	default:
		return StreamLifecycleContractPlan{}
	}
}

func runtimeTestSessionOperations(sessionKind SessionKind) []SessionOperationPlan {
	op := func(kind SessionOperationKind) SessionOperationPlan {
		return SessionOperationPlan{Kind: kind}
	}
	switch sessionKind {
	case SessionKindClient:
		return []SessionOperationPlan{op(SessionOperationStart), op(SessionOperationSend), op(SessionOperationFinish), op(SessionOperationCancel)}
	case SessionKindServer:
		return []SessionOperationPlan{op(SessionOperationStart), op(SessionOperationReceive), op(SessionOperationDone), op(SessionOperationCancel)}
	case SessionKindBidi:
		return []SessionOperationPlan{op(SessionOperationStart), op(SessionOperationSend), op(SessionOperationReceive), op(SessionOperationCloseSend), op(SessionOperationDone), op(SessionOperationCancel)}
	default:
		return nil
	}
}

func runtimeTestTerminalPolicy(sessionKind SessionKind) TerminalRenderPlan {
	switch sessionKind {
	case SessionKindClient:
		return TerminalRenderPlan{Kind: TerminalKindFinish, Operation: SessionOperationFinish, ReleasesHandle: true, RequiresResponseConvert: true}
	case SessionKindServer:
		return TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true}
	case SessionKindBidi:
		return TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true}
	default:
		return TerminalRenderPlan{}
	}
}

func TestRenderRuntimeGeneratedSourceCompiles(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire (\n\trpccgo v0.0.0\n\tgoogle.golang.org/protobuf v1.36.11\n)\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.sum"), goSum, 0o644); err != nil {
		t.Fatalf("write go.sum: %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !strings.Contains(name, ".runtime.rpccgo.go") {
			continue
		}
		target := filepath.Join(tmp, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir generated dir: %v", err)
		}
		if err := os.WriteFile(target, []byte(generated.GetContent()), 0o644); err != nil {
			t.Fatalf("write generated file %s: %v", name, err)
		}
	}
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime go test failed: %v\n%s", err, out)
	}
}

func TestRenderRuntimeGeneratedSourceCompilesWithImportedMessages(t *testing.T) {
	common := importedNativeCommonFile()
	service := importedNativeServiceFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", service.GetName(), common, service)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go")
	})
	target := filepath.Join(tmp, "common/v1/common_stubs.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir common stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(`package commonv1

import protoreflect "google.golang.org/protobuf/reflect/protoreflect"

type CommonRequest struct {
	Name string
}

type CommonReply struct {
	Payload []byte
}

func (*CommonRequest) ProtoReflect() protoreflect.Message { return nil }
func (*CommonReply) ProtoReflect() protoreflect.Message { return nil }
`), 0o644); err != nil {
		t.Fatalf("write common stubs: %v", err)
	}
	serviceTarget := filepath.Join(tmp, "test/v1/imported_native_stubs.go")
	if err := os.WriteFile(serviceTarget, []byte(`package testv1

import (
	context "context"

	commonv1 "example.com/generated/common/v1"
)

type ImportedNativeHandler interface {
	Call(context.Context, *commonv1.CommonRequest) (*commonv1.CommonReply, error)
}
`), 0o644); err != nil {
		t.Fatalf("write imported native stubs: %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime with imported messages go test failed: %v\n%s", err, out)
	}
}

func importedNativeCommonFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("common/v1/common.proto"),
		Package: proto.String("common.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/generated/common/v1;commonv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("CommonRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
			{
				Name: proto.String("CommonReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("payload", 1, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
		},
	}
}

func importedNativeServiceFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/v1/imported_native.proto"),
		Package:    proto.String("test.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"common/v1/common.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/generated/test/v1;testv1"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("ImportedNative"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Call", ".common.v1.CommonRequest", ".common.v1.CommonReply", false, false),
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
