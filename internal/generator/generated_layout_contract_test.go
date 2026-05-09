package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
)

func TestStage7GeneratedLayoutContract(t *testing.T) {
	plugin := newStage7GeneratedLayoutPlugin(t)

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.server.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.server.connect.rpccgo.go",
		"test/v1/greeter.greeter.server.grpc.rpccgo.go",
		"test/v1/greeter.greeter.remote.connect.rpccgo.go",
		"test/v1/greeter.greeter.remote.grpc.rpccgo.go",
		"test/v1/greeter.greeter.codec.rpccgo.go",
	})
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.server.connect.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.server.grpc.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.remote.connect.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", "package testv1")
	assertStage7GeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.server.cgo.rpccgo.go", "package main")
	assertStage7GeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.client.cgo.rpccgo.go", "package main")
	assertStage7GeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go", "package main")
	assertStage7GeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go", "package main")
}

func TestStage7PublicAPIContract(t *testing.T) {
	plugin := newStage7GeneratedLayoutPlugin(t)

	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"type GreeterNativeAdapter interface {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"type GreeterMessageAdapter interface {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func NewGreeterCGONativeClientBridge() GreeterCGONativeClientBridge {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func RegisterGreeterCGOMessageActiveServer(kind rpcruntime.ServerKind, adapter GreeterMessageAdapter) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"func RegisterGreeterGoNativeServer(server GreeterNativeServer) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go",
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) (*HelloRequest, error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.connect.rpccgo.go",
		"func NewGreeterConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.grpc.rpccgo.go",
		"func RegisterGreeterGRPCServer(registrar grpc.ServiceRegistrar) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.remote.connect.rpccgo.go",
		"func RegisterGreeterConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.remote.grpc.rpccgo.go",
		"func RegisterGreeterGRPCRemoteServer(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.server.cgo.rpccgo.go",
		"func RegisterGreeterCGONativeServer(callbacks *C.GreeterCGONativeServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterNativeAdapter], error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
		"func CallGreeterSayHelloNativeUnary(ctx context.Context, input *GreeterSayHelloNativeUnaryInput, output *GreeterSayHelloNativeUnaryOutput) int32 {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go",
		"func RegisterGreeterCGOMessageServer(callbacks *C.GreeterCGOMessageServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterMessageAdapter], error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
		"func CallGreeterSayHelloMessageUnary(ctx context.Context, requestPtr uintptr, requestLen int32, output *GreeterMessageOutput) int32 {",
	)
}

func TestStage7GeneratedLayoutRejectsOldBootstrapNames(t *testing.T) {
	plugin := newStage7GeneratedLayoutPlugin(t)
	banned := []string{
		"provider registry",
		"framework selector",
		"multi provider",
		"dual provider",
		"bootstrap",
		"goclient.export",
		"goserver.export",
		"native_forwarding_client",
		"native_forwarding_server",
	}

	for _, generated := range plugin.Response().GetFile() {
		name := strings.ToLower(generated.GetName())
		content := strings.ToLower(generated.GetContent())
		for _, token := range banned {
			if strings.Contains(name, token) || strings.Contains(content, token) {
				t.Fatalf("generated %s contains old layout token %q", generated.GetName(), token)
			}
		}
	}
}

func newStage7GeneratedLayoutPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()

	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect|msg-grpc|native\n")
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", file)
	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}
	return plugin
}

func assertStage7GeneratedPackage(t *testing.T, plugin *protogen.Plugin, filename string, want string) {
	t.Helper()

	assertGeneratedContentContains(t, plugin, filename, want)
}
