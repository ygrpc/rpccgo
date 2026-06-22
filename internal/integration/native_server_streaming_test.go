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

func TestNativeServerStreamingRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeServerStreamingTestPlugin(t, "example.com/nativeserverstream/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeServerStreamingFixture(t, tmp, plugin, "example.com/nativeserverstream")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_go_test.go"), nativeServerStreamingGoFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeServerStreamingGo", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native server streaming go fixture failed: %v\n%s", err, out)
	}
}

func TestNativeServerStreamingRoutesToCGONativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeServerStreamingTestPlugin(t, "example.com/nativeserverstreamcgo/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeNativeServerStreamingFixture(t, tmp, plugin, "example.com/nativeserverstreamcgo")
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_cgo_callbacks.go"), nativeServerStreamingCGOFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_cgo_test.go"), nativeServerStreamingCGOFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeServerStreamingCGO", "-count=1")
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
	writeFile(t, filepath.Join(tmp, "test/v1/native_server_streaming_stubs.go"), nativeServerStreamingStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_server_streaming_cgo_client_bridge.go"), nativeServerStreamingCGOClientBridgeSource)
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

import (
	context "context"

	connect "connectrpc.com/connect"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type ListRequest struct {
	Prefix string
	Limit int32
}

func (*ListRequest) ProtoReflect() protoreflect.Message { return nil }

type ListReply struct {
	Index int32
	Name string
}

func (*ListReply) ProtoReflect() protoreflect.Message { return nil }

type GreeterHandler interface {
	List(context.Context, *ListRequest, *connect.ServerStream[ListReply]) error
}
type GreeterClient interface {
	List(context.Context, *ListRequest) (*connect.ServerStreamForClient[ListReply], error)
}
type GreeterServer interface {
	List(*ListRequest, Greeter_ListServer) error
}

type Greeter_ListServer interface {
	Send(*ListReply) error
	SendMsg(any) error
	RecvMsg(any) error
	Context() context.Context
}
`

const nativeServerStreamingCGOClientBridgeSource = `package main

/*
#include <stdint.h>
*/
import "C"

import context "context"

func GreeterListNativeServerStreamStart(ctx context.Context, PrefixPtr uintptr, PrefixLen int32, PrefixOwnership int32, Limit int32) (int32, int32) {
	var stream C.int32_t
	errID := rpccgoNativeTestv1GreeterListStart(C.uintptr_t(PrefixPtr), C.int32_t(PrefixLen), C.int32_t(PrefixOwnership), C.int32_t(Limit), &stream, nil, nil)
	return int32(stream), int32(errID)
}

func GreeterListNativeServerStreamRecv(ctx context.Context, stream int32, outIndex *int32, outNamePtr *uintptr, outNameLen *int32) int32 {
	var index C.int32_t
	var namePtr C.uintptr_t
	var nameLen C.int32_t
	var nameOwnership C.int32_t
	errID := rpccgoNativeTestv1GreeterListRecv(C.int32_t(stream), &index, &namePtr, &nameLen, &nameOwnership)
	*outIndex = int32(index)
	*outNamePtr = uintptr(namePtr)
	*outNameLen = int32(nameLen)
	return int32(errID)
}

func GreeterListNativeServerStreamFinish(ctx context.Context, stream int32) int32 {
	return int32(rpccgoNativeTestv1GreeterListCancel(C.int32_t(stream)))
}

func GreeterListNativeServerStreamCancel(ctx context.Context, stream int32) int32 {
	return int32(rpccgoNativeTestv1GreeterListCancel(C.int32_t(stream)))
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
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

type listGoServer struct {
	label string
	stream *listGoStream
}

func (s *listGoServer) List(ctx context.Context, prefix *rpcruntime.RpcString, limit int32, stream v1.GreeterListNativeServerStream) error {
	s.stream = &listGoStream{prefix: s.label + prefix.SafeString(), limit: limit}
	for s.stream.index < s.stream.limit {
		s.stream.index++
		if err := stream.Send(ctx, s.stream.index, s.stream.prefix+":"+string(rune('0'+s.stream.index))); err != nil {
			s.stream.canceled = true
			return err
		}
	}
	return nil
}

type listGoStream struct {
	prefix string
	limit int32
	index int32
	canceled bool
}

func (s *listGoStream) Recv(ctx context.Context) (int32, string, error) {
	if s.index >= s.limit {
		return 0, "", io.EOF
	}
	s.index++
	return s.index, s.prefix + ":" + string(rune('0'+s.index)), nil
}

func (s *listGoStream) Cancel(ctx context.Context) error {
	s.canceled = true
	return nil
}

type listInputABI struct {
	PrefixPtr uintptr
	PrefixLen int32
	PrefixOwnership int32
	Limit int32
}

type listOutput struct {
	Index int32
	NamePtr uintptr
	NameLen int32
}

func startList(ctx context.Context, input *listInputABI) (int32, int32) {
	if input == nil {
		input = &listInputABI{}
	}
	return GreeterListNativeServerStreamStart(ctx, input.PrefixPtr, input.PrefixLen, input.PrefixOwnership, input.Limit)
}

func readList(ctx context.Context, handle int32, output *listOutput) int32 {
	if output == nil {
		output = &listOutput{}
	}
	return GreeterListNativeServerStreamRecv(ctx, handle, &output.Index, &output.NamePtr, &output.NameLen)
}

func TestNativeServerStreamingGoServerFinishFinalizesHandle(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &listGoServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	input := listInput("go", 2)
	handle, errID := startList(context.Background(), input)
	if errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamStart() errID = %d", errID)
	}
	assertListRecv(t, handle, 1, "go:1")
	assertListRecv(t, handle, 2, "go:2")
	if errID := GreeterListNativeServerStreamFinish(context.Background(), handle); errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamFinish() errID = %d", errID)
	}
	if errID := readList(context.Background(), handle, &listOutput{}); errID == 0 {
		t.Fatal("Read after Finish returned errID 0")
	}
}

func TestNativeServerStreamingGoServerStartCapturesActiveServerSnapshot(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	serverA := &listGoServer{label: "A:"}
	if err := v1.RegisterGreeterGoNativeServer(serverA); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(A) error = %v", err)
	}
	handle, errID := startList(context.Background(), listInput("x", 1))
	if errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamStart() errID = %d", errID)
	}
	serverB := &listGoServer{label: "B:"}
	if err := v1.RegisterGreeterGoNativeServer(serverB); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer(B) error = %v", err)
	}
	assertListRecv(t, handle, 1, "A:x:1")
	if serverB.stream != nil {
		t.Fatalf("server B unexpectedly received stream: %#v", serverB.stream)
	}
}

func TestNativeServerStreamingGoServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &listGoServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	handle, errID := startList(context.Background(), listInput("go", 32))
	if errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamStart() errID = %d", errID)
	}
	if errID := GreeterListNativeServerStreamCancel(context.Background(), handle); errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamCancel() errID = %d", errID)
	}
	if server.stream == nil || !server.stream.canceled {
		t.Fatal("Cancel did not propagate to Go native stream")
	}
	if errID := GreeterListNativeServerStreamCancel(context.Background(), handle); errID == 0 {
		t.Fatal("second Cancel returned errID 0")
	}
}

func TestNativeServerStreamingGoServerMissingActiveServer(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	handle, errID := startList(context.Background(), listInput("none", 1))
	if handle != 0 {
		t.Fatalf("handle = %d, want 0", handle)
	}
	assertErrorTextContainsServerStream(t, errID, "registered server")
}

func listInput(prefix string, limit int32) *listInputABI {
	data := []byte(prefix)
	return &listInputABI{
		PrefixPtr: uintptr(unsafe.Pointer(&data[0])),
		PrefixLen: int32(len(data)),
		Limit: limit,
	}
}

func assertListRecv(t *testing.T, handle int32, wantIndex int32, wantName string) {
	t.Helper()
	output := &listOutput{}
	if errID := readList(context.Background(), handle, output); errID != 0 {
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("GreeterListNativeServerStreamRecv() errID = %d, text = %q, ok = %v", errID, text, ok)
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
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

type listInputABI struct {
	PrefixPtr uintptr
	PrefixLen int32
	PrefixOwnership int32
	Limit int32
}

type listOutput struct {
	Index int32
	NamePtr uintptr
	NameLen int32
}

func startList(ctx context.Context, input *listInputABI) (int32, int32) {
	if input == nil {
		input = &listInputABI{}
	}
	return GreeterListNativeServerStreamStart(ctx, input.PrefixPtr, input.PrefixLen, input.PrefixOwnership, input.Limit)
}

func readList(ctx context.Context, handle int32, output *listOutput) int32 {
	if output == nil {
		output = &listOutput{}
	}
	return GreeterListNativeServerStreamRecv(ctx, handle, &output.Index, &output.NamePtr, &output.NameLen)
}

func TestNativeServerStreamingCGOServerEOFFinalizesHandle(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	frees := registerServerStreamCFreeCallback()
	if err := registerGreeterServerStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterServerStreamCGONativeServerCallbacks() error = %v", err)
	}

	handle, errID := startList(context.Background(), listInputCGO("cgo", 2))
	if errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamStart() errID = %d", errID)
	}
	assertListRecvCGO(t, handle, 1, "cgo:1")
	assertListRecvCGO(t, handle, 2, "cgo:2")
	if got := frees(); got != 2 {
		t.Fatalf("free count after reads = %d, want 2", got)
	}
	assertErrorTextContainsCGOServerStream(t, readList(context.Background(), handle, &listOutput{}), "EOF")
	if got := greeterServerStreamFinishCount(); got != 1 {
		t.Fatalf("finish count = %d, want 1", got)
	}
	if errID := readList(context.Background(), handle, &listOutput{}); errID == 0 {
		t.Fatal("second read returned errID 0")
	}
}

