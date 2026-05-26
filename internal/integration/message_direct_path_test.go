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

func TestMessageClientToCGONativeRoutesThroughConverter(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageClientToCGONative")
}

func TestNativeContractMismatchRoutesThroughConverter(t *testing.T) {
	runMessageDirectPathFixture(t, "TestNativeContractMismatch")
}

func TestMessageKnownErrorTextIsConsumedOnce(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageKnownErrorTextIsConsumedOnce")
}

func TestMessageUnknownErrorIDReturnsErrorText(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageUnknownErrorIDReturnsErrorText")
}

func TestMessageBytesRejectInvalidUnaryRequest(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBytesRejectInvalidUnaryRequest")
}

func TestMessageBytesRejectInvalidClientStreamSend(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBytesRejectInvalidClientStreamSend")
}

func TestMessageBytesRejectInvalidServerStreamStart(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBytesRejectInvalidServerStreamStart")
}

func TestMessageBytesRejectInvalidBidiSend(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBytesRejectInvalidBidiSend")
}

func TestMessageBytesRejectInvalidCallbackResponse(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBytesRejectInvalidCallbackResponse")
}

func TestMessageClientStreamRejectsOperationsAfterFinish(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageClientStreamRejectsOperationsAfterFinish")
}

func TestMessageServerStreamRejectsReadAfterDone(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageServerStreamRejectsReadAfterDone")
}

func TestMessageBidiRejectsSendAfterCloseSendAndReadAfterCancel(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBidiRejectsSendAfterCloseSendAndReadAfterCancel")
}

func TestMessageBidiCloseSendErrorKeepsSendSideOpen(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageBidiCloseSendErrorKeepsSendSideOpen")
}

func TestMessageStreamCancelTwiceCallsDownstreamOnce(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageStreamCancelTwiceCallsDownstreamOnce")
}

func TestMessageStreamInvalidHandleReturnsError(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageStreamInvalidHandleReturnsError")
}

func TestMessageStreamStartCapturesActiveServerSnapshot(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageStreamStartCapturesActiveServerSnapshot")
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
	return messageContractMethod(name, ".google.protobuf.Empty", ".google.protobuf.Empty", clientStreaming, serverStreaming)
}

func messageContractMethod(name, inputType, outputType string, clientStreaming, serverStreaming bool) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:            proto.String(name),
		InputType:       proto.String(inputType),
		OutputType:      proto.String(outputType),
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
	writeFile(t, filepath.Join(root, "go.mod"), "module "+module+"\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\tgoogle.golang.org/grpc v1.79.3\n\tgoogle.golang.org/protobuf v1.36.11\n\trpccgo v0.0.0\n)\n\nreplace rpccgo => "+repoRoot+"\n")
	writeFile(t, filepath.Join(root, "go.sum"), "google.golang.org/protobuf v1.36.11 h1:fV6ZwhNocDyBLK0dj+fg8ektcVegBBuEolpbTQyBNVE=\ngoogle.golang.org/protobuf v1.36.11/go.mod h1:HTf+CrKn2C3g5S8VImy6tdcUvCska2kB7j23XfzDpco=\n")
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".server.connect.rpccgo.go") ||
			strings.Contains(name, ".server.grpc.rpccgo.go") ||
			strings.Contains(name, ".server.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go") ||
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

typedef int32_t (*GreeterUnaryCGONativeUnaryCallback)(void);
typedef int32_t (*GreeterUploadCGONativeClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamSendCallback)(int32_t stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterListCGONativeServerStreamRecvCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamRecvCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamDoneCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamCancelCallback)(int32_t stream);

extern int32_t greeterMessageStreamEOFErrorIDForIntegration(void);
extern int32_t greeterMessageStoredErrorIDForIntegration(void);

typedef struct GreeterCGONativeServerCallbacks {
	GreeterUnaryCGONativeUnaryCallback Unary;
	GreeterUploadCGONativeClientStreamStartCallback UploadStart;
	GreeterUploadCGONativeClientStreamSendCallback UploadSend;
	GreeterUploadCGONativeClientStreamFinishCallback UploadFinish;
	GreeterUploadCGONativeClientStreamCancelCallback UploadCancel;
	GreeterListCGONativeServerStreamStartCallback ListStart;
	GreeterListCGONativeServerStreamRecvCallback ListRecv;
	GreeterListCGONativeServerStreamDoneCallback ListDone;
	GreeterListCGONativeServerStreamCancelCallback ListCancel;
	GreeterChatCGONativeBidiStreamStartCallback ChatStart;
	GreeterChatCGONativeBidiStreamSendCallback ChatSend;
	GreeterChatCGONativeBidiStreamRecvCallback ChatRecv;
	GreeterChatCGONativeBidiStreamCloseSendCallback ChatCloseSend;
	GreeterChatCGONativeBidiStreamDoneCallback ChatDone;
	GreeterChatCGONativeBidiStreamCancelCallback ChatCancel;
} GreeterCGONativeServerCallbacks;

