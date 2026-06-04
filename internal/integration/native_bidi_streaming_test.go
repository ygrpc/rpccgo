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
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestNativeBidiStreamingRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeBidiStreamingTestPlugin(t, "example.com/nativebidi/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeBidiStreamingFixture(t, tmp, plugin, "example.com/nativebidi")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_bidi_streaming_go_test.go"), nativeBidiStreamingGoFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeBidiStreamingGo", "-count=1", "-timeout=30s")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native bidi streaming go fixture failed: %v\n%s", err, out)
	}
}

func TestNativeBidiStreamingRoutesToCGONativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeBidiStreamingTestPlugin(t, "example.com/nativebidicgo/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeBidiStreamingFixture(t, tmp, plugin, "example.com/nativebidicgo")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_bidi_streaming_cgo_callbacks.go"), nativeBidiStreamingCGOFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_bidi_streaming_cgo_test.go"), nativeBidiStreamingCGOFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeBidiStreamingCGO", "-count=1", "-timeout=30s")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native bidi streaming cgo fixture failed: %v\n%s", err, out)
	}
}

func writeNativeBidiStreamingFixture(t *testing.T, tmp string, plugin *protogen.Plugin, module string) {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.mod"), "module "+module+"\n\ngo 1.24.4\n\nrequire (\n\tgoogle.golang.org/protobuf v1.36.11\n\trpccgo v0.0.0\n)\n\nreplace rpccgo => "+repoRoot+"\n")
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
	writeFile(t, filepath.Join(tmp, "test/v1/native_bidi_streaming_stubs.go"), nativeBidiStreamingStubSource)
}

func newNativeBidiStreamingTestPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"test/v1/native_bidi_streaming.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("test/v1/native_bidi_streaming.proto"),
			Package: proto.String("test.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String(goPackage),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ChatRequest"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("seq", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
				{
					Name: proto.String("ChatReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("ack", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("message", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:            proto.String("Chat"),
					InputType:       proto.String(".test.v1.ChatRequest"),
					OutputType:      proto.String(".test.v1.ChatReply"),
					ClientStreaming: proto.Bool(true),
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

const nativeBidiStreamingStubSource = `package testv1

import (
	context "context"

	connect "connectrpc.com/connect"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type ChatRequest struct {
	Name string
	Seq int32
}

func (*ChatRequest) ProtoReflect() protoreflect.Message { return nil }

type ChatReply struct {
	Ack int32
	Message string
}

func (*ChatReply) ProtoReflect() protoreflect.Message { return nil }

type GreeterHandler interface {
	Chat(context.Context, *connect.BidiStream[ChatRequest, ChatReply]) error
}
type GreeterClient interface {
	Chat(context.Context) (*connect.BidiStreamForClientSimple[ChatRequest, ChatReply], error)
}
type GreeterServer interface {
	Chat(Greeter_ChatServer) error
}

type Greeter_ChatServer interface {
	Recv() (*ChatRequest, error)
	RecvMsg(any) error
	Send(*ChatReply) error
	SendMsg(any) error
	Context() context.Context
}
`

const nativeBidiStreamingGoFixtureTestSource = `package main

import (
	context "context"
	"errors"
	"io"
	"strings"
	"testing"
	"unsafe"

	v1 "example.com/nativebidi/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type chatGoServer struct {
	stream *chatGoStream
}

func (s *chatGoServer) Chat(ctx context.Context, stream v1.GreeterChatNativeBidiStream) error {
	s.stream = &chatGoStream{}
	for {
		name, seq, err := stream.Recv(ctx)
		if err == io.EOF {
			s.stream.closed = true
			return nil
		}
		if err != nil {
			s.stream.canceled = true
			return err
		}
		s.stream.names = append(s.stream.names, name.SafeString())
		s.stream.seqs = append(s.stream.seqs, seq)
		if err := stream.Send(ctx, seq, name.SafeString()); err != nil {
			s.stream.canceled = true
			return err
		}
	}
}

type chatGoStream struct {
	names []string
	seqs []int32
	read int
	closed bool
	canceled bool
}

func (s *chatGoStream) Send(ctx context.Context, name *rpcruntime.RpcString, seq int32) error {
	if s.closed {
		return errors.New("go bidi send closed")
	}
	s.names = append(s.names, name.SafeString())
	s.seqs = append(s.seqs, seq)
	return nil
}

func (s *chatGoStream) Recv(ctx context.Context) (int32, string, error) {
	if s.read >= len(s.names) {
		return 0, "", io.EOF
	}
	index := s.read
	s.read++
	return s.seqs[index], s.names[index], nil
}

func (s *chatGoStream) CloseSend(ctx context.Context) error {
	s.closed = true
	return nil
}

func (s *chatGoStream) Cancel(ctx context.Context) error {
	s.canceled = true
	return nil
}

type chatInputABI struct {
	NamePtr uintptr
	NameLen int32
	NameOwnership int32
	Seq int32
}

type chatOutput struct {
	Ack int32
	MessagePtr uintptr
	MessageLen int32
}

func sendChatErr(ctx context.Context, handle int32, input *chatInputABI) int32 {
	if input == nil {
		input = &chatInputABI{}
	}
	return SendGreeterChatNativeBidiStream(ctx, handle, input.NamePtr, input.NameLen, input.NameOwnership, input.Seq)
}

func readChat(ctx context.Context, handle int32, output *chatOutput) int32 {
	if output == nil {
		output = &chatOutput{}
	}
	return ReadGreeterChatNativeBidiStream(ctx, handle, &output.Ack, &output.MessagePtr, &output.MessageLen)
}

func TestNativeBidiStreamingGoServerCloseSendSemantics(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &chatGoServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	handle, errID := StartGreeterChatNativeBidiStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() errID = %d", errID)
	}
	sendChat(t, handle, "first", 1)
	sendChat(t, handle, "second", 2)
	assertChatRead(t, handle, 1, "first")
	if errID := CloseSendGreeterChatNativeBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatNativeBidiStream() errID = %d", errID)
	}
	assertErrorTextContainsBidi(t, sendChatErr(context.Background(), handle, chatInput("third", 3)), "native stream is closed")
	assertChatRead(t, handle, 2, "second")
	if errID := FinishGreeterChatNativeBidiStream(context.Background(), handle); errID != 0 {
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("FinishGreeterChatNativeBidiStream() errID = %d, text = %q, ok = %v", errID, text, ok)
	}
	if errID := readChat(context.Background(), handle, &chatOutput{}); errID == 0 {
		t.Fatal("Read after Finish returned errID 0")
	}
}

func TestNativeBidiStreamingGoServerCancelFinalizesOnce(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &chatGoServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	handle, errID := StartGreeterChatNativeBidiStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() errID = %d", errID)
	}
	if errID := CancelGreeterChatNativeBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CancelGreeterChatNativeBidiStream() errID = %d", errID)
	}
	if server.stream == nil || !server.stream.canceled {
		t.Fatal("Cancel did not propagate to Go native stream")
	}
	if errID := CancelGreeterChatNativeBidiStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Cancel returned errID 0")
	}
}

func sendChat(t *testing.T, handle int32, name string, seq int32) {
	t.Helper()
	if errID := sendChatErr(context.Background(), handle, chatInput(name, seq)); errID != 0 {
		t.Fatalf("SendGreeterChatNativeBidiStream() errID = %d", errID)
	}
}

func chatInput(name string, seq int32) *chatInputABI {
	data := []byte(name)
	return &chatInputABI{
		NamePtr: uintptr(unsafe.Pointer(&data[0])),
		NameLen: int32(len(data)),
		Seq: seq,
	}
}

func assertChatRead(t *testing.T, handle int32, wantAck int32, wantMessage string) {
	t.Helper()
	output := &chatOutput{}
	if errID := readChat(context.Background(), handle, output); errID != 0 {
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("ReadGreeterChatNativeBidiStream() errID = %d, text = %q, ok = %v", errID, text, ok)
	}
	if output.Ack != wantAck {
		t.Fatalf("Ack = %d, want %d", output.Ack, wantAck)
	}
	message := unsafe.Slice((*byte)(unsafe.Pointer(output.MessagePtr)), output.MessageLen)
	if string(message) != wantMessage {
		t.Fatalf("Message = %q, want %q", message, wantMessage)
	}
	rpcruntime.Release(output.MessagePtr)
}

func assertErrorTextContainsBidi(t *testing.T, errID int32, want string) {
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

const nativeBidiStreamingCGOFixtureTestSource = `package main

import (
	context "context"
	"strings"
	"testing"
	"unsafe"

	v1 "example.com/nativebidicgo/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type chatInputABI struct {
	NamePtr uintptr
	NameLen int32
	NameOwnership int32
	Seq int32
}

type chatOutput struct {
	Ack int32
	MessagePtr uintptr
	MessageLen int32
}

func sendChatErr(ctx context.Context, handle int32, input *chatInputABI) int32 {
	if input == nil {
		input = &chatInputABI{}
	}
	return SendGreeterChatNativeBidiStream(ctx, handle, input.NamePtr, input.NameLen, input.NameOwnership, input.Seq)
}

func readChat(ctx context.Context, handle int32, output *chatOutput) int32 {
	if output == nil {
		output = &chatOutput{}
	}
	return ReadGreeterChatNativeBidiStream(ctx, handle, &output.Ack, &output.MessagePtr, &output.MessageLen)
}

func TestNativeBidiStreamingCGOServerCloseSendSemantics(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerBidiCFreeCallback()
	if err := registerGreeterBidiCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterBidiCGONativeServerCallbacks() error = %v", err)
	}

	handle, errID := StartGreeterChatNativeBidiStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() errID = %d", errID)
	}
	sendChatCGO(t, handle, "one", 1)
	sendChatCGO(t, handle, "two", 2)
	assertChatReadCGO(t, handle, 1, "one")
	if got := frees(); got != 1 {
		t.Fatalf("free count after first read = %d, want 1", got)
	}
	if errID := CloseSendGreeterChatNativeBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CloseSendGreeterChatNativeBidiStream() errID = %d", errID)
	}
	assertErrorTextContainsBidiCGO(t, sendChatErr(context.Background(), handle, chatInputCGO("three", 3)), "native stream is closed")
	assertChatReadCGO(t, handle, 2, "two")
	if errID := FinishGreeterChatNativeBidiStream(context.Background(), handle); errID != 0 {
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("FinishGreeterChatNativeBidiStream() errID = %d, text = %q, ok = %v", errID, text, ok)
	}
	if got := greeterBidiFinishCount(); got != 1 {
		t.Fatalf("finish count = %d, want 1", got)
	}
	if errID := FinishGreeterChatNativeBidiStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Finish returned errID 0")
	}
}

