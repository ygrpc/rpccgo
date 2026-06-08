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

func TestNativeUnaryClientRoutesToGoNativeServer(t *testing.T) {
	tmp := t.TempDir()
	plugin := newNativeUnaryTestPlugin(t)
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/nativeunary\n\ngo 1.24.4\n\nrequire (\n\tgoogle.golang.org/protobuf v1.36.11\n\trpccgo v0.0.0\n)\n\nreplace rpccgo => "+repoRoot+"\n")
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.sum"), string(goSum))
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !strings.Contains(name, ".runtime.rpccgo.go") &&
			!strings.Contains(name, ".codec.rpccgo.go") &&
			!strings.Contains(name, ".server.message.rpccgo.go") &&
			!strings.Contains(name, ".server.native.rpccgo.go") &&
			!strings.Contains(name, ".exports.cgo.rpccgo.go") &&
			!strings.Contains(name, ".client.native.cgo.rpccgo.go") {
			continue
		}
		writeFile(t, filepath.Join(tmp, name), generated.GetContent())
	}
	writeFile(t, filepath.Join(tmp, "test/v1/native_unary_stubs.go"), nativeUnaryStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_unary_cgo_client_bridge.go"), nativeUnaryCGOClientBridgeSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_unary_test.go"), nativeUnaryFixtureTestSource)

	cmd := exec.Command("go", "test", "-mod=mod", "./test/v1/cgo", "-run", "TestNativeUnary", "-count=1")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native unary fixture failed: %v\n%s", err, out)
	}
}

func newNativeUnaryTestPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()
	return newNativeUnaryTestPluginForPackage(t, "example.com/nativeunary/test/v1;testv1")
}

