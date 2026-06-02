package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
)

func TestGeneratedLayoutContract(t *testing.T) {
	plugin := newGeneratedLayoutPlugin(t)

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/v1/greeter.greeter.server.message.rpccgo.go",
		"test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.server.native.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.client.native.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go",
		"test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.codec.rpccgo.go",
	})
	assertGeneratedPackage(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go", "package testv1")
	assertGeneratedPackage(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go", "package testv1")
	assertGeneratedPackage(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go", "package testv1")
	assertGeneratedPackage(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go", "package testv1")
	assertGeneratedPackage(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go", "package main")
	assertGeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.server.native.cgo.rpccgo.go", "package main")
	assertGeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.client.native.cgo.rpccgo.go", "package main")
	assertGeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go", "package main")
	assertGeneratedPackage(t, plugin, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go", "package main")
}

func TestGeneratedLayoutPublicAPIContract(t *testing.T) {
	plugin := newGeneratedLayoutPlugin(t)

	assertGeneratedFileContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"type GreeterNativeAdapter interface {",
		"type GreeterMessageAdapter interface {",
		"type GreeterNativeServer interface {",
		"type UnimplementedGreeterNativeServer struct{}",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"type GreeterNativeServer interface {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"type UnimplementedGreeterNativeServer struct{}",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go",
		"type GreeterCGOMessageServer interface {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go",
		"SayHello(ctx context.Context, req []byte) ([]byte, error)",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func InvokeGreeterNativeSayHello(ctx context.Context) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func InvokeGreeterMessageSayHello(ctx context.Context, req []byte) ([]byte, error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go",
		"func RegisterGreeterCGOMessageServer(server GreeterCGOMessageServer) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"func RegisterGreeterGoNativeServer(server GreeterNativeServer) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go",
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) (*greeterSayHelloNativeRequestView, error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func RegisterGreeterConnectRemoteServer(client GreeterClient) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.server.native.cgo.rpccgo.go",
		"//export rpccgo_native_testv1_Greeter_register",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.client.native.cgo.rpccgo.go",
		"//export rpccgo_native_testv1_Greeter_SayHello",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"//export rpccgo_take_error_text",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"//export rpccgo_release",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go",
		"//export rpccgo_msg_testv1_Greeter_register",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
		"func CallGreeterSayHelloMessageUnary",
	)
}

func TestGeneratedLayoutRejectsOldBootstrapNames(t *testing.T) {
	plugin := newGeneratedLayoutPlugin(t)
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

func newGeneratedLayoutPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()

	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect|native\n")
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", file)
	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}
	return plugin
}

func assertGeneratedPackage(t *testing.T, plugin *protogen.Plugin, filename string, want string) {
	t.Helper()

	assertGeneratedContentContains(t, plugin, filename, want)
}
