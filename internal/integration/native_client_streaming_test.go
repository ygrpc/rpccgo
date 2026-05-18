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
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_go_test.go"), nativeClientStreamingGoFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "TestNativeClientStreamingGo", "-count=1")
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
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_cgo_callbacks.go"), nativeClientStreamingCGOFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_cgo_test.go"), nativeClientStreamingCGOFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "TestNativeClientStreamingCGO", "-count=1")
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

const nativeClientStreamingGoFixtureTestSource = `package main

import (
	context "context"
	io "io"
	strings "strings"
	"testing"
	"unsafe"

	v1 "example.com/nativeclientstream/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type uploadGoServer struct {
	label string
	stream *uploadGoStream
}

func (s *uploadGoServer) Upload(ctx context.Context, stream v1.GreeterUploadNativeClientStream) (int32, string, error) {
	s.stream = &uploadGoStream{label: s.label}
	for {
		name, payload, err := stream.Recv(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			s.stream.canceled = true
			return 0, "", err
		}
		s.stream.names = append(s.stream.names, name.SafeString())
		s.stream.payloads = append(s.stream.payloads, string(payload.SafeBytes()))
	}
	return s.stream.Finish(ctx)
}

type uploadGoStream struct {
	label string
	names []string
	payloads []string
	canceled bool
}

func (s *uploadGoStream) Send(ctx context.Context, name *rpcruntime.RpcString, payload *rpcruntime.RpcBytes) error {
	s.names = append(s.names, name.SafeString())
	s.payloads = append(s.payloads, string(payload.SafeBytes()))
	return nil
}

func (s *uploadGoStream) Finish(ctx context.Context) (int32, string, error) {
	prefix := s.label
	if prefix != "" {
		prefix += ":"
	}
	return int32(len(s.payloads)), prefix+strings.Join(s.names, ",")+":"+strings.Join(s.payloads, "|"), nil
}

func (s *uploadGoStream) Cancel(ctx context.Context) error {
	s.canceled = true
	return nil
}

type uploadOutput struct {
	Count int32
	SummaryPtr uintptr
	SummaryLen int32
}

func finishUpload(ctx context.Context, handle int32, output *uploadOutput) int32 {
	if output == nil {
		output = &uploadOutput{}
	}
	return FinishGreeterUploadNativeClientStream(ctx, handle, &output.Count, &output.SummaryPtr, &output.SummaryLen)
}

func TestNativeClientStreamingGoServerFinishFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	server := &uploadGoServer{}
	if _, err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	sendUpload(t, handle, "first", "aa")
	sendUpload(t, handle, "second", "bbb")

	output := &uploadOutput{}
	if errID := finishUpload(context.Background(), handle, output); errID != 0 {
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

	errID = SendGreeterUploadNativeClientStream(context.Background(), handle, 0, 0, 0, 0, 0, 0)
	if errID == 0 {
		t.Fatal("Send after Finish returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "native client stream handle is invalid") {
		t.Fatalf("Send after Finish error text = %q, ok=%v", text, ok)
	}
}

func TestNativeClientStreamingGoServerStartCapturesActiveServerSnapshot(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	serverA := &uploadGoServer{label: "A"}
	if _, err := v1.RegisterGreeterGoNativeServer(serverA); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(A) error = %v", err)
	}
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	serverB := &uploadGoServer{label: "B"}
	if _, err := v1.RegisterGreeterGoNativeServer(serverB); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(B) error = %v", err)
	}

	sendUpload(t, handle, "first", "aa")
	output := &uploadOutput{}
	if errID := finishUpload(context.Background(), handle, output); errID != 0 {
		t.Fatalf("FinishGreeterUploadNativeClientStream() errID = %d", errID)
	}
	summary := unsafe.Slice((*byte)(unsafe.Pointer(output.SummaryPtr)), output.SummaryLen)
	if string(summary) != "A:first:aa" {
		t.Fatalf("Summary = %q, want A stream response", summary)
	}
	rpcruntime.Release(output.SummaryPtr)
	if serverA.stream == nil || len(serverA.stream.payloads) != 1 {
		t.Fatalf("server A stream payloads = %#v", serverA.stream)
	}
	if serverB.stream != nil {
		t.Fatalf("server B unexpectedly received stream: %#v", serverB.stream)
	}
}

func TestNativeClientStreamingGoServerStartReportsMissingActiveServer(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if handle != 0 {
		t.Fatalf("handle = %d, want 0", handle)
	}
	if errID == 0 {
		t.Fatal("missing active server returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "active server") {
		t.Fatalf("missing active server error text = %q, ok=%v", text, ok)
	}
}

func TestNativeClientStreamingGoServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	server := &uploadGoServer{}
	if _, err := v1.RegisterGreeterGoNativeServer(server); err != nil {
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
	if errID := SendGreeterUploadNativeClientStream(context.Background(), handle, uintptr(unsafe.Pointer(&name[0])), int32(len(name)), 0, uintptr(unsafe.Pointer(&payload[0])), int32(len(payload)), 0); errID != 0 {
		t.Fatalf("SendGreeterUploadNativeClientStream() errID = %d", errID)
	}
}
`

