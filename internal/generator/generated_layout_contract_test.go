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
		"test/cmd/rpc/main.go",
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
	assertGeneratedPackage(t, plugin, "test/cmd/rpc/main.go", "package main")
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
		"type greeterNativeBinding struct {",
		"type greeterMessageBinding struct {",
		"type greeterNativeActiveBinding struct {",
		"type greeterMessageActiveBinding struct {",
		"greeterCurrentNativeBinding",
		"greeterCurrentMessageBinding",
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
		"SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error)",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func InvokeGreeterNativeSayHello(ctx context.Context) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func InvokeGreeterMessageSayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go",
		"func RegisterGreeterCGOMessageServer(server GreeterCGOMessageServer) error {",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"type greeterGoNativeEntry struct {",
		"func (a *greeterGoNativeEntry)",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.server.message.rpccgo.go",
		"type greeterCGOMessageEntry struct {",
		"func (a *greeterCGOMessageEntry)",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"func RegisterGreeterGoNativeServer(server GreeterNativeServer) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go",
		"func convertGreeterSayHelloMessageToNativeRequest(msg *HelloRequest) (any, error) {",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go",
		"NativeRequestView",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"func RegisterGreeterConnectRemoteServer(client GreeterClient) error {",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		`const greeterServiceID rpcruntime.ServiceID = "test.v1.Greeter"`,
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"rpcruntime.RegisterServer(greeterServiceID, rpcruntime.RegisteredServer{",
	)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go",
		"Kind:   rpcruntime.ServerKindConnectRemote",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.server.native.cgo.rpccgo.go",
		"//export rpccgoNativeTestv1GreeterRegister",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.client.native.cgo.rpccgo.go",
		"//export rpccgoNativeTestv1GreeterSayHello",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"typedef void (*rpccgo_free_callback)(void*);",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"//export rpccgoRegisterFree",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"func rpccgoRegisterFree(callback C.rpccgo_free_callback) C.int32_t {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"//export rpccgoStoreErrorText",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"func rpccgoStoreErrorText(text *C.char, textLen C.int32_t) C.int32_t {",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"//export rpccgoTakeErrorText",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/main.go",
		"func main() {}",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"func main() {}",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"if textPtr == nil || textLen == nil {\n\t\treturn -1\n\t}",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"*textPtr = 0",
		"*textLen = 0",
		"return 1",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/cmd/rpc/greeter.greeter.server.native.cgo.rpccgo.go",
		"CGONativeServerErrorTextForExport",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"//export rpccgoRelease",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go",
		"//export rpccgoMsgTestv1GreeterRegister",
	)
	assertGeneratedContentContains(t, plugin, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
		"//export rpccgoMsgTestv1GreeterSayHello",
	)
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
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
