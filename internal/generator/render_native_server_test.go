package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderNativeServerDefinesInterfaceAdapterAndRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/complete_service_plan.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceNativeServer interface {",
		"Unary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error)",
		"ClientStream(ctx context.Context, stream AllServiceClientStreamNativeClientStream) (bool, []byte, error)",
		"ServerStream(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes, stream AllServiceServerStreamNativeServerStream) error",
		"BidiStream(ctx context.Context, stream AllServiceBidiStreamNativeBidiStream) error",
		"type UnimplementedAllServiceNativeServer struct{}",
		`errors.New("rpccgo: AllService.Unary native server method is not implemented")`,
		"func RegisterAllServiceGoNativeServer(server AllServiceNativeServer) error {",
		`errors.New("rpccgo: AllService go native server is nil")`,
		"return registerAllServiceGoNativeServer(server)",
		"func registerAllServiceGoNativeServer(server AllServiceNativeServer) error {",
		"Kind:   rpcruntime.ServerKindGoNative,",
		"func RegisterAllServiceCGONativeServer(server AllServiceNativeServer) error {",
		"Kind:   rpcruntime.ServerKindCGONative,",
		"Server: server,",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeServerFile,
		"Unary(ctx context.Context, req *AllRequest)",
		"(*AllReply, error)",
		"StartServerStream(ctx context.Context, req *AllRequest)",
		"Send(ctx context.Context, req *AllRequest) error",
		"Recv(ctx context.Context) (*AllReply, error)",
		"ctx, nil",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeServerFile,
		"connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf",
		"type allServiceGoNativeAdapter struct {", "type allServiceNativeServerAdapter struct {",
		"type allServiceNativeBinding struct {",
		"rpcruntime.AdapterSnapshot", "registerAllServiceActiveServer",
	)
}