const nativeClientStreamingCGOFixtureTestSource = `package main

import (
	context "context"
	strings "strings"
	"testing"
	"unsafe"

	v1 "example.com/nativeclientstreamcgo/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type uploadOutput struct {
	Count int32
	SummaryPtr uintptr
	SummaryLen int32
}

func finishUpload(ctx context.Context, handle int32, output *uploadOutput) int32 {
	if output == nil {
		output = &uploadOutput{}
	}
	return FinishGreeterUploadNativeClientStream(ctx, handle, &output.Count, &output.SummaryPtr, &output.SummaryLen)
}

func TestNativeClientStreamingCGOServerFinishFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerClientStreamCFreeCallback()
	if err := registerGreeterClientStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterClientStreamCGONativeServerCallbacks() error = %v", err)
	}

	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	sendUploadCGO(t, handle, "one", "ab")
	sendUploadCGO(t, handle, "two", "cde")

	output := &uploadOutput{}
	if errID := finishUpload(context.Background(), handle, output); errID != 0 {
		t.Fatalf("FinishGreeterUploadNativeClientStream() errID = %d", errID)
	}
	if errID := finishUpload(context.Background(), handle, &uploadOutput{}); errID == 0 {
		t.Fatal("second Finish returned errID 0")
	}
	if output.Count != 2 {
		t.Fatalf("Count = %d, want 2", output.Count)
	}
	summary := unsafe.Slice((*byte)(unsafe.Pointer(output.SummaryPtr)), output.SummaryLen)
	if string(summary) != "cgo:2:5" {
		t.Fatalf("Summary = %q", summary)
	}
	if got := frees(); got != 1 {
		t.Fatalf("free count after Finish = %d, want 1", got)
	}

	if errID := SendGreeterUploadNativeClientStream(context.Background(), handle, 0, 0, 0, 0, 0, 0); errID == 0 {
		t.Fatal("Send after cgo Finish returned errID 0")
	}
}

func TestNativeClientStreamingCGOServerCancelTwiceInvalidatesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
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
	if errID := CancelGreeterUploadNativeClientStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Cancel returned errID 0")
	}
	if got := greeterClientStreamCancelCount(); got != 1 {
		t.Fatalf("cancel count = %d, want 1", got)
	}
}

func TestNativeClientStreamingCGOServerCallbackErrorsPropagate(t *testing.T) {
	tests := []struct {
		name string
		mode int32
		run func(t *testing.T, handle int32) int32
		want string
	}{
		{name: "start", mode: 1, run: func(t *testing.T, handle int32) int32 { return 0 }, want: "forced start error"},
		{name: "send", mode: 2, run: func(t *testing.T, handle int32) int32 {
			return sendUploadCGOErr(handle, "one", "ab")
		}, want: "forced send error"},
		{name: "finish", mode: 3, run: func(t *testing.T, handle int32) int32 {
			return finishUpload(context.Background(), handle, &uploadOutput{})
		}, want: "forced finish error"},
		{name: "cancel", mode: 4, run: func(t *testing.T, handle int32) int32 {
			return CancelGreeterUploadNativeClientStream(context.Background(), handle)
		}, want: "forced cancel error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1.ResetGreeterDispatcherForIntegrationTest()
			setGreeterClientStreamErrorMode(tt.mode)
			t.Cleanup(func() { setGreeterClientStreamErrorMode(0) })
			if err := registerGreeterClientStreamCGONativeServerCallbacks(); err != nil {
				t.Fatalf("registerGreeterClientStreamCGONativeServerCallbacks() error = %v", err)
			}
			handle, errID := StartGreeterUploadNativeClientStream(context.Background())
			if tt.name == "start" {
				assertErrorTextContains(t, errID, tt.want)
				if handle != 0 {
					t.Fatalf("handle = %d, want 0", handle)
				}
				return
			}
			if errID != 0 {
				t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
			}
			assertErrorTextContains(t, tt.run(t, handle), tt.want)
		})
	}
}

func TestNativeClientStreamingCGOServerFinishErrorCleansOwnedOutput(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerClientStreamCFreeCallback()
	setGreeterClientStreamErrorMode(5)
	t.Cleanup(func() { setGreeterClientStreamErrorMode(0) })
	if err := registerGreeterClientStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterClientStreamCGONativeServerCallbacks() error = %v", err)
	}
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	output := &uploadOutput{}
	errID = finishUpload(context.Background(), handle, output)
	assertErrorTextContains(t, errID, "forced finish output error")
	if got := frees(); got != 1 {
		t.Fatalf("free count after Finish error = %d, want 1", got)
	}
}

func TestNativeClientStreamingCGOServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
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
	errID = SendGreeterUploadNativeClientStream(context.Background(), handle, 0, 0, 0, 0, 0, 0)
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
	if errID := sendUploadCGOErr(handle, nameValue, payloadValue); errID != 0 {
		t.Fatalf("SendGreeterUploadNativeClientStream() errID = %d", errID)
	}
}

func sendUploadCGOErr(handle int32, nameValue, payloadValue string) int32 {
	name := []byte(nameValue)
	payload := []byte(payloadValue)
	return SendGreeterUploadNativeClientStream(context.Background(), handle, uintptr(unsafe.Pointer(&name[0])), int32(len(name)), 0, uintptr(unsafe.Pointer(&payload[0])), int32(len(payload)), 0)
}

func assertErrorTextContains(t *testing.T, errID int32, want string) {
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

const nativeClientStreamingCGOFixtureCallbackSource = `package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

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
static int32_t greeterStreamErrorMode;

