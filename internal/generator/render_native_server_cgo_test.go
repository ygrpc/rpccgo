package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRenderNativeServerCGODefinesFlatServiceCallbackRegistration(t *testing.T) {
	file := completeServicePlanTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/cgo/complete_service_plan.all_service.server.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"package main",
		`import "C"`,
		`v1 "example.com/test/v1"`,
		"typedef int32_t (*AllServiceUnaryCGONativeUnaryCallback)(uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, int8_t Enabled, uintptr_t ChildPtr, int32_t ChildLen, int32_t ChildOwnership, int8_t *outAccepted, uintptr_t *outPayloadPtr, int32_t *outPayloadLen, int32_t *outPayloadOwnership);",
		"static inline int32_t callAllServiceUnaryCGONativeUnaryCallback",
		"return callback(NamePtr, NameLen, NameOwnership, Enabled, ChildPtr, ChildLen, ChildOwnership, outAccepted, outPayloadPtr, outPayloadLen, outPayloadOwnership);",
		`rpcruntime "rpccgo/rpcruntime"`,
		`sync "sync"`,
		`unsafe "unsafe"`,
		"allServiceCGONativeServerAdapterMu            sync.Mutex",
		"allServiceCGONativeServerAdapter              = &allServiceCGONativeAdapter{}",
		"type allServiceCGONativeAdapter struct {",
		"UnaryCallback       C.AllServiceUnaryCGONativeUnaryCallback",
		"ClientStreamSend    C.AllServiceClientStreamCGONativeClientStreamSendCallback",
		"ServerStreamRecv    C.AllServiceServerStreamCGONativeServerStreamRecvCallback",
		"BidiStreamCloseSend C.AllServiceBidiStreamCGONativeBidiStreamCloseSendCallback",
		"//export rpccgo_native_testv1_AllService_register",
		"func rpccgo_native_testv1_AllService_register(unaryCallback C.AllServiceUnaryCGONativeUnaryCallback, clientStreamStart C.AllServiceClientStreamCGONativeClientStreamStartCallback, clientStreamSend C.AllServiceClientStreamCGONativeClientStreamSendCallback, clientStreamFinish C.AllServiceClientStreamCGONativeClientStreamFinishCallback, clientStreamCancel C.AllServiceClientStreamCGONativeClientStreamCancelCallback, serverStreamStart C.AllServiceServerStreamCGONativeServerStreamStartCallback, serverStreamRecv C.AllServiceServerStreamCGONativeServerStreamRecvCallback, serverStreamFinish C.AllServiceServerStreamCGONativeServerStreamFinishCallback, serverStreamCancel C.AllServiceServerStreamCGONativeServerStreamCancelCallback, bidiStreamStart C.AllServiceBidiStreamCGONativeBidiStreamStartCallback, bidiStreamSend C.AllServiceBidiStreamCGONativeBidiStreamSendCallback, bidiStreamRecv C.AllServiceBidiStreamCGONativeBidiStreamRecvCallback, bidiStreamCloseSend C.AllServiceBidiStreamCGONativeBidiStreamCloseSendCallback, bidiStreamFinish C.AllServiceBidiStreamCGONativeBidiStreamFinishCallback, bidiStreamCancel C.AllServiceBidiStreamCGONativeBidiStreamCancelCallback) C.int32_t {",
		"next := &allServiceCGONativeAdapter{}",
		"next.UnaryCallback = unaryCallback",
		"next.ClientStreamStart = clientStreamStart",
		"next.ClientStreamSend = clientStreamSend",
		"next.ClientStreamFinish = clientStreamFinish",
		"next.ClientStreamCancel = clientStreamCancel",
		"if err := v1.RegisterAllServiceCGONativeServer(next); err != nil {",
		"allServiceCGONativeServerAdapter = next",
		"return &allServiceClientStreamCGONativeClientStreamSession{send: a.ClientStreamSend, finish: a.ClientStreamFinish, cancel: a.ClientStreamCancel, stream: stream}, nil",
		"type allServiceClientStreamCGONativeClientStreamSession struct {",
		"send   C.AllServiceClientStreamCGONativeClientStreamSendCallback",
		"finish C.AllServiceClientStreamCGONativeClientStreamFinishCallback",
		"cancel C.AllServiceClientStreamCGONativeClientStreamCancelCallback",
		"errID := int32(C.callAllServiceClientStreamCGONativeClientStreamSendCallback(s.send, s.stream",
		"return &allServiceServerStreamCGONativeServerStreamSession{recv: a.ServerStreamRecv, finish: a.ServerStreamFinish, cancel: a.ServerStreamCancel, stream: stream}, nil",
		"recv   C.AllServiceServerStreamCGONativeServerStreamRecvCallback",
		"finish C.AllServiceServerStreamCGONativeServerStreamFinishCallback",
		"errID := int32(C.callAllServiceServerStreamCGONativeServerStreamRecvCallback(s.recv, s.stream",
		"return &allServiceBidiStreamCGONativeBidiStreamSession{send: a.BidiStreamSend, recv: a.BidiStreamRecv, closeSend: a.BidiStreamCloseSend, finish: a.BidiStreamFinish, cancel: a.BidiStreamCancel, stream: stream}, nil",
		"closeSend C.AllServiceBidiStreamCGONativeBidiStreamCloseSendCallback",
		"errID := int32(C.callAllServiceBidiStreamCGONativeBidiStreamCloseSendCallback(s.closeSend, s.stream))",
		`errors.New("rpccgo: AllService cgo native server callbacks are nil")`,
		`errors.New("rpccgo: AllService cgo native server unary callback is missing")`,
		`errors.New("rpccgo: cgo native server streaming is not implemented")`,
		"callback := a.UnaryCallback",
		"errID := int32(C.callAllServiceUnaryCGONativeUnaryCallback(callback, namePtr, nameLen, nameOwnership, enabledValue, childPtr, childLen, childOwnership, &outAcceptedValue, &outPayloadPtr, &outPayloadLen, &outPayloadOwnership))",
		"callbackErr := allServiceCGONativeServerErrorFromID(errID)",
		"return false, nil, errors.Join(callbackErr, cleanupErr)",
		"_, namePtrValue, err := rpcruntime.PinString(name.SafeString())",
		"pinned = append(pinned, namePtrValue)",
		"rpcruntime.Release(pinned[i])",
		"payloadWrapper := rpcruntime.NewRpcBytes((*byte)(unsafe.Pointer(uintptr(payloadPtr))), int32(payloadLen), false)",
		"payloadResult := payloadWrapper.SafeBytes()",
		"func cleanupAllServiceUnaryCGONativeUnaryResponse(acceptedValue C.int8_t, payloadPtr C.uintptr_t, payloadLen C.int32_t, payloadOwnership C.int32_t) error {",
		`if err := rpcruntime.ReleaseC(unsafe.Pointer(uintptr(payloadPtr)), true, "test.v1.AllReply.payload"); err != nil {`,
		"cleanupErr = errors.Join(cleanupErr, err)",
		"func allServiceCGONativeServerErrorFromID(errID int32) error {",
		"rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))",
		"//export StoreAllServiceCGONativeServerErrorTextForExport",
		"func StoreAllServiceCGONativeServerErrorTextForExport(text *C.char, textLen C.int32_t) C.int32_t {",
		"return C.int32_t(rpcruntime.StoreError(errors.New(string(data))))",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, cgoServerFile,
		"typedef struct AllServiceCGONativeServerCallbacks",
		"callbacks C.AllServiceCGONativeServerCallbacks",
		"func RegisterAllServiceCGONativeServer(callbacks *C.AllServiceCGONativeServerCallbacks)",
		"callbacksCopy := *callbacks",
		"callbacks: callbacksCopy",
		"registerAllServiceActiveServer",
		"connectrpc.com/connect",
		"google.golang.org/grpc",
		"google.golang.org/protobuf",
		"type AllServiceUnaryCGONativeUnaryRequest struct",
		"type AllServiceUnaryCGONativeUnaryResponse struct",
		"*AllServiceUnaryCGONativeUnaryRequest",
		"*AllServiceUnaryCGONativeUnaryResponse",
		"* input",
		"* output",
		"GoCGONativeServerCallbacks",
		"GoCGONativeServerForTesting",
		"GoCGONativeAdapter",
		"allServiceCGONativeServerAdapter.UnaryCallback = callback",
		"allServiceCGONativeServerAdapter.ClientStreamStart = start",
		"rpccgo_native_testv1_AllService_Unary_register",
	)
	for _, generated := range plugin.Response().GetFile() {
		if generated.GetName() != cgoServerFile {
			continue
		}
		content := generated.GetContent()
		register := "if err := v1.RegisterAllServiceCGONativeServer(next); err != nil {"
		commit := "allServiceCGONativeServerAdapter = next"
		if registerIndex, commitIndex := strings.Index(content, register), strings.Index(content, commit); registerIndex < 0 || commitIndex < 0 || commitIndex < registerIndex {
			t.Fatalf("generated registration side-effect order invalid: register index=%d commit index=%d", registerIndex, commitIndex)
		}
		return
	}
	t.Fatalf("generated file %q not found", cgoServerFile)
}

