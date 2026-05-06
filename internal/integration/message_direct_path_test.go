package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestMessageUnaryDirectPathRoutesToCGOMessageServer(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageUnaryDirectPath")
}

func TestMessageClientStreamingDirectPathRoutesToCGOMessageServer(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageClientStreamingDirectPath")
}

func TestMessageServerStreamingDirectPathRoutesToCGOMessageServer(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageServerStreamingDirectPath")
}

func TestMessageBidiStreamingDirectPathRoutesToCGOMessageServer(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBidiStreamingDirectPath")
}

func TestMessageContractMismatchRoutesThroughConverter(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageContractMismatch")
}

func TestNativeContractMismatchRoutesThroughConverter(t *testing.T) {
	runMessageDirectPathFixture(t, "TestNativeContractMismatch")
}

func runMessageDirectPathFixture(t *testing.T, testName string) {
	t.Helper()
	tmp := t.TempDir()
	plugin := newMessageDirectPathTestPlugin(t, "example.com/messagedirect/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeMessageDirectPathGeneratedModule(t, tmp, plugin, "example.com/messagedirect")
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_reset.go"), messageDirectPathResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_callbacks.go"), messageDirectPathFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_test.go"), messageDirectPathFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "^"+testName+"$", "-count=1")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("message direct path fixture %s failed: %v\n%s", testName, err, out)
	}
}

func newMessageDirectPathTestPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	return newMessageDirectPathTestPluginWithParameter(t, "paths=source_relative", goPackage)
}

func newMessageDirectPathTestPluginWithParameter(t *testing.T, parameter, goPackage string) *protogen.Plugin {
	t.Helper()
	emptyFile := protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto)
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(parameter),
		FileToGenerate: []string{"test/v1/message_direct.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			emptyFile,
			{
				Name:       proto.String("test/v1/message_direct.proto"),
				Package:    proto.String("test.v1"),
				Syntax:     proto.String("proto3"),
				Dependency: []string{"google/protobuf/empty.proto"},
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String(goPackage),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{{
					Name: proto.String("Greeter"),
					Method: []*descriptorpb.MethodDescriptorProto{
						messageDirectPathMethod("Unary", false, false),
						messageDirectPathMethod("Upload", true, false),
						messageDirectPathMethod("List", false, true),
						messageDirectPathMethod("Chat", true, true),
					},
				}},
				SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
					Path:            []int32{6, 0},
					Span:            []int32{0, 0, 0},
					LeadingComments: proto.String("@rpccgo: native\n"),
				}}},
			},
		},
	}
	plugin, err := generator.ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func messageDirectPathMethod(name string, clientStreaming, serverStreaming bool) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:            proto.String(name),
		InputType:       proto.String(".google.protobuf.Empty"),
		OutputType:      proto.String(".google.protobuf.Empty"),
		ClientStreaming: proto.Bool(clientStreaming),
		ServerStreaming: proto.Bool(serverStreaming),
	}
}

func writeMessageDirectPathGeneratedModule(t *testing.T, root string, plugin *protogen.Plugin, module string) {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module "+module+"\n\ngo 1.24.4\n\nrequire (\n\tgoogle.golang.org/protobuf v1.36.11\n\trpccgo v0.0.0\n)\n\nreplace rpccgo => "+repoRoot+"\n")
	writeFile(t, filepath.Join(root, "go.sum"), "google.golang.org/protobuf v1.36.11 h1:fV6ZwhNocDyBLK0dj+fg8ektcVegBBuEolpbTQyBNVE=\ngoogle.golang.org/protobuf v1.36.11/go.mod h1:HTf+CrKn2C3g5S8VImy6tdcUvCska2kB7j23XfzDpco=\n")
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".client.cgo.rpccgo.go") ||
			strings.Contains(name, ".message.cgo.rpccgo.go")
		if !include {
			continue
		}
		writeFile(t, filepath.Join(root, name), generated.GetContent())
	}
}

const messageDirectPathResetSource = `package testv1

import rpcruntime "rpccgo/rpcruntime"

func ResetGreeterDispatcherForIntegrationTest() {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterActiveAdapter]{}
}
`