static int unaryCalls;
static int unaryError;
static int unaryStoredError;
static int nativeUnaryError;
static int chatCloseSendFailuresRemaining;
static int uploadStarts;
static int uploadSends;
static int uploadFinishes;
static int uploadCancels;
static int listStarts;
static int listRecvs;
static int listDones;
static int listCancels;
static int chatStarts;
static int chatSends;
static int chatRecvs;
static int chatCloseSends;
static int chatDones;
static int chatCancels;
static int nativeUnaryCalls;
static int nativeUploadStarts;
static int nativeUploadSends;
static int nativeUploadFinishes;
static int nativeUploadCancels;
static int nativeListStarts;
static int nativeListRecvs;
static int nativeListDones;
static int nativeListCancels;
static int nativeChatStarts;
static int nativeChatSends;
static int nativeChatRecvs;
static int nativeChatCloseSends;
static int nativeChatDones;
static int nativeChatCancels;
static int messageStreamEOFMode;
static int invalidMessageResponse;

static void resetMessageCounters(void) {
	unaryCalls = 0;
	unaryError = 0;
	unaryStoredError = 0;
	nativeUnaryError = 0;
	chatCloseSendFailuresRemaining = 0;
	uploadStarts = 0;
	uploadSends = 0;
	uploadFinishes = 0;
	uploadCancels = 0;
	listStarts = 0;
	listRecvs = 0;
	listDones = 0;
	listCancels = 0;
	chatStarts = 0;
	chatSends = 0;
	chatRecvs = 0;
	chatCloseSends = 0;
	chatDones = 0;
	chatCancels = 0;
	nativeUnaryCalls = 0;
	nativeUploadStarts = 0;
	nativeUploadSends = 0;
	nativeUploadFinishes = 0;
	nativeUploadCancels = 0;
	nativeListStarts = 0;
	nativeListRecvs = 0;
	nativeListDones = 0;
	nativeListCancels = 0;
	nativeChatStarts = 0;
	nativeChatSends = 0;
	nativeChatRecvs = 0;
	nativeChatCloseSends = 0;
	nativeChatDones = 0;
	nativeChatCancels = 0;
	messageStreamEOFMode = 0;
	invalidMessageResponse = 0;
}

static int32_t emptyResponse(uintptr_t* response_ptr, int32_t* response_len) {
	static unsigned char invalid_response[] = {0xff};
	if (invalidMessageResponse) {
		*response_ptr = (uintptr_t)&invalid_response[0];
		*response_len = 1;
		return 0;
	}
	*response_ptr = 0;
	*response_len = 0;
	return 0;
}