static int32_t greeterForcedError(const char* text) {
	return StoreGreeterCGONativeServerErrorTextForExport((char*)text, (int32_t)strlen(text));
}

static int32_t greeterUploadStart(int32_t* stream) {
	if (greeterStreamErrorMode == 1) {
		return greeterForcedError("forced start error");
	}
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
	if (greeterStreamErrorMode == 2) {
		return greeterForcedError("forced send error");
	}
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
	if (greeterStreamErrorMode == 3) {
		return greeterForcedError("forced finish error");
	}
	if (greeterStreamErrorMode == 5) {
		return greeterForcedError("forced finish output error");
	}
	return 0;
}

static int32_t greeterUploadCancel(int32_t stream) {
	if (greeterStreamErrorMode == 4) {
		return greeterForcedError("forced cancel error");
	}
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

static void setGreeterClientStreamErrorMode(int32_t mode) {
	greeterStreamErrorMode = mode;
}
*/
import "C"

import (
	"errors"
	"sync/atomic"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func registerGreeterClientStreamCGONativeServerCallbacks() error {
	callbacks := C.greeterClientStreamCallbacks()
	errID := rpccgo_native_testv1_Greeter_Upload_register(callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel)
	if errID == 0 {
		return nil
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok {
		return nil
	}
	if ptr != 0 {
		rpcruntime.Release(ptr)
	}
	return errors.New(string(text))
}

func greeterClientStreamCancelCount() int32 {
	return int32(C.greeterClientStreamCancelCount())
}

func setGreeterClientStreamErrorMode(mode int32) {
	C.setGreeterClientStreamErrorMode(C.int32_t(mode))
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
