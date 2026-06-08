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
		"rpccgo native message codec generated file for Greeter",
		"func convertGreeterSayHelloMessageToNativeRequest(msg *HelloRequest) (any, error) {",
		`return nil, errors.New("rpccgo: message request is nil")`,
		"reqOwner := []any{msg}",
		"func convertGreeterSayHelloNativeToMessageRequest() (*HelloRequest, error) {",
		"msg := &HelloRequest{}",
		"return msg, nil",
		"func convertGreeterSayHelloMessageToNativeResponse(msg *HelloReply) error {",
		`errors.New("rpccgo: message response is nil")`,
		"func convertGreeterSayHelloNativeToMessageResponse() (*HelloReply, error) {",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeRequestView", "connectrpc.com/connect", "google.golang.org/grpc", ".remote.", "panic"+"(", "TODO", "proto.Marshal", "proto.Unmarshal", "[]byte, error")
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeMessageCodecNotReadyErr", "native message codec is "+"not implemented")
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

func TestCodecMessageToNativeRendersTypedMessagesAndErrors(t *testing.T) {
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
		"func convertGreeterSayHelloMessageToNativeRequest(msg *HelloRequest) (any, error) {",
		`return nil, errors.New("rpccgo: message request is nil")`,
		"reqOwner := []any{msg}",
		"return reqOwner, nil",
		"func convertGreeterSayHelloMessageToNativeResponse(msg *HelloReply) error {",
		`err := errors.New("rpccgo: message response is nil")`,
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
}

func TestCodecNativeToMessageRendersTypedMessages(t *testing.T) {
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
		"func convertGreeterSayHelloNativeToMessageRequest() (*HelloRequest, error) {",
		"msg := &HelloRequest{}",
		"return msg, nil",
		"func convertGreeterSayHelloNativeToMessageResponse() (*HelloReply, error) {",
		"msg := &HelloReply{}",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "proto.Marshal", "protobuf marshal failed")
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
		"func convertAllServiceUnaryNativeToMessageRequest(name *rpcruntime.RpcString, enabled bool, child *rpcruntime.RpcBytes) (*AllRequest, error) {",
		"msg.Name = name.UnsafeString()",
		"msg.Child = child.UnsafeBytes()",
		"goruntime.KeepAlive(name)",
		"goruntime.KeepAlive(child)",
		"return msg, nil",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "msg.Name = name.SafeString()", "msg.Child = child.SafeBytes()", "proto.Marshal")
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
		"func convertAllServiceUnaryMessageToNativeRequest(msg *AllRequest) (*rpcruntime.RpcString, bool, *rpcruntime.RpcBytes, any, error) {",
		`return nil, false, nil, nil, errors.New("rpccgo: message request is nil")`,
		"// Returned native wrappers borrow from msg and reqOwner-owned buffers.",
		"// Callers must keep the returned owner alive until the synchronous native call returns.",
		"reqOwner := []any{msg}",
		"var err error",
		"name = rpcruntime.EmptyRpcString()",
		"child = rpcruntime.EmptyRpcBytes()",
		"name, err = rpcruntime.NewRpcStringChecked(unsafe.StringData(msg.Name), int32(len(msg.Name)), false)",
		"child, err = rpcruntime.NewRpcBytesChecked(unsafe.SliceData(msg.Child), int32(len(msg.Child)), false)",
		"return nil, false, nil, reqOwner, err",
		"return name, enabled, child, reqOwner, nil",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeRequestView", "NewRpcStringView", "NewRpcBytesView", "fn func(", "goruntime.KeepAlive(&msg)", "proto.Unmarshal", "PinString(", "PinBytes(", "PinSlice(", "defer rpcruntime.Release(ptr)", "cleanup := func()", "rpcruntime.NewRpcString(nil, 0, false)", "rpcruntime.NewRpcBytes(nil, 0, false)")
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
		"flags, err = rpcruntime.NewRpcBoolRepeatChecked(unsafe.SliceData(flagsRaw), int32(len(flagsRaw)), false)",
		"moodsRaw := make([]int32, len(msg.Moods))",
		"reqOwner = append(reqOwner, moodsRaw)",
		"moods, err = rpcruntime.NewRpcRepeatChecked[int32](unsafe.SliceData(moodsRaw), int32(len(moodsRaw)), false)",
		"unsignedScores, err = rpcruntime.NewRpcRepeatChecked[uint32](unsafe.SliceData(msg.UnsignedScores), int32(len(msg.UnsignedScores)), false)",
		"unsignedTotals, err = rpcruntime.NewRpcRepeatChecked[uint64](unsafe.SliceData(msg.UnsignedTotals), int32(len(msg.UnsignedTotals)), false)",
		"return nil, nil, nil, nil, nil, nil, nil, reqOwner, err",
		"return scores, flags, counts, ratios, moods, unsignedScores, unsignedTotals, reqOwner, nil",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "NativeRequestView", "NewRpcBoolRepeatView", "NewRpcRepeatView", "fn func(", "goruntime.KeepAlive(&msg)", "proto.Unmarshal", "goruntime.KeepAlive(flagsRaw)", "goruntime.KeepAlive(moodsRaw)")
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
		"func convertRepeatedServiceCheckNativeToMessageRequest(scores *rpcruntime.RpcRepeat[int32], flags *rpcruntime.RpcBoolRepeat, counts *rpcruntime.RpcRepeat[int64], ratios *rpcruntime.RpcRepeat[float64], moods *rpcruntime.RpcRepeat[int32], unsignedScores *rpcruntime.RpcRepeat[uint32], unsignedTotals *rpcruntime.RpcRepeat[uint64]) (*RepeatedRequest, error) {",
		"msg.Scores = scores.UnsafeSlice()",
		"msg.Flags = flags.SafeSlice()",
		"msg.Counts = counts.UnsafeSlice()",
		"msg.Ratios = ratios.UnsafeSlice()",
		"moodsRaw := moods.UnsafeSlice()",
		"msg.UnsignedScores = unsignedScores.UnsafeSlice()",
		"msg.UnsignedTotals = unsignedTotals.UnsafeSlice()",
		"goruntime.KeepAlive(scores)",
		"goruntime.KeepAlive(flags)",
		"goruntime.KeepAlive(counts)",
		"goruntime.KeepAlive(ratios)",
		"goruntime.KeepAlive(moods)",
		"goruntime.KeepAlive(unsignedScores)",
		"goruntime.KeepAlive(unsignedTotals)",
		"return msg, nil",
	} {
		assertGeneratedContentContains(t, plugin, codecFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, codecFile, "msg.Scores = scores.SafeSlice()", "moodsRaw := moods.SafeSlice()", "proto.Marshal")
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
		"test/v1/cgo/main.go",
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