func TestRenderNativeServerCGOScalarOnlyGeneratedSourceCompiles(t *testing.T) {
	file := nativeServerScalarOnlyFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/cgo/native_scalar.scalar.server.native.cgo.rpccgo.go"
	assertGeneratedContentContains(t, plugin, cgoServerFile, `unsafe "unsafe"`)
	assertGeneratedContentContains(t, plugin, cgoServerFile, "func StoreScalarCGONativeServerErrorTextForExport(text *C.char, textLen C.int32_t) C.int32_t {")

	tmp := t.TempDir()
	writeNativeGeneratedModule(t, tmp, plugin, func(name string) bool {
		return strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.message.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".server.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go")
	})
	writeNativeServerCGOTestFile(t, filepath.Join(tmp, "test/v1/native_scalar_stubs.go"), `package testv1

import (
	context "context"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type ScalarRequest struct {
	Enabled bool
	Count int32
}

type ScalarReply struct {
	Accepted bool
	Count int32
}

type ScalarHandler interface {
	Unary(context.Context, *ScalarRequest) (*ScalarReply, error)
}
type ScalarClient interface {
	Unary(context.Context, *ScalarRequest) (*ScalarReply, error)
}

func (*ScalarRequest) ProtoReflect() protoreflect.Message { return nil }
func (*ScalarReply) ProtoReflect() protoreflect.Message { return nil }
`)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scalar-only generated cgo native server go test failed: %v\n%s", err, out)
	}
}

