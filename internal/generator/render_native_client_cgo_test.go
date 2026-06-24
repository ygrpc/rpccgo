package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRenderNativeClientCGODefinesUnaryExportSurface(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/complete_service_plan.all_service.client.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"package main",
		`v1 "example.com/test/v1"`,
		`rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"//export rpccgoNativeTestv1AllServiceUnary",
		"func rpccgoNativeTestv1AllServiceUnary(NamePtr C.uintptr_t, NameLen C.int32_t, NameOwnership C.int32_t, Enabled C.int8_t, ChildPtr C.uintptr_t, ChildLen C.int32_t, ChildOwnership C.int32_t, outAccepted *C.int8_t, outPayloadPtr *C.uintptr_t, outPayloadLen *C.int32_t, outPayloadOwnership *C.int32_t) C.int32_t {",
		"ctx := context.Background()",
		"if err := validateAllServiceUnaryNativeUnaryResponse((*int8)(unsafe.Pointer(outAccepted)), (*uintptr)(unsafe.Pointer(outPayloadPtr)), (*int32)(unsafe.Pointer(outPayloadLen))); err != nil {",
		"nameValue, enabledValue, childValue, err := decodeAllServiceUnaryNativeUnaryRequest(uintptr(NamePtr), int32(NameLen), int32(NameOwnership), int8(Enabled), uintptr(ChildPtr), int32(ChildLen), int32(ChildOwnership))",
		"acceptedResult, payloadResult, err := v1.InvokeAllServiceNativeUnary(ctx, nameValue, enabledValue, childValue)",
		"return C.int32_t(rpcruntime.StoreError(err))",
		"var decoded rpcruntime.NativeReleaseStack",
		"if _, err := rpcruntime.LengthFromInt32(NameLen); err != nil {",
		"if NamePtr == 0 || NameLen == 0 {",
		"nameValue = rpcruntime.EmptyRpcString()",
		"var decodeErr error",
		"nameValue, decodeErr = rpcruntime.NewRpcStringChecked((*byte)(unsafe.Pointer(NamePtr)), NameLen, NameOwnership > 0)",
		"if decodeErr != nil {",
		`fmt.Errorf("test.v1.AllRequest.name: %w", decodeErr)`,
		"return nil, false, nil, errors.Join(fmt.Errorf(\"test.v1.AllRequest.name: %w\", decodeErr), decoded.Release())",
		"decoded = append(decoded, nameValue)",
		"NameOwnership > 0",
		"var acceptedResultValue int8",
		"acceptedResultValue = 1",
		"payloadLenValue, err := rpcruntime.LengthToInt32(len(payloadResult))",
		"payloadPtrValue, err := rpcruntime.PinBytes(payloadResult)",
		"*outAccepted = acceptedResultValue",
		"*outPayloadPtr = payloadPtrValue",
		"*outPayloadLen = payloadLenValue",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeClientFile, "func CallAllServiceUnaryNativeUnary(", "type allServiceNativeClientDecodedResource interface", "type allServiceNativeClientDecodedResources struct", "decoded.Add(", "cleanupDecoded := func() error", "cleanupDecoded()", "type AllServiceUnaryNativeUnaryInput struct", "type AllServiceUnaryNativeUnaryOutput struct", "PayloadOwnership *int32", "allServiceDispatcher", "loadAllService", "takeAllService", "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf", "rpcruntime.NewRpcString((*byte)(unsafe.Pointer(NamePtr))")
}

func TestRenderNativeClientCGOStreamsUseRuntimeStreamOperations(t *testing.T) {
	file := messageCgoTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/message_cgo.greeter.client.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"//export rpccgoNativeTestv1GreeterUploadStart",
		"//export rpccgoNativeTestv1GreeterUploadSend",
		"//export rpccgoNativeTestv1GreeterListRecv",
		"//export rpccgoNativeTestv1GreeterChatCloseSend",
		"typedef void (*GreeterListCGONativeOnRecvCallback)",
		"typedef void (*RpccgoNativeOnDoneCallback)(int32_t stream, int32_t err_id);",
		"static inline void callGreeterListCGONativeOnRecvCallback",
		"static inline void callRpccgoNativeOnDoneCallback",
		"func rpccgoNativeTestv1GreeterListStart(",
		"onRecv C.GreeterListCGONativeOnRecvCallback, onDone C.RpccgoNativeOnDoneCallback",
		"func rpccgoNativeTestv1GreeterChatStart(stream *C.int32_t, onRecv C.GreeterChatCGONativeOnRecvCallback, onDone C.RpccgoNativeOnDoneCallback) C.int32_t {",
		"callbackState, err := rpcruntime.EnableStreamCallbackReceive(rpcruntime.StreamHandle(handle))",
		"if rpcruntime.StreamCallbackReceiveEnabled(rpcruntime.StreamHandle(handle)) {",
		`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: stream receive is owned by callback receive mode")))`,
		"err = v1.GreeterNativeUploadSend(ctx, rpcruntime.StreamHandle(handle)",
		"v1.GreeterNativeUploadFinish(ctx, rpcruntime.StreamHandle(handle))",
		"err = v1.GreeterNativeChatCloseSend(ctx, rpcruntime.StreamHandle(handle))",
		"err = v1.GreeterNativeChatFinish(ctx, rpcruntime.StreamHandle(handle))",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeClientFile,
		"func StartGreeterUploadNativeClientStream(",
		"func SendGreeterUploadNativeClientStream(",
		"GreeterNativeListFinish(ctx",
		"func FinishGreeterListNativeServerStream(",
		"func CloseSendGreeterChatNativeBidiStream(",
		"func FinishGreeterChatNativeBidiStream(",
		"NewGreeterUploadNative"+"Stream",
		"NewGreeterListNative"+"Stream",
		"NewGreeterChatNative"+"Stream",
		"LoadUploadNativeStream",
		"TakeUploadNativeStream",
		"LoadListNativeStream",
		"TakeListNativeStream",
		"LoadChatNativeStream",
		"TakeChatNativeStream",
		"rpcruntime.DispatcherStreamSend[",
		"rpcruntime.DispatcherStreamReceive[",
		"rpcruntime.DispatcherStreamFinish[",
		"rpcruntime.DispatcherStreamCancel[",
		"rpcruntime.DispatcherStreamCloseSend[",
		"DoneGreeterListNativeServerStream",
		"DoneGreeterChatNativeBidiStream",
		"typedef int32_t (*GreeterListCGONativeOnRecvCallback)",
		"typedef int32_t (*RpccgoNativeOnDoneCallback)",
		"errID := int32(C.callGreeterListCGONativeOnRecvCallback",
		"_ = C.callRpccgoNativeOnDoneCallback",
	)
}