func TestNativeBidiStreamingCGOServerCancelFinalizesOnce(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := registerGreeterBidiCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterBidiCGONativeServerCallbacks() error = %v", err)
	}
	handle, errID := StartGreeterChatNativeBidiStream(context.Background())
	if errID != 0 {
		t.Fatalf("StartGreeterChatNativeBidiStream() errID = %d", errID)
	}
	if errID := CancelGreeterChatNativeBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("CancelGreeterChatNativeBidiStream() errID = %d", errID)
	}
	if got := greeterBidiCancelCount(); got != 1 {
		t.Fatalf("cancel count = %d, want 1", got)
	}
	if errID := CancelGreeterChatNativeBidiStream(context.Background(), handle); errID == 0 {
		t.Fatal("second Cancel returned errID 0")
	}
	if got := greeterBidiCancelCount(); got != 1 {
		t.Fatalf("cancel count after second cancel = %d, want 1", got)
	}
}

func sendChatCGO(t *testing.T, handle int32, name string, seq int32) {
	t.Helper()
	if errID := sendChatErr(context.Background(), handle, chatInputCGO(name, seq)); errID != 0 {
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("SendGreeterChatNativeBidiStream() errID = %d, text = %q, ok = %v", errID, text, ok)
	}
}

func chatInputCGO(name string, seq int32) *chatInputABI {
	data := []byte(name)
	return &chatInputABI{
		NamePtr: uintptr(unsafe.Pointer(&data[0])),
		NameLen: int32(len(data)),
		Seq: seq,
	}
}

func assertChatReadCGO(t *testing.T, handle int32, wantAck int32, wantMessage string) {
	t.Helper()
	output := &chatOutput{}
	if errID := readChat(context.Background(), handle, output); errID != 0 {
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("ReadGreeterChatNativeBidiStream() errID = %d, text = %q, ok = %v", errID, text, ok)
	}
	if output.Ack != wantAck {
		t.Fatalf("Ack = %d, want %d", output.Ack, wantAck)
	}
	message := unsafe.Slice((*byte)(unsafe.Pointer(output.MessagePtr)), output.MessageLen)
	if string(message) != wantMessage {
		t.Fatalf("Message = %q, want %q", message, wantMessage)
	}
}