func TestNativeServerStreamingCGOServerCancelFinalizesHandle(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := registerGreeterServerStreamCGONativeServerCallbacks(); err != nil {
		t.Fatalf("registerGreeterServerStreamCGONativeServerCallbacks() error = %v", err)
	}
	handle, errID := startList(context.Background(), listInputCGO("cgo", 1))
	if errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamStart() errID = %d", errID)
	}
	if errID := GreeterListNativeServerStreamCancel(context.Background(), handle); errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamCancel() errID = %d", errID)
	}
	if got := greeterServerStreamCancelCount(); got != 1 {
		t.Fatalf("cancel count = %d, want 1", got)
	}
	if errID := readList(context.Background(), handle, &listOutput{}); errID == 0 {
		t.Fatal("Read after Cancel returned errID 0")
	}
}

func listInputCGO(prefix string, limit int32) *listInputABI {
	data := []byte(prefix)
	return &listInputABI{
		PrefixPtr: uintptr(unsafe.Pointer(&data[0])),
		PrefixLen: int32(len(data)),
		Limit: limit,
	}
}

func assertListRecvCGO(t *testing.T, handle int32, wantIndex int32, wantName string) {
	t.Helper()
	output := &listOutput{}
	if errID := readList(context.Background(), handle, output); errID != 0 {
		t.Fatalf("GreeterListNativeServerStreamRecv() errID = %d", errID)
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

extern int32_t rpccgoStoreErrorText(char* text, int32_t textLen);
extern int32_t greeterNativeStreamEOFErrorIDForIntegration(void);

typedef int32_t (*GreeterListCGONativeServerStreamStartCallback)(uintptr_t PrefixPtr, int32_t PrefixLen, int32_t PrefixOwnership, int32_t Limit, int32_t* stream);
typedef int32_t (*GreeterListCGONativeServerStreamRecvCallback)(int32_t stream, int32_t* outIndex, uintptr_t* outNamePtr, int32_t* outNameLen, int32_t* outNameOwnership);
typedef int32_t (*GreeterListCGONativeServerStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGONativeServerCallbacks {
GreeterListCGONativeServerStreamStartCallback ListStart;
GreeterListCGONativeServerStreamRecvCallback ListRecv;
GreeterListCGONativeServerStreamFinishCallback ListFinish;
GreeterListCGONativeServerStreamCancelCallback ListCancel;
} GreeterCGONativeServerCallbacks;

static int32_t greeterServerStreamID;
static int32_t greeterServerStreamIndex;
static int32_t greeterServerStreamLimit;
static int32_t greeterServerStreamCancels;
static int32_t greeterServerStreamFinishes;
static int32_t greeterServerStreamErrorMode;
static char greeterServerStreamPrefix[64];

static int32_t greeterServerStreamError(const char* text) {
	return rpccgoStoreErrorText((char*)text, (int32_t)strlen(text));
}

static int32_t greeterListStart(uintptr_t PrefixPtr, int32_t PrefixLen, int32_t PrefixOwnership, int32_t Limit, int32_t* stream) {
	if (greeterServerStreamErrorMode == 1) {
		return greeterServerStreamError("forced start error");
	}
	if (stream == NULL) {
		return greeterServerStreamError("server stream start missing input");
	}
	int32_t n = PrefixLen;
	if (n < 0 || n >= 60) {
		return greeterServerStreamError("server stream bad prefix");
	}
	memcpy(greeterServerStreamPrefix, (void*)PrefixPtr, (size_t)n);
	greeterServerStreamPrefix[n] = 0;
	greeterServerStreamID = 71;
	greeterServerStreamIndex = 0;
	greeterServerStreamLimit = Limit;
	greeterServerStreamCancels = 0;
	greeterServerStreamFinishes = 0;
	*stream = greeterServerStreamID;
	return 0;
}

static int32_t greeterListStartForcedError(uintptr_t PrefixPtr, int32_t PrefixLen, int32_t PrefixOwnership, int32_t Limit, int32_t* stream) {
	return greeterServerStreamError("forced start error");
}

static int32_t greeterListRecv(int32_t stream, int32_t* outIndex, uintptr_t* outNamePtr, int32_t* outNameLen, int32_t* outNameOwnership) {
	if (greeterServerStreamErrorMode == 2) {
		return greeterServerStreamError("forced recv error");
	}
	if (stream != greeterServerStreamID || outIndex == NULL || outNamePtr == NULL || outNameLen == NULL || outNameOwnership == NULL) {
		return greeterServerStreamError("server stream recv did not reach cgo callback");
	}
	if (greeterServerStreamIndex >= greeterServerStreamLimit) {
		return greeterNativeStreamEOFErrorIDForIntegration();
	}
	greeterServerStreamIndex += 1;
	char buf[96];
	int n = snprintf(buf, sizeof(buf), "%s:%d", greeterServerStreamPrefix, greeterServerStreamIndex);
	char* name = (char*)malloc((size_t)n);
	if (name == NULL) {
		return greeterServerStreamError("name malloc failed");
	}
	memcpy(name, buf, (size_t)n);
	*outIndex = greeterServerStreamIndex;
	*outNamePtr = (uintptr_t)name;
	*outNameLen = n;
	*outNameOwnership = 1;
	return 0;
}

static int32_t greeterListRecvForcedError(int32_t stream, int32_t* outIndex, uintptr_t* outNamePtr, int32_t* outNameLen, int32_t* outNameOwnership) {
	return greeterServerStreamError("forced recv error");
}

static int32_t greeterListFinish(int32_t stream) {
	if (greeterServerStreamErrorMode == 3) {
		return greeterServerStreamError("forced finish error");
	}
	if (stream != greeterServerStreamID) {
		return greeterServerStreamError("server stream finish did not reach cgo callback");
	}
	greeterServerStreamFinishes += 1;
	return 0;
}

static int32_t greeterListFinishForcedError(int32_t stream) {
	return greeterServerStreamError("forced finish error");
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

static int32_t greeterListCancelForcedError(int32_t stream) {
	return greeterServerStreamError("forced cancel error");
}

static GreeterCGONativeServerCallbacks greeterServerStreamCallbacks(void) {
	GreeterCGONativeServerCallbacks callbacks;
	callbacks.ListStart = greeterListStart;
	callbacks.ListRecv = greeterListRecv;
	callbacks.ListFinish = greeterListFinish;
	callbacks.ListCancel = greeterListCancel;
	return callbacks;
}

static int32_t greeterServerStreamCancelCount(void) {
	return greeterServerStreamCancels;
}

static int32_t greeterServerStreamFinishCount(void) {
	return greeterServerStreamFinishes;
}

static void setGreeterServerStreamErrorMode(int32_t mode) {
	greeterServerStreamErrorMode = mode;
}

static GreeterCGONativeServerCallbacks greeterServerStreamCallbacksWithMode(int32_t mode) {
	GreeterCGONativeServerCallbacks callbacks = greeterServerStreamCallbacks();
	if (mode == 1) {
		callbacks.ListStart = greeterListStartForcedError;
	} else if (mode == 2) {
		callbacks.ListRecv = greeterListRecvForcedError;
	} else if (mode == 3) {
		callbacks.ListFinish = greeterListFinishForcedError;
	} else if (mode == 4) {
		callbacks.ListCancel = greeterListCancelForcedError;
	}
	return callbacks;
}
*/
import "C"

import (
	"errors"
	"io"
	"sync/atomic"
	"unsafe"

	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

//export greeterNativeStreamEOFErrorIDForIntegration
func greeterNativeStreamEOFErrorIDForIntegration() C.int32_t {
	return C.int32_t(rpcruntime.StoreError(io.EOF))
}

func registerGreeterServerStreamCGONativeServerCallbacks() error {
	callbacks := C.greeterServerStreamCallbacks()
	return registerGreeterServerStreamCGONativeServerCallbackTable(callbacks)
}

func registerGreeterServerStreamCGONativeServerCallbacksWithMode(mode int32) error {
	callbacks := C.greeterServerStreamCallbacksWithMode(C.int32_t(mode))
	return registerGreeterServerStreamCGONativeServerCallbackTable(callbacks)
}

func registerGreeterServerStreamCGONativeServerCallbackTable(callbacks C.GreeterCGONativeServerCallbacks) error {
	errID := rpccgoNativeTestv1GreeterRegister(callbacks.ListStart, callbacks.ListRecv, callbacks.ListFinish, callbacks.ListCancel)
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

func greeterServerStreamCancelCount() int32 {
	return int32(C.greeterServerStreamCancelCount())
}

func greeterServerStreamFinishCount() int32 {
	return int32(C.greeterServerStreamFinishCount())
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
