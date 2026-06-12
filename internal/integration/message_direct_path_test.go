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

func TestMessageServerStreamRejectsReadAfterFinish(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageServerStreamRejectsReadAfterFinish")
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

func TestMessageWrongTerminalOperationPreservesStreamHandle(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageWrongTerminalOperationPreservesStreamHandle")
}

func TestMessageServiceLevelRegistrationAccumulatesExistingCallbacks(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageServiceLevelRegistrationAccumulatesExistingCallbacks")
}

func TestNativeServiceLevelRegistrationAccumulatesExistingCallbacks(t *testing.T) {
	runMessageDirectPathFixture(t, "TestNativeServiceLevelRegistrationAccumulatesExistingCallbacks")
}

func TestMessagePartialRegistrationReportsRejectedMethods(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessagePartialRegistrationReportsRejectedMethods")
}

func TestNativePartialRegistrationReportsRejectedMethods(t *testing.T) {
	runMessageDirectPathFixture(t, "TestNativePartialRegistrationReportsRejectedMethods")
}

func TestMessageDirectConnectHandlerRegistrationRoutesUnaryAndStreaming(t *testing.T) {
	runMessageDirectRegistrationFixture(t, "@rpccgo: msg-connect\n", "TestDirectConnectHandlerRegistration")
}

func TestMessageDirectGRPCServerRegistrationRoutesUnaryAndStreaming(t *testing.T) {
	runMessageDirectRegistrationFixture(t, "@rpccgo: msg-grpc\n", "TestDirectGRPCServerRegistration")
}

func runMessageDirectRegistrationFixture(t *testing.T, serviceComment, testName string) {
	t.Helper()
	tmp := t.TempDir()
	plugin := newMessageDirectPathTestPluginWithServiceComment(t, "paths=source_relative", "example.com/messagedirect/test/v1;testv1", serviceComment)
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeMessageDirectPathGeneratedModule(t, tmp, plugin, "example.com/messagedirect")
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_stubs.go"), messageDirectPathHandlerStubSource)
	if strings.Contains(serviceComment, "msg-grpc") {
		writeFile(t, filepath.Join(tmp, "test/v1/message_integration_client_stubs.go"), messageDirectPathGRPCClientStubSource)
	} else {
		writeFile(t, filepath.Join(tmp, "test/v1/message_integration_client_stubs.go"), messageDirectPathClientStubSource)
	}
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_reset.go"), messageDirectPathMessageOnlyResetSource)
	if strings.Contains(serviceComment, "msg-connect") {
		writeFile(t, filepath.Join(tmp, "test/v1/message_direct_registration_test.go"), messageDirectConnectRegistrationTestSource)
	} else {
		writeFile(t, filepath.Join(tmp, "test/v1/message_direct_registration_test.go"), messageDirectGRPCRegistrationTestSource)
	}

	cmd := exec.Command("go", "test", "./test/v1", "-run", "^"+testName+"$", "-count=1", "-timeout=30s")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("message direct registration fixture %s failed: %v\n%s", testName, err, out)
	}
}

func runMessageDirectPathFixture(t *testing.T, testName string) {
	t.Helper()
	tmp := t.TempDir()
	plugin := newMessageDirectPathTestPlugin(t, "example.com/messagedirect/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeMessageDirectPathGeneratedModule(t, tmp, plugin, "example.com/messagedirect")
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_stubs.go"), messageDirectPathHandlerStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_client_stubs.go"), messageDirectPathClientStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_reset.go"), messageDirectPathResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_callbacks.go"), messageDirectPathFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_cgo_client_bridge.go"), messageDirectPathCGOClientBridgeSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_test.go"), messageDirectPathFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "^"+testName+"$", "-count=1", "-timeout=30s")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("message direct path fixture %s failed: %v\n%s", testName, err, out)
	}
}

func newMessageDirectPathTestPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	return newMessageDirectPathTestPluginWithServiceComment(t, "paths=source_relative", goPackage, "@rpccgo: native\n")
}

func newMessageDirectPathTestPluginWithParameter(t *testing.T, parameter, goPackage string) *protogen.Plugin {
	t.Helper()
	return newMessageDirectPathTestPluginWithServiceComment(t, parameter, goPackage, "@rpccgo: native\n")
}

func newMessageDirectPathTestPluginWithServiceComment(t *testing.T, parameter, goPackage, serviceComment string) *protogen.Plugin {
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
					LeadingComments: proto.String(serviceComment),
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
	writeFile(t, filepath.Join(root, "go.mod"), "module "+module+"\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\tgoogle.golang.org/grpc v1.79.3\n\tgoogle.golang.org/protobuf v1.36.11\n\tgithub.com/ygrpc/rpccgo v0.0.0\n)\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n")
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	writeFile(t, filepath.Join(root, "go.sum"), string(goSum))
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		include := strings.Contains(name, ".runtime.rpccgo.go") ||
			strings.Contains(name, ".codec.rpccgo.go") ||
			strings.Contains(name, ".server.message.rpccgo.go") ||
			strings.Contains(name, ".server.native.rpccgo.go") ||
			strings.Contains(name, ".exports.cgo.rpccgo.go") ||
			strings.Contains(name, ".server.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".client.native.cgo.rpccgo.go") ||
			strings.Contains(name, ".message.cgo.rpccgo.go")
		if !include {
			continue
		}
		writeFile(t, filepath.Join(root, name), generated.GetContent())
	}
}

const messageDirectPathClientStubSource = `package testv1

import (
	context "context"

	connect "connectrpc.com/connect"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GreeterClient interface {
	Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	Upload(context.Context) (*connect.ClientStreamForClientSimple[emptypb.Empty, emptypb.Empty], error)
	List(context.Context, *emptypb.Empty) (*connect.ServerStreamForClient[emptypb.Empty], error)
	Chat(context.Context) (*connect.BidiStreamForClientSimple[emptypb.Empty, emptypb.Empty], error)
}

`

const messageDirectPathGRPCClientStubSource = `package testv1

import (
	context "context"

	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GreeterClient interface {
	Unary(context.Context, *emptypb.Empty, ...grpc.CallOption) (*emptypb.Empty, error)
	Upload(context.Context, ...grpc.CallOption) (grpc.ClientStreamingClient[emptypb.Empty, emptypb.Empty], error)
	List(context.Context, *emptypb.Empty, ...grpc.CallOption) (grpc.ServerStreamingClient[emptypb.Empty], error)
	Chat(context.Context, ...grpc.CallOption) (grpc.BidiStreamingClient[emptypb.Empty, emptypb.Empty], error)
}

`