func newNativeUnaryTestPluginForPackage(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"test/v1/native_unary.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("test/v1/native_unary.proto"),
			Package: proto.String("test.v1"),
			Syntax:  proto.String("proto3"),
			Options: &descriptorpb.FileOptions{
				GoPackage: proto.String(goPackage),
			},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("HelloRequest"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("enabled", 3, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
				{
					Name: proto.String("HelloReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("note", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("extra_payload", 4, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
				{
					Name: proto.String("UnsupportedReply"),
					Field: []*descriptorpb.FieldDescriptorProto{
						fieldDescriptor("payload", 1, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("note", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
						fieldDescriptor("unsupported", 4, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{{
				Name: proto.String("Greeter"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:       proto.String("SayHello"),
					InputType:  proto.String(".test.v1.HelloRequest"),
					OutputType: proto.String(".test.v1.HelloReply"),
				}, {
					Name:       proto.String("SayUnsupported"),
					InputType:  proto.String(".test.v1.HelloRequest"),
					OutputType: proto.String(".test.v1.UnsupportedReply"),
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

func fieldDescriptor(name string, number int32, fieldType descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label, typeName string) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(number),
		Type:   fieldType.Enum(),
		Label:  label.Enum(),
	}
	if typeName != "" {
		field.TypeName = proto.String(typeName)
	}
	return field
}

func writeFile(t *testing.T, target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

const nativeUnaryStubSource = `package testv1

import (
	context "context"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type HelloRequest struct {
	Name string
	Payload []byte
	Enabled bool
}

func (*HelloRequest) ProtoReflect() protoreflect.Message { return nil }

type HelloReply struct {
	Accepted bool
	Payload []byte
	Note string
	ExtraPayload []byte
}

func (*HelloReply) ProtoReflect() protoreflect.Message { return nil }

type UnsupportedReply struct {
	Payload []byte
	Note string
	Unsupported []byte
}

func (*UnsupportedReply) ProtoReflect() protoreflect.Message { return nil }

type GreeterHandler interface {
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
	SayUnsupported(context.Context, *HelloRequest) (*UnsupportedReply, error)
}
type GreeterClient interface {
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
	SayUnsupported(context.Context, *HelloRequest) (*UnsupportedReply, error)
}

type GreeterServer interface {
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
	SayUnsupported(context.Context, *HelloRequest) (*UnsupportedReply, error)
}
`

const nativeIntegrationResetSource = `package testv1

import rpcruntime "rpccgo/rpcruntime"

func ResetGreeterServerForIntegrationTest() {
	_ = ClearGreeterServer()
	rpcruntime.ResetStreamSessionsForTesting()
}
`

const nativeUnaryCGOClientBridgeSource = `package main

/*
#include <stdint.h>
*/
import "C"

import context "context"

func CallGreeterSayHelloNativeUnary(ctx context.Context, NamePtr uintptr, NameLen int32, NameOwnership int32, PayloadPtr uintptr, PayloadLen int32, PayloadOwnership int32, Enabled int8, outAccepted *int8, outPayloadPtr *uintptr, outPayloadLen *int32, outNotePtr *uintptr, outNoteLen *int32, outExtraPayloadPtr *uintptr, outExtraPayloadLen *int32) int32 {
	var accepted C.int8_t
	var payloadPtr C.uintptr_t
	var payloadLen C.int32_t
	var payloadOwnership C.int32_t
	var notePtr C.uintptr_t
	var noteLen C.int32_t
	var noteOwnership C.int32_t
	var extraPayloadPtr C.uintptr_t
	var extraPayloadLen C.int32_t
	var extraPayloadOwnership C.int32_t
	errID := rpccgo_native_testv1_Greeter_SayHello(C.uintptr_t(NamePtr), C.int32_t(NameLen), C.int32_t(NameOwnership), C.uintptr_t(PayloadPtr), C.int32_t(PayloadLen), C.int32_t(PayloadOwnership), C.int8_t(Enabled), &accepted, &payloadPtr, &payloadLen, &payloadOwnership, &notePtr, &noteLen, &noteOwnership, &extraPayloadPtr, &extraPayloadLen, &extraPayloadOwnership)
	*outAccepted = int8(accepted)
	*outPayloadPtr = uintptr(payloadPtr)
	*outPayloadLen = int32(payloadLen)
	*outNotePtr = uintptr(notePtr)
	*outNoteLen = int32(noteLen)
	*outExtraPayloadPtr = uintptr(extraPayloadPtr)
	*outExtraPayloadLen = int32(extraPayloadLen)
	return int32(errID)
}

func CallGreeterSayUnsupportedNativeUnary(ctx context.Context, NamePtr uintptr, NameLen int32, NameOwnership int32, PayloadPtr uintptr, PayloadLen int32, PayloadOwnership int32, Enabled int8, outPayloadPtr *uintptr, outPayloadLen *int32, outNotePtr *uintptr, outNoteLen *int32, outUnsupportedPtr *uintptr, outUnsupportedLen *int32) int32 {
	var payloadPtr C.uintptr_t
	var payloadLen C.int32_t
	var payloadOwnership C.int32_t
	var notePtr C.uintptr_t
	var noteLen C.int32_t
	var noteOwnership C.int32_t
	var unsupportedPtr C.uintptr_t
	var unsupportedLen C.int32_t
	var unsupportedOwnership C.int32_t
	errID := rpccgo_native_testv1_Greeter_SayUnsupported(C.uintptr_t(NamePtr), C.int32_t(NameLen), C.int32_t(NameOwnership), C.uintptr_t(PayloadPtr), C.int32_t(PayloadLen), C.int32_t(PayloadOwnership), C.int8_t(Enabled), &payloadPtr, &payloadLen, &payloadOwnership, &notePtr, &noteLen, &noteOwnership, &unsupportedPtr, &unsupportedLen, &unsupportedOwnership)
	*outPayloadPtr = uintptr(payloadPtr)
	*outPayloadLen = int32(payloadLen)
	*outNotePtr = uintptr(notePtr)
	*outNoteLen = int32(noteLen)
	*outUnsupportedPtr = uintptr(unsupportedPtr)
	*outUnsupportedLen = int32(unsupportedLen)
	return int32(errID)
}
`

const nativeUnaryFixtureTestSource = `package main

import (
	context "context"
	errors "errors"
	strings "strings"
	"testing"
	"unsafe"

	v1 "example.com/nativeunary/test/v1"
	rpcruntime "rpccgo/rpcruntime"
)

type recordingServer struct {
	called bool
	err error
	response *v1.HelloReply
	received *v1.HelloRequest
	allowAnyRequest bool
}

func (s *recordingServer) SayHello(ctx context.Context, name *rpcruntime.RpcString, payload *rpcruntime.RpcBytes, enabled bool) (bool, []byte, string, []byte, error) {
	s.called = true
	req := &v1.HelloRequest{Name: name.SafeString(), Payload: payload.SafeBytes(), Enabled: enabled}
	s.received = req
	if s.err != nil {
		return false, nil, "", nil, s.err
	}
	if !s.allowAnyRequest && (req.Name != "native" || string(req.Payload) != "bytes" || !req.Enabled) {
		return false, nil, "", nil, errors.New("request did not reach native server")
	}
	if s.response != nil {
		return s.response.Accepted, s.response.Payload, s.response.Note, s.response.ExtraPayload, nil
	}
	return true, []byte("entry:"+req.Name), "ok", nil, nil
}

func (s *recordingServer) SayUnsupported(ctx context.Context, name *rpcruntime.RpcString, payload *rpcruntime.RpcBytes, enabled bool) ([]byte, string, []byte, error) {
	return []byte("pinned"), "note", []byte("unsupported"), nil
}

type sayHelloInput struct {
	NamePtr uintptr
	NameLen int32
	NameOwnership int32
	PayloadPtr uintptr
	PayloadLen int32
	PayloadOwnership int32
	Enabled int8
}

type sayHelloOutput struct {
	Accepted int8
	PayloadPtr uintptr
	PayloadLen int32
	NotePtr uintptr
	NoteLen int32
	ExtraPayloadPtr uintptr
	ExtraPayloadLen int32
}

func callSayHello(ctx context.Context, input *sayHelloInput, output *sayHelloOutput) int32 {
	if input == nil {
		input = &sayHelloInput{}
	}
	if output == nil {
		output = &sayHelloOutput{}
	}
	return CallGreeterSayHelloNativeUnary(ctx,
		input.NamePtr, input.NameLen, input.NameOwnership,
		input.PayloadPtr, input.PayloadLen, input.PayloadOwnership,
		input.Enabled,
		&output.Accepted,
		&output.PayloadPtr, &output.PayloadLen,
		&output.NotePtr, &output.NoteLen,
		&output.ExtraPayloadPtr, &output.ExtraPayloadLen,
	)
}

type unsupportedOutput struct {
	PayloadPtr uintptr
	PayloadLen int32
	NotePtr uintptr
	NoteLen int32
	UnsupportedPtr uintptr
	UnsupportedLen int32
}

func callSayUnsupported(ctx context.Context, input *sayHelloInput, output *unsupportedOutput) int32 {
	if input == nil {
		input = &sayHelloInput{}
	}
	if output == nil {
		output = &unsupportedOutput{}
	}
	return CallGreeterSayUnsupportedNativeUnary(ctx,
		input.NamePtr, input.NameLen, input.NameOwnership,
		input.PayloadPtr, input.PayloadLen, input.PayloadOwnership,
		input.Enabled,
		&output.PayloadPtr, &output.PayloadLen,
		&output.NotePtr, &output.NoteLen,
		&output.UnsupportedPtr, &output.UnsupportedLen,
	)
}

func TestNativeUnaryClientRoutesToGoNativeServer(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &recordingServer{}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if !server.called {
		t.Fatal("server was not called through runtime entry")
	}
	if output.Accepted != 1 {
		t.Fatalf("Accepted = %d, want 1", output.Accepted)
	}
	got := unsafe.Slice((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen)
	if string(got) != "entry:native" {
		t.Fatalf("Payload = %q", got)
	}
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.ExtraPayloadPtr)
}

func TestNativeUnaryTreatsNilPointerAsEmptyRequestInput(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	server := &recordingServer{allowAnyRequest: true}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &sayHelloInput{
		NamePtr: 0,
		NameLen: 5,
		PayloadPtr: 0,
		PayloadLen: 5,
		Enabled: 1,
	}
	output := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if !server.called || server.received == nil {
		t.Fatal("server did not receive request")
	}
	if server.received.Name != "" || len(server.received.Payload) != 0 || !server.received.Enabled {
		t.Fatalf("received request = %#v, want empty name/payload and enabled", server.received)
	}
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.ExtraPayloadPtr)
}

func TestNativeUnaryRejectsNegativeRequestLength(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: -1,
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &sayHelloOutput{}
	errID := callSayHello(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("negative length returned errID 0")
	}
	assertNativeErrContains(t, errID, "negative")
}

func TestNativeUnaryInputOwnershipRelease(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	var released []uintptr
	rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {
		released = append(released, uintptr(ptr))
	})

	borrowedName := []byte("native")
	borrowedPayload := []byte("bytes")
	borrowedInput := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&borrowedName[0])),
		NameLen: int32(len(borrowedName)),
		PayloadPtr: uintptr(unsafe.Pointer(&borrowedPayload[0])),
		PayloadLen: int32(len(borrowedPayload)),
		Enabled: 1,
	}
	borrowedOutput := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), borrowedInput, borrowedOutput); errID != 0 {
		t.Fatalf("borrowed CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if len(released) != 0 {
		t.Fatalf("borrowed released = %#v, want none", released)
	}
	rpcruntime.Release(borrowedOutput.PayloadPtr)
	rpcruntime.Release(borrowedOutput.NotePtr)
	rpcruntime.Release(borrowedOutput.ExtraPayloadPtr)

	ownedName := []byte("native")
	ownedPayload := []byte("bytes")
	ownedNamePtr := uintptr(unsafe.Pointer(&ownedName[0]))
	ownedPayloadPtr := uintptr(unsafe.Pointer(&ownedPayload[0]))
	ownedInput := &sayHelloInput{
		NamePtr: ownedNamePtr,
		NameLen: int32(len(ownedName)),
		NameOwnership: 1,
		PayloadPtr: ownedPayloadPtr,
		PayloadLen: int32(len(ownedPayload)),
		PayloadOwnership: 1,
		Enabled: 1,
	}
	ownedOutput := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), ownedInput, ownedOutput); errID != 0 {
		t.Fatalf("owned CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if len(released) != 2 || released[0] != ownedNamePtr || released[1] != ownedPayloadPtr {
		t.Fatalf("owned released = %#v, want name then payload", released)
	}
	rpcruntime.Release(ownedOutput.PayloadPtr)
	rpcruntime.Release(ownedOutput.NotePtr)
	rpcruntime.Release(ownedOutput.ExtraPayloadPtr)
}

func TestNativeUnaryOutputReleaseCanBeCalledOnce(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &sayHelloOutput{}
	if errID := callSayHello(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if output.PayloadPtr == 0 || output.NotePtr == 0 {
		t.Fatalf("output missing releasable pointers: %#v", output)
	}
	if !rpcruntime.Release(output.PayloadPtr) {
		t.Fatal("first payload release = false, want true")
	}
	if rpcruntime.Release(output.PayloadPtr) {
		t.Fatal("second payload release = true, want false")
	}
	if !rpcruntime.Release(output.NotePtr) {
		t.Fatal("first note release = false, want true")
	}
	if rpcruntime.Release(output.NotePtr) {
		t.Fatal("second note release = true, want false")
	}
	if rpcruntime.Release(output.ExtraPayloadPtr) {
		t.Fatal("empty extra payload release = true, want false")
	}
}

func TestNativeUnaryPinFailureReleasesStagedOutput(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	shared := []byte("shared")
	server := &recordingServer{response: &v1.HelloReply{Accepted: true, Payload: shared, ExtraPayload: shared}}
	if err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	name := []byte("native")
	payload := []byte("bytes")
	input := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &sayHelloOutput{}
	errID := callSayHello(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("duplicate response backing slice returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "already pinned") {
		t.Fatalf("duplicate response backing slice error text = %q, ok=%v", text, ok)
	}
	if output.PayloadPtr != 0 || output.PayloadLen != 0 || output.ExtraPayloadPtr != 0 || output.ExtraPayloadLen != 0 {
		t.Fatalf("output was partially committed on pin failure: %#v", output)
	}
	ptr, err := rpcruntime.PinBytes(shared)
	if err != nil {
		t.Fatalf("PinBytes(shared) after failed call = %v, want staged pin released", err)
	}
	rpcruntime.Release(ptr)
}

func TestNativeUnaryOwnedReleaseErrorStoresError(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := v1.RegisterGreeterGoNativeServer(&recordingServer{allowAnyRequest: true}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	name := []byte("native")
	input := &sayHelloInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		NameOwnership: 1,
	}
	output := &sayHelloOutput{}
	errID := callSayHello(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("owned release error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "ownership requires registered free func") {
		t.Fatalf("release error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryMissingActiveServerStoresError(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	input := &sayHelloInput{}
	output := &sayHelloOutput{}
	errID := callSayHello(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("missing registered server returned errID 0")
	}
	assertNativeErrContains(t, errID, "registered server")
}

func TestNativeUnaryServerErrorStoresError(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := v1.RegisterGreeterGoNativeServer(&recordingServer{err: errors.New("server exploded")}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &sayHelloInput{}
	output := &sayHelloOutput{}
	errID := callSayHello(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("server error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "server exploded") {
		t.Fatalf("server error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryMessageResponseFieldUsesBytesBoundary(t *testing.T) {
	v1.ResetGreeterServerForIntegrationTest()
	if err := v1.RegisterGreeterGoNativeServer(&recordingServer{allowAnyRequest: true}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &sayHelloInput{}
	output := &unsupportedOutput{}
	if errID := callSayUnsupported(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayUnsupportedNativeUnary() errID = %d", errID)
	}
	if output.PayloadPtr == 0 || output.PayloadLen == 0 || output.NotePtr == 0 || output.NoteLen == 0 || output.UnsupportedPtr == 0 || output.UnsupportedLen == 0 {
		t.Fatalf("output missing bytes-boundary fields: %#v", output)
	}
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.UnsupportedPtr)
}

func assertNativeErrContains(t *testing.T, errID int32, want string) {
	t.Helper()
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), want) {
		t.Fatalf("error text = %q, ok=%v, want substring %q", text, ok, want)
	}
}
`
