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

func TestRenderRuntimeGlueDefinesServiceDispatcherAndRegistration(t *testing.T) {
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
		"type AllServiceActiveAdapter struct {",
		"Native  AllServiceNativeAdapter",
		"Message AllServiceMessageAdapter",
		"var allServiceDispatcher rpcruntime.Dispatcher[AllServiceActiveAdapter]",
		"func AllServiceDispatcherForRuntime() *rpcruntime.Dispatcher[AllServiceActiveAdapter] {",
		"return &allServiceDispatcher",
		"func registerAllServiceActiveServer(kind rpcruntime.ServerKind, adapter AllServiceNativeAdapter) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		"snapshot, err := allServiceDispatcher.Register(kind, rpcruntime.ServerContractNative, AllServiceActiveAdapter{Native: adapter})",
		"type AllServiceCGONativeClientBridge struct{}",
		"func (AllServiceCGONativeClientBridge) Unary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error) {",
		"if snapshot.Adapter.Native == nil {",
		"return AllServiceNativeAdapterUnavailableErr",
		"var acceptedResult bool",
		"var payloadResult []byte",
		"acceptedResult, payloadResult, callErr = snapshot.Adapter.Native.Unary(ctx, name, enabled, child)",
		"func NewAllServiceCGONativeClientBridge() AllServiceCGONativeClientBridge {",
		"func RegisterAllServiceCGONativeActiveServer(kind rpcruntime.ServerKind, adapter AllServiceNativeAdapter) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesMessageContractDispatcherAndRegistration(t *testing.T) {
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
		"snapshot, err := allServiceDispatcher.Register(kind, rpcruntime.ServerContractMessage, AllServiceActiveAdapter{Message: adapter})",
		"type AllServiceCGOMessageClientBridge struct{}",
		"func (AllServiceCGOMessageClientBridge) Unary(ctx context.Context, req []byte) ([]byte, error) {",
		"case rpcruntime.ServerContractMessage:",
		"return AllServiceMessageAdapterUnavailableErr",
		"resp, callErr = snapshot.Adapter.Message.UnaryMessage(ctx, req)",
		"AllServiceNativeMessageConverterUnavailableErr",
		"func NewAllServiceCGOMessageClientBridge() AllServiceCGOMessageClientBridge {",
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
		"rpcruntime.DispatcherStreamSend[AllServiceActiveAdapter, AllServiceClientStreamNativeStreamSession](AllServiceDispatcherForRuntime(), s.handle, func(session AllServiceClientStreamNativeStreamSession) error {",
		"func (s AllServiceClientStreamNativeStream) Finish(ctx context.Context) (bool, []byte, error) {",
		"rpcruntime.DispatcherStreamFinish[AllServiceActiveAdapter, AllServiceClientStreamNativeStreamSession](AllServiceDispatcherForRuntime(), s.handle, func(session AllServiceClientStreamNativeStreamSession) error {",
		"func (s AllServiceServerStreamNativeStream) Recv(ctx context.Context) (bool, []byte, error) {",
		"func (s AllServiceServerStreamNativeStream) Done(ctx context.Context) error {",
		"rpcruntime.DispatcherStreamDone[AllServiceActiveAdapter, AllServiceServerStreamNativeStreamSession](AllServiceDispatcherForRuntime(), s.handle, func(session AllServiceServerStreamNativeStreamSession) error {",
		"func (s AllServiceBidiStreamNativeStream) CloseSend(ctx context.Context) error {",
		"type AllServiceClientStreamMessageStream struct {",
		"func NewAllServiceClientStreamMessageStream(handle rpcruntime.StreamHandle) AllServiceClientStreamMessageStream {",
		"func (s AllServiceClientStreamMessageStream) Send(ctx context.Context, req []byte) error {",
		"rpcruntime.DispatcherStreamSend[AllServiceActiveAdapter, AllServiceClientStreamMessageStreamSession](AllServiceDispatcherForRuntime(), s.handle, func(session AllServiceClientStreamMessageStreamSession) error {",
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
		"func (AllServiceCGONativeClientBridge) StartClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"func AllServiceDispatcherForRuntime() *rpcruntime.Dispatcher[AllServiceActiveAdapter] {",
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
		"func (AllServiceCGOMessageClientBridge) StartClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"func AllServiceDispatcherForRuntime() *rpcruntime.Dispatcher[AllServiceActiveAdapter] {",
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
		"nativeSession, err := snapshot.Adapter.Native.StartClientStream(ctx)",
		"return &allServiceClientStreamNativeToMessageStreamSession{native: nativeSession}, nil",
		"type allServiceClientStreamNativeToMessageStreamSession struct {",
		"native AllServiceClientStreamNativeStreamSession",
		"func (s *allServiceClientStreamNativeToMessageStreamSession) Send(ctx context.Context, req []byte) error {",
		"return s.native.Send(ctx, name, enabled, child)",
		"acceptedResult, payloadResult, err := s.native.Finish(ctx)",
		"return convertAllServiceClientStreamNativeToMessageResponse(acceptedResult, payloadResult)",
		"err := withAllServiceServerStreamMessageToNativeRequest(req, func(name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error {",
		"nativeSession, err := snapshot.Adapter.Native.StartServerStream(ctx, name, enabled, child)",
		"session = &allServiceServerStreamNativeToMessageStreamSession{native: nativeSession}",
		"acceptedResult, payloadResult, err := s.native.Recv(ctx)",
		"return convertAllServiceServerStreamNativeToMessageResponse(acceptedResult, payloadResult)",
		"nativeSession, err := snapshot.Adapter.Native.StartBidiStream(ctx)",
		"return &allServiceBidiStreamNativeToMessageStreamSession{native: nativeSession}, nil",
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
		"messageSession, err := snapshot.Adapter.Message.StartClientStreamMessage(ctx)",
		"return &allServiceClientStreamMessageToNativeStreamSession{message: messageSession}, nil",
		"type allServiceClientStreamMessageToNativeStreamSession struct {",
		"message AllServiceClientStreamMessageStreamSession",
		"messageReq, err := convertAllServiceClientStreamNativeToMessageRequest(name, enabled, child)",
		"return s.message.Send(ctx, messageReq)",
		"messageResp, err := s.message.Finish(ctx)",
		"return convertAllServiceClientStreamMessageToNativeResponse(messageResp)",
		"messageReq, err := convertAllServiceServerStreamNativeToMessageRequest(name, enabled, child)",
		"messageSession, err := snapshot.Adapter.Message.StartServerStreamMessage(ctx, messageReq)",
		"return &allServiceServerStreamMessageToNativeStreamSession{message: messageSession}, nil",
		"messageResp, err := s.message.Recv(ctx)",
		"return convertAllServiceServerStreamMessageToNativeResponse(messageResp)",
		"return s.message.Done(ctx)",
		"messageSession, err := snapshot.Adapter.Message.StartBidiStreamMessage(ctx)",
		"return &allServiceBidiStreamMessageToNativeStreamSession{message: messageSession}, nil",
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
		Session: SessionRenderPlan{Kind: sessionKind},
		Symbols: RenderSymbolsPlan{NativeAdapterMethod: nativeAdapterMethod, MessageAdapterMethod: messageAdapterMethod},
		Errors:  RenderErrorsPlan{NativeAdapterUnavailableErr: "GreeterNativeAdapterUnavailableErr", MessageAdapterUnavailableErr: "GreeterMessageAdapterUnavailableErr", UnknownActiveContractErr: "GreeterUnknownActiveContractErr", NativeMessageConverterErr: "GreeterNativeMessageConverterUnavailableErr"},
	}
	if sessionKind != SessionKindNone {
		method.Contract.Lifecycle = runtimeTestLifecycle(sessionKind)
		method.RenderPlan.Session.Operations = runtimeTestSessionOperations(sessionKind)
		method.RenderPlan.Terminal = runtimeTestTerminalPolicy(sessionKind)
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
	op := func(kind SessionOperationKind, terminal bool) SessionOperationPlan {
		return SessionOperationPlan{Kind: kind, Enabled: true, RequiresTerminal: terminal}
	}
	switch sessionKind {
	case SessionKindClient:
		return []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationSend, false), op(SessionOperationFinish, true), op(SessionOperationCancel, true)}
	case SessionKindServer:
		return []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationReceive, false), op(SessionOperationDone, true), op(SessionOperationCancel, true)}
	case SessionKindBidi:
		return []SessionOperationPlan{op(SessionOperationStart, false), op(SessionOperationSend, false), op(SessionOperationReceive, false), op(SessionOperationCloseSend, false), op(SessionOperationDone, true), op(SessionOperationCancel, true)}
	default:
		return nil
	}
}

func runtimeTestTerminalPolicy(sessionKind SessionKind) TerminalRenderPlan {
	switch sessionKind {
	case SessionKindClient:
		return TerminalRenderPlan{Kind: TerminalKindFinish, Operation: SessionOperationFinish, ReleasesHandle: true, RequiresResponseConvert: true, AllowsCancel: true}
	case SessionKindServer:
		return TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true, AllowsCancel: true}
	case SessionKindBidi:
		return TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true, AllowsCancel: true, AllowsCloseSend: true}
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
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
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

type CommonRequest struct {
	Name string
}

type CommonReply struct {
	Payload []byte
}
`), 0o644); err != nil {
		t.Fatalf("write common stubs: %v", err)
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
