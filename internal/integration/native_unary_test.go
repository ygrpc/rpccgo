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
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/nativeunary\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n")
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !strings.Contains(name, ".runtime.rpccgo.go") &&
			!strings.Contains(name, ".server.native.rpccgo.go") &&
			!strings.Contains(name, ".client.cgo.rpccgo.go") {
			continue
		}
		writeFile(t, filepath.Join(tmp, name), generated.GetContent())
	}
	writeFile(t, filepath.Join(tmp, "test/v1/native_unary_stubs.go"), nativeUnaryStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/native_integration_reset.go"), nativeIntegrationResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/native_unary_test.go"), nativeUnaryFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "TestNativeUnary", "-count=1")
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
						fieldDescriptor("unsupported", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.Child"),
					},
				},
				{Name: proto.String("Child")},
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

type HelloRequest struct {
	Name string
	Payload []byte
	Enabled bool
}

type HelloReply struct {
	Accepted bool
	Payload []byte
	Note string
	ExtraPayload []byte
}

type UnsupportedReply struct {
	Payload []byte
	Note string
	Unsupported *Child
}

type Child struct{}
`

const nativeIntegrationResetSource = `package testv1

import rpcruntime "rpccgo/rpcruntime"

func ResetGreeterDispatcherForIntegrationTest() {
	greeterDispatcher = rpcruntime.Dispatcher[GreeterNativeAdapter]{}
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
}

func (s *recordingServer) SayHello(ctx context.Context, req *v1.HelloRequest) (*v1.HelloReply, error) {
	s.called = true
	if s.err != nil {
		return nil, s.err
	}
	if req.Name != "stage3" || string(req.Payload) != "bytes" || !req.Enabled {
		return nil, errors.New("request did not cross native bridge")
	}
	if s.response != nil {
		return s.response, nil
	}
	return &v1.HelloReply{Accepted: true, Payload: []byte("dispatcher:"+req.Name), Note: "ok"}, nil
}

func (s *recordingServer) SayUnsupported(ctx context.Context, req *v1.HelloRequest) (*v1.UnsupportedReply, error) {
	return &v1.UnsupportedReply{Payload: []byte("pinned"), Note: "note", Unsupported: &v1.Child{}}, nil
}

func TestNativeUnaryClientRoutesToGoNativeServer(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	server := &recordingServer{}
	if _, err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	name := []byte("stage3")
	payload := []byte("bytes")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if !server.called {
		t.Fatal("server was not called through dispatcher")
	}
	if output.Accepted != 1 {
		t.Fatalf("Accepted = %d, want 1", output.Accepted)
	}
	got := unsafe.Slice((*byte)(unsafe.Pointer(output.PayloadPtr)), output.PayloadLen)
	if string(got) != "dispatcher:stage3" {
		t.Fatalf("Payload = %q", got)
	}
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.ExtraPayloadPtr)
}

func TestNativeUnaryNegativeLengthStoresError(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &GreeterSayHelloNativeUnaryInput{NameLen: -1}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("negative length returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "cannot be negative") {
		t.Fatalf("negative length error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryOwnedStringAndBytesRelease(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	var released []uintptr
	rpcruntime.RegisterFreeCallback(func(ptr unsafe.Pointer) {
		released = append(released, uintptr(ptr))
	})

	name := []byte("stage3")
	payload := []byte("bytes")
	namePtr := uintptr(unsafe.Pointer(&name[0]))
	payloadPtr := uintptr(unsafe.Pointer(&payload[0]))
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr: namePtr,
		NameLen: int32(len(name)),
		NameOwnership: 1,
		PayloadPtr: payloadPtr,
		PayloadLen: int32(len(payload)),
		PayloadOwnership: 1,
		Enabled: 1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	if errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output); errID != 0 {
		t.Fatalf("CallGreeterSayHelloNativeUnary() errID = %d", errID)
	}
	if len(released) != 2 || released[0] != namePtr || released[1] != payloadPtr {
		t.Fatalf("released = %#v, want name then payload", released)
	}
	rpcruntime.Release(output.PayloadPtr)
	rpcruntime.Release(output.NotePtr)
	rpcruntime.Release(output.ExtraPayloadPtr)
}

func TestNativeUnaryPinFailureReleasesStagedOutput(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	shared := []byte("shared")
	server := &recordingServer{response: &v1.HelloReply{Accepted: true, Payload: shared, ExtraPayload: shared}}
	if _, err := v1.RegisterGreeterGoNativeServer(server); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	name := []byte("stage3")
	payload := []byte("bytes")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		PayloadPtr: uintptr(unsafe.Pointer(&payload[0])),
		PayloadLen: int32(len(payload)),
		Enabled: 1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
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
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	rpcruntime.ResetFreeCallbackForTesting()
	t.Cleanup(rpcruntime.ResetFreeCallbackForTesting)
	name := []byte("stage3")
	input := &GreeterSayHelloNativeUnaryInput{
		NamePtr: uintptr(unsafe.Pointer(&name[0])),
		NameLen: int32(len(name)),
		NameOwnership: 1,
	}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("owned release error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "ownership requires registered free func") {
		t.Fatalf("release error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryMissingActiveServerStoresError(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	input := &GreeterSayHelloNativeUnaryInput{}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("missing active server returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "active server") {
		t.Fatalf("missing active server error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryServerErrorStoresError(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGoNativeServer(&recordingServer{err: errors.New("server exploded")}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &GreeterSayHelloNativeUnaryInput{}
	output := &GreeterSayHelloNativeUnaryOutput{}
	errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("server error returned errID 0")
	}
	text, _, ok := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
	if !ok || !strings.Contains(string(text), "server exploded") {
		t.Fatalf("server error text = %q, ok=%v", text, ok)
	}
}

func TestNativeUnaryOutputStagingLeavesOutputUntouchedOnUnsupportedResponse(t *testing.T) {
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGoNativeServer(&recordingServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
	input := &GreeterSayUnsupportedNativeUnaryInput{}
	output := &GreeterSayUnsupportedNativeUnaryOutput{}
	errID := CallGreeterSayUnsupportedNativeUnary(context.Background(), input, output)
	if errID == 0 {
		t.Fatal("unsupported response returned errID 0")
	}
	if output.PayloadPtr != 0 || output.PayloadLen != 0 || output.NotePtr != 0 || output.NoteLen != 0 {
		t.Fatalf("output was partially committed on error: %#v", output)
	}
}
`