func TestRenderNativeServerDefinesStreamingMethodSignatures(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/complete_service_plan.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceClientStreamNativeClientStream interface {",
		"Recv(ctx context.Context) (*rpcruntime.RpcString, bool, *rpcruntime.RpcBytes, error)",
		"type AllServiceServerStreamNativeServerStream interface {",
		"Send(ctx context.Context, accepted bool, payload []byte) error",
		"type AllServiceBidiStreamNativeBidiStream interface {",
		"Recv(ctx context.Context) (*rpcruntime.RpcString, bool, *rpcruntime.RpcBytes, error)",
		"Send(ctx context.Context, accepted bool, payload []byte) error",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}

	for _, fragment := range []string{
		"type AllServiceClientStreamNativeStreamRequest struct {",
		"type AllServiceClientStreamNativeStreamResponse struct {",
		"type allServiceClientStreamGoNativeClientStreamingServer struct {",
		"stream rpcruntime.ClientStreamingServer[AllServiceClientStreamNativeStreamRequest]",
		"func allServiceClientStreamGoNativeStart(ctx context.Context, server AllServiceNativeServer) (rpcruntime.ClientStreamingClient[AllServiceClientStreamNativeStreamRequest, AllServiceClientStreamNativeStreamResponse], error)",
		"func allServiceServerStreamGoNativeStart(ctx context.Context, server AllServiceNativeServer, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (rpcruntime.ServerStreamingClient[AllServiceServerStreamNativeStreamResponse], error)",
		"func allServiceBidiStreamGoNativeStart(ctx context.Context, server AllServiceNativeServer) (rpcruntime.BidiStreamingClient[AllServiceBidiStreamNativeStreamRequest, AllServiceBidiStreamNativeStreamResponse], error)",
		"client, stream, streamCtx := rpcruntime.NewClientStreaming[AllServiceClientStreamNativeStreamRequest, AllServiceClientStreamNativeStreamResponse]",
		"stream.Complete(AllServiceClientStreamNativeStreamResponse{Accepted: accepted, Payload: payload}, err)",
		"req, err := s.stream.Recv(ctx)",
		"type allServiceServerStreamGoNativeServerStreamingServer struct {",
		"stream rpcruntime.ServerStreamingServer[AllServiceServerStreamNativeStreamResponse]",
		"client, stream, streamCtx := rpcruntime.NewServerStreaming[AllServiceServerStreamNativeStreamResponse]",
		"return s.stream.Send(ctx, AllServiceServerStreamNativeStreamResponse{Accepted: accepted, Payload: payload})",
		"type allServiceBidiStreamGoNativeBidiStreamingServer struct {",
		"stream rpcruntime.BidiStreamingServer[AllServiceBidiStreamNativeStreamRequest, AllServiceBidiStreamNativeStreamResponse]",
		"func (s *allServiceBidiStreamGoNativeBidiStreamingServer) Recv(ctx context.Context) (*rpcruntime.RpcString, bool, *rpcruntime.RpcBytes, error)",
		"return s.stream.Send(ctx, AllServiceBidiStreamNativeStreamResponse{Accepted: accepted, Payload: payload})",
		"client, stream, streamCtx := rpcruntime.NewBidiStreaming[AllServiceBidiStreamNativeStreamRequest, AllServiceBidiStreamNativeStreamResponse]",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	nativeServerContent := generatedFileContent(t, plugin, nativeServerFile)
	assertContentOrder(t, nativeServerContent,
		"type AllServiceClientStreamNativeStreamRequest struct {",
		"type AllServiceClientStreamNativeStreamResponse struct {",
		"type allServiceClientStreamGoNativeClientStreamingServer struct {",
		"func (s *allServiceClientStreamGoNativeClientStreamingServer) Recv(ctx context.Context)",
		"type AllServiceServerStreamNativeStreamResponse struct {",
		"type allServiceServerStreamGoNativeServerStreamingServer struct {",
		"func (s *allServiceServerStreamGoNativeServerStreamingServer) Send(ctx context.Context",
		"type AllServiceBidiStreamNativeStreamRequest struct {",
		"type AllServiceBidiStreamNativeStreamResponse struct {",
		"type allServiceBidiStreamGoNativeBidiStreamingServer struct {",
		"func (s *allServiceBidiStreamGoNativeBidiStreamingServer) Recv(ctx context.Context)",
		"func (s *allServiceBidiStreamGoNativeBidiStreamingServer) Send(ctx context.Context",
		"func AllServiceNativeClientStreamSend(ctx context.Context, handle rpcruntime.StreamHandle",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeServerFile,
		"type allServiceGoNativeEntry struct {",
		"func (a *allServiceGoNativeEntry)",
		"type allServiceClientStreamGoNativeClientStreamSessionRequest struct {",
		"type allServiceClientStreamGoNativeClientStreamSessionResult struct {",
		"case s.requests <- req:",
		"responses     chan allServiceServerStreamGoNativeServerStreamSessionResponse",
		"received      chan struct{}",
		"doneRequested bool",
		"return AllServiceServerStreamNativeStreamResponse{}, io.EOF",
		"s.closeSendOnce.Do(func() { close(s.sendDone) })",
		"GoNativeBidiStreamFacade",
		"GoNativeBidiStreamSession",
	)
}

func TestRenderNativeServerRejectsUnknownStreamingKind(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := nativeServerCollisionTestFilePlan("Greeter", []MethodPlan{{
		Name:      "Mystery",
		GoName:    "Mystery",
		FullName:  "test.v1.Greeter.Mystery",
		Streaming: StreamingKind(99),
		Request:   MethodIOPlan{GoName: "HelloRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloRequest"},
		Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
	}})

	err := renderNativeServerFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
	if err == nil {
		t.Fatal("renderNativeServerFile() error = nil, want unknown native server streaming kind error")
	}
	if got := err.Error(); !strings.Contains(got, "Mystery") || !strings.Contains(got, "unknown native server streaming kind") {
		t.Fatalf("renderNativeServerFile() error = %q, want method name and unknown native server streaming kind", got)
	}
}

func TestRenderNativeServerRejectsGeneratedSymbolCollisions(t *testing.T) {
	tests := []struct {
		name      string
		service   ServicePlan
		wantError string
	}{
		{
			name: "service native server collides with request message",
			service: nativeServerCollisionTestService("AllService", []MethodPlan{{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllServiceNativeServer", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllServiceNativeServer"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
			}}),
			wantError: "AllServiceNativeServer",
		},
		{
			name: "unimplemented native server collides with request message",
			service: nativeServerCollisionTestService("AllService", []MethodPlan{{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "UnimplementedAllServiceNativeServer", GoImportPath: "example.com/test/v1", FullName: "test.v1.UnimplementedAllServiceNativeServer"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
			}}),
			wantError: "UnimplementedAllServiceNativeServer",
		},
		{
			name: "method stream interface collides with response message",
			service: nativeServerCollisionTestService("AllService", []MethodPlan{{
				Name:      "ClientStream",
				GoName:    "ClientStream",
				FullName:  "test.v1.AllService.ClientStream",
				Streaming: StreamingKindClientStreaming,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllServiceClientStreamNativeClientStream", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllServiceClientStreamNativeClientStream"},
			}}),
			wantError: "AllServiceClientStreamNativeClientStream",
		},
		{
			name: "private server mapper collision",
			service: nativeServerCollisionTestService("AllService", []MethodPlan{
				{
					Name:      "Foo",
					GoName:    "Foo",
					FullName:  "test.v1.AllService.Foo",
					Streaming: StreamingKindClientStreaming,
					Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
					Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
				},
				{
					Name:      "FooGoNativeClientStreamingServer",
					GoName:    "FooGoNativeClientStreamingServer",
					FullName:  "test.v1.AllService.FooGoNativeClientStreamingServer",
					Streaming: StreamingKindClientStreaming,
					Request:   MethodIOPlan{GoName: "allServiceFooGoNativeClientStreamingServer", GoImportPath: "example.com/test/v1", FullName: "test.v1.allServiceFooGoNativeClientStreamingServer"},
					Response:  MethodIOPlan{GoName: "OtherReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.OtherReply"},
				},
			}),
			wantError: "allServiceFooGoNativeClientStreamingServer",
		},
		{
			name: "registration function collides with response message",
			service: nativeServerCollisionTestService("AllService", []MethodPlan{{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "RegisterAllServiceGoNativeServer", GoImportPath: "example.com/test/v1", FullName: "test.v1.RegisterAllServiceGoNativeServer"},
			}}),
			wantError: "RegisterAllServiceGoNativeServer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
			plan := nativeServerCollisionTestFilePlan(tt.service.GoName, tt.service.Methods)
			err := renderNativeServerFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
			if err == nil {
				t.Fatal("renderNativeServerFile() error = nil, want native server symbol collision")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantError) || !strings.Contains(got, "collides") {
				t.Fatalf("renderNativeServerFile() error = %q, want collision for %q", got, tt.wantError)
			}
		})
	}
}