const messageDirectPathHandlerStubSource = `package testv1

import (
	context "context"
	errors "errors"
	io "io"
	http "net/http"
	strings "strings"

	connect "connectrpc.com/connect"
	metadata "google.golang.org/grpc/metadata"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GreeterHandler interface {
	Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	Upload(context.Context, *connect.ClientStream[emptypb.Empty]) (*emptypb.Empty, error)
	List(context.Context, *emptypb.Empty, *connect.ServerStream[emptypb.Empty]) error
	Chat(context.Context, *connect.BidiStream[emptypb.Empty, emptypb.Empty]) error
}

type GreeterServer interface {
	Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	Upload(Greeter_UploadServer) error
	List(*emptypb.Empty, Greeter_ListServer) error
	Chat(Greeter_ChatServer) error
}

type Greeter_UploadServer interface {
	Context() context.Context
	Recv() (*emptypb.Empty, error)
	RecvMsg(any) error
	SendAndClose(*emptypb.Empty) error
	SendMsg(any) error
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)
}
type Greeter_ListServer interface {
	Context() context.Context
	Send(*emptypb.Empty) error
	SendMsg(any) error
	RecvMsg(any) error
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)
}
type Greeter_ChatServer interface {
	Context() context.Context
	Recv() (*emptypb.Empty, error)
	RecvMsg(any) error
	Send(*emptypb.Empty) error
	SendMsg(any) error
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)
}

const GreeterUnaryConnectProcedure = "/test.v1.Greeter/Unary"
const GreeterUploadConnectProcedure = "/test.v1.Greeter/Upload"
const GreeterListConnectProcedure = "/test.v1.Greeter/List"
const GreeterChatConnectProcedure = "/test.v1.Greeter/Chat"

func GreeterEntryForIntegrationTest() GreeterHandler { return greeterEntryHandler{} }

type greeterEntryHandler struct{}

func (greeterEntryHandler) Unary(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	resp, err := InvokeGreeterMessageUnary(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (greeterEntryHandler) Upload(ctx context.Context, stream *connect.ClientStream[emptypb.Empty]) (*emptypb.Empty, error) {
	handle, err := StartGreeterMessageUpload(ctx)
	if err != nil {
		return nil, err
	}
	for stream.Receive() {
		if err := SendGreeterMessageUpload(ctx, handle, &emptypb.Empty{}); err != nil {
			_ = CancelGreeterMessageUpload(ctx, handle)
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		_ = CancelGreeterMessageUpload(ctx, handle)
		return nil, err
	}
	resp, err := FinishGreeterMessageUpload(ctx, handle)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (greeterEntryHandler) List(ctx context.Context, req *emptypb.Empty, stream *connect.ServerStream[emptypb.Empty]) error {
	handle, err := StartGreeterMessageList(ctx, req)
	if err != nil {
		return err
	}
	for {
		resp, err := RecvGreeterMessageList(ctx, handle)
		if err == io.EOF {
			return FinishGreeterMessageList(ctx, handle)
		}
		if err != nil {
			_ = CancelGreeterMessageList(ctx, handle)
			return err
		}
		if err := stream.Send(resp); err != nil {
			_ = CancelGreeterMessageList(ctx, handle)
			return err
		}
	}
}

func (greeterEntryHandler) Chat(ctx context.Context, stream *connect.BidiStream[emptypb.Empty, emptypb.Empty]) error {
	handle, err := StartGreeterMessageChat(ctx)
	if err != nil {
		return err
	}
	type chatResponse struct {
		msg *emptypb.Empty
		err error
	}
	requestErr := make(chan error, 1)
	response := make(chan chatResponse, 1)
	go func() {
		for {
			_, err := stream.Receive()
			if errors.Is(err, io.EOF) || err != nil && strings.Contains(err.Error(), "EOF") {
				requestErr <- CloseSendGreeterMessageChat(ctx, handle)
				return
			}
			if err != nil {
				requestErr <- err
				return
			}
			if err := SendGreeterMessageChat(ctx, handle, &emptypb.Empty{}); err != nil {
				requestErr <- err
				return
			}
		}
	}()
	go func() {
		for {
			resp, err := RecvGreeterMessageChat(ctx, handle)
			if err != nil {
				response <- chatResponse{err: err}
				return
			}
			response <- chatResponse{msg: resp}
		}
	}()
	for {
		select {
		case err := <-requestErr:
			requestErr = nil
			if err != nil {
				if strings.Contains(err.Error(), "EOF") {
					if err := FinishGreeterMessageChat(ctx, handle); err != nil && !strings.Contains(err.Error(), "EOF") {
						return err
					}
					return nil
				}
				_ = CancelGreeterMessageChat(ctx, handle)
				return err
			}
		case resp := <-response:
			if errors.Is(resp.err, io.EOF) || resp.err != nil && strings.Contains(resp.err.Error(), "EOF") {
				if err := FinishGreeterMessageChat(ctx, handle); err != nil && !strings.Contains(err.Error(), "EOF") {
					return err
				}
				return nil
			}
			if resp.err != nil {
				_ = CancelGreeterMessageChat(ctx, handle)
				return resp.err
			}
			if err := stream.Send(resp.msg); err != nil {
				_ = CancelGreeterMessageChat(ctx, handle)
				return err
			}
		}
	}
}

func NewGreeterHandler(svc GreeterHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(GreeterUnaryConnectProcedure, connect.NewUnaryHandler(GreeterUnaryConnectProcedure, func(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
		resp, err := svc.Unary(ctx, req.Msg)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(resp), nil
	}, opts...))
	mux.Handle(GreeterUploadConnectProcedure, connect.NewClientStreamHandler(GreeterUploadConnectProcedure, func(ctx context.Context, stream *connect.ClientStream[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
		resp, err := svc.Upload(ctx, stream)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(resp), nil
	}, opts...))
	mux.Handle(GreeterListConnectProcedure, connect.NewServerStreamHandler(GreeterListConnectProcedure, func(ctx context.Context, req *connect.Request[emptypb.Empty], stream *connect.ServerStream[emptypb.Empty]) error {
		return svc.List(ctx, req.Msg, stream)
	}, opts...))
	mux.Handle(GreeterChatConnectProcedure, connect.NewBidiStreamHandler(GreeterChatConnectProcedure, func(ctx context.Context, stream *connect.BidiStream[emptypb.Empty, emptypb.Empty]) error {
		return svc.Chat(ctx, stream)
	}, opts...))
	return "/test.v1.Greeter/", mux
}
`