func assertErrorTextContainsBidiCGO(t *testing.T, errID int32, want string) {
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

const nativeBidiStreamingCGOFixtureCallbackSource = `package main

/*
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

extern int32_t StoreGreeterCGONativeServerErrorTextForExport(char* text, int32_t textLen);

typedef int32_t (*GreeterChatCGONativeBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamSendCallback)(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, int32_t Seq);
typedef int32_t (*GreeterChatCGONativeBidiStreamRecvCallback)(int32_t stream, int32_t* outAck, uintptr_t* outMessagePtr, int32_t* outMessageLen, int32_t* outMessageOwnership);
typedef int32_t (*GreeterChatCGONativeBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGONativeServerCallbacks {
GreeterChatCGONativeBidiStreamStartCallback ChatStart;
GreeterChatCGONativeBidiStreamSendCallback ChatSend;
GreeterChatCGONativeBidiStreamRecvCallback ChatRecv;
GreeterChatCGONativeBidiStreamCloseSendCallback ChatCloseSend;
GreeterChatCGONativeBidiStreamFinishCallback ChatFinish;
GreeterChatCGONativeBidiStreamCancelCallback ChatCancel;
} GreeterCGONativeServerCallbacks;

static int32_t greeterBidiStreamID;
static int32_t greeterBidiCount;
static int32_t greeterBidiRead;
static int32_t greeterBidiClosed;
static int32_t greeterBidiCancels;
static int32_t greeterBidiFinishes;
static int32_t greeterBidiErrorMode;
static char greeterBidiNames[8][64];
static int32_t greeterBidiSeqs[8];

static int32_t greeterBidiError(const char* text) {
	return StoreGreeterCGONativeServerErrorTextForExport((char*)text, (int32_t)strlen(text));
}

static int32_t greeterChatStart(int32_t* stream) {
	if (greeterBidiErrorMode == 1) {
		return greeterBidiError("forced start error");
	}
	if (stream == NULL) {
		return greeterBidiError("bidi start missing stream");
	}
	greeterBidiStreamID = 81;
	greeterBidiCount = 0;
	greeterBidiRead = 0;
	greeterBidiClosed = 0;
	greeterBidiCancels = 0;
	greeterBidiFinishes = 0;
	*stream = greeterBidiStreamID;
	return 0;
}

static int32_t greeterChatStartForcedError(int32_t* stream) {
	return greeterBidiError("forced start error");
}

static int32_t greeterChatSend(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, int32_t Seq) {
	if (greeterBidiErrorMode == 2) {
		return greeterBidiError("forced send error");
	}
	if (stream != greeterBidiStreamID) {
		return greeterBidiError("bidi send did not reach cgo callback");
	}
	if (greeterBidiClosed) {
		return greeterBidiError("native stream is closed");
	}
	if (greeterBidiCount >= 8 || NameLen < 0 || NameLen >= 60) {
		return greeterBidiError("bidi bad input");
	}
	memcpy(greeterBidiNames[greeterBidiCount], (void*)NamePtr, (size_t)NameLen);
	greeterBidiNames[greeterBidiCount][NameLen] = 0;
	greeterBidiSeqs[greeterBidiCount] = Seq;
	greeterBidiCount += 1;
	return 0;
}

static int32_t greeterChatSendForcedError(int32_t stream, uintptr_t NamePtr, int32_t NameLen, int32_t NameOwnership, int32_t Seq) {
	return greeterBidiError("forced send error");
}

static int32_t greeterChatRecv(int32_t stream, int32_t* outAck, uintptr_t* outMessagePtr, int32_t* outMessageLen, int32_t* outMessageOwnership) {
	if (greeterBidiErrorMode == 3) {
		return greeterBidiError("forced recv error");
	}
	if (stream != greeterBidiStreamID || outAck == NULL || outMessagePtr == NULL || outMessageLen == NULL || outMessageOwnership == NULL) {
		return greeterBidiError("bidi recv did not reach cgo callback");
	}
	for (int i = 0; i < 1000 && greeterBidiRead >= greeterBidiCount && !greeterBidiClosed; i++) {
		usleep(1000);
	}
	if (greeterBidiRead >= greeterBidiCount) {
		char finished[64];
		snprintf(finished, sizeof(finished), "cgo bidi finished read=%d count=%d", greeterBidiRead, greeterBidiCount);
		return greeterBidiError(finished);
	}
	char* msg = (char*)malloc((size_t)strlen(greeterBidiNames[greeterBidiRead]));
	if (msg == NULL) {
		return greeterBidiError("bidi malloc failed");
	}
	memcpy(msg, greeterBidiNames[greeterBidiRead], strlen(greeterBidiNames[greeterBidiRead]));
	*outAck = greeterBidiSeqs[greeterBidiRead];
	*outMessagePtr = (uintptr_t)msg;
	*outMessageLen = (int32_t)strlen(greeterBidiNames[greeterBidiRead]);
	*outMessageOwnership = 1;
	greeterBidiRead += 1;
	return 0;
}

static int32_t greeterChatRecvForcedError(int32_t stream, int32_t* outAck, uintptr_t* outMessagePtr, int32_t* outMessageLen, int32_t* outMessageOwnership) {
	return greeterBidiError("forced recv error");
}

static int32_t greeterChatCloseSend(int32_t stream) {
	if (greeterBidiErrorMode == 4) {
		return greeterBidiError("forced close error");
	}
	if (stream != greeterBidiStreamID) {
		return greeterBidiError("bidi close did not reach cgo callback");
	}
	greeterBidiClosed = 1;
	return 0;
}

static int32_t greeterChatCloseSendForcedError(int32_t stream) {
	return greeterBidiError("forced close error");
}

static int32_t greeterChatFinish(int32_t stream) {
	if (greeterBidiErrorMode == 5) {
		return greeterBidiError("forced finish error");
	}
	if (stream != greeterBidiStreamID) {
		return greeterBidiError("bidi finish did not reach cgo callback");
	}
	greeterBidiFinishes += 1;
	return 0;
}

static int32_t greeterChatFinishForcedError(int32_t stream) {
	return greeterBidiError("forced finish error");
}

static int32_t greeterChatCancel(int32_t stream) {
	if (greeterBidiErrorMode == 6) {
		return greeterBidiError("forced cancel error");
	}
	if (stream != greeterBidiStreamID) {
		return greeterBidiError("bidi cancel did not reach cgo callback");
	}
	greeterBidiCancels += 1;
	return 0;
}

static int32_t greeterChatCancelForcedError(int32_t stream) {
	return greeterBidiError("forced cancel error");
}

static GreeterCGONativeServerCallbacks greeterBidiCallbacks(void) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.ChatStart = greeterChatStart;
	callbacks.ChatSend = greeterChatSend;
	callbacks.ChatRecv = greeterChatRecv;
	callbacks.ChatCloseSend = greeterChatCloseSend;
	callbacks.ChatFinish = greeterChatFinish;
	callbacks.ChatCancel = greeterChatCancel;
	return callbacks;
}

static int32_t greeterBidiCancelCount(void) {
	return greeterBidiCancels;
}

static int32_t greeterBidiFinishCount(void) {
	return greeterBidiFinishes;
}

static void setGreeterBidiErrorMode(int32_t mode) {
	greeterBidiErrorMode = mode;
}

static GreeterCGONativeServerCallbacks greeterBidiCallbacksWithMode(int32_t mode) {
	GreeterCGONativeServerCallbacks callbacks = greeterBidiCallbacks();
	if (mode == 1) {
		callbacks.ChatStart = greeterChatStartForcedError;
	} else if (mode == 2) {
		callbacks.ChatSend = greeterChatSendForcedError;
	} else if (mode == 3) {
		callbacks.ChatRecv = greeterChatRecvForcedError;
	} else if (mode == 4) {
		callbacks.ChatCloseSend = greeterChatCloseSendForcedError;
	} else if (mode == 5) {
		callbacks.ChatFinish = greeterChatFinishForcedError;
	} else if (mode == 6) {
		callbacks.ChatCancel = greeterChatCancelForcedError;
	}
	return callbacks;
}
*/
import "C"

import (
	"errors"
	"sync/atomic"
	"unsafe"

	rpcruntime "rpccgo/rpcruntime"
)

func registerGreeterBidiCGONativeServerCallbacks() error {
	callbacks := C.greeterBidiCallbacks()
	return registerGreeterBidiCGONativeServerCallbackTable(callbacks)
}

func registerGreeterBidiCGONativeServerCallbacksWithMode(mode int32) error {
	callbacks := C.greeterBidiCallbacksWithMode(C.int32_t(mode))
	return registerGreeterBidiCGONativeServerCallbackTable(callbacks)
}

func registerGreeterBidiCGONativeServerCallbackTable(callbacks C.GreeterCGONativeServerCallbacks) error {
	errID := rpccgo_native_testv1_Greeter_register(callbacks.ChatStart, callbacks.ChatSend, callbacks.ChatRecv, callbacks.ChatCloseSend, callbacks.ChatFinish, callbacks.ChatCancel)
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

func greeterBidiCancelCount() int32 {
	return int32(C.greeterBidiCancelCount())
}

func greeterBidiFinishCount() int32 {
	return int32(C.greeterBidiFinishCount())
}

func setGreeterBidiErrorMode(mode int32) {
	C.setGreeterBidiErrorMode(C.int32_t(mode))
}

func registerBidiCFreeCallback() func() int32 {
	var frees atomic.Int32
	rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {
		frees.Add(1)
		C.free(ptr)
	})
	return frees.Load
}
`
