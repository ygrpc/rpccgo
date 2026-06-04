package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestRepeatedNativeABIAcceptance(t *testing.T) {
	tmp := t.TempDir()
	plugin := newRepeatedNativeABIPlugin(t, "example.com/repeatednativeabi/repeated/v1;repeatedv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeMessageDirectPathGeneratedModule(t, tmp, plugin, "example.com/repeatednativeabi")
	writeFile(t, filepath.Join(tmp, "repeated/v1/repeated.pb.go"), repeatedNativeABIPBGoSource)
	writeFile(t, filepath.Join(tmp, "repeated/v1/repeated_connect_stubs.go"), repeatedNativeABIConnectStubSource)
	writeFile(t, filepath.Join(tmp, "repeated/v1/repeated_integration_reset.go"), repeatedNativeABIResetSource)
	writeFile(t, filepath.Join(tmp, "repeated/v1/cgo/repeated_callbacks.go"), repeatedNativeABICallbackSource)
	writeFile(t, filepath.Join(tmp, "repeated/v1/cgo/repeated_native_abi_test.go"), repeatedNativeABIFixtureTestSource)

	cmd := exec.Command("go", "test", "./repeated/v1/cgo", "-run", "^TestRepeatedNativeABI$", "-count=1")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("repeated native ABI fixture failed: %v\n%s", err, out)
	}
}

func newRepeatedNativeABIPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"repeated/v1/repeated.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("repeated/v1/repeated.proto"),
			Package: proto.String("repeated.abi.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String(goPackage),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("RepeatedRequest"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("scores", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
						fieldDescriptor("flags", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					},
				},
				{
					Name: proto.String("RepeatedReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("scores", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
						fieldDescriptor("flags", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("RepeatedGreeter"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:       proto.String("Echo"),
					InputType:  proto.String(".repeated.abi.v1.RepeatedRequest"),
					OutputType: proto.String(".repeated.abi.v1.RepeatedReply"),
				}},
			}},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
				Path:            []int32{6, 0},
				Span:            []int32{0, 0, 0},
				LeadingComments: proto.String("@rpccgo: msg-connect|native\n"),
			}}},
		}},
	}
	plugin, err := generator.ProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

const repeatedNativeABIConnectStubSource = `package repeatedv1

import context "context"

type RepeatedGreeterHandler interface {
	Echo(context.Context, *RepeatedRequest) (*RepeatedReply, error)
}

type RepeatedGreeterClient interface {
	Echo(context.Context, *RepeatedRequest) (*RepeatedReply, error)
}

type RepeatedGreeterServer interface {
	Echo(context.Context, *RepeatedRequest) (*RepeatedReply, error)
}
`

const repeatedNativeABIResetSource = `package repeatedv1

import rpcruntime "rpccgo/rpcruntime"

func ResetRepeatedGreeterServerForIntegrationTest() {
	_ = ClearRepeatedGreeterServer()
	repeatedGreeterStreamRegistry = rpcruntime.StreamRegistry{}
}
`

const repeatedNativeABICallbackSource = `package main

/*
#include <stdint.h>

typedef int32_t (*RepeatedGreeterEchoCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);

typedef struct RepeatedGreeterCGOMessageServerCallbacks {
	RepeatedGreeterEchoCGOMessageUnaryCallback Echo;
} RepeatedGreeterCGOMessageServerCallbacks;

static int repeatedMessageUnaryCalls;
static uint8_t repeatedMessageUnaryResponse[] = {0x0A, 0x02, 0x01, 0x02, 0x12, 0x02, 0x01, 0x00};

static int32_t repeatedGreeterMessageUnary(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {
	repeatedMessageUnaryCalls++;
	*response_ptr = (uintptr_t)repeatedMessageUnaryResponse;
	*response_len = (int32_t)sizeof(repeatedMessageUnaryResponse);
	return 0;
}

static RepeatedGreeterCGOMessageServerCallbacks repeatedGreeterMessageCallbacks(void) {
	RepeatedGreeterCGOMessageServerCallbacks callbacks;
	callbacks.Echo = repeatedGreeterMessageUnary;
	return callbacks;
}

static int getRepeatedMessageUnaryCalls(void) { return repeatedMessageUnaryCalls; }
*/
import "C"

import (
	errors "errors"

	repeatedv1 "example.com/repeatednativeabi/repeated/v1"
	rpcruntime "rpccgo/rpcruntime"
)

func registerRepeatedGreeterMessageCallbacksForIntegration() error {
	repeatedv1.ResetRepeatedGreeterServerForIntegrationTest()
	callbacks := C.repeatedGreeterMessageCallbacks()
	errID := rpccgo_msg_repeatedv1_RepeatedGreeter_register(callbacks.Echo)
	if errID == 0 {
		return nil
	}
	text, ptr, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok {
		return errors.New("missing repeated message registration error")
	}
	if ptr != 0 {
		rpcruntime.Release(ptr)
	}
	return errors.New(string(text))
}

func repeatedMessageUnaryCallsForIntegration() int {
	return int(C.getRepeatedMessageUnaryCalls())
}
`

