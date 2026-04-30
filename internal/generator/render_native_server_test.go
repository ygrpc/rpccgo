package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderNativeServerDefinesInterfaceAdapterAndRegistration(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/stage1_acceptance.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceNativeServer interface {",
		"Unary(ctx context.Context, req *AllRequest) (*AllReply, error)",
		"ClientStream(ctx context.Context) (AllServiceClientStreamNativeClientStream, error)",
		"ServerStream(ctx context.Context, req *AllRequest) (AllServiceServerStreamNativeServerStream, error)",
		`allServiceNativeRequestBridgeNotImplemented`,
		`allServiceNativeStreamBridgeNotImplemented`,
		`allServiceNativeStreamIsNil`,
		`errors.New("rpccgo: native request bridge is not implemented")`,
		`errors.New("rpccgo: native stream bridge is not implemented")`,
		`errors.New("rpccgo: native stream is nil")`,
		"type allServiceGoNativeAdapter struct {",
		"server AllServiceNativeServer",
		"func (a *allServiceGoNativeAdapter) Unary(ctx context.Context, req *AllRequest) (*AllReply, error) {",
		"return nil, allServiceNativeRequestBridgeNotImplemented",
		"return a.server.Unary(ctx, req)",
		"func (a *allServiceGoNativeAdapter) StartClientStream(ctx context.Context) (AllServiceClientStreamNativeStreamSession, error) {",
		"if stream == nil {",
		"return nil, allServiceNativeStreamIsNil",
		"func (a *allServiceGoNativeAdapter) StartServerStream(ctx context.Context, req *AllRequest) (AllServiceServerStreamNativeStreamSession, error) {",
		"stream, err := a.server.ServerStream(ctx, req)",
		"return &allServiceServerStreamGoNativeServerStreamSession{stream: stream}, nil",
		"func (a *allServiceGoNativeAdapter) StartBidiStream(ctx context.Context) error {",
		"return allServiceNativeStreamBridgeNotImplemented",
		"func RegisterAllServiceGoNativeServer(server AllServiceNativeServer) (rpcruntime.AdapterSnapshot[AllServiceNativeAdapter], error) {",
		`errors.New("rpccgo: AllService go native server is nil")`,
		"return registerAllServiceActiveServer(rpcruntime.ServerKindGoNative, &allServiceGoNativeAdapter{server: server})",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin, "ctx, nil")
	assertGeneratedContentDoesNotContain(t, plugin,
		"BidiStream(ctx context.Context) (AllServiceBidiStreamNativeBidiStream, error)",
	)
	assertGeneratedContentDoesNotContain(t, plugin, "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf")
}

func TestRenderNativeServerDefinesStreamingMethodSignatures(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeServerFile = "test/v1/stage1_acceptance.all_service.server.native.rpccgo.go"
	for _, fragment := range []string{
		"type AllServiceClientStreamNativeClientStream interface {",
		"Send(ctx context.Context, req *AllRequest) error",
		"Finish(ctx context.Context) (*AllReply, error)",
		"Cancel(ctx context.Context) error",
		"return s.stream.Send(ctx, req)",
		"return s.stream.Finish(ctx)",
		"if s.stream == nil {",
		"return allServiceNativeStreamIsNil",
		"return s.stream.Cancel(ctx)",
		"type AllServiceServerStreamNativeServerStream interface {",
		"Recv(ctx context.Context) (*AllReply, error)",
		"type allServiceServerStreamGoNativeServerStreamSession struct {",
		"stream AllServiceServerStreamNativeServerStream",
		"return s.stream.Recv(ctx)",
	} {
		assertGeneratedContentContains(t, plugin, nativeServerFile, fragment)
	}
	assertGeneratedContentDoesNotContain(t, plugin,
		"type AllServiceBidiStreamNativeBidiStream interface {",
		"return s.stream.CloseSend(ctx)",
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

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want unknown native server streaming kind error")
	}
	if got := err.Error(); !strings.Contains(got, "Mystery") || !strings.Contains(got, "unknown native server streaming kind") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want method name and unknown native server streaming kind", got)
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
			name: "private session struct collision",
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
					Name:      "FooGoNativeClientStreamSession",
					GoName:    "FooGoNativeClientStreamSession",
					FullName:  "test.v1.AllService.FooGoNativeClientStreamSession",
					Streaming: StreamingKindClientStreaming,
					Request:   MethodIOPlan{GoName: "allServiceFooGoNativeClientStreamSession", GoImportPath: "example.com/test/v1", FullName: "test.v1.allServiceFooGoNativeClientStreamSession"},
					Response:  MethodIOPlan{GoName: "OtherReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.OtherReply"},
				},
			}),
			wantError: "allServiceFooGoNativeClientStreamSession",
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
			err := RenderNativeStageFiles(plugin, nativeServerCollisionTestFilePlan(tt.service.GoName, tt.service.Methods))
			if err == nil {
				t.Fatal("RenderNativeStageFiles() error = nil, want native server symbol collision")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantError) || !strings.Contains(got, "collides") {
				t.Fatalf("RenderNativeStageFiles() error = %q, want collision for %q", got, tt.wantError)
			}
		})
	}
}

func TestRenderNativeServerGeneratedSourceCompiles(t *testing.T) {
	file := stage1AcceptanceFile()
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
		if !strings.Contains(name, ".runtime.rpccgo.go") && !strings.Contains(name, ".server.native.rpccgo.go") {
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
		t.Fatalf("generated native server go test failed: %v\n%s", err, out)
	}
}

func writeNativeServerCompileStubs(t *testing.T, root string) {
	t.Helper()

	const content = `package testv1

type AllRequest struct {
	Name string
	Enabled bool
}
type AllReply struct {
	Accepted bool
	Payload []byte
}
type DefaultRequest struct {
	Name string
	Enabled bool
}
type DefaultReply struct {
	Accepted bool
	Payload []byte
}
type ConnectRequest struct {
	Name string
	Enabled bool
}
type ConnectReply struct {
	Accepted bool
	Payload []byte
}
type GrpcRequest struct {
	Name string
	Enabled bool
}
type GrpcReply struct {
	Accepted bool
	Payload []byte
}
type MessageRequest struct {
	Name string
	Enabled bool
}
type MessageReply struct {
	Accepted bool
	Payload []byte
}
type ConnectNativeRequest struct {
	Name string
	Enabled bool
}
type ConnectNativeReply struct {
	Accepted bool
	Payload []byte
}
type NativeOnlyRequest struct {
	Name string
	Enabled bool
}
type NativeOnlyReply struct {
	Accepted bool
	Payload []byte
}
`
	target := filepath.Join(root, "test/v1/stage1_acceptance_stubs.go")
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
		Name:     serviceName,
		GoName:   serviceName,
		FullName: "test.v1." + serviceName,
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenNative}},
		Methods:  methods,
		NativeFileFamily: NativeFileFamilyPlan{
			NativeServer: GeneratedFilePlan{Filename: "test/v1/collision.server.native.rpccgo.go", Enabled: true},
		},
	}
}