const messageDirectConnectRegistrationTestSource = `package testv1

import (
	context "context"
	io "io"
	testing "testing"

	connect "connectrpc.com/connect"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type directConnectGreeter struct{}

func (directConnectGreeter) Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error) { return &emptypb.Empty{}, nil }
func (directConnectGreeter) Upload(ctx context.Context, stream *connect.ClientStream[emptypb.Empty]) (*emptypb.Empty, error) {
	for stream.Receive() {
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
func (directConnectGreeter) List(context.Context, *emptypb.Empty, *connect.ServerStream[emptypb.Empty]) error { return nil }
func (directConnectGreeter) Chat(context.Context, *connect.BidiStream[emptypb.Empty, emptypb.Empty]) error { return nil }

func TestDirectConnectHandlerRegistration(t *testing.T) {
	ResetGreeterServerForIntegrationTest()
	if err := RegisterGreeterConnectHandler(directConnectGreeter{}); err != nil {
		t.Fatalf("RegisterGreeterConnectHandler() error = %v", err)
	}
	if _, err := InvokeGreeterMessageUnary(context.Background(), &emptypb.Empty{}); err != nil {
		t.Fatalf("InvokeGreeterMessageUnary() error = %v", err)
	}
	uploadHandle, err := StartGreeterMessageUpload(context.Background())
	if err != nil {
		t.Fatalf("StartGreeterMessageUpload() error = %v", err)
	}
	if err := SendGreeterMessageUpload(context.Background(), uploadHandle, &emptypb.Empty{}); err != nil {
		t.Fatalf("upload Send() error = %v", err)
	}
	if _, err := FinishGreeterMessageUpload(context.Background(), uploadHandle); err != nil {
		t.Fatalf("upload Finish() error = %v", err)
	}
	listHandle, err := StartGreeterMessageList(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("StartGreeterMessageList() error = %v", err)
	}
	if _, err := RecvGreeterMessageList(context.Background(), listHandle); err != io.EOF {
		t.Fatalf("list Recv() error = %v, want EOF", err)
	}
	if err := FinishGreeterMessageList(context.Background(), listHandle); err != nil {
		t.Fatalf("list Finish() error = %v", err)
	}
	chatHandle, err := StartGreeterMessageChat(context.Background())
	if err != nil {
		t.Fatalf("StartGreeterMessageChat() error = %v", err)
	}
	if err := CloseSendGreeterMessageChat(context.Background(), chatHandle); err != nil {
		t.Fatalf("chat CloseSend() error = %v", err)
	}
	if _, err := RecvGreeterMessageChat(context.Background(), chatHandle); err != io.EOF {
		t.Fatalf("chat Recv() error = %v, want EOF", err)
	}
	if err := FinishGreeterMessageChat(context.Background(), chatHandle); err != nil {
		t.Fatalf("chat Finish() error = %v", err)
	}
}
`

const messageDirectGRPCRegistrationTestSource = `package testv1

import (
	context "context"
	io "io"
	testing "testing"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type directGRPCGreeter struct{}

func (directGRPCGreeter) Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error) { return &emptypb.Empty{}, nil }
func (directGRPCGreeter) Upload(stream Greeter_UploadServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&emptypb.Empty{})
		}
		if err != nil {
			return err
		}
	}
}
func (directGRPCGreeter) List(*emptypb.Empty, Greeter_ListServer) error { return nil }
func (directGRPCGreeter) Chat(Greeter_ChatServer) error { return nil }

func TestDirectGRPCServerRegistration(t *testing.T) {
	ResetGreeterServerForIntegrationTest()
	if err := RegisterGreeterGRPCServer(directGRPCGreeter{}); err != nil {
		t.Fatalf("RegisterGreeterGRPCServer() error = %v", err)
	}
	if _, err := InvokeGreeterMessageUnary(context.Background(), &emptypb.Empty{}); err != nil {
		t.Fatalf("InvokeGreeterMessageUnary() error = %v", err)
	}
	uploadHandle, err := StartGreeterMessageUpload(context.Background())
	if err != nil {
		t.Fatalf("StartGreeterMessageUpload() error = %v", err)
	}
	if err := SendGreeterMessageUpload(context.Background(), uploadHandle, &emptypb.Empty{}); err != nil {
		t.Fatalf("upload Send() error = %v", err)
	}
	if _, err := FinishGreeterMessageUpload(context.Background(), uploadHandle); err != nil {
		t.Fatalf("upload Finish() error = %v", err)
	}
	listHandle, err := StartGreeterMessageList(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("StartGreeterMessageList() error = %v", err)
	}
	if _, err := RecvGreeterMessageList(context.Background(), listHandle); err != io.EOF {
		t.Fatalf("list Recv() error = %v, want EOF", err)
	}
	if err := FinishGreeterMessageList(context.Background(), listHandle); err != nil {
		t.Fatalf("list Finish() error = %v", err)
	}
	chatHandle, err := StartGreeterMessageChat(context.Background())
	if err != nil {
		t.Fatalf("StartGreeterMessageChat() error = %v", err)
	}
	if err := CloseSendGreeterMessageChat(context.Background(), chatHandle); err != nil {
		t.Fatalf("chat CloseSend() error = %v", err)
	}
	if _, err := RecvGreeterMessageChat(context.Background(), chatHandle); err != io.EOF {
		t.Fatalf("chat Recv() error = %v, want EOF", err)
	}
	if err := FinishGreeterMessageChat(context.Background(), chatHandle); err != nil {
		t.Fatalf("chat Finish() error = %v", err)
	}
}
`

const messageDirectPathResetSource = `package testv1

import rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"

func ResetGreeterServerForIntegrationTest() {
	_ = ClearGreeterServer()
	rpcruntime.ResetStreamSessionsForTesting()
}
`