static int32_t greeterUnary(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {
	unaryCalls++;
	if (unaryStoredError) {
		return greeterMessageStoredErrorIDForIntegration();
	}
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

static int32_t greeterUploadCancel(int32_t stream) {
	uploadCancels++;
	return 0;
}

static int32_t greeterListStart(uintptr_t request_ptr, int32_t request_len, int32_t* stream) {
	listStarts++;
	*stream = 202;
	return 0;
}

static int32_t greeterListRecv(int32_t stream, uintptr_t* response_ptr, int32_t* response_len) {
	listRecvs++;
	if (messageStreamEOFMode && listRecvs > 1) {
		return greeterMessageStreamEOFErrorIDForIntegration();
	}
	return emptyResponse(response_ptr, response_len);
}

static int32_t greeterListDone(int32_t stream) {
	listDones++;
	return 0;
}

static int32_t greeterListCancel(int32_t stream) {
	listCancels++;
	return 0;
}

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
	if (messageStreamEOFMode && chatRecvs > 1) {
		return greeterMessageStreamEOFErrorIDForIntegration();
	}
	return emptyResponse(response_ptr, response_len);
}

static int32_t greeterChatCloseSend(int32_t stream) {
	chatCloseSends++;
	if (chatCloseSendFailuresRemaining > 0) {
		chatCloseSendFailuresRemaining--;
		return 99996;
	}
	return 0;
}

static int32_t greeterChatDone(int32_t stream) {
	chatDones++;
	return 0;
}

static int32_t greeterChatCancel(int32_t stream) {
	chatCancels++;
	return 0;
}

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

static int32_t nativeGreeterUnary(void) {
	nativeUnaryCalls++;
	if (nativeUnaryError) {
		return 99997;
	}
	return 0;
}

static int32_t nativeGreeterUploadStart(int32_t* stream) {
	nativeUploadStarts++;
	*stream = 404;
	return 0;
}

static int32_t nativeGreeterUploadSend(int32_t stream) {
	nativeUploadSends++;
	return 0;
}

static int32_t nativeGreeterUploadFinish(int32_t stream) {
	nativeUploadFinishes++;
	return 0;
}

static int32_t nativeGreeterUploadCancel(int32_t stream) {
	nativeUploadCancels++;
	return 0;
}

static int32_t nativeGreeterListStart(int32_t* stream) {
	nativeListStarts++;
	*stream = 505;
	return 0;
}

static int32_t nativeGreeterListRecv(int32_t stream) {
	nativeListRecvs++;
	return 0;
}

static int32_t nativeGreeterListDone(int32_t stream) {
	nativeListDones++;
	return 0;
}

static int32_t nativeGreeterListCancel(int32_t stream) {
	nativeListCancels++;
	return 0;
}

static int32_t nativeGreeterChatStart(int32_t* stream) {
	nativeChatStarts++;
	*stream = 606;
	return 0;
}

static int32_t nativeGreeterChatSend(int32_t stream) {
	nativeChatSends++;
	return 0;
}

static int32_t nativeGreeterChatRecv(int32_t stream) {
	nativeChatRecvs++;
	return 0;
}

static int32_t nativeGreeterChatCloseSend(int32_t stream) {
	nativeChatCloseSends++;
	return 0;
}

static int32_t nativeGreeterChatDone(int32_t stream) {
	nativeChatDones++;
	return 0;
}

static int32_t nativeGreeterChatCancel(int32_t stream) {
	nativeChatCancels++;
	return 0;
}

static GreeterCGONativeServerCallbacks greeterNativeCallbacks(void) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.Unary = nativeGreeterUnary;
	callbacks.UploadStart = nativeGreeterUploadStart;
	callbacks.UploadSend = nativeGreeterUploadSend;
	callbacks.UploadFinish = nativeGreeterUploadFinish;
	callbacks.UploadCancel = nativeGreeterUploadCancel;
	callbacks.ListStart = nativeGreeterListStart;
	callbacks.ListRecv = nativeGreeterListRecv;
	callbacks.ListDone = nativeGreeterListDone;
	callbacks.ListCancel = nativeGreeterListCancel;
	callbacks.ChatStart = nativeGreeterChatStart;
	callbacks.ChatSend = nativeGreeterChatSend;
	callbacks.ChatRecv = nativeGreeterChatRecv;
	callbacks.ChatCloseSend = nativeGreeterChatCloseSend;
	callbacks.ChatDone = nativeGreeterChatDone;
	callbacks.ChatCancel = nativeGreeterChatCancel;
	return callbacks;
}

static void setUnaryError(int enabled) { unaryError = enabled; }
static void setUnaryStoredError(int enabled) { unaryStoredError = enabled; }
static void setNativeUnaryError(int enabled) { nativeUnaryError = enabled; }
static void setChatCloseSendFailuresRemaining(int remaining) { chatCloseSendFailuresRemaining = remaining; }
static void setMessageStreamEOFMode(int enabled) { messageStreamEOFMode = enabled; }
static void setInvalidMessageResponse(int enabled) { invalidMessageResponse = enabled; }
static int getUnaryCalls(void) { return unaryCalls; }
static int getUploadStarts(void) { return uploadStarts; }
static int getUploadSends(void) { return uploadSends; }
static int getUploadFinishes(void) { return uploadFinishes; }
static int getUploadCancels(void) { return uploadCancels; }
static int getListStarts(void) { return listStarts; }
static int getListRecvs(void) { return listRecvs; }
static int getListDones(void) { return listDones; }
static int getListCancels(void) { return listCancels; }
static int getChatStarts(void) { return chatStarts; }
static int getChatSends(void) { return chatSends; }
static int getChatRecvs(void) { return chatRecvs; }
static int getChatCloseSends(void) { return chatCloseSends; }
static int getChatDones(void) { return chatDones; }
static int getChatCancels(void) { return chatCancels; }
static int getNativeUnaryCalls(void) { return nativeUnaryCalls; }
static int getNativeUploadStarts(void) { return nativeUploadStarts; }
static int getNativeUploadSends(void) { return nativeUploadSends; }
static int getNativeUploadFinishes(void) { return nativeUploadFinishes; }
static int getNativeUploadCancels(void) { return nativeUploadCancels; }
static int getNativeListStarts(void) { return nativeListStarts; }
static int getNativeListRecvs(void) { return nativeListRecvs; }
static int getNativeListDones(void) { return nativeListDones; }
static int getNativeListCancels(void) { return nativeListCancels; }
static int getNativeChatStarts(void) { return nativeChatStarts; }
static int getNativeChatSends(void) { return nativeChatSends; }
static int getNativeChatRecvs(void) { return nativeChatRecvs; }
static int getNativeChatCloseSends(void) { return nativeChatCloseSends; }
static int getNativeChatDones(void) { return nativeChatDones; }
static int getNativeChatCancels(void) { return nativeChatCancels; }
*/
import "C"