func TestRenderNativeClientCGOHandlesBytesOwnershipAndPinnedOutputRelease(t *testing.T) {
	file := nativeClientBytesOwnershipFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/native_unary.greeter.client.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"PayloadOwnership int32",
		"if PayloadPtr == 0 || PayloadLen == 0 {",
		"payloadValue = rpcruntime.EmptyRpcBytes()",
		"var decodeErr error",
		"payloadValue, decodeErr = rpcruntime.NewRpcBytesChecked((*byte)(unsafe.Pointer(PayloadPtr)), PayloadLen, PayloadOwnership > 0)",
		"if decodeErr != nil {",
		`fmt.Errorf("test.v1.HelloRequest.payload: %w", decodeErr)`,
		"PayloadOwnership > 0",
		"payloadResult, noteResult, extraPayloadResult, err := v1.InvokeGreeterNativeSayHello(ctx, payloadValue)",
		"payloadPtrValue, err := rpcruntime.PinBytes(payloadResult)",
		"data, notePtrValue, err := rpcruntime.PinString(noteResult)",
		"rpcruntime.Release(payloadPtrValue)",
		"rpcruntime.Release(notePtrValue)",
		"*outPayloadPtr = payloadPtrValue",
		"*outNotePtr = notePtrValue",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeClientFile, "PayloadOwnership *int32", "rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(PayloadPtr))")
}

