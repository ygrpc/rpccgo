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

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/complete_service_plan.all_service.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		"package main",
		`v1 "example.com/test/v1"`,
		`rpcruntime "rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"func CallAllServiceUnaryNativeUnary(ctx context.Context, NamePtr uintptr, NameLen int32, NameOwnership int32, Enabled int8, ChildPtr uintptr, ChildLen int32, ChildOwnership int32, outAccepted *int8, outPayloadPtr *uintptr, outPayloadLen *int32) int32 {",
		"if err := validateAllServiceUnaryNativeUnaryResponse(outAccepted, outPayloadPtr, outPayloadLen); err != nil {",
		"nameValue, enabledValue, childValue, err := decodeAllServiceUnaryNativeUnaryRequest(NamePtr, NameLen, NameOwnership, Enabled, ChildPtr, ChildLen, ChildOwnership)",
		"acceptedResult, payloadResult, err := v1.NewAllServiceCGONativeClientBridge().Unary(ctx, nameValue, enabledValue, childValue)",
		"return int32(rpcruntime.StoreError(err))",
		"if _, err := rpcruntime.LengthFromInt32(NameLen); err != nil {",
		"if NamePtr == 0 || NameLen == 0 {",
		"nameValue = rpcruntime.EmptyRpcString()",
		"var decodeErr error",
		"nameValue, decodeErr = rpcruntime.NewRpcStringChecked((*byte)(unsafe.Pointer(NamePtr)), NameLen, NameOwnership > 0)",
		"if decodeErr != nil {",
		`fmt.Errorf("test.v1.AllRequest.name: %w", decodeErr)`,
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
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeClientFile, "type AllServiceUnaryNativeUnaryInput struct", "type AllServiceUnaryNativeUnaryOutput struct", "PayloadOwnership *int32", "allServiceDispatcher", "loadAllService", "takeAllService", "connectrpc.com/connect", "google.golang.org/grpc", "google.golang.org/protobuf", "rpcruntime.NewRpcString((*byte)(unsafe.Pointer(NamePtr))")
}

func TestRenderNativeClientCGOStreamsUseDispatcherAccessor(t *testing.T) {
	file := messageCgoTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/message_cgo.greeter.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		"v1.NewGreeterUploadNativeStream(rpcruntime.StreamHandle(handle)).Send(ctx",
		"err = v1.NewGreeterUploadNativeStream(rpcruntime.StreamHandle(handle)).Finish(ctx)",
		"CloseSendGreeterChatNativeBidiStream(ctx context.Context, handle int32) int32",
		"err = v1.NewGreeterChatNativeStream(rpcruntime.StreamHandle(handle)).CloseSend(ctx)",
	} {
		assertGeneratedContentContains(t, plugin, nativeClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, nativeClientFile,
		"LoadUploadNativeStream",
		"TakeUploadNativeStream",
		"LoadListNativeStream",
		"TakeListNativeStream",
		"LoadChatNativeStream",
		"TakeChatNativeStream",
		"rpcruntime.DispatcherStreamSend[",
		"rpcruntime.DispatcherStreamReceive[",
		"rpcruntime.DispatcherStreamFinish[",
		"rpcruntime.DispatcherStreamDone[",
		"rpcruntime.DispatcherStreamCancel[",
		"rpcruntime.DispatcherStreamCloseSend[",
	)
}

func TestRenderNativeClientCGOHandlesBytesOwnershipAndPinnedOutputRelease(t *testing.T) {
	file := nativeClientBytesOwnershipFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/native_unary.greeter.client.cgo.rpccgo.go"
	for _, fragment := range []string{
		"PayloadOwnership int32",
		"if PayloadPtr == 0 || PayloadLen == 0 {",
		"payloadValue = rpcruntime.EmptyRpcBytes()",
		"var decodeErr error",
		"payloadValue, decodeErr = rpcruntime.NewRpcBytesChecked((*byte)(unsafe.Pointer(PayloadPtr)), PayloadLen, PayloadOwnership > 0)",
		"if decodeErr != nil {",
		`fmt.Errorf("test.v1.HelloRequest.payload: %w", decodeErr)`,
		"PayloadOwnership > 0",
		"payloadResult, noteResult, extraPayloadResult, err := v1.NewGreeterCGONativeClientBridge().SayHello(ctx, payloadValue)",
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

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/native_enum.enum_service.client.cgo.rpccgo.go"
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

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const nativeClientFile = "test/v1/cgo/native_repeated.repeated_service.client.cgo.rpccgo.go"
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
		"rpcruntime.LengthFromInt32(ScoresLen)",
		"rpcruntime.LengthFromInt32(FlagsLen)",
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
			name: "unrelated message collides with call func",
			method: MethodPlan{
				Name:      "Unary",
				GoName:    "Unary",
				FullName:  "test.v1.AllService.Unary",
				Streaming: StreamingKindUnary,
				Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
				Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
			},
			topLevelSymbols: []TopLevelSymbolPlan{
				{GoName: "CallAllServiceUnaryNativeUnary", FullName: "test.v1.CallAllServiceUnaryNativeUnary", Kind: TopLevelSymbolKindMessage},
			},
			wantError: "CallAllServiceUnaryNativeUnary",
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
			err := RenderNativeStageFiles(plugin, plan)
			if err == nil {
				t.Fatal("RenderNativeStageFiles() error = nil, want native client cgo symbol collision")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantError) || !strings.Contains(got, "collides") {
				t.Fatalf("RenderNativeStageFiles() error = %q, want collision for %q", got, tt.wantError)
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
		NativeFileFamily: NativeFileFamilyPlan{
			CGONativeClient: GeneratedFilePlan{Filename: "test/v1/collision_sibling.client.cgo.rpccgo.go", Enabled: true},
		},
	})

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want sibling native client cgo symbol collision")
	}
	if got := err.Error(); !strings.Contains(got, "CallAllServiceUnaryNativeUnary") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want sibling collision for CallAllServiceUnaryNativeUnary", got)
	}
}

func TestRenderNativeClientCGORejectsPackageLevelMultiFileSymbolCollisions(t *testing.T) {
	tests := []struct {
		name      string
		file      *descriptorpb.FileDescriptorProto
		otherFile *descriptorpb.FileDescriptorProto
		wantError string
	}{
		{
			name:      "other file enum collides with native call",
			file:      simpleTestFile(),
			otherFile: nativeClientPackageCollisionEnumFile("test/v1/other.proto", "example.com/test/v1;testv1", "CallGreeterSayHelloNativeUnary"),
			wantError: "CallGreeterSayHelloNativeUnary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := tt.file
			setSimpleServiceComment(t, file, "@rpccgo: native\n")
			request := newTestCodeGeneratorRequest("paths=source_relative", file, tt.otherFile)
			request.FileToGenerate = []string{file.GetName(), tt.otherFile.GetName()}
			plugin, err := ProtogenOptions().New(request)
			if err != nil {
				t.Fatalf("protogen.Options.New() error = %v", err)
			}

			_, err = GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
			if err == nil {
				t.Fatal("GenerateWithOptions() error = nil, want package-level symbol collision")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantError) || !strings.Contains(got, "collides") {
				t.Fatalf("GenerateWithOptions() error = %q, want collision for %q", got, tt.wantError)
			}
		})
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
	plan.Services[0].NativeFileFamily.CGONativeClient.Enabled = true
	plan.TopLevelSymbols = append(plan.TopLevelSymbols, TopLevelSymbolPlan{
		GoName:   "CallDecodeSayHelloNativeUnaryInputNativeUnary",
		FullName: "test.v1.Parent.Nested",
		Kind:     TopLevelSymbolKindMessage,
	})

	err := RenderNativeStageFiles(plugin, plan)
	if err == nil {
		t.Fatal("RenderNativeStageFiles() error = nil, want nested protobuf symbol collision")
	}
	if got := err.Error(); !strings.Contains(got, "CallDecodeSayHelloNativeUnaryInputNativeUnary") || !strings.Contains(got, "collides") {
		t.Fatalf("RenderNativeStageFiles() error = %q, want nested collision for CallDecodeSayHelloNativeUnaryInputNativeUnary", got)
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

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v, want different GoImportPath ignored", err)
	}
}

func TestRenderNativeClientCGOGeneratedSourceCompiles(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin, GenerateOptions{RenderNativeStageFiles: true})
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
	})
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "./...")
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+modulePath+"\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
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
			Name:     "AllService",
			GoName:   "AllService",
			FullName: "test.v1.AllService",
			Methods:  []MethodPlan{method},
			NativeFileFamily: NativeFileFamilyPlan{
				CGONativeClient: GeneratedFilePlan{Filename: "test/v1/collision.client.cgo.rpccgo.go", Enabled: true},
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
