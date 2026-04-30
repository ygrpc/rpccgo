package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestNativeServerStreamingRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeServerStreamingTestPlugin(t, "example.com/nativeserverstream/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeServerStreamingFixture(t, tmp, plugin, "example.com/nativeserverstream")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_go_test.go"), nativeServerStreamingGoFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "TestNativeServerStreamingGo", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native server streaming go fixture failed: %v\n%s", err, out)
	}
}

func TestNativeServerStreamingRoutesToCGONativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeServerStreamingTestPlugin(t, "example.com/nativeserverstreamcgo/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeServerStreamingFixture(t, tmp, plugin, "example.com/nativeserverstreamcgo")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_cgo_callbacks.go"), nativeServerStreamingCGOFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_cgo_test.go"), nativeServerStreamingCGOFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "TestNativeServerStreamingCGO", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native server streaming cgo fixture failed: %v\n%s", err, out)
	}
}

func writeNativeServerStreamingFixture(t *testing.T, tmp string, plugin *protogen.Plugin, module string) {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.mod"), "module "+module+"\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n")
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".server.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go")
		if !include {
			continue
		}
		writeFile(t, filepath.Join(tmp, name), generated.GetContent())
	}
	writeFile(t, filepath.Join(tmp, "test/v1/native_server_streaming_stubs.go"), nativeServerStreamingStubSource)
}

func newNativeServerStreamingTestPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"test/v1/native_server_streaming.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("test/v1/native_server_streaming.proto"),
			Package: proto.String("test.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String(goPackage),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ListRequest"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("prefix", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("limit", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
				{
					Name: proto.String("ListReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("index", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("name", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:            proto.String("List"),
					InputType:       proto.String(".test.v1.ListRequest"),
					OutputType:      proto.String(".test.v1.ListReply"),
					ServerStreaming: proto.Bool(true),
				}},
			}},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
				Path:            []int32{6, 0},
				Span:            []int32{0, 0, 0},
				LeadingComments: proto.String("@rpccgo: native\n"),
			}}},
		}},
	}
	plugin, err := generator.ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

const nativeServerStreamingStubSource = `package testv1

type ListRequest struct {
	Prefix string
	Limit int32
}

type ListReply struct {
	Index int32
	Name string
}
`