const repeatedNativeABIFixtureTestSource = `package main

import (
	context "context"
	slices "slices"
	strings "strings"
	testing "testing"
	unsafe "unsafe"

	repeatedv1 "example.com/repeatednativeabi/repeated/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type repeatedGoNativeServer struct{}

func (repeatedGoNativeServer) Echo(ctx context.Context, scores *rpcruntime.RpcRepeat[int32], flags *rpcruntime.RpcBoolRepeat) ([]int32, []bool, error) {
	outScores := append([]int32(nil), scores.SafeSlice()...)
	for i := range outScores {
		outScores[i] += 10
	}
	inFlags := flags.SafeSlice()
	outFlags := make([]bool, len(inFlags))
	for i, flag := range inFlags {
		outFlags[i] = !flag
	}
	return outScores, outFlags, nil
}

type repeatedInput struct {
	ScoresPtr uintptr
	ScoresLen int32
	ScoresOwnership int32
	FlagsPtr uintptr
	FlagsLen int32
	FlagsOwnership int32
}

type repeatedOutput struct {
	ScoresPtr uintptr
	ScoresLen int32
	FlagsPtr uintptr
	FlagsLen int32
}

func callRepeatedEcho(ctx context.Context, input *repeatedInput, output *repeatedOutput) int32 {
	if input == nil {
		input = &repeatedInput{}
	}
	if output == nil {
		output = &repeatedOutput{}
	}
	return CallRepeatedGreeterEchoNativeUnary(ctx,
		input.ScoresPtr, input.ScoresLen, input.ScoresOwnership,
		input.FlagsPtr, input.FlagsLen, input.FlagsOwnership,
		&output.ScoresPtr, &output.ScoresLen,
		&output.FlagsPtr, &output.FlagsLen,
	)
}

func TestRepeatedNativeABI(t *testing.T) {
	t.Run("native client routes to go native server with repeated fields", func(t *testing.T) {
		repeatedv1.ResetRepeatedGreeterServerForIntegrationTest()
		if err := repeatedv1.RegisterRepeatedGreeterGoNativeServer(repeatedGoNativeServer{}); err != nil {
			t.Fatalf("RegisterRepeatedGreeterGoNativeServer() error = %v", err)
		}

		scores := []int32{1, 2, 3}
		flags := []byte{1, 0, 1}
		input := &repeatedInput{
			ScoresPtr:       uintptr(unsafe.Pointer(&scores[0])),
			ScoresLen:       int32(len(scores)),
			ScoresOwnership: 0,
			FlagsPtr:        uintptr(unsafe.Pointer(&flags[0])),
			FlagsLen:        int32(len(flags)),
			FlagsOwnership:  0,
		}
		output := &repeatedOutput{}
		if errID := callRepeatedEcho(context.Background(), input, output); errID != 0 {
			text, _, _ := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
			t.Fatalf("CallRepeatedGreeterEchoNativeUnary() errID = %d: %s", errID, text)
		}
		t.Cleanup(func() { releaseRepeatedOutput(output) })

		if got, want := int32SliceFromOutput(output.ScoresPtr, output.ScoresLen), []int32{11, 12, 13}; !slices.Equal(got, want) {
			t.Fatalf("scores = %v, want %v", got, want)
		}
		if got, want := boolSliceFromOutput(output.FlagsPtr, output.FlagsLen), []bool{false, true, false}; !slices.Equal(got, want) {
			t.Fatalf("flags = %v, want %v", got, want)
		}
	})

	t.Run("negative repeated length returns error id instead of panic", func(t *testing.T) {
		repeatedv1.ResetRepeatedGreeterServerForIntegrationTest()
		if err := repeatedv1.RegisterRepeatedGreeterGoNativeServer(repeatedGoNativeServer{}); err != nil {
			t.Fatalf("RegisterRepeatedGreeterGoNativeServer() error = %v", err)
		}

		input := &repeatedInput{
			ScoresLen: -1,
		}
		output := &repeatedOutput{}
		errID := callRepeatedEcho(context.Background(), input, output)
		if errID == 0 {
			t.Fatal("negative length returned errID 0")
		}
		text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		if !ok || !strings.Contains(string(text), "negative") {
			t.Fatalf("negative length error text = %q, ok=%v", text, ok)
		}
	})
}

func int32SliceFromOutput(ptr uintptr, length int32) []int32 {
	if ptr == 0 || length == 0 {
		return nil
	}
	data := unsafe.Slice((*int32)(unsafe.Pointer(ptr)), int(length))
	return append([]int32(nil), data...)
}

func boolSliceFromOutput(ptr uintptr, length int32) []bool {
	if ptr == 0 || length == 0 {
		return nil
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))
	out := make([]bool, len(raw))
	for i := range raw {
		out[i] = raw[i] != 0
	}
	return out
}

func bytesFromOutput(ptr uintptr, length int32) []byte {
	if ptr == 0 || length == 0 {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))...)
}

func requestPtr(data []byte) uintptr {
	if len(data) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&data[0]))
}

func releaseRepeatedOutput(output *repeatedOutput) {
	if output == nil {
		return
	}
	rpcruntime.Release(output.ScoresPtr)
	rpcruntime.Release(output.FlagsPtr)
}
`

