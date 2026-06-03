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

	_, err := GenerateWithOptions(plugin)
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

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"CGONativeClientBridge",
		"CGOMessageClientBridge",
		"type AllServiceNativeServer interface {",
		"type AllServiceClientStreamNativeClientStream interface {",
		"type AllServiceServerStreamNativeServerStream interface {",
		"type AllServiceBidiStreamNativeBidiStream interface {",
	)
	for _, fragment := range []string{
		"type allServiceNativeServerAdapter struct {",
		"func (a *allServiceNativeServerAdapter) StartClientStream(ctx context.Context) (AllServiceClientStreamNativeStreamSession, error)",
		"type AllServiceClientStreamNativeStreamSession interface {",
		"Send(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error",
		"Finish(ctx context.Context) (bool, []byte, error)",
		"Cancel(ctx context.Context) error",
		"type AllServiceServerStreamNativeStreamSession interface {",
		"Recv(ctx context.Context) (bool, []byte, error)",
		"Finish(ctx context.Context) error",
		"type AllServiceBidiStreamNativeStreamSession interface {",
		"CloseSend(ctx context.Context) error",
		"Finish(ctx context.Context) error",
		"var allServiceStreamRegistry rpcruntime.StreamRegistry",
		"type allServiceActiveServerRecord struct {",
		"invokeNativeUnary",
		"var allServiceActiveServer atomic.Pointer[allServiceActiveServerRecord]",
		"func InvokeAllServiceNativeUnary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error) {",
		"return active.invokeNativeUnary(ctx, name, enabled, child)",
		"return adapter.Unary(ctx, name, enabled, child)",
		"func RegisterAllServiceCGONativeServer(server AllServiceNativeServer) error {",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesMessageContractActiveSlotAndRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceClientStreamMessageStreamSession interface {",
		"Send(ctx context.Context, req []byte) error",
		"Finish(ctx context.Context) ([]byte, error)",
		"Cancel(ctx context.Context) error",
		"type AllServiceServerStreamMessageStreamSession interface {",
		"Recv(ctx context.Context) ([]byte, error)",
		"Finish(ctx context.Context) error",
		"type AllServiceBidiStreamMessageStreamSession interface {",
		"CloseSend(ctx context.Context) error",
		"Finish(ctx context.Context) error",
		"func registerAllServiceCGOMessageServer(server AllServiceCGOMessageServer) error {",
		"record := &allServiceActiveServerRecord{}",
		"record.invokeMessageUnary = adapter.Unary",
		"func InvokeAllServiceMessageUnary(ctx context.Context, req []byte) ([]byte, error) {",
		"return active.invokeMessageUnary(ctx, req)",
		"return AllServiceMessageServerUnavailableErr",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"allServiceMessageDispatcher",
		"rpcruntime.Dispatcher[AllServiceNativeAdapter]",
		"rpcruntime.Dispatcher[AllServiceMessageAdapter]",
		"type AllServiceMessageAdapter interface {",
		"renderRuntimeMessageContractMismatchCheck",
		"if _, nativeErr :=",
	)
}