const messageDirectPathFixtureCallbackSource = `package main

/*
#include <stdint.h>

typedef int32_t (*GreeterUnaryCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterUploadCGOMessageClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterUploadCGOMessageClientStreamSendCallback)(int32_t stream, uintptr_t request_ptr, int32_t request_len);
typedef int32_t (*GreeterUploadCGOMessageClientStreamFinishCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterUploadCGOMessageClientStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterListCGOMessageServerStreamStartCallback)(uintptr_t request_ptr, int32_t request_len, int32_t* stream);
typedef int32_t (*GreeterListCGOMessageServerStreamRecvCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterListCGOMessageServerStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterListCGOMessageServerStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamSendCallback)(int32_t stream, uintptr_t request_ptr, int32_t request_len);
typedef int32_t (*GreeterChatCGOMessageBidiStreamRecvCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterChatCGOMessageBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGOMessageServerCallbacks {
	GreeterUnaryCGOMessageUnaryCallback Unary;
	GreeterUploadCGOMessageClientStreamStartCallback UploadStart;
	GreeterUploadCGOMessageClientStreamSendCallback UploadSend;
	GreeterUploadCGOMessageClientStreamFinishCallback UploadFinish;
	GreeterUploadCGOMessageClientStreamCancelCallback UploadCancel;
	GreeterListCGOMessageServerStreamStartCallback ListStart;
	GreeterListCGOMessageServerStreamRecvCallback ListRecv;
	GreeterListCGOMessageServerStreamDoneCallback ListDone;
	GreeterListCGOMessageServerStreamCancelCallback ListCancel;
	GreeterChatCGOMessageBidiStreamStartCallback ChatStart;
	GreeterChatCGOMessageBidiStreamSendCallback ChatSend;
	GreeterChatCGOMessageBidiStreamRecvCallback ChatRecv;
	GreeterChatCGOMessageBidiStreamCloseSendCallback ChatCloseSend;
	GreeterChatCGOMessageBidiStreamDoneCallback ChatDone;
	GreeterChatCGOMessageBidiStreamCancelCallback ChatCancel;
} GreeterCGOMessageServerCallbacks;

static int unaryCalls;
static int unaryError;
static int uploadStarts;
static int uploadSends;
static int uploadFinishes;
static int listStarts;
static int listRecvs;
static int listDones;
static int chatStarts;
static int chatSends;
static int chatRecvs;
static int chatCloseSends;
static int chatDones;

static void resetMessageCounters(void) {
	unaryCalls = 0;
	unaryError = 0;
	uploadStarts = 0;
	uploadSends = 0;
	uploadFinishes = 0;
	listStarts = 0;
	listRecvs = 0;
	listDones = 0;
	chatStarts = 0;
	chatSends = 0;
	chatRecvs = 0;
	chatCloseSends = 0;
	chatDones = 0;
}

static int32_t emptyResponse(uintptr_t* response_ptr, int32_t* response_len) {
	*response_ptr = 0;
	*response_len = 0;
	return 0;
}

static int32_t greeterUnary(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {
	unaryCalls++;
	if (unaryError) {
		return 99999;
	}
	return emptyResponse(response_ptr, response_len);
}

static int32_t greeterUploadStart(int32_t* stream) {
	uploadStarts++;
	*stream = 101;
	return 0;
}

static int32_t greeterUploadSend(int32_t stream, uintptr_t request_ptr, int32_t request_len) {
	if (stream != 101) {
		return 99998;
	}
	uploadSends++;
	return 0;
}

static int32_t greeterUploadFinish(int32_t stream, uintptr_t* response_ptr, int32_t* response_len) {
	uploadFinishes++;
	return emptyResponse(response_ptr, response_len);
}

static int32_t greeterUploadCancel(int32_t stream) { return 0; }

static int32_t greeterListStart(uintptr_t request_ptr, int32_t request_len, int32_t* stream) {
	listStarts++;
	*stream = 202;
	return 0;
}

static int32_t greeterListRecv(int32_t stream, uintptr_t* response_ptr, int32_t* response_len) {
	listRecvs++;
	return emptyResponse(response_ptr, response_len);
}

static int32_t greeterListDone(int32_t stream) {
	listDones++;
	return 0;
}

static int32_t greeterListCancel(int32_t stream) { return 0; }

static int32_t greeterChatStart(int32_t* stream) {
	chatStarts++;
	*stream = 303;
	return 0;
}

static int32_t greeterChatSend(int32_t stream, uintptr_t request_ptr, int32_t request_len) {
	chatSends++;
	return 0;
}

static int32_t greeterChatRecv(int32_t stream, uintptr_t* response_ptr, int32_t* response_len) {
	chatRecvs++;
	return emptyResponse(response_ptr, response_len);
}

static int32_t greeterChatCloseSend(int32_t stream) {
	chatCloseSends++;
	return 0;
}

static int32_t greeterChatDone(int32_t stream) {
	chatDones++;
	return 0;
}

static int32_t greeterChatCancel(int32_t stream) { return 0; }

static GreeterCGOMessageServerCallbacks greeterMessageCallbacks(void) {
	GreeterCGOMessageServerCallbacks callbacks;
	callbacks.Unary = greeterUnary;
	callbacks.UploadStart = greeterUploadStart;
	callbacks.UploadSend = greeterUploadSend;
	callbacks.UploadFinish = greeterUploadFinish;
	callbacks.UploadCancel = greeterUploadCancel;
	callbacks.ListStart = greeterListStart;
	callbacks.ListRecv = greeterListRecv;
	callbacks.ListDone = greeterListDone;
	callbacks.ListCancel = greeterListCancel;
	callbacks.ChatStart = greeterChatStart;
	callbacks.ChatSend = greeterChatSend;
	callbacks.ChatRecv = greeterChatRecv;
	callbacks.ChatCloseSend = greeterChatCloseSend;
	callbacks.ChatDone = greeterChatDone;
	callbacks.ChatCancel = greeterChatCancel;
	return callbacks;
}

static void setUnaryError(int enabled) { unaryError = enabled; }
static int getUnaryCalls(void) { return unaryCalls; }
static int getUploadStarts(void) { return uploadStarts; }
static int getUploadSends(void) { return uploadSends; }
static int getUploadFinishes(void) { return uploadFinishes; }
static int getListStarts(void) { return listStarts; }
static int getListRecvs(void) { return listRecvs; }
static int getListDones(void) { return listDones; }
static int getChatStarts(void) { return chatStarts; }
static int getChatSends(void) { return chatSends; }
static int getChatRecvs(void) { return chatRecvs; }
static int getChatCloseSends(void) { return chatCloseSends; }
static int getChatDones(void) { return chatDones; }
*/
import "C"

import (
	v1 "example.com/messagedirect/test/v1"
)

func registerGreeterMessageCallbacksForIntegration() error {
	v1.ResetGreeterDispatcherForIntegrationTest()
	C.resetMessageCounters()
	C.setUnaryError(0)
	callbacks := C.greeterMessageCallbacks()
	_, err := RegisterGreeterCGOMessageServer(&callbacks)
	return err
}

func setGreeterMessageUnaryErrorForIntegration(enabled bool) {
	if enabled {
		C.setUnaryError(1)
		return
	}
	C.setUnaryError(0)
}

func greeterMessageUnaryCallsForIntegration() int { return int(C.getUnaryCalls()) }
func greeterMessageUploadStartsForIntegration() int { return int(C.getUploadStarts()) }
func greeterMessageUploadSendsForIntegration() int { return int(C.getUploadSends()) }
func greeterMessageUploadFinishesForIntegration() int { return int(C.getUploadFinishes()) }
func greeterMessageListStartsForIntegration() int { return int(C.getListStarts()) }
func greeterMessageListRecvsForIntegration() int { return int(C.getListRecvs()) }
func greeterMessageListDonesForIntegration() int { return int(C.getListDones()) }
func greeterMessageChatStartsForIntegration() int { return int(C.getChatStarts()) }
func greeterMessageChatSendsForIntegration() int { return int(C.getChatSends()) }
func greeterMessageChatRecvsForIntegration() int { return int(C.getChatRecvs()) }
func greeterMessageChatCloseSendsForIntegration() int { return int(C.getChatCloseSends()) }
func greeterMessageChatDonesForIntegration() int { return int(C.getChatDones()) }
`