const messageDirectPathMessageOnlyResetSource = `package testv1

import rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"

func ResetGreeterServerForIntegrationTest() {
	_ = ClearGreeterServer()
	rpcruntime.ResetStreamSessionsForTesting()
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
typedef int32_t (*GreeterListCGOMessageServerStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterListCGOMessageServerStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamSendCallback)(int32_t stream, uintptr_t request_ptr, int32_t request_len);
typedef int32_t (*GreeterChatCGOMessageBidiStreamRecvCallback)(int32_t stream, uintptr_t* response_ptr, int32_t* response_len);
typedef int32_t (*GreeterChatCGOMessageBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGOMessageBidiStreamCancelCallback)(int32_t stream);

typedef struct GreeterCGOMessageServerCallbacks {
	GreeterUnaryCGOMessageUnaryCallback Unary;
	GreeterUploadCGOMessageClientStreamStartCallback UploadStart;
	GreeterUploadCGOMessageClientStreamSendCallback UploadSend;
	GreeterUploadCGOMessageClientStreamFinishCallback UploadFinish;
	GreeterUploadCGOMessageClientStreamCancelCallback UploadCancel;
	GreeterListCGOMessageServerStreamStartCallback ListStart;
	GreeterListCGOMessageServerStreamRecvCallback ListRecv;
	GreeterListCGOMessageServerStreamFinishCallback ListFinish;
	GreeterListCGOMessageServerStreamCancelCallback ListCancel;
	GreeterChatCGOMessageBidiStreamStartCallback ChatStart;
	GreeterChatCGOMessageBidiStreamSendCallback ChatSend;
	GreeterChatCGOMessageBidiStreamRecvCallback ChatRecv;
	GreeterChatCGOMessageBidiStreamCloseSendCallback ChatCloseSend;
	GreeterChatCGOMessageBidiStreamFinishCallback ChatFinish;
	GreeterChatCGOMessageBidiStreamCancelCallback ChatCancel;
} GreeterCGOMessageServerCallbacks;

typedef int32_t (*GreeterUnaryCGONativeUnaryCallback)(void);
typedef int32_t (*GreeterUploadCGONativeClientStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamSendCallback)(int32_t stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterUploadCGONativeClientStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterListCGONativeServerStreamRecvCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamFinishCallback)(int32_t stream);
typedef int32_t (*GreeterListCGONativeServerStreamCancelCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamStartCallback)(int32_t* stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamRecvCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamCloseSendCallback)(int32_t stream);
typedef int32_t (*GreeterChatCGONativeBidiStreamFinishCallback)(int32_t stream);
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
	GreeterListCGONativeServerStreamFinishCallback ListFinish;
	GreeterListCGONativeServerStreamCancelCallback ListCancel;
	GreeterChatCGONativeBidiStreamStartCallback ChatStart;
	GreeterChatCGONativeBidiStreamSendCallback ChatSend;
	GreeterChatCGONativeBidiStreamRecvCallback ChatRecv;
	GreeterChatCGONativeBidiStreamCloseSendCallback ChatCloseSend;
	GreeterChatCGONativeBidiStreamFinishCallback ChatFinish;
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
static int listFinishes;
static int listCancels;
static int chatStarts;
static int chatSends;
static int chatRecvs;
static int chatCloseSends;
static int chatFinishes;
static int chatCancels;
static int nativeUnaryCalls;
static int nativeUploadStarts;
static int nativeUploadSends;
static int nativeUploadFinishes;
static int nativeUploadCancels;
static int nativeListStarts;
static int nativeListRecvs;
static int nativeListFinishs;
static int nativeListCancels;
static int nativeChatStarts;
static int nativeChatSends;
static int nativeChatRecvs;
static int nativeChatCloseSends;
static int nativeChatFinishs;
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
	listFinishes = 0;
	listCancels = 0;
	chatStarts = 0;
	chatSends = 0;
	chatRecvs = 0;
	chatCloseSends = 0;
	chatFinishes = 0;
	chatCancels = 0;
	nativeUnaryCalls = 0;
	nativeUploadStarts = 0;
	nativeUploadSends = 0;
	nativeUploadFinishes = 0;
	nativeUploadCancels = 0;
	nativeListStarts = 0;
	nativeListRecvs = 0;
	nativeListFinishs = 0;
	nativeListCancels = 0;
	nativeChatStarts = 0;
	nativeChatSends = 0;
	nativeChatRecvs = 0;
	nativeChatCloseSends = 0;
	nativeChatFinishs = 0;
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

static int32_t greeterListFinish(int32_t stream) {
	listFinishes++;
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

static int32_t greeterChatFinish(int32_t stream) {
	chatFinishes++;
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
	callbacks.ListFinish = greeterListFinish;
	callbacks.ListCancel = greeterListCancel;
	callbacks.ChatStart = greeterChatStart;
	callbacks.ChatSend = greeterChatSend;
	callbacks.ChatRecv = greeterChatRecv;
	callbacks.ChatCloseSend = greeterChatCloseSend;
	callbacks.ChatFinish = greeterChatFinish;
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

static int32_t nativeGreeterListFinish(int32_t stream) {
	nativeListFinishs++;
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

static int32_t nativeGreeterChatFinish(int32_t stream) {
	nativeChatFinishs++;
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
	callbacks.ListFinish = nativeGreeterListFinish;
	callbacks.ListCancel = nativeGreeterListCancel;
	callbacks.ChatStart = nativeGreeterChatStart;
	callbacks.ChatSend = nativeGreeterChatSend;
	callbacks.ChatRecv = nativeGreeterChatRecv;
	callbacks.ChatCloseSend = nativeGreeterChatCloseSend;
	callbacks.ChatFinish = nativeGreeterChatFinish;
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
static int getListFinishs(void) { return listFinishes; }
static int getListCancels(void) { return listCancels; }
static int getChatStarts(void) { return chatStarts; }
static int getChatSends(void) { return chatSends; }
static int getChatRecvs(void) { return chatRecvs; }
static int getChatCloseSends(void) { return chatCloseSends; }
static int getChatFinishs(void) { return chatFinishes; }
static int getChatCancels(void) { return chatCancels; }
static int getNativeUnaryCalls(void) { return nativeUnaryCalls; }
static int getNativeUploadStarts(void) { return nativeUploadStarts; }
static int getNativeUploadSends(void) { return nativeUploadSends; }
static int getNativeUploadFinishes(void) { return nativeUploadFinishes; }
static int getNativeUploadCancels(void) { return nativeUploadCancels; }
static int getNativeListStarts(void) { return nativeListStarts; }
static int getNativeListRecvs(void) { return nativeListRecvs; }
static int getNativeListFinishs(void) { return nativeListFinishs; }
static int getNativeListCancels(void) { return nativeListCancels; }
static int getNativeChatStarts(void) { return nativeChatStarts; }
static int getNativeChatSends(void) { return nativeChatSends; }
static int getNativeChatRecvs(void) { return nativeChatRecvs; }
static int getNativeChatCloseSends(void) { return nativeChatCloseSends; }
static int getNativeChatFinishs(void) { return nativeChatFinishs; }
static int getNativeChatCancels(void) { return nativeChatCancels; }
*/
import "C"

import (
	errors "errors"

	v1 "example.com/messagedirect/test/v1"
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
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
	v1.ResetGreeterServerForIntegrationTest()
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
	v1.ResetGreeterServerForIntegrationTest()
	C.resetMessageCounters()
	C.setNativeUnaryError(0)
	callbacks := C.greeterNativeCallbacks()
	return registerGreeterNativeCallbacks(callbacks)
}

func registerGreeterMessageCallbacks(callbacks C.GreeterCGOMessageServerCallbacks) error {
	errID := rpccgo_msg_testv1_Greeter_register(callbacks.Unary, callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel, callbacks.ListStart, callbacks.ListRecv, callbacks.ListFinish, callbacks.ListCancel, callbacks.ChatStart, callbacks.ChatSend, callbacks.ChatRecv, callbacks.ChatCloseSend, callbacks.ChatFinish, callbacks.ChatCancel)
	if errID != 0 {
		return cgoFixtureStoredError(errID)
	}
	return nil
}

func registerGreeterMessageUnaryOnlyForIntegration() error {
	v1.ResetGreeterServerForIntegrationTest()
	C.resetMessageCounters()
	C.setUnaryError(0)
	C.setUnaryStoredError(0)
	C.setMessageStreamEOFMode(0)
	C.setInvalidMessageResponse(0)
	callbacks := C.greeterMessageCallbacks()
	errID := rpccgo_msg_testv1_Greeter_register_Unary(callbacks.Unary)
	if errID == 0 {
		return nil
	}
	return cgoFixtureStoredError(errID)
}

func registerGreeterMessageUploadOnlyServiceLevelWithoutResetForIntegration() error {
	C.setUnaryError(0)
	C.setUnaryStoredError(0)
	C.setMessageStreamEOFMode(0)
	C.setInvalidMessageResponse(0)
	callbacks := C.greeterMessageCallbacks()
	errID := rpccgo_msg_testv1_Greeter_register(nil, callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if errID == 0 {
		return nil
	}
	return cgoFixtureStoredError(errID)
}

func registerGreeterMessagePartialStreamingMethodsForIntegration() error {
	v1.ResetGreeterServerForIntegrationTest()
	C.resetMessageCounters()
	C.setUnaryError(0)
	C.setUnaryStoredError(0)
	C.setMessageStreamEOFMode(0)
	C.setInvalidMessageResponse(0)
	callbacks := C.greeterMessageCallbacks()
	errID := rpccgo_msg_testv1_Greeter_register(nil, callbacks.UploadStart, nil, callbacks.UploadFinish, callbacks.UploadCancel, nil, nil, nil, nil, callbacks.ChatStart, callbacks.ChatSend, nil, callbacks.ChatCloseSend, callbacks.ChatFinish, callbacks.ChatCancel)
	if errID == 0 {
		return nil
	}
	return cgoFixtureStoredError(errID)
}

func registerGreeterNativeCallbacks(callbacks C.GreeterCGONativeServerCallbacks) error {
	errID := rpccgo_native_testv1_Greeter_register(callbacks.Unary, callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel, callbacks.ListStart, callbacks.ListRecv, callbacks.ListFinish, callbacks.ListCancel, callbacks.ChatStart, callbacks.ChatSend, callbacks.ChatRecv, callbacks.ChatCloseSend, callbacks.ChatFinish, callbacks.ChatCancel)
	if errID != 0 {
		return cgoFixtureStoredError(errID)
	}
	return nil
}

func registerGreeterNativeUnaryOnlyForIntegration() error {
	v1.ResetGreeterServerForIntegrationTest()
	C.resetMessageCounters()
	C.setNativeUnaryError(0)
	callbacks := C.greeterNativeCallbacks()
	errID := rpccgo_native_testv1_Greeter_register_Unary(callbacks.Unary)
	if errID == 0 {
		return nil
	}
	return cgoFixtureStoredError(errID)
}

func registerGreeterNativeUploadOnlyServiceLevelWithoutResetForIntegration() error {
	C.setNativeUnaryError(0)
	callbacks := C.greeterNativeCallbacks()
	errID := rpccgo_native_testv1_Greeter_register(nil, callbacks.UploadStart, callbacks.UploadSend, callbacks.UploadFinish, callbacks.UploadCancel, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if errID == 0 {
		return nil
	}
	return cgoFixtureStoredError(errID)
}

func registerGreeterNativePartialStreamingMethodsForIntegration() error {
	v1.ResetGreeterServerForIntegrationTest()
	C.resetMessageCounters()
	C.setNativeUnaryError(0)
	callbacks := C.greeterNativeCallbacks()
	errID := rpccgo_native_testv1_Greeter_register(nil, callbacks.UploadStart, nil, callbacks.UploadFinish, callbacks.UploadCancel, nil, nil, nil, nil, callbacks.ChatStart, callbacks.ChatSend, nil, callbacks.ChatCloseSend, callbacks.ChatFinish, callbacks.ChatCancel)
	if errID == 0 {
		return nil
	}
	return cgoFixtureStoredError(errID)
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
func greeterMessageListFinishsForIntegration() int { return int(C.getListFinishs()) }
func greeterMessageListCancelsForIntegration() int { return int(C.getListCancels()) }
func greeterMessageChatStartsForIntegration() int { return int(C.getChatStarts()) }
func greeterMessageChatSendsForIntegration() int { return int(C.getChatSends()) }
func greeterMessageChatRecvsForIntegration() int { return int(C.getChatRecvs()) }
func greeterMessageChatCloseSendsForIntegration() int { return int(C.getChatCloseSends()) }
func greeterMessageChatFinishsForIntegration() int { return int(C.getChatFinishs()) }
func greeterMessageChatCancelsForIntegration() int { return int(C.getChatCancels()) }
func greeterNativeUnaryCallsForIntegration() int { return int(C.getNativeUnaryCalls()) }
func greeterNativeUploadStartsForIntegration() int { return int(C.getNativeUploadStarts()) }
func greeterNativeUploadSendsForIntegration() int { return int(C.getNativeUploadSends()) }
func greeterNativeUploadFinishesForIntegration() int { return int(C.getNativeUploadFinishes()) }
func greeterNativeUploadCancelsForIntegration() int { return int(C.getNativeUploadCancels()) }
func greeterNativeListStartsForIntegration() int { return int(C.getNativeListStarts()) }
func greeterNativeListRecvsForIntegration() int { return int(C.getNativeListRecvs()) }
func greeterNativeListFinishsForIntegration() int { return int(C.getNativeListFinishs()) }
func greeterNativeListCancelsForIntegration() int { return int(C.getNativeListCancels()) }
func greeterNativeChatStartsForIntegration() int { return int(C.getNativeChatStarts()) }
func greeterNativeChatSendsForIntegration() int { return int(C.getNativeChatSends()) }
func greeterNativeChatRecvsForIntegration() int { return int(C.getNativeChatRecvs()) }
func greeterNativeChatCloseSendsForIntegration() int { return int(C.getNativeChatCloseSends()) }
func greeterNativeChatFinishsForIntegration() int { return int(C.getNativeChatFinishs()) }
func greeterNativeChatCancelsForIntegration() int { return int(C.getNativeChatCancels()) }
`