const nativeServerStreamingGoFixtureTestSource = `package main

import (
	context "context"
	"errors"
	"io"
	"strings"
	"testing"
	"unsafe"

	v1 "example.com/nativeserverstream/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type listGoServer struct {
	label string
	stream *listGoStream
}

func (s *listGoServer) List(ctx context.Context, req *v1.ListRequest) (v1.GreeterListNativeServerStream, error) {
	s.stream = &listGoStream{prefix: s.label + req.Prefix, limit: req.Limit}
	return s.stream, nil
}

type listGoStream struct {
	prefix string
	limit int32
	index int32
	canceled bool
}

func (s *listGoStream) Recv(ctx context.Context) (*v1.ListReply, error) {
	if s.index >= s.limit {
		return nil, io.EOF
	}
	s.index++
	return &v1.ListReply{Index: s.index, Name: s.prefix + ":" + string(rune('0'+s.index))}, nil
}

func (s *listGoStream) Cancel(ctx context.Context) error {
	s.canceled = true
	return nil
}

func TestNativeServerStreamingGoServerReadDoneFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	server := &listGoServer{}
	if _, err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	input := listInput("go", 2)
	handle, errID := StartGreeterListNativeServerStream(context.Background(), input)
	if errID != 0 {
		t.Fatalf("StartGreeterListNativeServerStream() errID = %d", errID)
	}
	assertListRead(t, handle, 1, "go:1")
	assertListRead(t, handle, 2, "go:2")
	assertErrorTextContainsServerStream(t, ReadGreeterListNativeServerStream(context.Background(), handle, &GreeterListNativeServerStreamOutput{}), "EOF")
	if errID := DoneGreeterListNativeServerStream(context.Background(), handle); errID != 0 {
		t.Fatalf("DoneGreeterListNativeServerStream() errID = %d", errID)
	}
	if errID := ReadGreeterListNativeServerStream(context.Background(), handle, &GreeterListNativeServerStreamOutput{}); errID == 0 {
		t.Fatal("Read after Done returned errID 0")
	}
}

func TestNativeServerStreamingGoServerStartCapturesActiveServerSnapshot(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	serverA := &listGoServer{label: "A:"}
	if _, err := v1.RegisterGreeterGoNativeServer(serverA); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(A) error = %v", err)
	}
	handle, errID := StartGreeterListNativeServerStream(context.Background(), listInput("x", 1))
	if errID != 0 {
		t.Fatalf("StartGreeterListNativeServerStream() errID = %d", errID)
	}
	serverB := &listGoServer{label: "B:"}
	if _, err := v1.RegisterGreeterGoNativeServer(serverB); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(B) error = %v", err)
	}
	assertListRead(t, handle, 1, "A:x:1")
	if serverB.stream != nil {
		t.Fatalf("server B unexpectedly received stream: %#v", serverB.stream)
	}
}

func TestNativeServerStreamingGoServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	server := &listGoServer{}
	if _, err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	handle, errID := StartGreeterListNativeServerStream(context.Background(), listInput("go", 1))
	if errID != 0 {
		t.Fatalf("StartGreeterListNativeServerStream() errID = %d", errID)
	}
	if errID := CancelGreeterListNativeServerStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CancelGreeterListNativeServerStream() errID = %d", errID)
	}
	if server.stream == nil || !server.stream.canceled {
		t.Fatal("Cancel did not propagate to Go native stream")
	}
	if errID := CancelGreeterListNativeServerStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Cancel returned errID 0")
	}
}

func TestNativeServerStreamingGoServerMissingActiveServer(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	handle, errID := StartGreeterListNativeServerStream(context.Background(), listInput("none", 1))
	if handle != 0 {
		t.Fatalf("handle = %d, want 0", handle)
	}
	assertErrorTextContainsServerStream(t, errID, "active server")
}

func listInput(prefix string, limit int32) *GreeterListNativeServerStreamInput {
	data := []byte(prefix)
	return &GreeterListNativeServerStreamInput{
		PrefixPtr: uintptr(unsafe.Pointer(&data[0])),
		PrefixLen: int32(len(data)),
		Limit: limit,
	}
}

func assertListRead(t *testing.T, handle int32, wantIndex int32, wantName string) {
	t.Helper()
	output := &GreeterListNativeServerStreamOutput{}
	if errID := ReadGreeterListNativeServerStream(context.Background(), handle, output); errID != 0 {
		t.Fatalf("ReadGreeterListNativeServerStream() errID = %d", errID)
	}
	if output.Index != wantIndex {
		t.Fatalf("Index = %d, want %d", output.Index, wantIndex)
	}
	name := unsafe.Slice((*byte)(unsafe.Pointer(output.NamePtr)), output.NameLen)
	if string(name) != wantName {
		t.Fatalf("Name = %q, want %q", name, wantName)
	}
	rpcruntime.Release(output.NamePtr)
}

func assertErrorTextContainsServerStream(t *testing.T, errID int32, want string) {
	t.Helper()
	if errID == 0 {
		t.Fatalf("errID = 0, want %q", want)
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || (!strings.Contains(string(text), want) && !errors.Is(errors.New(string(text)), io.EOF)) {
		t.Fatalf("error text = %q, ok=%v, want %q", text, ok, want)
	}
}
`