func TestRenderNativeServerGeneratedSourceCompiles(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/generated\n\ngo 1.24.4\n\nrequire (\n\tgithub.com/ygrpc/rpccgo v0.0.0\n\tgoogle.golang.org/protobuf v1.36.11\n)\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
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
		t.Fatalf("generated native server go test failed: %v\n%s", err, out)
	}
}

func TestRenderNativeServerUnimplementedHelperSupportsPartialImplementation(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

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
	writeNativeServerCompileStubs(t, tmp)
	writePartialNativeServerBehaviorTest(t, tmp)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated partial native server go test failed: %v\n%s", err, out)
	}
}

func TestRenderNativeServerNilRegistrationClearsCurrentServer(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPluginGenerating(t, "paths=source_relative", "test/v1/complete_service_plan.proto", file)

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
	writeNativeServerCompileStubs(t, tmp)
	writeNativeRegistrationClearBehaviorTest(t, tmp)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated native registration clear test failed: %v\n%s", err, out)
	}
}

func writePartialNativeServerBehaviorTest(t *testing.T, root string) {
	t.Helper()

	const content = `package generated_test

import (
	context "context"
	strings "strings"
	testing "testing"

	testv1 "example.com/test/v1"
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

type partialAllServiceNativeServer struct {
	testv1.UnimplementedAllServiceNativeServer
}

func (partialAllServiceNativeServer) ServerStream(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes, stream testv1.AllServiceServerStreamNativeServerStream) error {
	return nil
}

func TestPartialNativeServerUsesUnimplementedFallback(t *testing.T) {
	if err := testv1.RegisterAllServiceGoNativeServer(partialAllServiceNativeServer{}); err != nil {
		t.Fatalf("RegisterAllServiceGoNativeServer() error = %v", err)
	}

	accepted, payload, err := testv1.InvokeAllServiceNativeUnary(context.Background(), rpcruntime.EmptyRpcString(), false, rpcruntime.EmptyRpcBytes())
	if err == nil {
		t.Fatal("InvokeAllServiceNativeUnary() error = nil, want unimplemented error")
	}
	if accepted || payload != nil {
		t.Fatalf("InvokeAllServiceNativeUnary() = (%v, %v, %v), want zero values and error", accepted, payload, err)
	}
	if got := err.Error(); !strings.Contains(got, "rpccgo: AllService.Unary native server method is not implemented") {
		t.Fatalf("InvokeAllServiceNativeUnary() error = %q, want method-specific unimplemented error", got)
	}
}
`
	target := filepath.Join(root, "partial_native_server_test.go")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write partial native server behavior test: %v", err)
	}
}