func TestRenderNativeServerCGOSupportsRepeatedNativeABI(t *testing.T) {
	file := nativeServerRepeatedFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	_, err := GenerateWithOptions(plugin)
	if err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/cgo/native_repeated.repeated_service.server.native.cgo.rpccgo.go"
	for _, fragment := range []string{
		"uintptr_t ScoresPtr, int32_t ScoresLen, int32_t ScoresOwnership",
		"uintptr_t *outScoresPtr, int32_t *outScoresLen, int32_t *outScoresOwnership",
		"var outScoresPtr C.uintptr_t",
		"var outScoresLen C.int32_t",
		"var outScoresOwnership C.int32_t",
		"var outFlagsPtr C.uintptr_t",
		"var outFlagsLen C.int32_t",
		"var outFlagsOwnership C.int32_t",
		"var outCountsPtr C.uintptr_t",
		"var outCountsLen C.int32_t",
		"var outCountsOwnership C.int32_t",
		"var outRatiosPtr C.uintptr_t",
		"var outRatiosLen C.int32_t",
		"var outRatiosOwnership C.int32_t",
		"var outMoodsPtr C.uintptr_t",
		"var outMoodsLen C.int32_t",
		"var outMoodsOwnership C.int32_t",
		"rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(uintptr(scoresPtr))), int32(scoresLen), false)",
		"rpcruntime.NewRpcRepeatChecked((*int64)(unsafe.Pointer(uintptr(countsPtr))), int32(countsLen), false)",
		"rpcruntime.NewRpcRepeatChecked((*float64)(unsafe.Pointer(uintptr(ratiosPtr))), int32(ratiosLen), false)",
		"rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(uintptr(moodsPtr))), int32(moodsLen), false)",
		"rpcruntime.NewRpcBoolRepeatChecked((*byte)(unsafe.Pointer(uintptr(flagsPtr))), int32(flagsLen), false)",
		"scoresOwnership > 0",
		"rpcruntime.ReleaseC(unsafe.Pointer(uintptr(scoresPtr)), true, \"test.v1.RepeatedReply.scores\")",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}
}

func TestRenderNativeServerCGORejectsGeneratedSymbolCollisions(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
	plan := nativeServerCGOCollisionTestFilePlan("Greeter", []MethodPlan{{
		Name:      "SayHello",
		GoName:    "SayHello",
		FullName:  "test.v1.Greeter.SayHello",
		Streaming: StreamingKindUnary,
		Request:   MethodIOPlan{GoName: "GreeterSayHelloCGONativeUnaryRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.GreeterSayHelloCGONativeUnaryRequest"},
		Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
	}})

	err := renderNativeServerCGOFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
	if err == nil {
		t.Fatal("renderNativeServerCGOFile() error = nil, want cgo native server symbol collision")
	}
	if got := err.Error(); !strings.Contains(got, "GreeterSayHelloCGONativeUnaryRequest") || !strings.Contains(got, "collides") {
		t.Fatalf("renderNativeServerCGOFile() error = %q, want collision for cgo request type", got)
	}
}