const nativeServerStreamingCGOFixtureTestSource = `package main

import (
	context "context"
	"strings"
	"testing"
	"unsafe"

	v1 "example.com/nativeserverstreamcgo/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

func TestNativeServerStreamingCGOServerReadDoneFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerServerStreamCFreeCallback()
	if err := registerGreeterServerStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterServerStreamCGONativeServerCallbacks() error = %v", err)
	}

	handle, errID := StartGreeterListNativeServerStream(context.Background(), listInputCGO("cgo", 2))
	if errID != 0 {
		t.Fatalf("StartGreeterListNativeServerStream() errID = %d", errID)
	}
	assertListReadCGO(t, handle, 1, "cgo:1")
	assertListReadCGO(t, handle, 2, "cgo:2")
	if got := frees(); got != 2 {
		t.Fatalf("free count after reads = %d, want 2", got)
	}
	assertErrorTextContainsCGOServerStream(t, ReadGreeterListNativeServerStream(context.Background(), handle, &GreeterListNativeServerStreamOutput{}), "server stream done")
	if errID := DoneGreeterListNativeServerStream(context.Background(), handle); errID != 0 {
		t.Fatalf("DoneGreeterListNativeServerStream() errID = %d", errID)
	}
	if got := greeterServerStreamDoneCount(); got != 1 {
		t.Fatalf("done count = %d, want 1", got)
	}
	if errID := DoneGreeterListNativeServerStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Done returned errID 0")
	}
}

func TestNativeServerStreamingCGOServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if err := registerGreeterServerStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterServerStreamCGONativeServerCallbacks() error = %v", err)
	}
	handle, errID := StartGreeterListNativeServerStream(context.Background(), listInputCGO("cgo", 1))
	if errID != 0 {
		t.Fatalf("StartGreeterListNativeServerStream() errID = %d", errID)
	}
	if errID := CancelGreeterListNativeServerStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CancelGreeterListNativeServerStream() errID = %d", errID)
	}
	if got := greeterServerStreamCancelCount(); got != 1 {
		t.Fatalf("cancel count = %d, want 1", got)
	}
	if errID := ReadGreeterListNativeServerStream(context.Background(), handle, &GreeterListNativeServerStreamOutput{}); errID == 0 {
		t.Fatal("Read after Cancel returned errID 0")
	}
}

func TestNativeServerStreamingCGOServerCallbackErrorsPropagate(t *testing.T) {
	tests := []struct {
		name string
		mode int32
		run func(t *testing.T, handle int32) int32
		want string
	}{
		{name: "start", mode: 1, run: func(t *testing.T, handle int32) int32 { return 0 }, want: "forced start error"},
		{name: "recv", mode: 2, run: func(t *testing.T, handle int32) int32 {
			return ReadGreeterListNativeServerStream(context.Background(), handle, &GreeterListNativeServerStreamOutput{})
		}, want: "forced recv error"},
		{name: "done", mode: 3, run: func(t *testing.T, handle int32) int32 {
			return DoneGreeterListNativeServerStream(context.Background(), handle)
		}, want: "forced done error"},
		{name: "cancel", mode: 4, run: func(t *testing.T, handle int32) int32 {
			return CancelGreeterListNativeServerStream(context.Background(), handle)
		}, want: "forced cancel error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1.ResetGreeterDispatcherForIntegrationTest()
			setGreeterServerStreamErrorMode(tt.mode)
			t.Cleanup(func() { setGreeterServerStreamErrorMode(0) })
			if err := registerGreeterServerStreamCGONativeServerCallbacks(); err != nil {
				t.Fatalf("registerGreeterServerStreamCGONativeServerCallbacks() error = %v", err)
			}
			handle, errID := StartGreeterListNativeServerStream(context.Background(), listInputCGO("cgo", 1))
			if tt.name == "start" {
				assertErrorTextContainsCGOServerStream(t, errID, tt.want)
				if handle != 0 {
					t.Fatalf("handle = %d, want 0", handle)
				}
				return
			}
			if errID != 0 {
				t.Fatalf("StartGreeterListNativeServerStream() errID = %d", errID)
			}
			assertErrorTextContainsCGOServerStream(t, tt.run(t, handle), tt.want)
		})
	}
}

func listInputCGO(prefix string, limit int32) *GreeterListNativeServerStreamInput {
	data := []byte(prefix)
	return &GreeterListNativeServerStreamInput{
		PrefixPtr: uintptr(unsafe.Pointer(&data[0])),
		PrefixLen: int32(len(data)),
		Limit: limit,
	}
}

func assertListReadCGO(t *testing.T, handle int32, wantIndex int32, wantName string) {
	t.Helper()
	output := &GreeterListNativeServerStreamOutput{}
	if errID := ReadGreeterListNativeServerStream(context.Background(), handle, output); errID != 0 {
		t.Fatalf("ReadGreeterListNativeServerStream() errID = %d", errID)
	}
	if output.Index != wantIndex {
		t.Fatalf("Index = %d, want %d", output.Index, wantIndex)
	}
	name := unsafe.Slice((*byte)(unsafe.Pointer(output.NamePtr)), output.NameLen)
	if string(name) != wantName {
		t.Fatalf("Name = %q, want %q", name, wantName)
	}
}

func assertErrorTextContainsCGOServerStream(t *testing.T, errID int32, want string) {
	t.Helper()
	if errID == 0 {
		t.Fatalf("errID = 0, want %q", want)
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), want) {
		t.Fatalf("error text = %q, ok=%v, want %q", text, ok, want)
	}
}
`