import (
	errors "errors"

	v1 "example.com/messagedirect/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

//export greeterMessageStreamEOFErrorIDForIntegration
func greeterMessageStreamEOFErrorIDForIntegration() C.int32_t {
	return C.int32_t(GreeterCGOMessageStreamEOFErrorID())
}

//export greeterMessageStoredErrorIDForIntegration
func greeterMessageStoredErrorIDForIntegration() C.int32_t {
	return C.int32_t(rpcruntime.StoreError(errors.New("expected callback error")))
}

func registerGreeterMessageCallbacksForIntegration() error {
	v1.ResetGreeterDispatcherForIntegrationTest()
	C.resetMessageCounters()
	C.setUnaryError(0)
	C.setUnaryStoredError(0)
	C.setMessageStreamEOFMode(0)
	C.setInvalidMessageResponse(0)
	callbacks := C.greeterMessageCallbacks()
	return registerGreeterMessageCallbacks(callbacks)
}

func registerGreeterMessageCallbacksWithoutResetForIntegration() error {
	C.setUnaryError(0)
	C.setUnaryStoredError(0)
	C.setMessageStreamEOFMode(0)
	C.setInvalidMessageResponse(0)
	callbacks := C.greeterMessageCallbacks()
	return registerGreeterMessageCallbacks(callbacks)
}

func registerGreeterNativeCallbacksForIntegration() error {
	v1.ResetGreeterDispatcherForIntegrationTest()
	C.resetMessageCounters()
	C.setNativeUnaryError(0)
	callbacks := C.greeterNativeCallbacks()
	return registerGreeterNativeCallbacks(callbacks)
}

func registerGreeterMessageCallbacks(callbacks C.GreeterCGOMessageServerCallbacks) error {
	for _, errID := range []C.int32_t{
		rpccgo_msg_testv1_Greeter_Unary_register(callbacks.Unary),
		rpccgo_msg_testv1_Greeter_Upload_register(callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel),
		rpccgo_msg_testv1_Greeter_List_register(callbacks.ListStart, callbacks.ListRecv, callbacks.ListDone, callbacks.ListCancel),
		rpccgo_msg_testv1_Greeter_Chat_register(callbacks.ChatStart, callbacks.ChatSend, callbacks.ChatRecv, callbacks.ChatCloseSend, callbacks.ChatDone, callbacks.ChatCancel),
	} {
		if errID != 0 {
			return cgoFixtureStoredError(errID)
		}
	}
	return nil
}

func registerGreeterNativeCallbacks(callbacks C.GreeterCGONativeServerCallbacks) error {
	for _, errID := range []C.int32_t{
		rpccgo_native_testv1_Greeter_Unary_register(callbacks.Unary),
		rpccgo_native_testv1_Greeter_Upload_register(callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel),
		rpccgo_native_testv1_Greeter_List_register(callbacks.ListStart, callbacks.ListRecv, callbacks.ListDone, callbacks.ListCancel),
		rpccgo_native_testv1_Greeter_Chat_register(callbacks.ChatStart, callbacks.ChatSend, callbacks.ChatRecv, callbacks.ChatCloseSend, callbacks.ChatDone, callbacks.ChatCancel),
	} {
		if errID != 0 {
			return cgoFixtureStoredError(errID)
		}
	}
	return nil
}

func cgoFixtureStoredError(errID C.int32_t) error {
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok {
		return errors.New("missing cgo fixture error")
	}
	if ptr != 0 {
		rpcruntime.Release(ptr)
	}
	return errors.New(string(text))
}

func setGreeterNativeUnaryErrorForIntegration(enabled bool) {
	if enabled {
		C.setNativeUnaryError(1)
		return
	}
	C.setNativeUnaryError(0)
}

func setGreeterMessageUnaryErrorForIntegration(enabled bool) {
	if enabled {
		C.setUnaryError(1)
		return
	}
	C.setUnaryError(0)
}

func setGreeterMessageUnaryStoredErrorForIntegration(enabled bool) {
	if enabled {
		C.setUnaryStoredError(1)
		return
	}
	C.setUnaryStoredError(0)
}

func setGreeterMessageChatCloseSendFailuresForIntegration(remaining int) {
	C.setChatCloseSendFailuresRemaining(C.int(remaining))
}

func setGreeterMessageStreamEOFModeForIntegration(enabled bool) {
	if enabled {
		C.setMessageStreamEOFMode(1)
		return
	}
	C.setMessageStreamEOFMode(0)
}

func setGreeterMessageInvalidResponseForIntegration(enabled bool) {
	if enabled {
		C.setInvalidMessageResponse(1)
		return
	}
	C.setInvalidMessageResponse(0)
}

func greeterMessageUnaryCallsForIntegration() int { return int(C.getUnaryCalls()) }
func greeterMessageUploadStartsForIntegration() int { return int(C.getUploadStarts()) }
func greeterMessageUploadSendsForIntegration() int { return int(C.getUploadSends()) }
func greeterMessageUploadFinishesForIntegration() int { return int(C.getUploadFinishes()) }
func greeterMessageUploadCancelsForIntegration() int { return int(C.getUploadCancels()) }
func greeterMessageListStartsForIntegration() int { return int(C.getListStarts()) }
func greeterMessageListRecvsForIntegration() int { return int(C.getListRecvs()) }
func greeterMessageListDonesForIntegration() int { return int(C.getListDones()) }
func greeterMessageListCancelsForIntegration() int { return int(C.getListCancels()) }
func greeterMessageChatStartsForIntegration() int { return int(C.getChatStarts()) }
func greeterMessageChatSendsForIntegration() int { return int(C.getChatSends()) }
func greeterMessageChatRecvsForIntegration() int { return int(C.getChatRecvs()) }
func greeterMessageChatCloseSendsForIntegration() int { return int(C.getChatCloseSends()) }
func greeterMessageChatDonesForIntegration() int { return int(C.getChatDones()) }
func greeterMessageChatCancelsForIntegration() int { return int(C.getChatCancels()) }
func greeterNativeUnaryCallsForIntegration() int { return int(C.getNativeUnaryCalls()) }
func greeterNativeUploadStartsForIntegration() int { return int(C.getNativeUploadStarts()) }
func greeterNativeUploadSendsForIntegration() int { return int(C.getNativeUploadSends()) }
func greeterNativeUploadFinishesForIntegration() int { return int(C.getNativeUploadFinishes()) }
func greeterNativeUploadCancelsForIntegration() int { return int(C.getNativeUploadCancels()) }
func greeterNativeListStartsForIntegration() int { return int(C.getNativeListStarts()) }
func greeterNativeListRecvsForIntegration() int { return int(C.getNativeListRecvs()) }
func greeterNativeListDonesForIntegration() int { return int(C.getNativeListDones()) }
func greeterNativeListCancelsForIntegration() int { return int(C.getNativeListCancels()) }
func greeterNativeChatStartsForIntegration() int { return int(C.getNativeChatStarts()) }
func greeterNativeChatSendsForIntegration() int { return int(C.getNativeChatSends()) }
func greeterNativeChatRecvsForIntegration() int { return int(C.getNativeChatRecvs()) }
func greeterNativeChatCloseSendsForIntegration() int { return int(C.getNativeChatCloseSends()) }
func greeterNativeChatDonesForIntegration() int { return int(C.getNativeChatDones()) }
func greeterNativeChatCancelsForIntegration() int { return int(C.getNativeChatCancels()) }
`

const messageDirectPathFixtureTestSource = `package main

import (
	context "context"
	io "io"
	strings "strings"
	"testing"
	"unsafe"

	v1 "example.com/messagedirect/test/v1"
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

func TestMessageKnownErrorTextIsConsumedOnce(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageUnaryStoredErrorForIntegration(true)

	errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	if errID == 0 {
		t.Fatal("CallGreeterUnaryMessageUnary() errID = 0, want stored callback error")
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "expected callback error") {
		t.Fatalf("first TakeErrorText = (%q, %d, %v), want expected callback error", text, ptr, ok)
	}
	rpcruntime.Release(ptr)
	secondText, secondPtr, secondOK := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if secondOK || len(secondText) != 0 || secondPtr != 0 {
		t.Fatalf("second TakeErrorText = (%q, %d, %v), want consumed record", secondText, secondPtr, secondOK)
	}
}

func TestMessageUnknownErrorIDReturnsErrorText(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageUnaryErrorForIntegration(true)

	errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "unknown error id 99999")
}

func TestMessageBytesRejectInvalidUnaryRequest(t *testing.T) {
	registerMessageServer(t)
	invalid := []byte{0xff}
	errID := CallGreeterUnaryMessageUnary(context.Background(), uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)), &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")
	if got := greeterMessageUnaryCallsForIntegration(); got != 0 {
		t.Fatalf("unary callback calls = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidClientStreamSend(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	t.Cleanup(func() {
		_ = CancelGreeterUploadMessageClientStream(context.Background(), handle)
	})
	invalid := []byte{0xff}
	errID = SendGreeterUploadMessageClientStream(context.Background(), handle, uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)))
	assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")
	if got := greeterMessageUploadSendsForIntegration(); got != 0 {
		t.Fatalf("upload sends = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidServerStreamStart(t *testing.T) {
	registerMessageServer(t)
	invalid := []byte{0xff}
	handle, errID := StartGreeterListMessageServerStream(context.Background(), uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)))
	assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")
	if handle != 0 {
		if cancelErrID := CancelGreeterListMessageServerStream(context.Background(), handle); cancelErrID == 0 {
			t.Fatalf("StartGreeterListMessageServerStream() returned usable handle %d after invalid request bytes", handle)
		}
	}
	if got := greeterMessageListStartsForIntegration(); got != 0 {
		t.Fatalf("list starts = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidBidiSend(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	t.Cleanup(func() {
		_ = CancelGreeterChatMessageBidiStream(context.Background(), handle)
	})
	invalid := []byte{0xff}
	errID = SendGreeterChatMessageBidiStream(context.Background(), handle, uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)))
	assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")
	if got := greeterMessageChatSendsForIntegration(); got != 0 {
		t.Fatalf("chat sends = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidCallbackResponse(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageInvalidResponseForIntegration(true)

	errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message response protobuf unmarshal failed")

	uploadHandle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, SendGreeterUploadMessageClientStream(context.Background(), uploadHandle, 0, 0))
	errID = FinishGreeterUploadMessageClientStream(context.Background(), uploadHandle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message response protobuf unmarshal failed")

	listHandle, errID := StartGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	errID = ReadGreeterListMessageServerStream(context.Background(), listHandle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message response protobuf unmarshal failed")
	assertMessageNoErr(t, DoneGreeterListMessageServerStream(context.Background(), listHandle))

	chatHandle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, SendGreeterChatMessageBidiStream(context.Background(), chatHandle, 0, 0))
	errID = ReadGreeterChatMessageBidiStream(context.Background(), chatHandle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "message response protobuf unmarshal failed")
	assertMessageNoErr(t, DoneGreeterChatMessageBidiStream(context.Background(), chatHandle))
}

func TestMessageClientStreamRejectsOperationsAfterFinish(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0))
	output := &GreeterMessageOutput{}
	assertMessageNoErr(t, FinishGreeterUploadMessageClientStream(context.Background(), handle, output))
	errID = FinishGreeterUploadMessageClientStream(context.Background(), handle, output)
	assertMessageErrContains(t, errID, "stream handle is invalid")
	errID = SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageUploadFinishesForIntegration(); got != 1 {
		t.Fatalf("upload finishes = %d, want 1", got)
	}
}