func TestRenderNativeClientCGOSupportsEnumAsInt32(t *testing.T) {
	file := nativeClientEnumFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/native_enum.enum_service.client.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"State int32",
		"stateValue := v1.State(State)",
		"stateResultValue := int32(stateResult)",
		"*outState = stateResultValue",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
}

func TestRenderNativeClientCGOSupportsRepeatedNativeABI(t *testing.T) {
	file := nativeClientRepeatedFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/native_repeated.repeated_service.client.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"ScoresPtr",
		"ScoresLen",
		"ScoresOwnership",
		"FlagsPtr",
		"FlagsLen",
		"FlagsOwnership",
		"CountsPtr",
		"CountsLen",
		"CountsOwnership",
		"RatiosPtr",
		"RatiosLen",
		"RatiosOwnership",
		"MoodsPtr",
		"MoodsLen",
		"MoodsOwnership",
		"UnsignedScoresPtr",
		"UnsignedScoresLen",
		"UnsignedScoresOwnership",
		"UnsignedTotalsPtr",
		"UnsignedTotalsLen",
		"UnsignedTotalsOwnership",
		"if ScoresPtr == 0 || ScoresLen == 0 {",
		"scoresValue = rpcruntime.EmptyRpcRepeat[int32]()",
		"ScoresOwnership > 0",
		"scoresValue, decodeErr = rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(ScoresPtr)), ScoresLen, ScoresOwnership > 0)",
		"if FlagsPtr == 0 || FlagsLen == 0 {",
		"flagsValue = rpcruntime.EmptyRpcBoolRepeat()",
		"FlagsOwnership > 0",
		"flagsValue, decodeErr = rpcruntime.NewRpcBoolRepeatChecked((*byte)(unsafe.Pointer(FlagsPtr)), FlagsLen, FlagsOwnership > 0)",
		"if CountsPtr == 0 || CountsLen == 0 {",
		"countsValue = rpcruntime.EmptyRpcRepeat[int64]()",
		"countsValue, decodeErr = rpcruntime.NewRpcRepeatChecked((*int64)(unsafe.Pointer(CountsPtr)), CountsLen, CountsOwnership > 0)",
		"if RatiosPtr == 0 || RatiosLen == 0 {",
		"ratiosValue = rpcruntime.EmptyRpcRepeat[float64]()",
		"ratiosValue, decodeErr = rpcruntime.NewRpcRepeatChecked((*float64)(unsafe.Pointer(RatiosPtr)), RatiosLen, RatiosOwnership > 0)",
		"if MoodsPtr == 0 || MoodsLen == 0 {",
		"moodsValue = rpcruntime.EmptyRpcRepeat[int32]()",
		"moodsValue, decodeErr = rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(MoodsPtr)), MoodsLen, MoodsOwnership > 0)",
		"if UnsignedScoresPtr == 0 || UnsignedScoresLen == 0 {",
		"unsignedScoresValue = rpcruntime.EmptyRpcRepeat[uint32]()",
		"unsignedScoresValue, decodeErr = rpcruntime.NewRpcRepeatChecked((*uint32)(unsafe.Pointer(UnsignedScoresPtr)), UnsignedScoresLen, UnsignedScoresOwnership > 0)",
		"if UnsignedTotalsPtr == 0 || UnsignedTotalsLen == 0 {",
		"unsignedTotalsValue = rpcruntime.EmptyRpcRepeat[uint64]()",
		"unsignedTotalsValue, decodeErr = rpcruntime.NewRpcRepeatChecked((*uint64)(unsafe.Pointer(UnsignedTotalsPtr)), UnsignedTotalsLen, UnsignedTotalsOwnership > 0)",
		"rpcruntime.LengthFromInt32(ScoresLen)",
		"rpcruntime.LengthFromInt32(FlagsLen)",
		"rpcruntime.LengthFromInt32(UnsignedScoresLen)",
		"rpcruntime.LengthFromInt32(UnsignedTotalsLen)",
		"rpcruntime.Release(scoresPtrValue)",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeClientFile, "return nil, repeatedServiceNativeClientUnsupportedField")
}

