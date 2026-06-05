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

	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
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
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) (any, error) {",
		"if err := proto.Unmarshal(data, &msg); err != nil {",
		"return nil, err",
		"func convertGreeterSayHelloNativeToMessageRequest() ([]byte, error) {",
		"msg := &HelloRequest{}",
		"data, err := proto.Marshal(msg)",
		"func convertGreeterSayHelloMessageToNativeResponse(data []byte) error {",
		"func convertGreeterSayHelloNativeToMessageResponse() ([]byte, error) {",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeRequestView", "connectrpc.com/connect", "google.golang.org/grpc", ".remote.", "panic(", "TODO")
}

func TestRenderCodecFilesSkipsServiceWithoutCodecArtifact(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := FilePlan{
		GoPackageName:           "testv1",
		GoImportPath:            "example.com/test/v1",
		GeneratedFilenamePrefix: "test/v1/greeter",
		Services: []ServicePlan{
			{
				GoName: "Greeter",
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
	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/greeter.greeter.codec.rpccgo.go"
	for _, fragment := range []string{
		"func convertGreeterSayHelloMessageToNativeRequest(data []byte) (any, error) {",
		"var msg HelloRequest",
		"if err := proto.Unmarshal(data, &msg); err != nil {",
		"return nil, err",
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
	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
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

func TestCodecNativeToMessageRequestUsesUnsafeWrappersAndKeepAlive(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/complete_service_plan.all_service.codec.rpccgo.go"
	for _, fragment := range []string{
		`goruntime "runtime"`,
		"func convertAllServiceUnaryNativeToMessageRequest(name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) ([]byte, error) {",
		"msg.Name = name.UnsafeString()",
		"msg.Child = child.UnsafeBytes()",
		"data, err := proto.Marshal(msg)",
		"goruntime.KeepAlive(name)",
		"goruntime.KeepAlive(child)",
		`return nil, fmt.Errorf("rpccgo: native request protobuf marshal failed: %w", err)`,
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "msg.Name = name.SafeString()", "msg.Child = child.SafeBytes()")
}

func TestCodecMessageToNativeRequestUsesBorrowedWrappersAndReturnedOwner(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/complete_service_plan.all_service.codec.rpccgo.go"
	for _, fragment := range []string{
		"func convertAllServiceUnaryMessageToNativeRequest(data []byte) (*rpcruntime.RpcString, bool, *rpcruntime.RpcBytes, any, error) {",
		"// Returned native wrappers borrow from msg and reqOwner-owned buffers.",
		"// Callers must keep the returned owner alive until the synchronous native call returns.",
		"reqOwner := []any{&msg}",
		"name = rpcruntime.EmptyRpcString()",
		"child = rpcruntime.EmptyRpcBytes()",
		"name = rpcruntime.NewRpcString(unsafe.StringData(msg.Name), int32(len(msg.Name)), false)",
		"child = rpcruntime.NewRpcBytes(unsafe.SliceData(msg.Child), int32(len(msg.Child)), false)",
		"return name, enabled, child, reqOwner, nil",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeRequestView", "NewRpcStringView", "NewRpcBytesView", "fn func(", "goruntime.KeepAlive(&msg)", "PinString(", "PinBytes(", "PinSlice(", "defer rpcruntime.Release(ptr)", "cleanup := func()", "rpcruntime.NewRpcString(nil, 0, false)", "rpcruntime.NewRpcBytes(nil, 0, false)")
}

func TestCodecMessageToNativeRequestKeepsRepeatedBoolAndEnumRawOwnersAlive(t *testing.T) {
	file := nativeServerRepeatedFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/native_repeated.repeated_service.codec.rpccgo.go"
	for _, fragment := range []string{
		"flagsRaw := make([]byte, len(msg.Flags))",
		"reqOwner = append(reqOwner, flagsRaw)",
		"flags = rpcruntime.NewRpcBoolRepeat(unsafe.SliceData(flagsRaw), int32(len(flagsRaw)), false)",
		"moodsRaw := make([]int32, len(msg.Moods))",
		"reqOwner = append(reqOwner, moodsRaw)",
		"moods = rpcruntime.NewRpcRepeat[int32](unsafe.SliceData(moodsRaw), int32(len(moodsRaw)), false)",
		"return scores, flags, counts, ratios, moods, reqOwner, nil",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeRequestView", "NewRpcBoolRepeatView", "NewRpcRepeatView", "fn func(", "goruntime.KeepAlive(&msg)", "goruntime.KeepAlive(flagsRaw)", "goruntime.KeepAlive(moodsRaw)")
}

func TestCodecNativeToMessageRequestUsesUnsafeRepeatedWrappersAndKeepAlive(t *testing.T) {
	file := nativeServerRepeatedFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := RenderCodecFiles(plugin, firstFilePlan(t, plans)); err != nil {
		t.Fatalf("RenderCodecFiles() error = %v", err)
	}

	const codecFile = "test/v1/native_repeated.repeated_service.codec.rpccgo.go"
	for _, fragment := range []string{
		`goruntime "runtime"`,
		"func convertRepeatedServiceCheckNativeToMessageRequest(scores *rpcruntime.RpcRepeat[int32], flags *rpcruntime.RpcBoolRepeat, counts *rpcruntime.RpcRepeat[int64], ratios *rpcruntime.RpcRepeat[float64], moods *rpcruntime.RpcRepeat[int32]) ([]byte, error) {",
		"msg.Scores = scores.UnsafeSlice()",
		"msg.Flags = flags.SafeSlice()",
		"msg.Counts = counts.UnsafeSlice()",
		"msg.Ratios = ratios.UnsafeSlice()",
		"moodsRaw := moods.UnsafeSlice()",
		"goruntime.KeepAlive(scores)",
		"goruntime.KeepAlive(flags)",
		"goruntime.KeepAlive(counts)",
		"goruntime.KeepAlive(ratios)",
		"goruntime.KeepAlive(moods)",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "msg.Scores = scores.SafeSlice()", "moodsRaw := moods.SafeSlice()")
}

func TestGenerateWithOptionsEmitsCodecWithoutRemoteAdapterFiles(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/cgo/rpccgo.exports.cgo.rpccgo.go",
		"test/v1/greeter.greeter.server.message.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.native.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.native.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.codec.rpccgo.go",
	})
	assertNoGeneratedFilenameContains(t, plugin, ".connect.", ".grpc.", ".remote.")
	assertGeneratedContentDoesNotContain(t, plugin, ".remote.")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go", "rpccgo native message codec generated file for Greeter")
}

func TestGenerateWithOptionsEmitsCodecFiles(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")

	nativePlugin := newTestPlugin(t, "paths=source_relative", file)
	if _, err := GenerateWithOptions(nativePlugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}
	assertGeneratedFilenameContains(t, nativePlugin, ".codec.rpccgo.go")
}