const messageDirectPathCGOClientBridgeSource = `package main

/*
#include <stdint.h>
*/
import "C"

import (
	context "context"
	errors "errors"

	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

type greeterMessageOutput struct {
	DataPtr uintptr
	DataLen int32
}

func callGreeterUnaryMessageUnary(ctx context.Context, requestPtr uintptr, requestLen int32, output *greeterMessageOutput) int32 {
	_ = ctx
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))
	}
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := rpccgo_msg_testv1_Greeter_Unary(C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen)
	output.DataPtr = uintptr(responsePtr)
	output.DataLen = int32(responseLen)
	return int32(errID)
}

func startGreeterUploadMessageClientStream(ctx context.Context) (int32, int32) {
	_ = ctx
	var handle C.int32_t
	errID := rpccgo_msg_testv1_Greeter_Upload_start(&handle)
	return int32(handle), int32(errID)
}

func sendGreeterUploadMessageClientStream(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_Upload_send(C.int32_t(handle), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
}

func finishGreeterUploadMessageClientStream(ctx context.Context, handle int32, output *greeterMessageOutput) int32 {
	_ = ctx
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))
	}
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := rpccgo_msg_testv1_Greeter_Upload_finish(C.int32_t(handle), &responsePtr, &responseLen)
	output.DataPtr = uintptr(responsePtr)
	output.DataLen = int32(responseLen)
	return int32(errID)
}

func cancelGreeterUploadMessageClientStream(ctx context.Context, handle int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_Upload_cancel(C.int32_t(handle)))
}

func startGreeterListMessageServerStream(ctx context.Context, requestPtr uintptr, requestLen int32) (int32, int32) {
	_ = ctx
	var handle C.int32_t
	errID := rpccgo_msg_testv1_Greeter_List_start(C.uintptr_t(requestPtr), C.int32_t(requestLen), &handle)
	return int32(handle), int32(errID)
}

func readGreeterListMessageServerStream(ctx context.Context, handle int32, output *greeterMessageOutput) int32 {
	_ = ctx
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))
	}
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := rpccgo_msg_testv1_Greeter_List_read(C.int32_t(handle), &responsePtr, &responseLen)
	output.DataPtr = uintptr(responsePtr)
	output.DataLen = int32(responseLen)
	return int32(errID)
}

func finishGreeterListMessageServerStream(ctx context.Context, handle int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_List_finish(C.int32_t(handle)))
}

func cancelGreeterListMessageServerStream(ctx context.Context, handle int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_List_cancel(C.int32_t(handle)))
}

func startGreeterChatMessageBidiStream(ctx context.Context) (int32, int32) {
	_ = ctx
	var handle C.int32_t
	errID := rpccgo_msg_testv1_Greeter_Chat_start(&handle)
	return int32(handle), int32(errID)
}

func sendGreeterChatMessageBidiStream(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_Chat_send(C.int32_t(handle), C.uintptr_t(requestPtr), C.int32_t(requestLen)))
}

func readGreeterChatMessageBidiStream(ctx context.Context, handle int32, output *greeterMessageOutput) int32 {
	_ = ctx
	if output == nil {
		return int32(rpcruntime.StoreError(errors.New("rpccgo: message stream output is nil")))
	}
	var responsePtr C.uintptr_t
	var responseLen C.int32_t
	errID := rpccgo_msg_testv1_Greeter_Chat_read(C.int32_t(handle), &responsePtr, &responseLen)
	output.DataPtr = uintptr(responsePtr)
	output.DataLen = int32(responseLen)
	return int32(errID)
}

func closeSendGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_Chat_close_send(C.int32_t(handle)))
}

func finishGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_Chat_finish(C.int32_t(handle)))
}

func cancelGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {
	_ = ctx
	return int32(rpccgo_msg_testv1_Greeter_Chat_cancel(C.int32_t(handle)))
}
`