func TestRenderNativeClientCGORejectsGeneratedHelperCollisions(t *testing.T) {
	tests := []struct {
		name            string
		method          MethodPlan
		topLevelSymbols []TopLevelSymbolPlan
		wantError       string
	}{
		{
			name: "decoder collides with request message",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "decodeAllServiceUnaryNativeUnaryRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.decodeAllServiceUnaryNativeUnaryRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
			},
			wantError: "decodeAllServiceUnaryNativeUnaryRequest",
		},
		{
			name: "encoder collides with response message",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "encodeAllServiceUnaryNativeUnaryResponse", GoImportPath: "example.com/test/v1", FullName: "test.v1.encodeAllServiceUnaryNativeUnaryResponse"},
			},
			wantError: "encodeAllServiceUnaryNativeUnaryResponse",
		},
		{
			name: "string suffix field collision",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
				Contract: MethodContractPlan{Native: NativeContractPlan{RequestFields: []FieldPlan{
					{GoName: "Name", FullName: "test.v1.AllRequest.name", Kind: FieldKindString, Native: NativeFieldPlan{Kind: NativeFieldKindString, Shape: NativeABIShapeScalar}},
					{GoName: "NamePtr", FullName: "test.v1.AllRequest.name_ptr", Kind: FieldKindSignedInt32, Native: NativeFieldPlan{Kind: NativeFieldKindSignedNumeric, Shape: NativeABIShapeScalar}},
				}}},
			},
			wantError: "NamePtr",
		},
		{
			name: "bytes response suffix field collision",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
				Contract: MethodContractPlan{Native: NativeContractPlan{ResponseFields: []FieldPlan{
					{GoName: "Payload", FullName: "test.v1.AllReply.payload", Kind: FieldKindBytes, Native: NativeFieldPlan{Kind: NativeFieldKindBytes, Shape: NativeABIShapeScalar}},
					{GoName: "PayloadLen", FullName: "test.v1.AllReply.payload_len", Kind: FieldKindSignedInt32, Native: NativeFieldPlan{Kind: NativeFieldKindSignedNumeric, Shape: NativeABIShapeScalar}},
				}}},
			},
			wantError: "PayloadLen",
		},
		{
			name: "unrelated enum collides with decoder",
			method: MethodPlan{
				Name:      "Decode",
				GoName:    "Decode",
				FullName:  "test.v1.AllService.Decode",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
			},
			topLevelSymbols: []TopLevelSymbolPlan{
				{GoName: "decodeAllServiceDecodeNativeUnaryRequest", FullName: "test.v1.decodeAllServiceDecodeNativeUnaryRequest", Kind: TopLevelSymbolKindEnum},
			},
			wantError: "decodeAllServiceDecodeNativeUnaryRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
			plan := nativeClientCollisionTestFilePlan(tt.method)
			plan.TopLevelSymbols = tt.topLevelSymbols
			err := renderNativeClientCGOFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
			if err == nil {
				t.Fatal("RenderGeneratedFiles() error = nil, want native client cgo symbol collision")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantError) || !strings.Contains(got, "collides") {
				t.Fatalf("RenderGeneratedFiles() error = %q, want collision for %q", got, tt.wantError)
			}
		})
	}
}