func TestMessageServerStreamRejectsReadAfterDone(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageStreamEOFModeForIntegration(true)
	handle, errID := StartGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, ReadGreeterListMessageServerStream(context.Background(), handle, &GreeterMessageOutput{}))
	errID = ReadGreeterListMessageServerStream(context.Background(), handle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "EOF")
	assertMessageNoErr(t, DoneGreeterListMessageServerStream(context.Background(), handle))
	errID = ReadGreeterListMessageServerStream(context.Background(), handle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageListDonesForIntegration(); got != 1 {
		t.Fatalf("list dones = %d, want 1", got)
	}
}

func TestMessageBidiRejectsSendAfterCloseSendAndReadAfterCancel(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, CloseSendGreeterChatMessageBidiStream(context.Background(), handle))
	errID = SendGreeterChatMessageBidiStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "stream send side is closed")
	assertMessageNoErr(t, CancelGreeterChatMessageBidiStream(context.Background(), handle))
	errID = ReadGreeterChatMessageBidiStream(context.Background(), handle, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageChatCloseSendsForIntegration(); got != 1 {
		t.Fatalf("chat close sends = %d, want 1", got)
	}
	if got := greeterMessageChatCancelsForIntegration(); got != 1 {
		t.Fatalf("chat cancels = %d, want 1", got)
	}
}

