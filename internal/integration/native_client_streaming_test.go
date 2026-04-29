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

func TestNativeClientStreamingRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeClientStreamingTestPlugin(t, "example.com/nativeclientstream/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeClientStreamingFixture(t, tmp, plugin, "example.com/nativeclientstream")
	writeFile(t, filepath.Join(tmp, "test/v1/native_client_streaming_go_test.go"), nativeClientStreamingGoFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1", "-run", "TestNativeClientStreamingGo", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native client streaming go fixture failed: %v\n%s", err, out)
	}
}

func TestNativeClientStreamingRoutesToCGONativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeClientStreamingTestPlugin(t, "example.com/nativeclientstreamcgo/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeClientStreamingFixture(t, tmp, plugin, "example.com/nativeclientstreamcgo")
	writeFile(t, filepath.Join(tmp, "test/v1/native_client_streaming_cgo_callbacks.go"), nativeClientStreamingCGOFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/native_client_streaming_cgo_test.go"), nativeClientStreamingCGOFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1", "-run", "TestNativeClientStreamingCGO", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native client streaming cgo fixture failed: %v\n%s", err, out)
	}
}

func writeNativeClientStreamingFixture(t *testing.T, tmp string, plugin *protogen.Plugin, module string) {
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
	writeFile(t, filepath.Join(tmp, "test/v1/native_client_streaming_stubs.go"), nativeClientStreamingStubSource)
}

func newNativeClientStreamingTestPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"test/v1/native_client_streaming.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("test/v1/native_client_streaming.proto"),
			Package: proto.String("test.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String(goPackage),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("UploadRequest"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
				{
					Name: proto.String("UploadReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("summary", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:            proto.String("Upload"),
					InputType:       proto.String(".test.v1.UploadRequest"),
					OutputType:      proto.String(".test.v1.UploadReply"),
					ClientStreaming: proto.Bool(true),
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

const nativeClientStreamingStubSource = `package testv1

type UploadRequest struct {
	Name string
	Payload []byte
}

type UploadReply struct {
	Count int32
	Summary string
}
`

const nativeClientStreamingGoFixtureTestSource = `package testv1

import (
	context "context"
	strings "strings"
	"testing"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

type uploadGoServer struct {
	stream *uploadGoStream
}

func (s *uploadGoServer) Upload(ctx context.Context) (GreeterUploadNativeClientStream, error) {
	s.stream = &uploadGoStream{}
	return s.stream, nil
}

type uploadGoStream struct {
	names []string
	payloads []string
	canceled bool
}

func (s *uploadGoStream) Send(ctx context.Context, req *UploadRequest) error {
	s.names = append(s.names, req.Name)
	s.payloads = append(s.payloads, string(req.Payload))
	return nil
}

func (s *uploadGoStream) Finish(ctx context.Context) (*UploadReply, error) {
	return &UploadReply{Count: int32(len(s.payloads)), Summary: strings.Join(s.names, ",")+":"+strings.Join(s.payloads, "|")}, nil
}

func (s *uploadGoStream) Cancel(ctx context.Context) error {
	s.canceled = true
	return nil
}

func TestNativeClientStreamingGoServerFinishFinalizesHandle(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	server := &uploadGoServer{}
	if _, err := RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	sendUpload(t, handle, "first", "aa")
	sendUpload(t, handle, "second", "bbb")

	output := &GreeterUploadNativeClientStreamOutput{}
	if errID := FinishGreeterUploadNativeClientStream(context.Background(), handle, output); errID != 0 {
		t.Fatalf("FinishGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if output.Count != 2 {
		t.Fatalf("Count = %d, want 2", output.Count)
	}
	summary := unsafe.Slice((*byte)(unsafe.Pointer(output.SummaryPtr)), output.SummaryLen)
	if string(summary) != "first,second:aa|bbb" {
		t.Fatalf("Summary = %q", summary)
	}
	rpcruntime.Release(output.SummaryPtr)

	errID = SendGreeterUploadNativeClientStream(context.Background(), handle, &GreeterUploadNativeClientStreamInput{})
	if errID == 0 {
		t.Fatal("Send after Finish returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "native client stream handle is invalid") {
		t.Fatalf("Send after Finish error text = %q, ok=%v", text, ok)
	}
}

func TestNativeClientStreamingGoServerCancelFinalizesHandle(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	server := &uploadGoServer{}
	if _, err := RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if errID := CancelGreeterUploadNativeClientStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CancelGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if server.stream == nil || !server.stream.canceled {
		t.Fatal("Cancel did not propagate to Go native stream")
	}
	if errID := CancelGreeterUploadNativeClientStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Cancel returned errID 0")
	}
}

func sendUpload(t *testing.T, handle int32, nameValue, payloadValue string) {
	t.Helper()
	name := []byte(nameValue)
	payload := []byte(payloadValue)
	input := &GreeterUploadNativeClientStreamInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
	}
	if errID := SendGreeterUploadNativeClientStream(context.Background(), handle, input); errID != 0 {
		t.Fatalf("SendGreeterUploadNativeClientStream() errID = %d", errID)
	}
}
`

const nativeClientStreamingCGOFixtureTestSource = `package testv1

import (
	context "context"
	strings "strings"
	"testing"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func TestNativeClientStreamingCGOServerFinishFinalizesHandle(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	registerClientStreamCFreeCallback()
	if err := registerGreeterClientStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterClientStreamCGONativeServerCallbacks() error = %v", err)
	}

	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	sendUploadCGO(t, handle, "one", "ab")
	sendUploadCGO(t, handle, "two", "cde")

	output := &GreeterUploadNativeClientStreamOutput{}
	if errID := FinishGreeterUploadNativeClientStream(context.Background(), handle, output); errID != 0 {
		t.Fatalf("FinishGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if output.Count != 2 {
		t.Fatalf("Count = %d, want 2", output.Count)
	}
	summary := unsafe.Slice((*byte)(unsafe.Pointer(output.SummaryPtr)), output.SummaryLen)
	if string(summary) != "cgo:2:5" {
		t.Fatalf("Summary = %q", summary)
	}
	rpcruntime.Release(output.SummaryPtr)

	if errID := SendGreeterUploadNativeClientStream(context.Background(), handle, &GreeterUploadNativeClientStreamInput{}); errID == 0 {
		t.Fatal("Send after cgo Finish returned errID 0")
	}
}

func TestNativeClientStreamingCGOServerCancelFinalizesHandle(t *testing.T) {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
	if err := registerGreeterClientStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterClientStreamCGONativeServerCallbacks() error = %v", err)
	}
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if errID := CancelGreeterUploadNativeClientStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CancelGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if got := greeterClientStreamCancelCount(); got != 1 {
		t.Fatalf("cancel count = %d, want 1", got)
	}
	errID = SendGreeterUploadNativeClientStream(context.Background(), handle, &GreeterUploadNativeClientStreamInput{})
	if errID == 0 {
		t.Fatal("Send after cgo Cancel returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "native client stream handle is invalid") {
		t.Fatalf("Send after cgo Cancel error text = %q, ok=%v", text, ok)
	}
}

func sendUploadCGO(t *testing.T, handle int32, nameValue, payloadValue string) {
	t.Helper()
	name := []byte(nameValue)
	payload := []byte(payloadValue)
	input := &GreeterUploadNativeClientStreamInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
	}
	if errID := SendGreeterUploadNativeClientStream(context.Background(), handle, input); errID != 0 {
		t.Fatalf("SendGreeterUploadNativeClientStream() errID = %d", errID)
	}
}
`

const nativeClientStreamingCGOFixtureCallbackSource = `package testv1

/*
#include <stdint.h>
#include <stdlib.h>

extern int32_t StoreGreeterCGONativeServerErrorTextForExport(char* text, int32_t textLen);

typedef struct GreeterUploadCGONativeClientStreamRequest {
uintptr_t NamePtr;
int32_t NameLen;
uintptr_t PayloadPtr;
int32_t PayloadLen;
} GreeterUploadCGONativeClientStreamRequest;

typedef struct GreeterUploadCGONativeClientStreamResponse {
int32_t Count;
uintptr_t SummaryPtr;
int32_t SummaryLen;
int32_t SummaryOwnership;
} GreeterUploadCGONativeClientStreamResponse;

typedef int32_t (*GreeterUploadCGONativeClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamSendCallback)(int32_t stream, GreeterUploadCGONativeClientStreamRequest* input);
typedef int32_t (*GreeterUploadCGONativeClientStreamFinishCallback)(int32_t stream, GreeterUploadCGONativeClientStreamResponse* output);
typedef int32_t (*GreeterUploadCGONativeClientStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGONativeServerCallbacks {
GreeterUploadCGONativeClientStreamStartCallback UploadStart;
GreeterUploadCGONativeClientStreamSendCallback UploadSend;
GreeterUploadCGONativeClientStreamFinishCallback UploadFinish;
GreeterUploadCGONativeClientStreamCancelCallback UploadCancel;
} GreeterCGONativeServerCallbacks;

static int32_t greeterStreamID;
static int32_t greeterStreamCount;
static int32_t greeterStreamBytes;
static int32_t greeterStreamCancels;

static int32_t greeterUploadStart(int32_t* stream) {
	if (stream == NULL) {
		char msg[] = "stream output missing";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	greeterStreamID = 41;
	greeterStreamCount = 0;
	greeterStreamBytes = 0;
	greeterStreamCancels = 0;
	*stream = greeterStreamID;
	return 0;
}

static int32_t greeterUploadSend(int32_t stream, GreeterUploadCGONativeClientStreamRequest* input) {
	if (stream != greeterStreamID || input == NULL) {
		char msg[] = "stream send did not reach cgo callback";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	greeterStreamCount += 1;
	greeterStreamBytes += input->PayloadLen;
	return 0;
}

static int32_t greeterUploadFinish(int32_t stream, GreeterUploadCGONativeClientStreamResponse* output) {
	if (stream != greeterStreamID || output == NULL) {
		char msg[] = "stream finish did not reach cgo callback";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	char* summary = (char*)malloc(7);
	if (summary == NULL) {
		char msg[] = "summary malloc failed";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	summary[0] = 'c'; summary[1] = 'g'; summary[2] = 'o'; summary[3] = ':'; summary[4] = '2'; summary[5] = ':'; summary[6] = '5';
	output->Count = greeterStreamCount;
	output->SummaryPtr = (uintptr_t)summary;
	output->SummaryLen = 7;
	output->SummaryOwnership = 1;
	return 0;
}

static int32_t greeterUploadCancel(int32_t stream) {
	if (stream != greeterStreamID) {
		char msg[] = "stream cancel did not reach cgo callback";
		return StoreGreeterCGONativeServerErrorTextForExport(msg, sizeof(msg)-1);
	}
	greeterStreamCancels += 1;
	return 0;
}

static GreeterCGONativeServerCallbacks greeterClientStreamCallbacks(void) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.UploadStart = greeterUploadStart;
	callbacks.UploadSend = greeterUploadSend;
	callbacks.UploadFinish = greeterUploadFinish;
	callbacks.UploadCancel = greeterUploadCancel;
	return callbacks;
}

static int32_t greeterClientStreamCancelCount(void) {
	return greeterStreamCancels;
}
*/
import "C"

import (
	"sync/atomic"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func registerGreeterClientStreamCGONativeServerCallbacks() error {
	callbacks := C.greeterClientStreamCallbacks()
	_, err := RegisterGreeterCGONativeServer(&callbacks)
	return err
}

func greeterClientStreamCancelCount() int32 {
	return int32(C.greeterClientStreamCancelCount())
}

func registerClientStreamCFreeCallback() func() int32 {
	var frees atomic.Int32
	rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {
		frees.Add(1)
		C.free(ptr)
	})
	return frees.Load
}
`