func writeNativeRegistrationClearBehaviorTest(t *testing.T, root string) {
	t.Helper()

	const content = `package testv1

import (
	context "context"
	errors "errors"
	strings "strings"
	testing "testing"

	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

type registrationClearAllServiceNativeServer struct {
	UnimplementedAllServiceNativeServer
}

func (registrationClearAllServiceNativeServer) Unary(ctx context.Context, name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (bool, []byte, error) {
	return true, []byte("ok"), nil
}

func TestRegisterAllServiceGoNativeServerNilClearsCurrentServer(t *testing.T) {
	if err := RegisterAllServiceGoNativeServer(registrationClearAllServiceNativeServer{}); err != nil {
		t.Fatalf("RegisterAllServiceGoNativeServer(valid) error = %v", err)
	}
	if _, err := rpcruntime.LoadServer(allServiceServiceID); err != nil {
		t.Fatalf("LoadServer(valid) error = %v", err)
	}

	err := RegisterAllServiceGoNativeServer(nil)
	if err == nil || !strings.Contains(err.Error(), "go native server is nil") {
		t.Fatalf("RegisterAllServiceGoNativeServer(nil) error = %v, want go native server is nil", err)
	}
	if _, err := rpcruntime.LoadServer(allServiceServiceID); !errors.Is(err, rpcruntime.ErrNoRegisteredServer) {
		t.Fatalf("LoadServer(after nil register) error = %v, want ErrNoRegisteredServer", err)
	}
}
`
	target := filepath.Join(root, "test/v1/native_registration_clear_test.go")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write native registration clear behavior test: %v", err)
	}
}