const nativeServerStreamingCGOFixtureCallbackSource = `package main

/*
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

extern int32_t StoreGreeterCGONativeServerErrorTextForExport(char* text, int32_t textLen);

typedef struct GreeterListCGONativeServerStreamRequest {
uintptr_t PrefixPtr;
int32_t PrefixLen;
int32_t Limit;
} GreeterListCGONativeServerStreamRequest;

typedef struct GreeterListCGONativeServerStreamResponse {
int32_t Index;
uintptr_t NamePtr;
int32_t NameLen;
int32_t NameOwnership;
} GreeterListCGONativeServerStreamResponse;

typedef int32_t (*GreeterListCGONativeServerStreamStartCallback)(GreeterListCGONativeServerStreamRequest* input, int32_t* stream);
typedef int32_t (*GreeterListCGONativeServerStreamRecvCallback)(int32_t stream, GreeterListCGONativeServerStreamResponse* output);
typedef int32_t (*GreeterListCGONativeServerStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGONativeServerCallbacks {
GreeterListCGONativeServerStreamStartCallback ListStart;
GreeterListCGONativeServerStreamRecvCallback ListRecv;
GreeterListCGONativeServerStreamDoneCallback ListDone;
GreeterListCGONativeServerStreamCancelCallback ListCancel;
} GreeterCGONativeServerCallbacks;

static int32_t greeterServerStreamID;
static int32_t greeterServerStreamIndex;
static int32_t greeterServerStreamLimit;
static int32_t greeterServerStreamCancels;
static int32_t greeterServerStreamDones;
static int32_t greeterServerStreamErrorMode;
static char greeterServerStreamPrefix[64];

static int32_t greeterServerStreamError(const char* text) {
	return StoreGreeterCGONativeServerErrorTextForExport((char*)text, (int32_t)strlen(text));
}

static int32_t greeterListStart(GreeterListCGONativeServerStreamRequest* input, int32_t* stream) {
	if (greeterServerStreamErrorMode == 1) {
		return greeterServerStreamError("forced start error");
	}
	if (input == NULL || stream == NULL) {
		return greeterServerStreamError("server stream start missing input");
	}
	int32_t n = input->PrefixLen;
	if (n < 0 || n >= 60) {
		return greeterServerStreamError("server stream bad prefix");
	}
	memcpy(greeterServerStreamPrefix, (void*)input->PrefixPtr, (size_t)n);
	greeterServerStreamPrefix[n] = 0;
	greeterServerStreamID = 71;
	greeterServerStreamIndex = 0;
	greeterServerStreamLimit = input->Limit;
	greeterServerStreamCancels = 0;
	greeterServerStreamDones = 0;
	*stream = greeterServerStreamID;
	return 0;
}

static int32_t greeterListRecv(int32_t stream, GreeterListCGONativeServerStreamResponse* output) {
	if (greeterServerStreamErrorMode == 2) {
		return greeterServerStreamError("forced recv error");
	}
	if (stream != greeterServerStreamID || output == NULL) {
		return greeterServerStreamError("server stream recv did not reach cgo callback");
	}
	if (greeterServerStreamIndex >= greeterServerStreamLimit) {
		return greeterServerStreamError("server stream done");
	}
	greeterServerStreamIndex += 1;
	char buf[96];
	int n = snprintf(buf, sizeof(buf), "%s:%d", greeterServerStreamPrefix, greeterServerStreamIndex);
	char* name = (char*)malloc((size_t)n);
	if (name == NULL) {
		return greeterServerStreamError("name malloc failed");
	}
	memcpy(name, buf, (size_t)n);
	output->Index = greeterServerStreamIndex;
	output->NamePtr = (uintptr_t)name;
	output->NameLen = n;
	output->NameOwnership = 1;
	return 0;
}

static int32_t greeterListDone(int32_t stream) {
	if (greeterServerStreamErrorMode == 3) {
		return greeterServerStreamError("forced done error");
	}
	if (stream != greeterServerStreamID) {
		return greeterServerStreamError("server stream done did not reach cgo callback");
	}
	greeterServerStreamDones += 1;
	return 0;
}

static int32_t greeterListCancel(int32_t stream) {
	if (greeterServerStreamErrorMode == 4) {
		return greeterServerStreamError("forced cancel error");
	}
	if (stream != greeterServerStreamID) {
		return greeterServerStreamError("server stream cancel did not reach cgo callback");
	}
	greeterServerStreamCancels += 1;
	return 0;
}

static GreeterCGONativeServerCallbacks greeterServerStreamCallbacks(void) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.ListStart = greeterListStart;
	callbacks.ListRecv = greeterListRecv;
	callbacks.ListDone = greeterListDone;
	callbacks.ListCancel = greeterListCancel;
	return callbacks;
}

static int32_t greeterServerStreamCancelCount(void) {
	return greeterServerStreamCancels;
}

static int32_t greeterServerStreamDoneCount(void) {
	return greeterServerStreamDones;
}

static void setGreeterServerStreamErrorMode(int32_t mode) {
	greeterServerStreamErrorMode = mode;
}
*/
import "C"

import (
	"sync/atomic"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func registerGreeterServerStreamCGONativeServerCallbacks() error {
	callbacks := C.greeterServerStreamCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func greeterServerStreamCancelCount() int32 {
	return int32(C.greeterServerStreamCancelCount())
}

func greeterServerStreamDoneCount() int32 {
	return int32(C.greeterServerStreamDoneCount())
}

func setGreeterServerStreamErrorMode(mode int32) {
	C.setGreeterServerStreamErrorMode(C.int32_t(mode))
}

func registerServerStreamCFreeCallback() func() int32 {
	var frees atomic.Int32
	rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {
		frees.Add(1)
		C.free(ptr)
	})
	return frees.Load
}
`