func TestRenderRuntimeGlueDefinesConnectDirectRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.default_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func RegisterDefaultServiceConnectHandler(handler DefaultServiceHandler) error {",
		"record := &defaultServiceActiveServerRecord{",
		"defaultServiceActiveServer.Store(record)",
		"messageResp, err := handler.DefaultUnary(ctx, messageReq)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueRoutesNativeUnaryToConnectHandler(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.connect_native_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func RegisterConnectNativeServiceConnectHandler(handler ConnectNativeServiceHandler) error {",
		"record := &connectNativeServiceActiveServerRecord{",
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

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.grpc_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func RegisterGrpcServiceGRPCServer(server GrpcServiceServer) error {",
		"record := &grpcServiceActiveServerRecord{",
		"grpcServiceActiveServer.Store(record)",
		"messageResp, err := server.GrpcUnary(ctx, messageReq)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesGRPCDirectStreamingSessions(t *testing.T) {
	file := grpcStreamingRuntimeTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/grpc_streaming_runtime.proto", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/grpc_streaming_runtime.grpc_streaming_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func RegisterGrpcStreamingServiceGRPCServer(server GrpcStreamingServiceServer) error {",
		"record := &grpcStreamingServiceActiveServerRecord{",
		"source := newgrpcStreamingServiceClientStreamGRPCDirectMessageStreamSession(ctx, server)",
		"source, err := newgrpcStreamingServiceServerStreamGRPCDirectMessageStreamSession(ctx, server, req)",
		"source := newgrpcStreamingServiceBidiStreamGRPCDirectMessageStreamSession(ctx, server)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueDefinesMethodSpecificStreamFacades(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceClientStreamNativeStream struct {",
		"handle rpcruntime.StreamHandle",
		"func NewAllServiceClientStreamNativeStream(handle rpcruntime.StreamHandle) AllServiceClientStreamNativeStream {",
		"func (s AllServiceClientStreamNativeStream) Send(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error {",
		"value, ok := allServiceStreamRegistry.Load(s.handle)",
		"session, ok := value.(*allServiceClientStreamNativeFinalSession)",
		"func (s AllServiceClientStreamNativeStream) Finish(ctx context.Context) (bool, []byte, error) {",
		"taken, ok := allServiceStreamRegistry.Take(s.handle)",
		"func (s AllServiceServerStreamNativeStream) Recv(ctx context.Context) (bool, []byte, error) {",
		"func (s AllServiceServerStreamNativeStream) Finish(ctx context.Context) error {",
		"func (s AllServiceBidiStreamNativeStream) CloseSend(ctx context.Context) error {",
		"if err := session.lifecycle.MarkCanceled(); err != nil {",
		"return session.cancel(ctx)",
		"type AllServiceClientStreamMessageStream struct {",
		"func NewAllServiceClientStreamMessageStream(handle rpcruntime.StreamHandle) AllServiceClientStreamMessageStream {",
		"func (s AllServiceClientStreamMessageStream) Send(ctx context.Context, req []byte) error {",
		"session, ok := value.(*allServiceClientStreamMessageFinalSession)",
		"func (s AllServiceClientStreamMessageStream) Finish(ctx context.Context) ([]byte, error) {",
		"func (s AllServiceServerStreamMessageStream) Recv(ctx context.Context) ([]byte, error) {",
		"func (s AllServiceBidiStreamMessageStream) CloseSend(ctx context.Context) error {",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile, "session.lifecycle.Cancel(", "type AllServiceUnaryNativeStream", "type AllServiceUnaryMessageStream")
}