const messageDirectPathFixtureTestSource = `package main

import (
	context "context"
	strings "strings"
	"testing"
	"unsafe"

	v1 "example.com/messagedirect/test/v1"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	rpcruntime "rpccgo/rpcruntime"
)

func registerMessageServer(t *testing.T) {
	t.Helper()
	if err := registerGreeterMessageCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageCallbacksForIntegration() error = %v", err)
	}
}

func TestMessageUnaryDirectPath(t *testing.T) {
	registerMessageServer(t)
	output := &GreeterMessageOutput{}
	if errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, output); errID != 0 {
		t.Fatalf("CallGreeterUnaryMessageUnary() errID = %d", errID)
	}
	if got := greeterMessageUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("unary callback calls = %d, want 1", got)
	}
	if output.DataPtr != 0 || output.DataLen != 0 {
		t.Fatalf("output = {%d, %d}, want zero empty message", output.DataPtr, output.DataLen)
	}

	invalid := []byte{0xff}
	errID := CallGreeterUnaryMessageUnary(context.Background(), uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)), &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")

	setGreeterMessageUnaryErrorForIntegration(true)
	errID = CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "unknown error id 99999")
}

func TestMessageClientStreamingDirectPath(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterUploadMessageClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterUploadMessageClientStream() errID = %d", errID)
	}
	if errID := SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0); errID != 0 {
		t.Fatalf("SendGreeterUploadMessageClientStream() first errID = %d", errID)
	}
	if errID := SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0); errID != 0 {
		t.Fatalf("SendGreeterUploadMessageClientStream() second errID = %d", errID)
	}
	if errID := FinishGreeterUploadMessageClientStream(context.Background(), handle, &GreeterMessageOutput{}); errID != 0 {
		t.Fatalf("FinishGreeterUploadMessageClientStream() errID = %d", errID)
	}
	if got := greeterMessageUploadStartsForIntegration(); got != 1 {
		t.Fatalf("upload starts = %d, want 1", got)
	}
	if got := greeterMessageUploadSendsForIntegration(); got != 2 {
		t.Fatalf("upload sends = %d, want 2", got)
	}
	if got := greeterMessageUploadFinishesForIntegration(); got != 1 {
		t.Fatalf("upload finishes = %d, want 1", got)
	}
	errID = SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "message client stream handle is invalid")
}

func TestMessageServerStreamingDirectPath(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterListMessageServerStream(context.Background(), 0, 0)
	if errID != 0 {
		t.Fatalf("StartGreeterListMessageServerStream() errID = %d", errID)
	}
	if errID := ReadGreeterListMessageServerStream(context.Background(), handle, &GreeterMessageOutput{}); errID != 0 {
		t.Fatalf("ReadGreeterListMessageServerStream() first errID = %d", errID)
	}
	if errID := ReadGreeterListMessageServerStream(context.Background(), handle, &GreeterMessageOutput{}); errID != 0 {
		t.Fatalf("ReadGreeterListMessageServerStream() second errID = %d", errID)
	}
	if errID := DoneGreeterListMessageServerStream(context.Background(), handle); errID != 0 {
		t.Fatalf("DoneGreeterListMessageServerStream() errID = %d", errID)
	}
	if got := greeterMessageListStartsForIntegration(); got != 1 {
		t.Fatalf("list starts = %d, want 1", got)
	}
	if got := greeterMessageListRecvsForIntegration(); got != 2 {
		t.Fatalf("list recvs = %d, want 2", got)
	}
	if got := greeterMessageListDonesForIntegration(); got != 1 {
		t.Fatalf("list dones = %d, want 1", got)
	}
	errID = DoneGreeterListMessageServerStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "message client stream handle is invalid")
}

func TestMessageBidiStreamingDirectPath(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterChatMessageBidiStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := SendGreeterChatMessageBidiStream(context.Background(), handle, 0, 0); errID != 0 {
		t.Fatalf("SendGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := ReadGreeterChatMessageBidiStream(context.Background(), handle, &GreeterMessageOutput{}); errID != 0 {
		t.Fatalf("ReadGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := CloseSendGreeterChatMessageBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := DoneGreeterChatMessageBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("DoneGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if got := greeterMessageChatStartsForIntegration(); got != 1 {
		t.Fatalf("chat starts = %d, want 1", got)
	}
	if got := greeterMessageChatSendsForIntegration(); got != 1 {
		t.Fatalf("chat sends = %d, want 1", got)
	}
	if got := greeterMessageChatRecvsForIntegration(); got != 1 {
		t.Fatalf("chat recvs = %d, want 1", got)
	}
	if got := greeterMessageChatCloseSendsForIntegration(); got != 1 {
		t.Fatalf("chat close sends = %d, want 1", got)
	}
	if got := greeterMessageChatDonesForIntegration(); got != 1 {
		t.Fatalf("chat dones = %d, want 1", got)
	}
	errID = ReadGreeterChatMessageBidiStream(context.Background(), handle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message client stream handle is invalid")
}

func TestMessageContractMismatch(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGoNativeServer(mismatchNativeServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	assertMessageNoErr(t, errID)

	_, errID = StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	uploadHandle, _ := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, SendGreeterUploadMessageClientStream(context.Background(), uploadHandle, 0, 0))
	assertMessageNoErr(t, FinishGreeterUploadMessageClientStream(context.Background(), uploadHandle, &GreeterMessageOutput{}))

	listHandle, errID := StartGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, ReadGreeterListMessageServerStream(context.Background(), listHandle, &GreeterMessageOutput{}))
	assertMessageNoErr(t, DoneGreeterListMessageServerStream(context.Background(), listHandle))

	chatHandle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, SendGreeterChatMessageBidiStream(context.Background(), chatHandle, 0, 0))
	assertMessageNoErr(t, ReadGreeterChatMessageBidiStream(context.Background(), chatHandle, &GreeterMessageOutput{}))
	assertMessageNoErr(t, CloseSendGreeterChatMessageBidiStream(context.Background(), chatHandle))
	assertMessageNoErr(t, DoneGreeterChatMessageBidiStream(context.Background(), chatHandle))
}

func TestNativeContractMismatch(t *testing.T) {
	registerMessageServer(t)
	errID := CallGreeterUnaryNativeUnary(context.Background(), &GreeterUnaryNativeUnaryInput{}, &GreeterUnaryNativeUnaryOutput{})
	assertMessageNoErr(t, errID)
}

type mismatchNativeServer struct{}

func (mismatchNativeServer) Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (mismatchNativeServer) Upload(context.Context) (v1.GreeterUploadNativeClientStream, error) {
	return mismatchNativeClientStream{}, nil
}

func (mismatchNativeServer) List(context.Context, *emptypb.Empty) (v1.GreeterListNativeServerStream, error) {
	return mismatchNativeServerStream{}, nil
}

func (mismatchNativeServer) Chat(context.Context) (v1.GreeterChatNativeBidiStream, error) {
	return mismatchNativeBidiStream{}, nil
}

type mismatchNativeClientStream struct{}

func (mismatchNativeClientStream) Send(context.Context, *emptypb.Empty) error { return nil }
func (mismatchNativeClientStream) Finish(context.Context) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (mismatchNativeClientStream) Cancel(context.Context) error { return nil }

type mismatchNativeServerStream struct{}

func (mismatchNativeServerStream) Recv(context.Context) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (mismatchNativeServerStream) Cancel(context.Context) error { return nil }

type mismatchNativeBidiStream struct{}

func (mismatchNativeBidiStream) Send(context.Context, *emptypb.Empty) error { return nil }
func (mismatchNativeBidiStream) Recv(context.Context) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
func (mismatchNativeBidiStream) CloseSend(context.Context) error { return nil }
func (mismatchNativeBidiStream) Cancel(context.Context) error { return nil }

func assertMessageErrContains(t *testing.T, errID int32, wants ...string) {
	t.Helper()
	if errID == 0 {
		t.Fatalf("errID = 0, want error containing %q", wants)
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok {
		t.Fatalf("error text = %q, ok=%v, want contains %q", text, ok, wants)
	}
	for _, want := range wants {
		if !strings.Contains(string(text), want) {
			t.Fatalf("error text = %q, want contains %q", text, want)
		}
	}
}

func assertMessageNoErr(t *testing.T, errID int32) {
	t.Helper()
	if errID != 0 {
		text, _, _ := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("errID = %d, error text = %q, want no error", errID, text)
	}
}
`
