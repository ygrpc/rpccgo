package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ygrpc/rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestNativeClientStreamingRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeClientStreamingTestPlugin(t, "example.com/nativeclientstream/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeClientStreamingFixture(t, tmp, plugin, "example.com/nativeclientstream")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_go_test.go"), nativeClientStreamingGoFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeClientStreamingGo", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native client streaming go fixture failed: %v\n%s", err, out)
	}
}

func TestNativeClientStreamingRoutesToCGONativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeClientStreamingTestPlugin(t, "example.com/nativeclientstreamcgo/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeClientStreamingFixture(t, tmp, plugin, "example.com/nativeclientstreamcgo")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_cgo_callbacks.go"), nativeClientStreamingCGOFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_cgo_test.go"), nativeClientStreamingCGOFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeClientStreamingCGO", "-count=1")
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
	writeFile(t, filepath.Join(tmp, "go.mod"), "module "+module+"\n\ngo 1.24.4\n\nrequire (\n\tgoogle.golang.org/protobuf v1.36.11\n\tgithub.com/ygrpc/rpccgo v0.0.0\n)\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n")
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.sum"), string(goSum))
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.message.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".exports.cgo.rpccgo.go") ||
			strings.Contains(name, ".server.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go")
		if !include {
			continue
		}
		writeFile(t, filepath.Join(tmp, name), generated.GetContent())
	}
	writeFile(t, filepath.Join(tmp, "test/v1/native_client_streaming_stubs.go"), nativeClientStreamingStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_client_streaming_cgo_client_bridge.go"), nativeClientStreamingCGOClientBridgeSource)
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

import (
	context "context"

	connect "connectrpc.com/connect"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type UploadRequest struct {
	Name string
	Payload []byte
}

func (*UploadRequest) ProtoReflect() protoreflect.Message { return nil }

type UploadReply struct {
	Count int32
	Summary string
}

func (*UploadReply) ProtoReflect() protoreflect.Message { return nil }

type GreeterHandler interface {
	Upload(context.Context, *connect.ClientStream[UploadRequest]) (*UploadReply, error)
}
type GreeterClient interface {
	Upload(context.Context) (*connect.ClientStreamForClientSimple[UploadRequest, UploadReply], error)
}
type GreeterServer interface {
	Upload(Greeter_UploadServer) error
}

type Greeter_UploadServer interface {
	Recv() (*UploadRequest, error)
	RecvMsg(any) error
	SendAndClose(*UploadReply) error
	SendMsg(any) error
	Context() context.Context
}
`

const nativeClientStreamingCGOClientBridgeSource = `package main

/*
#include <stdint.h>
*/
import "C"

import context "context"

func StartGreeterUploadNativeClientStream(ctx context.Context) (int32, int32) {
	var stream C.int32_t
	errID := rpccgo_native_testv1_Greeter_Upload_start(&stream)
	return int32(stream), int32(errID)
}

func SendGreeterUploadNativeClientStream(ctx context.Context, stream int32, NamePtr uintptr, NameLen int32, NameOwnership int32, PayloadPtr uintptr, PayloadLen int32, PayloadOwnership int32) int32 {
	return int32(rpccgo_native_testv1_Greeter_Upload_send(C.int32_t(stream), C.uintptr_t(NamePtr), C.int32_t(NameLen), C.int32_t(NameOwnership), C.uintptr_t(PayloadPtr), C.int32_t(PayloadLen), C.int32_t(PayloadOwnership)))
}

func FinishGreeterUploadNativeClientStream(ctx context.Context, stream int32, outCount *int32, outSummaryPtr *uintptr, outSummaryLen *int32) int32 {
	var count C.int32_t
	var summaryPtr C.uintptr_t
	var summaryLen C.int32_t
	var summaryOwnership C.int32_t
	errID := rpccgo_native_testv1_Greeter_Upload_finish(C.int32_t(stream), &count, &summaryPtr, &summaryLen, &summaryOwnership)
	*outCount = int32(count)
	*outSummaryPtr = uintptr(summaryPtr)
	*outSummaryLen = int32(summaryLen)
	return int32(errID)
}

func CancelGreeterUploadNativeClientStream(ctx context.Context, stream int32) int32 {
	return int32(rpccgo_native_testv1_Greeter_Upload_cancel(C.int32_t(stream)))
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
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
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
	v1.ResetGreeterServerForIntegrationTest()
	server := &uploadGoServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
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
	if !ok || !strings.Contains(string(text), "stream handle is invalid") {
		t.Fatalf("Send after Finish error text = %q, ok=%v", text, ok)
	}
}

func TestNativeClientStreamingGoServerStartCapturesActiveServerSnapshot(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	serverA := &uploadGoServer{label: "A"}
	if err := v1.RegisterGreeterGoNativeServer(serverA); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(A) error = %v", err)
	}
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadNativeClientStream() errID = %d", errID)
	}
	serverB := &uploadGoServer{label: "B"}
	if err := v1.RegisterGreeterGoNativeServer(serverB); err != nil {
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
	v1.ResetGreeterServerForIntegrationTest()
	handle, errID := StartGreeterUploadNativeClientStream(context.Background())
	if handle != 0 {
		t.Fatalf("handle = %d, want 0", handle)
	}
	if errID == 0 {
		t.Fatal("missing registered server returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "registered server") {
		t.Fatalf("missing registered server error text = %q, ok=%v", text, ok)
	}
}

func TestNativeClientStreamingGoServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &uploadGoServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
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
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
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
	v1.ResetGreeterServerForIntegrationTest()
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
	v1.ResetGreeterServerForIntegrationTest()
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

func TestNativeClientStreamingCGOServerFinishErrorCleansOwnedOutput(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerClientStreamCFreeCallback()
	t.Cleanup(func() { setGreeterClientStreamErrorMode(0) })
	if err := registerGreeterClientStreamCGONativeServerCallbacksWithMode(5); err != nil {
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
	v1.ResetGreeterServerForIntegrationTest()
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
	if !ok || !strings.Contains(string(text), "stream handle is invalid") {
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

extern int32_t rpccgo_store_error_text(char* text, int32_t textLen);

typedef int32_t (*GreeterUploadCGONativeClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamSendCallback)(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership);
typedef int32_t (*GreeterUploadCGONativeClientStreamFinishCallback)(int32_t stream, int32_t *outCount, uintptr_t *outSummaryPtr, int32_t *outSummaryLen, int32_t *outSummaryOwnership);
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
	return rpccgo_store_error_text((char*)text, (int32_t)strlen(text));
}

static int32_t greeterUploadStart(int32_t* stream) {
	if (greeterStreamErrorMode == 1) {
		return greeterForcedError("forced start error");
	}
	if (stream == NULL) {
		char msg[] = "stream output missing";
		return rpccgo_store_error_text(msg, sizeof(msg)-1);
	}
	greeterStreamID = 41;
	greeterStreamCount = 0;
	greeterStreamBytes = 0;
	greeterStreamCancels = 0;
	*stream = greeterStreamID;
	return 0;
}

static int32_t greeterUploadStartForcedError(int32_t* stream) {
	return greeterForcedError("forced start error");
}

static int32_t greeterUploadSend(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership) {
	if (greeterStreamErrorMode == 2) {
		return greeterForcedError("forced send error");
	}
	if (stream != greeterStreamID) {
		char msg[] = "stream send did not reach cgo callback";
		return rpccgo_store_error_text(msg, sizeof(msg)-1);
	}
	greeterStreamCount += 1;
	greeterStreamBytes += PayloadLen;
	return 0;
}

static int32_t greeterUploadSendForcedError(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, uintptr_t PayloadPtr, int32_t PayloadLen, int32_t PayloadOwnership) {
	return greeterForcedError("forced send error");
}

static int32_t greeterUploadFinish(int32_t stream, int32_t *outCount, uintptr_t *outSummaryPtr, int32_t *outSummaryLen, int32_t *outSummaryOwnership) {
	if (stream != greeterStreamID || outCount == NULL || outSummaryPtr == NULL || outSummaryLen == NULL || outSummaryOwnership == NULL) {
		char msg[] = "stream finish did not reach cgo callback";
		return rpccgo_store_error_text(msg, sizeof(msg)-1);
	}
	char* summary = (char*)malloc(7);
	if (summary == NULL) {
		char msg[] = "summary malloc failed";
		return rpccgo_store_error_text(msg, sizeof(msg)-1);
	}
	summary[0] = 'c'; summary[1] = 'g'; summary[2] = 'o'; summary[3] = ':'; summary[4] = '2'; summary[5] = ':'; summary[6] = '5';
	*outCount = greeterStreamCount;
	*outSummaryPtr = (uintptr_t)summary;
	*outSummaryLen = 7;
	*outSummaryOwnership = 1;
	if (greeterStreamErrorMode == 3) {
		return greeterForcedError("forced finish error");
	}
	if (greeterStreamErrorMode == 5) {
		return greeterForcedError("forced finish output error");
	}
	return 0;
}

static int32_t greeterUploadFinishForcedError(int32_t stream, int32_t *outCount, uintptr_t *outSummaryPtr, int32_t *outSummaryLen, int32_t *outSummaryOwnership) {
	return greeterForcedError("forced finish error");
}

static int32_t greeterUploadFinishForcedOutputError(int32_t stream, int32_t *outCount, uintptr_t *outSummaryPtr, int32_t *outSummaryLen, int32_t *outSummaryOwnership) {
	int32_t err = greeterUploadFinish(stream, outCount, outSummaryPtr, outSummaryLen, outSummaryOwnership);
	if (err != 0) {
		return err;
	}
	return greeterForcedError("forced finish output error");
}

static int32_t greeterUploadCancel(int32_t stream) {
	if (greeterStreamErrorMode == 4) {
		return greeterForcedError("forced cancel error");
	}
	if (stream != greeterStreamID) {
		char msg[] = "stream cancel did not reach cgo callback";
		return rpccgo_store_error_text(msg, sizeof(msg)-1);
	}
	greeterStreamCancels += 1;
	return 0;
}

static int32_t greeterUploadCancelForcedError(int32_t stream) {
	return greeterForcedError("forced cancel error");
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

static GreeterCGONativeServerCallbacks greeterClientStreamCallbacksWithMode(int32_t mode) {
	GreeterCGONativeServerCallbacks callbacks = greeterClientStreamCallbacks();
	if (mode == 1) {
		callbacks.UploadStart = greeterUploadStartForcedError;
	} else if (mode == 2) {
		callbacks.UploadSend = greeterUploadSendForcedError;
	} else if (mode == 3) {
		callbacks.UploadFinish = greeterUploadFinishForcedError;
	} else if (mode == 4) {
		callbacks.UploadCancel = greeterUploadCancelForcedError;
	} else if (mode == 5) {
		callbacks.UploadFinish = greeterUploadFinishForcedOutputError;
	}
	return callbacks;
}
*/
import "C"

import (
	"errors"
	"sync/atomic"
	"unsafe"

	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

func registerGreeterClientStreamCGONativeServerCallbacks() error {
	callbacks := C.greeterClientStreamCallbacks()
	return registerGreeterClientStreamCGONativeServerCallbackTable(callbacks)
}

func registerGreeterClientStreamCGONativeServerCallbacksWithMode(mode int32) error {
	callbacks := C.greeterClientStreamCallbacksWithMode(C.int32_t(mode))
	return registerGreeterClientStreamCGONativeServerCallbackTable(callbacks)
}

func registerGreeterClientStreamCGONativeServerCallbackTable(callbacks C.GreeterCGONativeServerCallbacks) error {
	errID := rpccgo_native_testv1_Greeter_register(callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel)
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
