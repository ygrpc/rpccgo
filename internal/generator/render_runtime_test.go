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
	assertGeneratedContentContains(t, plugin, runtimeFile, `rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"`)
	assertGeneratedContentContains(t, plugin, runtimeFile, `errors "errors"`)
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc")
}

func TestRenderRuntimeGlueDefinesServerRegistryAndTransportRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"CGONativeClientEntryObject",
		"CGOMessageClientEntryObject",
		"type AllServiceNativeServer interface {",
		"type AllServiceClientStreamNativeClientStream interface {",
		"type AllServiceServerStreamNativeServerStream interface {",
		"type AllServiceBidiStreamNativeBidiStream interface {",
		"type allServiceNativeBinding struct {",
		"type allServiceNativeActiveBinding struct {",
		"type allServiceMessageActiveBinding struct {",
		"allServiceCurrentNativeBinding",
		"allServiceCurrentMessageBinding",
	)
	for _, fragment := range []string{
		`const allServiceServiceID rpcruntime.ServiceID = "test.v1.AllService"`,
		"func ClearAllServiceServer() error {",
		"return rpcruntime.ClearServer(allServiceServiceID)",
		"func LoadAllServiceRegisteredServer() (rpcruntime.RegisteredServer, error) {",
		"return rpcruntime.LoadServer(allServiceServiceID)",
		"func InvokeAllServiceNativeUnary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error) {",
		"registered, err := rpcruntime.LoadServer(allServiceServiceID)",
		"case rpcruntime.ServerKindGoNative:",
		"server, ok := registered.Server.(AllServiceNativeServer)",
		"return server.Unary(ctx, name, enabled, child)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"type allServiceBinding struct {",
		"invokeNativeUnary",
		"invokeMessageUnary",
		"func registerAllServiceGoNativeServer(server AllServiceNativeServer) error {",
		"func RegisterAllServiceCGONativeServer(server AllServiceNativeServer) error {",
		"func registerAllServiceCGOMessageServer(server AllServiceCGOMessageServer) error {",
		"type AllServiceClientStreamNativeStreamSession interface {",
		"type AllServiceClientStream"+"NativeStream struct {",
		"var allServiceStream"+"Registry rpcruntime.StreamRegistry",
		"rpcruntime.ActiveServerSlot",
		"rpcruntime.AdapterSnapshot",
	)
}

func TestRenderRuntimeGlueDefinesMessageContractRegistryRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	for _, fragment := range []string{
		"func InvokeAllServiceMessageUnary(ctx context.Context, req *AllRequest) (*AllReply, error) {",
		`return nil, errors.New("rpccgo: message request is nil")`,
		"registered, err := rpcruntime.LoadServer(allServiceServiceID)",
		"case rpcruntime.ServerKindCGOMessage:",
		"resp, err := server.Unary(ctx, req)",
		`return nil, errors.New("rpccgo: message response is nil")`,
		"return resp, nil",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"allServiceMessageDispatcher",
		"rpcruntime.Dispatcher[AllServiceNativeAdapter]",
		"rpcruntime.Dispatcher[AllServiceMessageAdapter]",
		"type AllServiceMessageAdapter interface {",
		"if _, nativeErr :=",
		"messageBinding := &allServiceMessageActiveBinding{}",
		"allServiceCurrentMessageBinding.Store(messageBinding)",
		"type AllServiceClientStreamMessageStreamSession interface {",
		"type AllServiceClientStream"+"MessageStream struct {",
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
		"rpcruntime.RegisterServer(defaultServiceServiceID, rpcruntime.RegisteredServer{",
		"Kind:   rpcruntime.ServerKindConnect,",
		"Server: handler,",
		"messageResp, err := server.DefaultUnary(ctx, req)",
		`return nil, errors.New("rpccgo: message response is nil")`,
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
		"Kind:   rpcruntime.ServerKindConnect,",
		"Server: handler,",
		"messageReq, err := convertConnectNativeServiceConnectNativeUnaryNativeToMessageRequest(name, enabled, child)",
		"messageResp, err = server.ConnectNativeUnary(ctx, messageReq)",
		"messageResp, err := server.ConnectNativeUnary(ctx, req)",
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
		"Kind:   rpcruntime.ServerKindGRPC,",
		"Server: server,",
		"messageResp, err := server.GrpcUnary(ctx, req)",
		`return nil, errors.New("rpccgo: message response is nil")`,
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
		"Kind:   rpcruntime.ServerKindGRPC,",
		"source := newgrpcStreamingServiceClientStreamGRPCDirectMessageStreamSession(ctx, server)",
		"source, err := newgrpcStreamingServiceServerStreamGRPCDirectMessageStreamSession(ctx, server, req)",
		"source := newgrpcStreamingServiceBidiStreamGRPCDirectMessageStreamSession(ctx, server)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"source := newgrpcStreamingServiceClientStreamGRPCDirectMessageStreamSession(ctx, server)\n\t\tvar err error\n\t\tif err != nil",
		"source := newgrpcStreamingServiceBidiStreamGRPCDirectMessageStreamSession(ctx, server)\n\t\tvar err error\n\t\tif err != nil",
	)
}