func TestMessageBidiCloseSendErrorClosesSendSide(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageChatCloseSendFailuresForIntegration(1)
	handle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)

	errID = CloseSendGreeterChatMessageBidiStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "unknown error id 99996")
	errID = SendGreeterChatMessageBidiStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "stream send side is closed")
	errID = CloseSendGreeterChatMessageBidiStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "stream send side is closed")

	if got := greeterMessageChatCloseSendsForIntegration(); got != 1 {
		t.Fatalf("chat close sends = %d, want 1 after failed close send", got)
	}
	if got := greeterMessageChatSendsForIntegration(); got != 0 {
		t.Fatalf("chat sends = %d, want 0 after failed close send", got)
	}
}

func TestMessageStreamCancelTwiceCallsDownstreamOnce(t *testing.T) {
	registerMessageServer(t)
	handle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, CancelGreeterUploadMessageClientStream(context.Background(), handle))
	errID = CancelGreeterUploadMessageClientStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageUploadCancelsForIntegration(); got != 1 {
		t.Fatalf("upload cancels = %d, want 1", got)
	}
}

func TestMessageStreamInvalidHandleReturnsError(t *testing.T) {
	registerMessageServer(t)
	const invalid int32 = 999999
	assertMessageErrContains(t, SendGreeterUploadMessageClientStream(context.Background(), invalid, 0, 0), "stream handle is invalid")
	assertMessageErrContains(t, FinishGreeterUploadMessageClientStream(context.Background(), invalid, &GreeterMessageOutput{}), "stream handle is invalid")
	assertMessageErrContains(t, ReadGreeterListMessageServerStream(context.Background(), invalid, &GreeterMessageOutput{}), "stream handle is invalid")
	assertMessageErrContains(t, DoneGreeterListMessageServerStream(context.Background(), invalid), "stream handle is invalid")
	assertMessageErrContains(t, CloseSendGreeterChatMessageBidiStream(context.Background(), invalid), "stream handle is invalid")
	assertMessageErrContains(t, CancelGreeterChatMessageBidiStream(context.Background(), invalid), "stream handle is invalid")
}