func TestRenderNativeClientCGORejectsSiblingServiceGeneratedSymbolCollisions(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := nativeClientCollisionTestFilePlan(MethodPlan{
		Name:      "Unary",
		GoName:    "Unary",
		FullName:  "test.v1.AllService.Unary",
		Streaming: StreamingKindUnary,
		Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
		Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
	})
	plan.Services = append(plan.Services, ServicePlan{
		Name:     "All",
		GoName:   "All",
		FullName: "test.v1.All",
		Methods: []MethodPlan{{
			Name:      "ServiceUnary",
			GoName:    "ServiceUnary",
			FullName:  "test.v1.All.ServiceUnary",
			Streaming: StreamingKindUnary,
			Request:   MethodIOPlan{GoName: "OtherRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.OtherRequest"},
			Response:  MethodIOPlan{GoName: "OtherReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.OtherReply"},
		}},
		Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
		Artifacts: []GeneratedArtifactPlan{
			{Kind: GeneratedArtifactKindCGONativeClient, Filename: "test/v1/collision_sibling.client.native.cgo.rpccgo.go"},
		},
	})

	err := renderNativeClientCGOFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
	if err == nil {
		t.Fatal("RenderGeneratedFiles() error = nil, want sibling native client cgo symbol collision")
	}
	if got := err.Error(); !strings.Contains(got, "decodeAllServiceUnaryNativeUnaryRequest") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderGeneratedFiles() error = %q, want sibling collision for decodeAllServiceUnaryNativeUnaryRequest", got)
	}
}

func TestRenderNativeClientCGORejectsNestedPackageLevelSymbolCollisions(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile(), nativeClientPackageCollisionNestedFile("test/v1/other.proto", "example.com/test/v1;testv1", "SayHelloNativeUnaryInput"))
	plan := nativeClientCollisionTestFilePlan(MethodPlan{
		Name:      "SayHello",
		GoName:    "SayHelloNativeUnaryInput",
		FullName:  "test.v1.Greeter.SayHello",
		Streaming: StreamingKindUnary,
		Request:   MethodIOPlan{GoName: "HelloRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloRequest"},
		Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
	})
	plan.Services[0].Name = "Greeter"
	plan.Services[0].GoName = "Decode"
	plan.Services[0].FullName = "test.v1.Greeter"
	plan.TopLevelSymbols = append(plan.TopLevelSymbols, TopLevelSymbolPlan{
		GoName:   "decodeDecodeSayHelloNativeUnaryInputNativeUnaryRequest",
		FullName: "test.v1.Parent.Nested",
		Kind:     TopLevelSymbolKindMessage,
	})

	err := renderNativeClientCGOFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
	if err == nil {
		t.Fatal("RenderGeneratedFiles() error = nil, want nested protobuf symbol collision")
	}
	if got := err.Error(); !strings.Contains(got, "decodeDecodeSayHelloNativeUnaryInputNativeUnaryRequest") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderGeneratedFiles() error = %q, want nested collision for decodeDecodeSayHelloNativeUnaryInputNativeUnaryRequest", got)
	}
}

func TestRenderNativeClientCGOIgnoresMultiFileSymbolsFromDifferentGoImportPath(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	otherFile := nativeClientPackageCollisionFile("other/v1/other.proto", "example.com/other/v1;otherv1", "GreeterSayHelloNativeUnaryInput")
	request := newTestCodeGeneratorRequest("paths=source_relative", file, otherFile)
	request.FileToGenerate = []string{file.GetName(), otherFile.GetName()}
	plugin, err := ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v, want different GoImportPath ignored", err)
	}
}

func TestRenderNativeClientCGOGeneratedSourceCompiles(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.message.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go")
	})
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated native client go test failed: %v\n%s", err, out)
	}
}