func TestRenderRuntimeGlueDefinesPackageLevelStreamOperations(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/complete_service_plan.all_service.server.native.rpccgo.go"
	const messageServerFile = "test/v1/complete_service_plan.all_service.server.message.rpccgo.go"
	for _, fragment := range []string{
		"func SendAllServiceNativeClientStream(ctx context.Context, handle rpcruntime.StreamHandle, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) error {",
		"entry, err := rpcruntime.SendStreamSession(handle)",
		"source, ok := entry.Session.(rpcruntime.ClientStreamingClient[AllServiceClientStreamNativeStreamRequest, AllServiceClientStreamNativeStreamResponse])",
		"return source.Send(ctx, AllServiceClientStreamNativeStreamRequest{Name: name, Enabled: enabled, Child: child})",
		"func FinishAllServiceNativeClientStream(ctx context.Context, handle rpcruntime.StreamHandle) (bool, []byte, error) {",
		"entry, err := rpcruntime.LoadStreamSession(handle)",
		"_, err = rpcruntime.FinishStreamSession(handle)",
		"return resp.Accepted, resp.Payload, nil",
		"func RecvAllServiceNativeServerStream(ctx context.Context, handle rpcruntime.StreamHandle) (bool, []byte, error) {",
		"entry, err := rpcruntime.RecvStreamSession(handle)",
		"func FinishAllServiceNativeServerStream(ctx context.Context, handle rpcruntime.StreamHandle) error {",
		"func CloseSendAllServiceNativeBidiStream(ctx context.Context, handle rpcruntime.StreamHandle) error {",
		"entry, err := rpcruntime.CloseSendStreamSession(handle)",
		"_, err = rpcruntime.CancelStreamSession(handle)",
		"return source.Cancel(ctx)",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	for _, fragment := range []string{
		"func SendAllServiceMessageClientStream(ctx context.Context, handle rpcruntime.StreamHandle, req *AllRequest) error {",
		`return errors.New("rpccgo: message request is nil")`,
		"entry, err := rpcruntime.SendStreamSession(handle)",
		"source, ok := entry.Session.(rpcruntime.ClientStreamingClient[*AllRequest, *AllReply])",
		"func FinishAllServiceMessageClientStream(ctx context.Context, handle rpcruntime.StreamHandle) (*AllReply, error) {",
		"entry, err := rpcruntime.LoadStreamSession(handle)",
		"_, err = rpcruntime.FinishStreamSession(handle)",
		"func RecvAllServiceMessageServerStream(ctx context.Context, handle rpcruntime.StreamHandle) (*AllReply, error) {",
		"entry, err := rpcruntime.RecvStreamSession(handle)",
		`return nil, errors.New("rpccgo: message response is nil")`,
		"func CloseSendAllServiceMessageBidiStream(ctx context.Context, handle rpcruntime.StreamHandle) error {",
		"entry, err := rpcruntime.CloseSendStreamSession(handle)",
		"_, err = rpcruntime.CancelStreamSession(handle)",
	} {
		assertGeneratedContentContains(t, plugin, messageServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeServerFile,
		"type AllServiceClientStream"+"NativeStream struct {",
		"func NewAllServiceClientStream"+"NativeStream(handle rpcruntime.StreamHandle)",
		"allServiceStreamRegistry",
		"."+"kind",
		"."+"session",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, messageServerFile,
		"type AllServiceClientStream"+"MessageStream struct {",
		"func NewAllServiceClientStream"+"MessageStream(handle rpcruntime.StreamHandle)",
		"allServiceStreamRegistry",
		"."+"kind",
		"."+"session",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/v1/complete_service_plan.all_service.runtime.rpccgo.go",
		"allServiceStreamRegistry.Load(s.handle)",
		"allServiceStreamRegistry.Take(s.handle)",
		"session.capability.EnsureCanSend()",
		"session.capability.MarkSendClosed()",
		"session.capability.MarkCanceled()",
		"type AllServiceUnaryNativeStream",
		"type AllServiceUnaryMessageStream",
	)
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
		"return rpcruntime.CreateStreamSession(rpcruntime.ServerKindGoNative, source)",
		"return rpcruntime.CreateStreamSession(rpcruntime.ServerKindConnect, source)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"var allServiceStream"+"Registry rpcruntime.StreamRegistry",
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
		"return rpcruntime.CreateStreamSession(rpcruntime.ServerKindGoNative, source)",
		"return rpcruntime.CreateStreamSession(rpcruntime.ServerKindCGOMessage, source)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"var allServiceStream"+"Registry rpcruntime.StreamRegistry",
		"LoadClientStreamMessageStream",
		"TakeClientStreamMessageStream",
		"func loadAllServiceClientStreamMessageStream",
		"func takeAllServiceClientStreamMessageStream",
		"func deleteAllServiceClientStreamMessageStream",
	)
}

func TestRenderRuntimeGlueRoutesMessageStreamsToNativeSessionsWithConverter(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	const messageServerFile = "test/v1/complete_service_plan.all_service.server.message.rpccgo.go"
	for _, fragment := range []string{
		"func StartAllServiceMessageClientStream(ctx context.Context) (rpcruntime.StreamHandle, error) {",
		"case rpcruntime.ServerKindGoNative:",
		"return rpcruntime.CreateStreamSession(rpcruntime.ServerKindGoNative, source)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	for _, fragment := range []string{
		"name, enabled, child, reqOwner, err := convertAllServiceClientStreamMessageToNativeRequest(req)",
		"goruntime.KeepAlive(reqOwner)",
		"err = source.Send(ctx, AllServiceClientStreamNativeStreamRequest{Name: name, Enabled: enabled, Child: child})",
		"resp, err := source.Finish(ctx)",
		"return convertAllServiceClientStreamNativeToMessageResponse(resp.Accepted, resp.Payload)",
	} {
		assertGeneratedContentContains(t, plugin, messageServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"startClientStream func(ctx context.Context) (*allServiceClientStreamMessageStreamSession, error)",
		"messageBinding := &allServiceMessageActiveBinding{}",
		"allServiceCurrentMessageBinding.Store(messageBinding)",
	)
}

func TestRenderRuntimeGlueRoutesNativeStreamsToMessageSessionsWithConverter(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	const nativeServerFile = "test/v1/complete_service_plan.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"source := newallServiceClientStreamConnectDirectMessageStreamSession(ctx, server)",
		"source, err := newallServiceServerStreamConnectDirectMessageStreamSession(ctx, server, req)",
		"source := newallServiceBidiStreamConnectDirectMessageStreamSession(ctx, server)",
		"rpcruntime.NewConnectClientStream[AllRequest](conn)",
		"rpcruntime.NewConnectServerStream[AllReply](conn)",
		"rpcruntime.NewConnectBidiStream[AllRequest, AllReply](conn)",
	} {
		assertGeneratedContentContains(t, plugin, runtimeFile, fragment)
	}
	for _, fragment := range []string{
		"messageReq, err := convertAllServiceClientStreamNativeToMessageRequest(name, enabled, child)",
		"return source.Send(ctx, messageReq)",
		"messageResp, err := source.Finish(ctx)",
		"return convertAllServiceClientStreamMessageToNativeResponse(messageResp)",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"source, err := serverBinding.StartServerStream(ctx, messageReq)",
		"source := newallServiceClientStreamConnectDirectMessageStreamSession(ctx, server)\n\t\tvar err error\n\t\tif err != nil",
		"source := newallServiceBidiStreamConnectDirectMessageStreamSession(ctx, server)\n\t\tvar err error\n\t\tif err != nil",
	)
}

func TestRenderRuntimeGlueDoesNotGenerateActiveStreamClosures(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const runtimeFile = "test/v1/complete_service_plan.all_service.runtime.rpccgo.go"
	assertGeneratedFileContentDoesNotContain(t, plugin, runtimeFile,
		"startClientStream func(ctx context.Context) (*allServiceClientStreamMessageStreamSession, error)",
		"send func(ctx context.Context",
		"recv func(ctx context.Context",
		"finish func(ctx context.Context",
		"cancel func(ctx context.Context",
		"allServiceCurrentNativeBinding",
		"allServiceCurrentMessageBinding",
	)
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
	if got := err.Error(); !strings.Contains(got, "Mystery") || !strings.Contains(got, "render capability does not match contract capabilities") {
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
		t.Fatal("renderRuntimeFile() error = nil, want adapter method symbol collision error")
	}
	if got := err.Error(); !strings.Contains(got, "StartFoo") || !strings.Contains(got, "collides") {
		t.Fatalf("renderRuntimeFile() error = %q, want colliding adapter method symbol", got)
	}
}

func runtimeTestMethod(name string, streaming StreamingKind, nativeEntryMethod string, messageEntryMethod string, streamShape runtimeStreamShape) MethodPlan {
	method := MethodPlan{Name: name, GoName: name, FullName: "test.v1.Greeter." + name, Streaming: streaming}
	method.RenderPlan = MethodRenderPlan{
		Stream: runtimeTestStreamCapabilityProjection(streamShape),
		Symbols: RenderSymbolsPlan{
			NativeEntryMethod:    nativeEntryMethod,
			MessageEntryMethod:   messageEntryMethod,
			NativeAdapterMethod:  nativeEntryMethod,
			MessageAdapterMethod: messageEntryMethod,
		},
		Errors: RenderErrorsPlan{
			NativeServerUnavailableErr:  "GreeterNativeServerUnavailableErr",
			MessageServerUnavailableErr: "GreeterMessageServerUnavailableErr",
			UnknownActiveContractErr:    "GreeterUnknownActiveContractErr",
		},
	}
	if streamShape != runtimeStreamUnary {
		method.Contract.Stream = runtimeTestStreamCapability(streamShape)
		method.RenderPlan.Symbols.NativeStreamRequestType = "Greeter" + name + "NativeStreamRequest"
		method.RenderPlan.Symbols.NativeStreamResponseType = "Greeter" + name + "NativeStreamResponse"
	}
	return method
}

func runtimeTestStreamCapabilityProjection(streamShape runtimeStreamShape) StreamCapabilityProjectionPlan {
	switch streamShape {
	case runtimeStreamClient:
		return StreamCapabilityProjectionPlan{Streaming: true, CanSend: true, FinishReturnsResponse: true, RequiresCodec: true}
	case runtimeStreamServer:
		return StreamCapabilityProjectionPlan{Streaming: true, CanRecv: true, RequiresCodec: true}
	case runtimeStreamBidi:
		return StreamCapabilityProjectionPlan{Streaming: true, CanSend: true, CanRecv: true, CanCloseSend: true, RequiresCodec: true}
	default:
		return StreamCapabilityProjectionPlan{RequiresCodec: true}
	}
}

func runtimeTestStreamCapability(streamShape runtimeStreamShape) StreamCapabilityContractPlan {
	switch streamShape {
	case runtimeStreamClient:
		return StreamCapabilityContractPlan{CanSend: true, FinishReturnsResponse: true}
	case runtimeStreamServer:
		return StreamCapabilityContractPlan{CanRecv: true}
	case runtimeStreamBidi:
		return StreamCapabilityContractPlan{CanSend: true, CanRecv: true, CanCloseSend: true}
	default:
		return StreamCapabilityContractPlan{}
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
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\tgithub.com/ygrpc/rpccgo v0.0.0\n\tgoogle.golang.org/protobuf v1.36.11\n)\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
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