const messageDirectPathFixtureTestSource = `package main

import (
	context "context"
	strings "strings"
	"testing"
	"unsafe"

	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

func registerMessageServer(t *testing.T) {
	t.Helper()
	if err := registerGreeterMessageCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageCallbacksForIntegration() error = %v", err)
	}
}

func TestMessageUnaryDirectPath(t *testing.T) {
	registerMessageServer(t)
	output := &greeterMessageOutput{}
	if errID := callGreeterUnaryMessageUnary(context.Background(), 0, 0, output); errID != 0 {
		t.Fatalf("callGreeterUnaryMessageUnary() errID = %d", errID)
	}
	if got := greeterMessageUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("unary callback calls = %d, want 1", got)
	}
	if output.DataPtr != 0 || output.DataLen != 0 {
		t.Fatalf("output = {%d, %d}, want zero empty message", output.DataPtr, output.DataLen)
	}

	invalid := []byte{0xff}
	errID := callGreeterUnaryMessageUnary(context.Background(), uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)), &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "message request decode failed")

	setGreeterMessageUnaryErrorForIntegration(true)
	errID = callGreeterUnaryMessageUnary(context.Background(), 0, 0, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "unknown error id 99999")
}

func TestMessageKnownErrorTextIsConsumedOnce(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageUnaryStoredErrorForIntegration(true)

	errID := callGreeterUnaryMessageUnary(context.Background(), 0, 0, &greeterMessageOutput{})
	if errID == 0 {
		t.Fatal("callGreeterUnaryMessageUnary() errID = 0, want stored callback error")
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

	errID := callGreeterUnaryMessageUnary(context.Background(), 0, 0, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "unknown error id 99999")
}

func TestMessageBytesRejectInvalidUnaryRequest(t *testing.T) {
	registerMessageServer(t)
	invalid := []byte{0xff}
	errID := callGreeterUnaryMessageUnary(context.Background(), uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)), &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "message request decode failed")
	if got := greeterMessageUnaryCallsForIntegration(); got != 0 {
		t.Fatalf("unary callback calls = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidClientStreamSend(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	t.Cleanup(func() {
		_ = cancelGreeterUploadMessageClientStream(context.Background(), handle)
	})
	invalid := []byte{0xff}
	errID = sendGreeterUploadMessageClientStream(context.Background(), handle, uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)))
	assertMessageErrContains(t, errID, "message request decode failed")
	if got := greeterMessageUploadSendsForIntegration(); got != 0 {
		t.Fatalf("upload sends = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidServerStreamStart(t *testing.T) {
	registerMessageServer(t)
	invalid := []byte{0xff}
	handle, errID := startGreeterListMessageServerStream(context.Background(), uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)))
	assertMessageErrContains(t, errID, "message request decode failed")
	if handle != 0 {
		if cancelErrID := cancelGreeterListMessageServerStream(context.Background(), handle); cancelErrID == 0 {
			t.Fatalf("startGreeterListMessageServerStream() returned usable handle %d after invalid request bytes", handle)
		}
	}
	if got := greeterMessageListStartsForIntegration(); got != 0 {
		t.Fatalf("list starts = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidBidiSend(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	t.Cleanup(func() {
		_ = cancelGreeterChatMessageBidiStream(context.Background(), handle)
	})
	invalid := []byte{0xff}
	errID = sendGreeterChatMessageBidiStream(context.Background(), handle, uintptr(unsafe.Pointer(&invalid[0])), int32(len(invalid)))
	assertMessageErrContains(t, errID, "message request decode failed")
	if got := greeterMessageChatSendsForIntegration(); got != 0 {
		t.Fatalf("chat sends = %d, want 0 after invalid request bytes", got)
	}
}

func TestMessageBytesRejectInvalidCallbackResponse(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageInvalidResponseForIntegration(true)

	errID := callGreeterUnaryMessageUnary(context.Background(), 0, 0, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "message server response decode failed")

	uploadHandle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, sendGreeterUploadMessageClientStream(context.Background(), uploadHandle, 0, 0))
	errID = finishGreeterUploadMessageClientStream(context.Background(), uploadHandle, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "message server response decode failed")

	listHandle, errID := startGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	errID = readGreeterListMessageServerStream(context.Background(), listHandle, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "message server response decode failed")
	assertMessageNoErr(t, finishGreeterListMessageServerStream(context.Background(), listHandle))

	chatHandle, errID := startGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	errID = sendGreeterChatMessageBidiStream(context.Background(), chatHandle, 0, 0)
	if errID != 0 {
		assertMessageErrContains(t, errID, "message server response decode failed")
		return
	}
	errID = readGreeterChatMessageBidiStream(context.Background(), chatHandle, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "message server response decode failed")
	assertMessageNoErr(t, finishGreeterChatMessageBidiStream(context.Background(), chatHandle))
}

func TestMessageClientStreamRejectsOperationsAfterFinish(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0))
	output := &greeterMessageOutput{}
	assertMessageNoErr(t, finishGreeterUploadMessageClientStream(context.Background(), handle, output))
	errID = finishGreeterUploadMessageClientStream(context.Background(), handle, output)
	assertMessageErrContains(t, errID, "stream handle is invalid")
	errID = sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageUploadFinishesForIntegration(); got != 1 {
		t.Fatalf("upload finishes = %d, want 1", got)
	}
}

func TestMessageServerStreamRejectsReadAfterFinish(t *testing.T) {
	registerMessageServer(t)
	setGreeterMessageStreamEOFModeForIntegration(true)
	handle, errID := startGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, readGreeterListMessageServerStream(context.Background(), handle, &greeterMessageOutput{}))
	errID = readGreeterListMessageServerStream(context.Background(), handle, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "EOF")
	assertMessageNoErr(t, finishGreeterListMessageServerStream(context.Background(), handle))
	errID = readGreeterListMessageServerStream(context.Background(), handle, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageListFinishsForIntegration(); got != 1 {
		t.Fatalf("list finishes = %d, want 1", got)
	}
}

func TestMessageBidiRejectsSendAfterCloseSendAndReadAfterCancel(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, closeSendGreeterChatMessageBidiStream(context.Background(), handle))
	errID = sendGreeterChatMessageBidiStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "message stream is closed")
	assertMessageNoErr(t, cancelGreeterChatMessageBidiStream(context.Background(), handle))
	errID = readGreeterChatMessageBidiStream(context.Background(), handle, &greeterMessageOutput{})
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
	handle, errID := startGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)

	errID = closeSendGreeterChatMessageBidiStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "unknown error id 99996")
	errID = sendGreeterChatMessageBidiStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "message stream is closed")
	errID = closeSendGreeterChatMessageBidiStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "message stream is closed")

	if got := greeterMessageChatCloseSendsForIntegration(); got != 1 {
		t.Fatalf("chat close sends = %d, want 1 after failed close send", got)
	}
	if got := greeterMessageChatSendsForIntegration(); got != 0 {
		t.Fatalf("chat sends = %d, want 0 after failed close send", got)
	}
}

func TestMessageStreamCancelTwiceCallsDownstreamOnce(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, cancelGreeterUploadMessageClientStream(context.Background(), handle))
	errID = cancelGreeterUploadMessageClientStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "stream handle is invalid")
	if got := greeterMessageUploadCancelsForIntegration(); got != 1 {
		t.Fatalf("upload cancels = %d, want 1", got)
	}
}

func TestMessageStreamInvalidHandleReturnsError(t *testing.T) {
	registerMessageServer(t)
	const invalid int32 = 999999
	assertMessageErrContains(t, sendGreeterUploadMessageClientStream(context.Background(), invalid, 0, 0), "stream handle is invalid")
	assertMessageErrContains(t, finishGreeterUploadMessageClientStream(context.Background(), invalid, &greeterMessageOutput{}), "stream handle is invalid")
	assertMessageErrContains(t, readGreeterListMessageServerStream(context.Background(), invalid, &greeterMessageOutput{}), "stream handle is invalid")
	assertMessageErrContains(t, finishGreeterListMessageServerStream(context.Background(), invalid), "stream handle is invalid")
	assertMessageErrContains(t, closeSendGreeterChatMessageBidiStream(context.Background(), invalid), "stream handle is invalid")
	assertMessageErrContains(t, cancelGreeterChatMessageBidiStream(context.Background(), invalid), "stream handle is invalid")
}

func TestMessageWrongTerminalOperationPreservesStreamHandle(t *testing.T) {
	registerMessageServer(t)

	uploadHandle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageErrContains(t, finishGreeterChatMessageBidiStream(context.Background(), uploadHandle), "stream handle is invalid")
	assertMessageNoErr(t, sendGreeterUploadMessageClientStream(context.Background(), uploadHandle, 0, 0))
	assertMessageNoErr(t, finishGreeterUploadMessageClientStream(context.Background(), uploadHandle, &greeterMessageOutput{}))

	listHandle, errID := startGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	assertMessageErrContains(t, cancelGreeterChatMessageBidiStream(context.Background(), listHandle), "stream handle is invalid")
	assertMessageNoErr(t, readGreeterListMessageServerStream(context.Background(), listHandle, &greeterMessageOutput{}))
	assertMessageNoErr(t, finishGreeterListMessageServerStream(context.Background(), listHandle))
}

func TestMessageServiceLevelRegistrationAccumulatesExistingCallbacks(t *testing.T) {
	if err := registerGreeterMessageUnaryOnlyForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageUnaryOnlyForIntegration() error = %v", err)
	}
	if err := registerGreeterMessageUploadOnlyServiceLevelWithoutResetForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageUploadOnlyServiceLevelWithoutResetForIntegration() error = %v", err)
	}

	assertMessageNoErr(t, callGreeterUnaryMessageUnary(context.Background(), 0, 0, &greeterMessageOutput{}))
	handle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0))
	assertMessageNoErr(t, finishGreeterUploadMessageClientStream(context.Background(), handle, &greeterMessageOutput{}))

	if got := greeterMessageUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("message unary calls = %d, want 1", got)
	}
	if got := greeterMessageUploadStartsForIntegration(); got != 1 {
		t.Fatalf("message upload starts = %d, want 1", got)
	}
	if got := greeterMessageUploadSendsForIntegration(); got != 1 {
		t.Fatalf("message upload sends = %d, want 1", got)
	}
	if got := greeterMessageUploadFinishesForIntegration(); got != 1 {
		t.Fatalf("message upload finishes = %d, want 1", got)
	}
}

func TestNativeServiceLevelRegistrationAccumulatesExistingCallbacks(t *testing.T) {
	if err := registerGreeterNativeUnaryOnlyForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeUnaryOnlyForIntegration() error = %v", err)
	}
	if err := registerGreeterNativeUploadOnlyServiceLevelWithoutResetForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeUploadOnlyServiceLevelWithoutResetForIntegration() error = %v", err)
	}

	assertMessageNoErr(t, callGreeterUnaryMessageUnary(context.Background(), 0, 0, &greeterMessageOutput{}))
	handle, errID := startGreeterUploadMessageClientStream(context.Background())
	assertMessageNoErr(t, errID)
	assertMessageNoErr(t, sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0))
	assertMessageNoErr(t, finishGreeterUploadMessageClientStream(context.Background(), handle, &greeterMessageOutput{}))

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
}

func TestMessagePartialRegistrationReportsRejectedMethods(t *testing.T) {
	err := registerGreeterMessagePartialStreamingMethodsForIntegration()
	if err == nil {
		t.Fatal("registerGreeterMessagePartialStreamingMethodsForIntegration() error = nil")
	}
	if !strings.Contains(err.Error(), "test.v1.Greeter.Upload") || !strings.Contains(err.Error(), "test.v1.Greeter.Chat") {
		t.Fatalf("message partial registration error = %q, want rejected method names", err)
	}
}

func TestNativePartialRegistrationReportsRejectedMethods(t *testing.T) {
	err := registerGreeterNativePartialStreamingMethodsForIntegration()
	if err == nil {
		t.Fatal("registerGreeterNativePartialStreamingMethodsForIntegration() error = nil")
	}
	if !strings.Contains(err.Error(), "test.v1.Greeter.Upload") || !strings.Contains(err.Error(), "test.v1.Greeter.Chat") {
		t.Fatalf("native partial registration error = %q, want rejected method names", err)
	}
}

func TestMessageStreamStartCapturesActiveServerSnapshot(t *testing.T) {
	if err := registerGreeterNativeCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterNativeCallbacksForIntegration() error = %v", err)
	}
	chatHandle, errID := startGreeterChatMessageBidiStream(context.Background())
	assertMessageNoErr(t, errID)
	listHandle, errID := startGreeterListMessageServerStream(context.Background(), 0, 0)
	assertMessageNoErr(t, errID)
	if err := registerGreeterMessageCallbacksWithoutResetForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageCallbacksWithoutResetForIntegration() error = %v", err)
	}

	assertMessageNoErr(t, sendGreeterChatMessageBidiStream(context.Background(), chatHandle, 0, 0))
	assertMessageNoErr(t, readGreeterChatMessageBidiStream(context.Background(), chatHandle, &greeterMessageOutput{}))
	assertMessageNoErr(t, finishGreeterChatMessageBidiStream(context.Background(), chatHandle))
	assertMessageNoErr(t, readGreeterListMessageServerStream(context.Background(), listHandle, &greeterMessageOutput{}))
	assertMessageNoErr(t, finishGreeterListMessageServerStream(context.Background(), listHandle))
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
	handle, errID := startGreeterUploadMessageClientStream(context.Background())
	if errID != 0 {
		t.Fatalf("startGreeterUploadMessageClientStream() errID = %d", errID)
	}
	if errID := sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0); errID != 0 {
		t.Fatalf("sendGreeterUploadMessageClientStream() first errID = %d", errID)
	}
	if errID := sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0); errID != 0 {
		t.Fatalf("sendGreeterUploadMessageClientStream() second errID = %d", errID)
	}
	if errID := finishGreeterUploadMessageClientStream(context.Background(), handle, &greeterMessageOutput{}); errID != 0 {
		t.Fatalf("finishGreeterUploadMessageClientStream() errID = %d", errID)
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
	errID = sendGreeterUploadMessageClientStream(context.Background(), handle, 0, 0)
	assertMessageErrContains(t, errID, "stream handle is invalid")
}

func TestMessageServerStreamingDirectPath(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterListMessageServerStream(context.Background(), 0, 0)
	if errID != 0 {
		t.Fatalf("startGreeterListMessageServerStream() errID = %d", errID)
	}
	if errID := readGreeterListMessageServerStream(context.Background(), handle, &greeterMessageOutput{}); errID != 0 {
		t.Fatalf("readGreeterListMessageServerStream() first errID = %d", errID)
	}
	if errID := readGreeterListMessageServerStream(context.Background(), handle, &greeterMessageOutput{}); errID != 0 {
		t.Fatalf("readGreeterListMessageServerStream() second errID = %d", errID)
	}
	if errID := finishGreeterListMessageServerStream(context.Background(), handle); errID != 0 {
		t.Fatalf("DoneGreeterListMessageServerStream() errID = %d", errID)
	}
	if got := greeterMessageListStartsForIntegration(); got != 1 {
		t.Fatalf("list starts = %d, want 1", got)
	}
	if got := greeterMessageListRecvsForIntegration(); got != 2 {
		t.Fatalf("list recvs = %d, want 2", got)
	}
	if got := greeterMessageListFinishsForIntegration(); got != 1 {
		t.Fatalf("list finishes = %d, want 1", got)
	}
	errID = finishGreeterListMessageServerStream(context.Background(), handle)
	assertMessageErrContains(t, errID, "stream handle is invalid")
}

func TestMessageBidiStreamingDirectPath(t *testing.T) {
	registerMessageServer(t)
	handle, errID := startGreeterChatMessageBidiStream(context.Background())
	if errID != 0 {
		t.Fatalf("startGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := sendGreeterChatMessageBidiStream(context.Background(), handle, 0, 0); errID != 0 {
		t.Fatalf("sendGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := readGreeterChatMessageBidiStream(context.Background(), handle, &greeterMessageOutput{}); errID != 0 {
		t.Fatalf("readGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := closeSendGreeterChatMessageBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("closeSendGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if errID := finishGreeterChatMessageBidiStream(context.Background(), handle); errID != 0 {
		t.Fatalf("DoneGreeterChatMessageBidiStream() errID = %d", errID)
	}
	if got := greeterMessageChatStartsForIntegration(); got != 1 {
		t.Fatalf("chat starts = %d, want 1", got)
	}
	if got := greeterMessageChatSendsForIntegration(); got != 1 {
		t.Fatalf("chat sends = %d, want 1", got)
	}
	if got := greeterMessageChatRecvsForIntegration(); got < 1 {
		t.Fatalf("chat recvs = %d, want at least 1", got)
	}
	if got := greeterMessageChatCloseSendsForIntegration(); got != 1 {
		t.Fatalf("chat close sends = %d, want 1", got)
	}
	if got := greeterMessageChatFinishsForIntegration(); got != 1 {
		t.Fatalf("chat finishes = %d, want 1", got)
	}
	errID = readGreeterChatMessageBidiStream(context.Background(), handle, &greeterMessageOutput{})
	assertMessageErrContains(t, errID, "stream handle is invalid")
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