func writeNativeServerCompileStubs(t *testing.T, root string) {
	t.Helper()

	const content = `package testv1

import (
	context "context"

	connect "connectrpc.com/connect"
	grpc "google.golang.org/grpc"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type AllRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type AllReply struct {
	Accepted bool
	Payload []byte
}
type DefaultRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type DefaultReply struct {
	Accepted bool
	Payload []byte
}
type ConnectRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type ConnectReply struct {
	Accepted bool
	Payload []byte
}
type GrpcRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type GrpcReply struct {
	Accepted bool
	Payload []byte
}
type MessageRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type MessageReply struct {
	Accepted bool
	Payload []byte
}
type ConnectNativeRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type ConnectNativeReply struct {
	Accepted bool
	Payload []byte
}
type NativeOnlyRequest struct {
	Name string
	Enabled bool
	Child []byte
}
type NativeOnlyReply struct {
	Accepted bool
	Payload []byte
}

type AllServiceHandler interface {
	Unary(context.Context, *AllRequest) (*AllReply, error)
	ClientStream(context.Context, *connect.ClientStream[AllRequest]) (*AllReply, error)
	ServerStream(context.Context, *AllRequest, *connect.ServerStream[AllReply]) error
	BidiStream(context.Context, *connect.BidiStream[AllRequest, AllReply]) error
}
type AllServiceClient interface {
	Unary(context.Context, *AllRequest) (*AllReply, error)
	ClientStream(context.Context) (*connect.ClientStreamForClientSimple[AllRequest, AllReply], error)
	ServerStream(context.Context, *AllRequest) (*connect.ServerStreamForClient[AllReply], error)
	BidiStream(context.Context) (*connect.BidiStreamForClientSimple[AllRequest, AllReply], error)
}
type DefaultServiceHandler interface {
	DefaultUnary(context.Context, *DefaultRequest) (*DefaultReply, error)
}
type DefaultServiceClient interface {
	DefaultUnary(context.Context, *DefaultRequest) (*DefaultReply, error)
}
type ConnectServiceHandler interface {
	ConnectUnary(context.Context, *ConnectRequest) (*ConnectReply, error)
}
type ConnectServiceClient interface {
	ConnectUnary(context.Context, *ConnectRequest) (*ConnectReply, error)
}
type MessageServiceHandler interface {
	MessageUnary(context.Context, *MessageRequest) (*MessageReply, error)
}
type MessageServiceClient interface {
	MessageUnary(context.Context, *MessageRequest) (*MessageReply, error)
}
type ConnectNativeServiceHandler interface {
	ConnectNativeUnary(context.Context, *ConnectNativeRequest) (*ConnectNativeReply, error)
}
type ConnectNativeServiceClient interface {
	ConnectNativeUnary(context.Context, *ConnectNativeRequest) (*ConnectNativeReply, error)
}
type NativeOnlyServiceHandler interface {
	NativeOnlyUnary(context.Context, *NativeOnlyRequest) (*NativeOnlyReply, error)
}
type NativeOnlyServiceClient interface {
	NativeOnlyUnary(context.Context, *NativeOnlyRequest) (*NativeOnlyReply, error)
}
type GrpcServiceServer interface {
	GrpcUnary(context.Context, *GrpcRequest) (*GrpcReply, error)
}
type GrpcServiceClient interface {
	GrpcUnary(context.Context, *GrpcRequest, ...grpc.CallOption) (*GrpcReply, error)
}

func (*AllRequest) ProtoReflect() protoreflect.Message { return nil }
func (*AllReply) ProtoReflect() protoreflect.Message { return nil }
func (*DefaultRequest) ProtoReflect() protoreflect.Message { return nil }
func (*DefaultReply) ProtoReflect() protoreflect.Message { return nil }
func (*ConnectRequest) ProtoReflect() protoreflect.Message { return nil }
func (*ConnectReply) ProtoReflect() protoreflect.Message { return nil }
func (*GrpcRequest) ProtoReflect() protoreflect.Message { return nil }
func (*GrpcReply) ProtoReflect() protoreflect.Message { return nil }
func (*MessageRequest) ProtoReflect() protoreflect.Message { return nil }
func (*MessageReply) ProtoReflect() protoreflect.Message { return nil }
func (*ConnectNativeRequest) ProtoReflect() protoreflect.Message { return nil }
func (*ConnectNativeReply) ProtoReflect() protoreflect.Message { return nil }
func (*NativeOnlyRequest) ProtoReflect() protoreflect.Message { return nil }
func (*NativeOnlyReply) ProtoReflect() protoreflect.Message { return nil }
`
	target := filepath.Join(root, "test/v1/complete_service_plan_stubs.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir stub dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write native server compile stubs: %v", err)
	}
}

func nativeServerCollisionTestFilePlan(serviceName string, methods []MethodPlan) FilePlan {
	return FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{
			nativeServerCollisionTestService(serviceName, methods),
		},
	}
}

func nativeServerCollisionTestService(serviceName string, methods []MethodPlan) ServicePlan {
	return ServicePlan{
		Name:       serviceName,
		GoName:     serviceName,
		FullName:   "test.v1." + serviceName,
		Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
		Methods:    methods,
		Artifacts: []GeneratedArtifactPlan{
			{Kind: GeneratedArtifactKindNativeServer, Filename: "test/v1/collision.server.native.rpccgo.go"},
		},
	}
}