func TestMessageStreamStartCapturesActiveServerSnapshot(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}
	chatHandle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	listHandle, errID := StartGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	if err := registerGreeterMessageCallbacksWithoutResetForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageCallbacksWithoutResetForIntegration() error = %v", err)
	}

	assertMessageNoErr(t, SendGreeterChatMessageBidiStream(context.Background(), chatHandle, 0, 0))
	assertMessageNoErr(t, ReadGreeterChatMessageBidiStream(context.Background(), chatHandle, &GreeterMessageOutput{}))
	assertMessageNoErr(t, DoneGreeterChatMessageBidiStream(context.Background(), chatHandle))
	assertMessageNoErr(t, ReadGreeterListMessageServerStream(context.Background(), listHandle, &GreeterMessageOutput{}))
	assertMessageNoErr(t, DoneGreeterListMessageServerStream(context.Background(), listHandle))
	if got := greeterNativeChatSendsForIntegration(); got != 1 {
		t.Fatalf("native chat sends = %d, want 1", got)
	}
	if got := greeterNativeChatRecvsForIntegration(); got != 1 {
		t.Fatalf("native chat recvs = %d, want 1", got)
	}
	if got := greeterNativeListRecvsForIntegration(); got != 1 {
		t.Fatalf("native list recvs = %d, want 1", got)
	}
	if got := greeterMessageChatSendsForIntegration(); got != 0 {
		t.Fatalf("message chat sends = %d, want 0 for existing native snapshot", got)
	}
	if got := greeterMessageListRecvsForIntegration(); got != 0 {
		t.Fatalf("message list recvs = %d, want 0 for existing native snapshot", got)
	}
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
	assertMessageErrContains(t, errID, "stream handle is invalid")
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
	assertMessageErrContains(t, errID, "stream handle is invalid")
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
	assertMessageErrContains(t, errID, "stream handle is invalid")
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

func TestMessageClientToCGONative(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}

	errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	assertMessageNoErr(t, errID)

	uploadHandle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
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

	if got := greeterNativeUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("native unary calls = %d, want 1", got)
	}
	if got := greeterNativeUploadStartsForIntegration(); got != 1 {
		t.Fatalf("native upload starts = %d, want 1", got)
	}
	if got := greeterNativeUploadSendsForIntegration(); got != 1 {
		t.Fatalf("native upload sends = %d, want 1", got)
	}
	if got := greeterNativeUploadFinishesForIntegration(); got != 1 {
		t.Fatalf("native upload finishes = %d, want 1", got)
	}
	if got := greeterNativeListStartsForIntegration(); got != 1 {
		t.Fatalf("native list starts = %d, want 1", got)
	}
	if got := greeterNativeListRecvsForIntegration(); got != 1 {
		t.Fatalf("native list recvs = %d, want 1", got)
	}
	if got := greeterNativeListDonesForIntegration(); got != 1 {
		t.Fatalf("native list dones = %d, want 1", got)
	}
	if got := greeterNativeChatStartsForIntegration(); got != 1 {
		t.Fatalf("native chat starts = %d, want 1", got)
	}
	if got := greeterNativeChatSendsForIntegration(); got != 1 {
		t.Fatalf("native chat sends = %d, want 1", got)
	}
	if got := greeterNativeChatRecvsForIntegration(); got != 1 {
		t.Fatalf("native chat recvs = %d, want 1", got)
	}
	if got := greeterNativeChatCloseSendsForIntegration(); got != 1 {
		t.Fatalf("native chat close sends = %d, want 1", got)
	}
	if got := greeterNativeChatDonesForIntegration(); got != 1 {
		t.Fatalf("native chat dones = %d, want 1", got)
	}
}

