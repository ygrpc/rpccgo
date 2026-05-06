package generator

import "testing"

func TestRenderCodecFilesEmitsServiceCodecFile(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if err := RenderCodecFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/greeter.greeter.codec.rpccgo.go"
	assertGeneratedFilenames(t, plugin, []string{codecFile})
	for _, fragment := range []string{
		"package testv1",
		`errors "errors"`,
		`fmt "fmt"`,
		`proto "google.golang.org/protobuf/proto"`,
		"rpccgo native message codec stage file for Greeter",
		`var greeterNativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")`,
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) (*HelloRequest, error) {",
		"if err := proto.Unmarshal(data, &msg); err != nil {",
		`return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)`,
		"func convertGreeterSayHelloNativeToMessageRequest(req *HelloRequest) ([]byte, error) {",
		`return nil, errors.New("rpccgo: native request is nil")`,
		"data, err := proto.Marshal(req)",
		"func convertGreeterSayHelloMessageToNativeResponse(data []byte) (*HelloReply, error) {",
		`return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)`,
		"func convertGreeterSayHelloNativeToMessageResponse(resp *HelloReply) ([]byte, error) {",
		`return nil, errors.New("rpccgo: native response is nil")`,
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "connectrpc.com/connect", "google.golang.org/grpc", ".remote.", "panic(", "TODO")
}

func TestRenderCodecFilesSkipsServiceWithoutCodecNeed(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		GoPackageName:           "testv1",
		GoImportPath:            "example.com/test/v1",
		GeneratedFilenamePrefix: "test/v1/greeter",
		Services: []ServicePlan{
			{
				GoName:     "Greeter",
				NeedsCodec: false,
			},
		},
	}

	if err := RenderCodecFiles(plugin, plan); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	assertGeneratedFilenames(t, plugin, nil)
}

func TestCodecMessageToNativeRendersProtobufUnmarshalAndErrors(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderCodecFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/greeter.greeter.codec.rpccgo.go"
	for _, fragment := range []string{
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) (*HelloRequest, error) {",
		"if err := proto.Unmarshal(data, &msg); err != nil {",
		`return nil, fmt.Errorf("rpccgo: message request protobuf unmarshal failed: %w", err)`,
		"func convertGreeterSayHelloMessageToNativeResponse(data []byte) (*HelloReply, error) {",
		`return nil, fmt.Errorf("rpccgo: message response protobuf unmarshal failed: %w", err)`,
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
}

func TestCodecNativeToMessageRendersProtobufMarshalAndErrors(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderCodecFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/greeter.greeter.codec.rpccgo.go"
	for _, fragment := range []string{
		"func convertGreeterSayHelloNativeToMessageRequest(req *HelloRequest) ([]byte, error) {",
		`return nil, errors.New("rpccgo: native request is nil")`,
		"data, err := proto.Marshal(req)",
		`return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)`,
		"func convertGreeterSayHelloNativeToMessageResponse(resp *HelloReply) ([]byte, error) {",
		`return nil, errors.New("rpccgo: native response is nil")`,
		`return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)`,
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
}

func TestRenderStageFilesEmitsCodecWithoutRemoteAdapterFiles(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.server.connect.rpccgo.go",
		"test/v1/greeter.greeter.codec.rpccgo.go",
	})
	assertNoGeneratedFilenameContains(t, plugin, ".grpc.", ".remote.")
	assertGeneratedContentDoesNotContain(t, plugin, ".remote.")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.connect.rpccgo.go", `connect "connectrpc.com/connect"`)
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go", "rpccgo native message codec stage file for Greeter")
}

func TestDirectPathRenderersDoNotEmitCodecFiles(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")

	nativePlugin := newTestPlugin(t, "paths=source_relative", file)
	if _, err := GenerateWithOptions(nativePlugin, GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderNativeStageFiles) error = %v", err)
	}
	assertNoGeneratedFilenameContains(t, nativePlugin, ".codec.")

	messagePlugin := newTestPlugin(t, "paths=source_relative", file)
	if _, err := GenerateWithOptions(messagePlugin, GenerateOptions{RenderMessageStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderMessageStageFiles) error = %v", err)
	}
	assertNoGeneratedFilenameContains(t, messagePlugin, ".codec.")
}