func TestRenderNativeServerCGORejectsPackageAndSiblingSymbolCollisions(t *testing.T) {
	t.Run("package enum collides with cgo adapter", func(t *testing.T) {
		plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
		plan := nativeServerCGOCollisionTestFilePlan("Greeter", []MethodPlan{{
			Name:      "SayHello",
			GoName:    "SayHello",
			FullName:  "test.v1.Greeter.SayHello",
			Streaming: StreamingKindUnary,
			Request:   MethodIOPlan{GoName: "HelloRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloRequest"},
			Response:  MethodIOPlan{GoName: "HelloReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.HelloReply"},
		}})
		plan.TopLevelSymbols = []TopLevelSymbolPlan{{
			GoName:   "greeterCGONativeAdapter",
			FullName: "test.v1.greeterCGONativeAdapter",
			Kind:     TopLevelSymbolKindEnum,
		}}

		err := renderNativeServerCGOFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
		if err == nil {
			t.Fatal("renderNativeServerCGOFile() error = nil, want package symbol collision")
		}
		if got := err.Error(); !strings.Contains(got, "greeterCGONativeAdapter") || !strings.Contains(got, "collides") {
			t.Fatalf("renderNativeServerCGOFile() error = %q, want package collision", got)
		}
	})

	t.Run("sibling service collides with generated helper", func(t *testing.T) {
		plugin := newTestPlugin(t, "paths=source_relative", simpleTestFile())
		plan := nativeServerCGOCollisionTestFilePlan("AllService", []MethodPlan{{
			Name:      "Unary",
			GoName:    "Unary",
			FullName:  "test.v1.AllService.Unary",
			Streaming: StreamingKindUnary,
			Request:   MethodIOPlan{GoName: "AllRequest", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllRequest"},
			Response:  MethodIOPlan{GoName: "AllReply", GoImportPath: "example.com/test/v1", FullName: "test.v1.AllReply"},
		}})
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
			Artifacts: []GeneratedArtifactPlan{
				{Kind: GeneratedArtifactKindCGONativeServer, Filename: "test/v1/collision_sibling.server.native.cgo.rpccgo.go"},
			},
		})

		err := renderNativeServerCGOFile(plugin, plan, plan.Services[0], plan.Services[0].Artifacts[0])
		if err == nil {
			t.Fatal("renderNativeServerCGOFile() error = nil, want sibling symbol collision")
		}
		if got := err.Error(); !strings.Contains(got, "AllServiceUnaryCGONativeUnaryRequest") || !strings.Contains(got, "collides") {
			t.Fatalf("renderNativeServerCGOFile() error = %q, want sibling collision", got)
		}
	})
}

func TestRenderNativeServerCGOGeneratedSourceCompiles(t *testing.T) {
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
			strings.Contains(name, ".server.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go")
	})
	writeNativeServerCompileStubs(t, tmp)

	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated cgo native server go test failed: %v\n%s", err, out)
	}
}

func nativeServerScalarOnlyFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_scalar.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("ScalarRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("enabled", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("count", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
			{
				Name: proto.String("ScalarReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("count", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Scalar"),
			Method: []*descriptorpb.MethodDescriptorProto{
				methodDescriptor("Unary", ".test.v1.ScalarRequest", ".test.v1.ScalarReply", false, false),
			},
		}},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
			Path:            []int32{6, 0},
			Span:            []int32{0, 0, 0},
			LeadingComments: proto.String("@rpccgo: native\n"),
		}}},
	}
}

func nativeServerRepeatedFile() *descriptorpb.FileDescriptorProto {
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

func writeNativeServerCGOTestFile(t *testing.T, target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

func nativeServerCGOCollisionTestFilePlan(serviceName string, methods []MethodPlan) FilePlan {
	return FilePlan{
		GoPackageName: "testv1",
		GoImportPath:  "example.com/test/v1",
		Services: []ServicePlan{{
			Name:       serviceName,
			GoName:     serviceName,
			FullName:   "test.v1." + serviceName,
			Generation: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
			Methods:    methods,
			Artifacts: []GeneratedArtifactPlan{
				{Kind: GeneratedArtifactKindCGONativeServer, Filename: "test/v1/collision.server.native.cgo.rpccgo.go"},
			},
		}},
	}
}