func TestConverterErrorDoesNotCallCGONativeServer(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}

	badRequest := []byte{0xff}
	errID := CallGreeterUnaryMessageUnary(context.Background(), uintptr(unsafe.Pointer(&badRequest[0])), int32(len(badRequest)), &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "protobuf unmarshal failed")
	if got := greeterNativeUnaryCallsForIntegration(); got != 0 {
		t.Fatalf("native unary calls = %d, want 0 after converter error", got)
	}
}

func TestDownstreamCGONativeErrorIsNotCoveredByConverter(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}
	setGreeterNativeUnaryErrorForIntegration(true)

	errID := CallGreeterUnaryMessageUnary(context.Background(), 0, 0, &GreeterMessageOutput{})
	assertMessageErrContains(t, errID, "unknown error id 99997")
	if got := greeterNativeUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("native unary calls = %d, want 1", got)
	}
}

func TestConverterStreamStartCapturesCGONativeSnapshot(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}
	handle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)

	if err := registerGreeterMessageCallbacksWithoutResetForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageCallbacksWithoutResetForIntegration() error = %v", err)
	}
	assertMessageNoErr(t, SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0))
	assertMessageNoErr(t, FinishGreeterUploadMessageClientStream(context.Background(), handle, &GreeterMessageOutput{}))
	if got := greeterNativeUploadSendsForIntegration(); got != 1 {
		t.Fatalf("native upload sends = %d, want 1", got)
	}
	if got := greeterMessageUploadSendsForIntegration(); got != 0 {
		t.Fatalf("message upload sends = %d, want 0 for existing native snapshot", got)
	}
}

func TestConverterCancelPropagatesToCGONativeAndFinalizesHandle(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}
	handle, errID := StartGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, CancelGreeterUploadMessageClientStream(context.Background(), handle))
	if got := greeterNativeUploadCancelsForIntegration(); got != 1 {
		t.Fatalf("native upload cancels = %d, want 1", got)
	}
	errID = SendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "stream handle is invalid")

	listHandle, errID := StartGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, CancelGreeterListMessageServerStream(context.Background(), listHandle))
	if got := greeterNativeListCancelsForIntegration(); got != 1 {
		t.Fatalf("native list cancels = %d, want 1", got)
	}

	chatHandle, errID := StartGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, CancelGreeterChatMessageBidiStream(context.Background(), chatHandle))
	if got := greeterNativeChatCancelsForIntegration(); got != 1 {
		t.Fatalf("native chat cancels = %d, want 1", got)
	}
}

func TestNativeContractMismatch(t *testing.T) {
	registerMessageServer(t)
	errID := CallGreeterUnaryNativeUnary(context.Background())
	assertMessageNoErr(t, errID)

	uploadHandle, errID := StartGreeterUploadNativeClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, SendGreeterUploadNativeClientStream(context.Background(), uploadHandle))
	assertMessageNoErr(t, FinishGreeterUploadNativeClientStream(context.Background(), uploadHandle))

	listHandle, errID := StartGreeterListNativeServerStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, ReadGreeterListNativeServerStream(context.Background(), listHandle))
	assertMessageNoErr(t, DoneGreeterListNativeServerStream(context.Background(), listHandle))

	chatHandle, errID := StartGreeterChatNativeBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, SendGreeterChatNativeBidiStream(context.Background(), chatHandle))
	assertMessageNoErr(t, ReadGreeterChatNativeBidiStream(context.Background(), chatHandle))
	assertMessageNoErr(t, CloseSendGreeterChatNativeBidiStream(context.Background(), chatHandle))
	assertMessageNoErr(t, DoneGreeterChatNativeBidiStream(context.Background(), chatHandle))
}

type mismatchNativeServer struct{}

func (mismatchNativeServer) Unary(context.Context) error {
	return nil
}

func (mismatchNativeServer) Upload(ctx context.Context, stream v1.GreeterUploadNativeClientStream) error {
	for {
		err := stream.Recv(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (mismatchNativeServer) List(ctx context.Context, stream v1.GreeterListNativeServerStream) error {
	if err := stream.Send(ctx); err != nil {
		return err
	}
	return nil
}

func (mismatchNativeServer) Chat(ctx context.Context, stream v1.GreeterChatNativeBidiStream) error {
	for {
		err := stream.Recv(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(ctx); err != nil {
			return err
		}
	}
}

func assertMessageErrContains(t *testing.T, errID int32, wants ...string) {
	t.Helper()
	if errID == 0 {
		t.Fatalf("errID = 0, want error containing %q", wants)
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok {
		t.Fatalf("error text = %q, ok=%v, want contains %q", text, ok, wants)
	}
	rpcruntime.Release(ptr)
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