const repeatedNativeABIPBGoSource = `// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.11
// 	protoc        v7.34.1
// source: repeated/v1/repeated.proto

package repeatedv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RepeatedRequest struct {
	state         protoimpl.MessageState ` + "`" + `protogen:"open.v1"` + "`" + `
	Scores        []int32                ` + "`" + `protobuf:"varint,1,rep,packed,name=scores,proto3" json:"scores,omitempty"` + "`" + `
	Flags         []bool                 ` + "`" + `protobuf:"varint,2,rep,packed,name=flags,proto3" json:"flags,omitempty"` + "`" + `
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RepeatedRequest) Reset() {
	*x = RepeatedRequest{}
	mi := &file_repeated_v1_repeated_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RepeatedRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepeatedRequest) ProtoMessage() {}

func (x *RepeatedRequest) ProtoReflect() protoreflect.Message {
	mi := &file_repeated_v1_repeated_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepeatedRequest.ProtoReflect.Descriptor instead.
func (*RepeatedRequest) Descriptor() ([]byte, []int) {
	return file_repeated_v1_repeated_proto_rawDescGZIP(), []int{0}
}

func (x *RepeatedRequest) GetScores() []int32 {
	if x != nil {
		return x.Scores
	}
	return nil
}

func (x *RepeatedRequest) GetFlags() []bool {
	if x != nil {
		return x.Flags
	}
	return nil
}

type RepeatedReply struct {
	state         protoimpl.MessageState ` + "`" + `protogen:"open.v1"` + "`" + `
	Scores        []int32                ` + "`" + `protobuf:"varint,1,rep,packed,name=scores,proto3" json:"scores,omitempty"` + "`" + `
	Flags         []bool                 ` + "`" + `protobuf:"varint,2,rep,packed,name=flags,proto3" json:"flags,omitempty"` + "`" + `
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RepeatedReply) Reset() {
	*x = RepeatedReply{}
	mi := &file_repeated_v1_repeated_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RepeatedReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepeatedReply) ProtoMessage() {}

func (x *RepeatedReply) ProtoReflect() protoreflect.Message {
	mi := &file_repeated_v1_repeated_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepeatedReply.ProtoReflect.Descriptor instead.
func (*RepeatedReply) Descriptor() ([]byte, []int) {
	return file_repeated_v1_repeated_proto_rawDescGZIP(), []int{1}
}

func (x *RepeatedReply) GetScores() []int32 {
	if x != nil {
		return x.Scores
	}
	return nil
}

func (x *RepeatedReply) GetFlags() []bool {
	if x != nil {
		return x.Flags
	}
	return nil
}

var File_repeated_v1_repeated_proto protoreflect.FileDescriptor

const file_repeated_v1_repeated_proto_rawDesc = "" +
	"\n" +
	"\x1arepeated/v1/repeated.proto\x12\x0frepeated.abi.v1\"?\n" +
	"\x0fRepeatedRequest\x12\x16\n" +
	"\x06scores\x18\x01 \x03(\x05R\x06scores\x12\x14\n" +
	"\x05flags\x18\x02 \x03(\bR\x05flags\"=\n" +
	"\rRepeatedReply\x12\x16\n" +
	"\x06scores\x18\x01 \x03(\x05R\x06scores\x12\x14\n" +
	"\x05flags\x18\x02 \x03(\bR\x05flagsB6Z4example.com/repeatednativeabi/repeated/v1;repeatedv1b\x06proto3"

var (
	file_repeated_v1_repeated_proto_rawDescOnce sync.Once
	file_repeated_v1_repeated_proto_rawDescData []byte
)

func file_repeated_v1_repeated_proto_rawDescGZIP() []byte {
	file_repeated_v1_repeated_proto_rawDescOnce.Do(func() {
		file_repeated_v1_repeated_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_repeated_v1_repeated_proto_rawDesc), len(file_repeated_v1_repeated_proto_rawDesc)))
	})
	return file_repeated_v1_repeated_proto_rawDescData
}

var file_repeated_v1_repeated_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_repeated_v1_repeated_proto_goTypes = []any{
	(*RepeatedRequest)(nil), // 0: repeated.abi.v1.RepeatedRequest
	(*RepeatedReply)(nil),   // 1: repeated.abi.v1.RepeatedReply
}
var file_repeated_v1_repeated_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_repeated_v1_repeated_proto_init() }
func file_repeated_v1_repeated_proto_init() {
	if File_repeated_v1_repeated_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_repeated_v1_repeated_proto_rawDesc), len(file_repeated_v1_repeated_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_repeated_v1_repeated_proto_goTypes,
		DependencyIndexes: file_repeated_v1_repeated_proto_depIdxs,
		MessageInfos:      file_repeated_v1_repeated_proto_msgTypes,
	}.Build()
	File_repeated_v1_repeated_proto = out.File
	file_repeated_v1_repeated_proto_goTypes = nil
	file_repeated_v1_repeated_proto_depIdxs = nil
}
`