func writeNativeGeneratedModule(t *testing.T, root string, plugin *protogen.Plugin, include func(string) bool) {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	modulePath := nativeGeneratedModulePath(t, plugin)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+modulePath+"\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\tgithub.com/ygrpc/rpccgo v0.0.0\n\tgoogle.golang.org/grpc v1.79.3\n\tgoogle.golang.org/protobuf v1.36.11\n)\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), goSum, 0o644); err != nil {
		t.Fatalf("write go.sum: %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !include(name) {
			continue
		}
		target := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir generated dir: %v", err)
		}
		if err := os.WriteFile(target, []byte(generated.GetContent()), 0o644); err != nil {
			t.Fatalf("write generated file %s: %v", name, err)
		}
	}
}

func nativeGeneratedModulePath(t *testing.T, plugin *protogen.Plugin) string {
	t.Helper()
	for _, file := range plugin.Files {
		if file == nil || !file.Generate {
			continue
		}
		generatedDir := filepath.ToSlash(filepath.Dir(file.GeneratedFilenamePrefix))
		importPath := string(file.GoImportPath)
		if generatedDir == "." {
			return importPath
		}
		suffix := "/" + generatedDir
		if strings.HasSuffix(importPath, suffix) {
			return strings.TrimSuffix(importPath, suffix)
		}
		return importPath
	}
	t.Fatal("no generated protogen file found")
	return ""
}

func nativeClientBytesOwnershipFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_unary.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("HelloRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("payload", 1, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
			{
				Name: proto.String("HelloReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("payload", 1, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("note", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("extra_payload", 3, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("SayHello", ".test.v1.HelloRequest", ".test.v1.HelloReply", false, false),
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

func nativeClientEnumFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_enum.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			stateEnumDescriptor(),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("EnumRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("state", 1, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.State"),
				},
			},
			{
				Name: proto.String("EnumReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("state", 1, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.State"),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("EnumService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Check", ".test.v1.EnumRequest", ".test.v1.EnumReply", false, false),
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

func nativeClientRepeatedFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_repeated.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Mood"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("MOOD_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("MOOD_OK"), Number: proto.Int32(1)},
					{Name: proto.String("MOOD_BUSY"), Number: proto.Int32(2)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("RepeatedRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("scores", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("flags", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("counts", 3, descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("ratios", 4, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("moods", 5, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ".test.v1.Mood"),
					fieldDescriptor("unsigned_scores", 6, descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("unsigned_totals", 7, descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
				},
			},
			{
				Name: proto.String("RepeatedReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("scores", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("flags", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("counts", 3, descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("ratios", 4, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("moods", 5, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ".test.v1.Mood"),
					fieldDescriptor("unsigned_scores", 6, descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("unsigned_totals", 7, descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("RepeatedService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					methodDescriptor("Check", ".test.v1.RepeatedRequest", ".test.v1.RepeatedReply", false, false),
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

func nativeClientCollisionTestFilePlan(method MethodPlan) FilePlan {
	return FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:       "AllService",
			GoName:     "AllService",
			FullName:   "test.v1.AllService",
			Methods:    []MethodPlan{method},
			Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
			Artifacts: []GeneratedArtifactPlan{
				{Kind: GeneratedArtifactKindCGONativeClient, Filename: "test/v1/collision.client.native.cgo.rpccgo.go"},
			},
		}},
	}
}

func nativeClientPackageCollisionFile(name, goPackage, messageName string) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String(name),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String(goPackage),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String(messageName)},
		},
	}
}

func nativeClientPackageCollisionEnumFile(name, goPackage, enumName string) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String(name),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String(goPackage),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String(enumName),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String(enumName + "_UNSPECIFIED"), Number: proto.Int32(0)},
				},
			},
		},
	}
}

func nativeClientPackageCollisionNestedFile(name, goPackage, nestedName string) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String(name),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String(goPackage),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Decode"),
				NestedType: []*descriptorpb.DescriptorProto{
					{Name: proto.String(nestedName)},
				},
			},
		},
	}
}
