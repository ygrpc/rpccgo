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
		"rpccgo native message codec generated file for Greeter",
		`var greeterNativeMessageCodecNotReadyErr = errors.New("rpccgo: native message codec is not implemented in this build")`,
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) error {",
		"if err := proto.Unmarshal(data, &msg); err != nil {",
		"return err",
		"func convertGreeterSayHelloNativeToMessageRequest() ([]byte, error) {",
		"msg := &HelloRequest{}",
		"data, err := proto.Marshal(msg)",
		"func convertGreeterSayHelloMessageToNativeResponse(data []byte) error {",
		"func convertGreeterSayHelloNativeToMessageResponse() ([]byte, error) {",
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
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) error {",
		"if err := proto.Unmarshal(data, &msg); err != nil {",
		"return err",
		"func convertGreeterSayHelloMessageToNativeResponse(data []byte) error {",
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
		"func convertGreeterSayHelloNativeToMessageRequest() ([]byte, error) {",
		"msg := &HelloRequest{}",
		"data, err := proto.Marshal(msg)",
		`return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)`,
		"func convertGreeterSayHelloNativeToMessageResponse() ([]byte, error) {",
		"msg := &HelloReply{}",
		`return nil, fmt.Errorf("rpccgo: native response protobuf marshal failed: %w", err)`,
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
}

func TestRenderStageFilesEmitsCodecWithoutRemoteAdapterFiles(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	plans[0].Services[0].Adapters = AdapterSelection{Tokens: []AdapterToken{AdapterTokenNative}}
	AttachNativeFileFamilyPlan(&plans[0])
	AttachMessageFileFamilyPlan(&plans[0])

	if err := RenderStageFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderStageFiles() error = %v", err)
	}

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/cgo/greeter.exports.cgo.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.codec.rpccgo.go",
	})
	assertNoGeneratedFilenameContains(t, plugin, ".connect.", ".grpc.", ".remote.")
	assertGeneratedContentDoesNotContain(t, plugin, ".remote.")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go", "rpccgo native message codec generated file for Greeter")
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