func TestRenderRuntimeGlueUsesRPCRuntimeStreamHandleAndCoreStreamRegistry(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func StartAllServiceNativeClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"var allServiceStreamRegistry rpcruntime.StreamRegistry",
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

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func StartAllServiceMessageClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"var allServiceStreamRegistry rpcruntime.StreamRegistry",
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

func TestRenderRuntimeGlueWrapsNativeStreamsForMessageClientCodec(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"startMessageClientStream func(ctx context.Context) (*allServiceClientStreamMessageFinalSession, error)",
		"source, err := adapter.StartClientStream(ctx)",
		"return &allServiceClientStreamMessageFinalSession{",
		"send: func(ctx context.Context, req []byte) error {",
		"reqView, err := convertAllServiceClientStreamMessageToNativeRequest(req)",
		"err = source.Send(ctx, reqView.name, reqView.enabled, reqView.child)",
		"goruntime.KeepAlive(reqView)",
		"acceptedResult, payloadResult, err := source.Finish(ctx)",
		"return convertAllServiceClientStreamNativeToMessageResponse(acceptedResult, payloadResult)",
		"reqView, err := convertAllServiceServerStreamMessageToNativeRequest(req)",
		"source, err := adapter.StartServerStream(ctx, reqView.name, reqView.enabled, reqView.child)",
		"goruntime.KeepAlive(reqView)",
		"return &allServiceServerStreamMessageFinalSession{",
		"acceptedResult, payloadResult, err := source.Recv(ctx)",
		"return convertAllServiceServerStreamNativeToMessageResponse(acceptedResult, payloadResult)",
		"source, err := adapter.StartBidiStream(ctx)",
		"return &allServiceBidiStreamMessageFinalSession{",
		"send: func(ctx context.Context, req []byte) error {",
		"return convertAllServiceBidiStreamNativeToMessageResponse(acceptedResult, payloadResult)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
}

func TestRenderRuntimeGlueWrapsMessageStreamsForNativeClientCodec(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"source, err := adapter.StartClientStream(ctx)",
		"return &allServiceClientStreamNativeFinalSession{",
		"messageReq, err := convertAllServiceClientStreamNativeToMessageRequest(name, enabled, child)",
		"return source.Send(ctx, messageReq)",
		"messageResp, err := source.Finish(ctx)",
		"return convertAllServiceClientStreamMessageToNativeResponse(messageResp)",
		"messageReq, err := convertAllServiceServerStreamNativeToMessageRequest(name, enabled, child)",
		"source, err := adapter.StartServerStream(ctx, messageReq)",
		"return &allServiceServerStreamNativeFinalSession{",
		"messageResp, err := source.Recv(ctx)",
		"return convertAllServiceServerStreamMessageToNativeResponse(messageResp)",
		"finish: source.Finish,",
		"source, err := adapter.StartBidiStream(ctx)",
		"return &allServiceBidiStreamNativeFinalSession{",
		"messageReq, err := convertAllServiceBidiStreamNativeToMessageRequest(name, enabled, child)",
		"return convertAllServiceBidiStreamMessageToNativeResponse(messageResp)",
		"closeSend: source.CloseSend,",
		"source, err := adapter.StartClientStream(ctx)",
		"source, err := adapter.StartServerStream(ctx, req)",
		"source, err := adapter.StartBidiStream(ctx)",
		"source := newallServiceClientStreamConnectDirectMessageStreamSession(ctx, handler)",
		"source, err := newallServiceServerStreamConnectDirectMessageStreamSession(ctx, handler, req)",
		"source := newallServiceBidiStreamConnectDirectMessageStreamSession(ctx, handler)",
		"rpcruntime.NewConnectClientStream[AllRequest](conn)",
		"rpcruntime.NewConnectServerStream[AllReply](conn)",
		"rpcruntime.NewConnectBidiStream[AllRequest, AllReply](conn)",
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
			Artifacts: []GeneratedArtifactPlan{
				{Kind: GeneratedArtifactKindRuntime, Filename: "test/v1/greeter.greeter.runtime.rpccgo.go"},
			},
		}},
	}

	err := renderRuntimeFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
	if err == nil {
		t.Fatal("renderRuntimeFile() error = nil, want unknown streaming kind error")
	}
	if got := err.Error(); !strings.Contains(got, "Mystery") || !strings.Contains(got, "render lifecycle does not match contract capabilities") {
		t.Fatalf("renderRuntimeFile() error = %q, want method name and render-shape validation error", got)
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
				runtimeTestMethod("StartFoo", StreamingKindUnary, "StartFoo", "StartFoo", runtimeStreamUnary),
				runtimeTestMethod("Foo", StreamingKindClientStreaming, "StartFoo", "StartFoo", runtimeStreamClient),
			},
			Artifacts: []GeneratedArtifactPlan{
				{Kind: GeneratedArtifactKindRuntime, Filename: "test/v1/greeter.greeter.runtime.rpccgo.go"},
			},
		}},
	}

	err := renderRuntimeFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
	if err == nil {
		t.Fatal("renderRuntimeFile() error = nil, want adapter method collision error")
	}
	if got := err.Error(); !strings.Contains(got, "StartFoo") || !strings.Contains(got, "collides") {
		t.Fatalf("renderRuntimeFile() error = %q, want colliding adapter method name", got)
	}
}

func runtimeTestMethod(name string, streaming StreamingKind, nativeAdapterMethod string, messageAdapterMethod string, streamShape runtimeStreamShape) MethodPlan {
	method := MethodPlan{Name: name, GoName: name, FullName: "test.v1.Greeter." + name, Streaming: streaming}
	method.RenderPlan = MethodRenderPlan{
		Lifecycle: runtimeTestLifecycleProjection(streamShape),
		Symbols:   RenderSymbolsPlan{NativeAdapterMethod: nativeAdapterMethod, MessageAdapterMethod: messageAdapterMethod},
		Errors: RenderErrorsPlan{
			NativeServerUnavailableErr:  "GreeterNativeServerUnavailableErr",
			MessageServerUnavailableErr: "GreeterMessageServerUnavailableErr",
			UnknownActiveContractErr:    "GreeterUnknownActiveContractErr",
		},
	}
	if streamShape != runtimeStreamUnary {
		method.Contract.Lifecycle = runtimeTestLifecycle(streamShape)
		method.RenderPlan.Symbols.NativeSessionType = "Greeter" + name + "NativeStreamSession"
		method.RenderPlan.Symbols.MessageSessionType = "Greeter" + name + "MessageStreamSession"
	}
	return method
}

func runtimeTestLifecycleProjection(streamShape runtimeStreamShape) StreamLifecycleProjectionPlan {
	switch streamShape {
	case runtimeStreamClient:
		return StreamLifecycleProjectionPlan{Streaming: true, CanSend: true, FinishReturnsResponse: true, RequiresCodec: true}
	case runtimeStreamServer:
		return StreamLifecycleProjectionPlan{Streaming: true, CanRecv: true, RequiresCodec: true}
	case runtimeStreamBidi:
		return StreamLifecycleProjectionPlan{Streaming: true, CanSend: true, CanRecv: true, CanCloseSend: true, RequiresCodec: true}
	default:
		return StreamLifecycleProjectionPlan{RequiresCodec: true}
	}
}

func runtimeTestLifecycle(streamShape runtimeStreamShape) StreamLifecycleContractPlan {
	switch streamShape {
	case runtimeStreamClient:
		return StreamLifecycleContractPlan{CanSend: true, FinishReturnsResponse: true}
	case runtimeStreamServer:
		return StreamLifecycleContractPlan{CanRecv: true}
	case runtimeStreamBidi:
		return StreamLifecycleContractPlan{CanSend: true, CanRecv: true, CanCloseSend: true}
	default:
		return StreamLifecycleContractPlan{}
	}
}

func TestRenderRuntimeGeneratedSourceCompiles(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\trpccgo v0.0.0\n\tgoogle.golang.org/protobuf v1.36.11\n)\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
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
		if !strings.Contains(name, ".runtime.rpccgo.go") && !strings.Contains(name, ".codec.rpccgo.go") && !strings.Contains(name, ".server.message.rpccgo.go") && !strings.Contains(name, ".server.native.rpccgo.go") {
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

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
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

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.message.rpccgo.go") ||
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
type ImportedNativeClient interface {
	Call(context.Context, *commonv1.CommonRequest) (*commonv1.CommonReply, error)
}
`), 0o644); err != nil {
		t.Fatalf("write imported native stubs: %v", err)
	}

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime with imported messages go test failed: %v\n%s", err, out)
	}
}

func grpcStreamingRuntimeTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/grpc_streaming_runtime.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/generated/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			completeServicePlanRequestDescriptor("GRPCStreamingRequest"),
			completeServicePlanReplyDescriptor("GRPCStreamingReply"),
			childMessageDescriptor(),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("GrpcStreamingService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("ClientStream", ".test.v1.GRPCStreamingRequest", ".test.v1.GRPCStreamingReply", true, false),
					methodDescriptor("ServerStream", ".test.v1.GRPCStreamingRequest", ".test.v1.GRPCStreamingReply", false, true),
					methodDescriptor("BidiStream", ".test.v1.GRPCStreamingRequest", ".test.v1.GRPCStreamingReply", true, true),
				},
			},
		},
		SourceCodeInfo: completeServicePlanServiceComments([]string{"@rpccgo: msg-grpc\n"}),
	}
}

func writeGRPCStreamingRuntimeCompileStubs(t *testing.T, root string) {
	t.Helper()

	const content = `package testv1

import (
	context "context"

	grpc "google.golang.org/grpc"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type GRPCStreamingRequest struct {
	Name    string
	Enabled bool
	Child   []byte
}

type GRPCStreamingReply struct {
	Accepted bool
	Payload  []byte
}

type GrpcStreamingServiceServer interface {
	ClientStream(grpc.ClientStreamingServer[GRPCStreamingRequest, GRPCStreamingReply]) error
	ServerStream(*GRPCStreamingRequest, grpc.ServerStreamingServer[GRPCStreamingReply]) error
	BidiStream(grpc.BidiStreamingServer[GRPCStreamingRequest, GRPCStreamingReply]) error
}
type GrpcStreamingServiceClient interface {
	ClientStream(context.Context, ...grpc.CallOption) (grpc.ClientStreamingClient[GRPCStreamingRequest, GRPCStreamingReply], error)
	ServerStream(context.Context, *GRPCStreamingRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[GRPCStreamingReply], error)
	BidiStream(context.Context, ...grpc.CallOption) (grpc.BidiStreamingClient[GRPCStreamingRequest, GRPCStreamingReply], error)
}

func (*GRPCStreamingRequest) ProtoReflect() protoreflect.Message { return nil }
func (*GRPCStreamingReply) ProtoReflect() protoreflect.Message { return nil }

var _ context.Context
`
	target := filepath.Join(root, "test/v1/grpc_streaming_runtime_stubs.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir grpc runtime stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write grpc runtime compile stubs: %v", err)
	}
}

func TestRenderRuntimeGeneratedSourceCompilesWithGRPCDirectStreaming(t *testing.T) {
	file := grpcStreamingRuntimeTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/grpc_streaming_runtime.proto", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.message.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go")
	})
	writeGRPCStreamingRuntimeCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runtime with grpc direct streaming go test failed: %v\n%s", err, out)
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
