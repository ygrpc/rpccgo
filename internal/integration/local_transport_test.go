package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestLocalTransportAcceptance(t *testing.T) {
	tmp := t.TempDir()
	plugin := newLocalTransportTestPlugin(t, "example.com/messagedirect/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeMessageDirectPathGeneratedModule(t, tmp, plugin, "example.com/messagedirect")
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_stubs.go"), messageDirectPathStubSource)
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_reset.go"), messageDirectPathResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_callbacks.go"), messageDirectPathFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/local_transport_test.go"), localTransportFixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "^TestLocalTransportAcceptance$", "-count=1", "-timeout=10s")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("local transport fixture failed: %v\n%s", err, out)
	}
}

func newLocalTransportTestPlugin(t *testing.T, goPackage string) *protogen.Plugin {
	t.Helper()
	emptyFile := protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto)
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
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
					LeadingComments: proto.String("@rpccgo: msg-connect|native\n"),
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

const localTransportFixtureTestSource = `package main

import (
	context "context"
	errors "errors"
	io "io"
	http "net/http"
	httptest "net/http/httptest"
	strings "strings"
	"testing"
	time "time"

	connect "connectrpc.com/connect"
	v1 "example.com/messagedirect/test/v1"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func TestLocalTransportAcceptance(t *testing.T) {
	t.Run("connect routes to cgo message server", func(t *testing.T) {
		registerTransportMessageServer(t)
		setGreeterMessageStreamEOFModeForIntegration(true)
		httpClient, baseURL, closeServer := startConnectTransport(t)
		defer closeServer()

		connectUnaryCall(t, httpClient, baseURL)
		connectClientStreamCall(t, httpClient, baseURL)
		if got := connectServerStreamCall(t, httpClient, baseURL); got != 1 {
			t.Fatalf("connect server stream messages = %d, want 1", got)
		}
		if got := connectBidiStreamCall(t, httpClient, baseURL); got != 1 {
			t.Fatalf("connect bidi responses = %d, want 1", got)
		}

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
		if got := greeterMessageListStartsForIntegration(); got != 1 {
			t.Fatalf("message list starts = %d, want 1", got)
		}
		if got := greeterMessageListRecvsForIntegration(); got != 2 {
			t.Fatalf("message list recvs = %d, want 2 including EOF probe", got)
		}
		if got := greeterMessageListDonesForIntegration(); got != 1 {
			t.Fatalf("message list dones = %d, want 1", got)
		}
		if got := greeterMessageChatStartsForIntegration(); got != 1 {
			t.Fatalf("message chat starts = %d, want 1", got)
		}
		if got := greeterMessageChatSendsForIntegration(); got != 1 {
			t.Fatalf("message chat sends = %d, want 1", got)
		}
		if got := greeterMessageChatRecvsForIntegration(); got != 2 {
			t.Fatalf("message chat recvs = %d, want 2 including EOF probe", got)
		}
		if got := greeterMessageChatCloseSendsForIntegration(); got > 1 {
			t.Fatalf("message chat close sends = %d, want at most 1", got)
		}
		if got := greeterMessageChatDonesForIntegration(); got != 1 {
			t.Fatalf("message chat dones = %d, want 1", got)
		}
	})


	t.Run("connect bidi supports independent response pump", func(t *testing.T) {
		resetTransportGoNativeCounters()
		setTransportGoNativeChatResponses(2)
		if _, err := v1.RegisterGreeterGoNativeServer(transportNativeServer{}); err != nil {
			t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
		}
		httpClient, baseURL, closeServer := startConnectTransport(t)
		defer closeServer()

		if got := connectBidiOneRequestTwoResponsesCall(t, httpClient, baseURL); got != 2 {
			t.Fatalf("connect bidi responses = %d, want 2", got)
		}

		if got := transportGoNativeChatStarts; got != 1 {
			t.Fatalf("transport native chat starts = %d, want 1", got)
		}
		if got := transportGoNativeChatSends; got != 1 {
			t.Fatalf("transport native chat sends = %d, want 1", got)
		}
		if got := transportGoNativeChatRecvs; got != 3 {
			t.Fatalf("transport native chat recvs = %d, want 3 including EOF", got)
		}
	})


	t.Run("connect converts into go native server", func(t *testing.T) {
		registerTransportGoNativeServer(t)
		httpClient, baseURL, closeServer := startConnectTransport(t)
		defer closeServer()

		connectUnaryCall(t, httpClient, baseURL)
		connectClientStreamCall(t, httpClient, baseURL)
		if got := connectServerStreamCall(t, httpClient, baseURL); got != 1 {
			t.Fatalf("connect server stream messages = %d, want 1", got)
		}
		if got := connectBidiStreamCall(t, httpClient, baseURL); got != 1 {
			t.Fatalf("connect bidi responses = %d, want 1", got)
		}

		assertGoNativeTransportCounters(t)
	})


	t.Run("connect surfaces downstream errors", func(t *testing.T) {
		registerTransportMessageServer(t)
		httpClient, baseURL, closeServer := startConnectTransport(t)
		defer closeServer()
		setGreeterMessageUnaryErrorForIntegration(true)

		client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterUnaryConnectProcedure)
		_, err := client.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err == nil || !strings.Contains(err.Error(), "unknown error id 99999") {
			t.Fatalf("connect unary error = %v, want unknown error id 99999", err)
		}
	})


	t.Run("connect stream snapshot stays on original server", func(t *testing.T) {
		registerTransportGoNativeServer(t)
		httpClient, baseURL, closeServer := startConnectTransport(t)
		defer closeServer()

		client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterListConnectProcedure)
		stream, err := client.CallServerStream(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err != nil {
			t.Fatalf("connect server stream CallServerStream() error = %v", err)
		}
		if err := registerGreeterMessageCallbacksWithoutResetForIntegration(); err != nil {
			t.Fatalf("registerGreeterMessageCallbacksWithoutResetForIntegration() error = %v", err)
		}
		count := 0
		for stream.Receive() {
			count++
		}
		if err := stream.Err(); err != nil {
			t.Fatalf("connect server stream Err() = %v", err)
		}
		if count != 1 {
			t.Fatalf("connect server stream messages = %d, want 1", count)
		}
		if got := transportGoNativeListRecvs; got != 2 {
			t.Fatalf("transport native list recvs = %d, want 2 including EOF", got)
		}
		if got := greeterMessageListRecvsForIntegration(); got != 0 {
			t.Fatalf("message list recvs = %d, want 0 for native snapshot", got)
		}
	})
}

func registerTransportMessageServer(t *testing.T) {
	t.Helper()
	if err := registerGreeterMessageCallbacksForIntegration(); err != nil {
		t.Fatalf("registerGreeterMessageCallbacksForIntegration() error = %v", err)
	}
}

func registerTransportGoNativeServer(t *testing.T) {
	t.Helper()
	v1.ResetGreeterDispatcherForIntegrationTest()
	resetTransportGoNativeCounters()
	if _, err := v1.RegisterGreeterGoNativeServer(transportNativeServer{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
}

func startConnectTransport(t *testing.T) (*http.Client, string, func()) {
	t.Helper()
	_, handler := v1.NewGreeterHandler(v1.GreeterBridgeForIntegrationTest())
	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = true
	server.StartTLS()
	return server.Client(), server.URL, server.Close
}

func connectUnaryCall(t *testing.T, httpClient *http.Client, baseURL string) {
	t.Helper()
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterUnaryConnectProcedure)
	if _, err := client.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{})); err != nil {
		t.Fatalf("connect unary CallUnary() error = %v", err)
	}
}

func connectClientStreamCall(t *testing.T, httpClient *http.Client, baseURL string) {
	t.Helper()
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterUploadConnectProcedure)
	stream := client.CallClientStream(context.Background())
	if err := stream.Send(&emptypb.Empty{}); err != nil {
		t.Fatalf("connect client stream Send() error = %v", err)
	}
	if _, err := stream.CloseAndReceive(); err != nil {
		t.Fatalf("connect client stream CloseAndReceive() error = %v", err)
	}
}

func connectServerStreamCall(t *testing.T, httpClient *http.Client, baseURL string) int {
	t.Helper()
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterListConnectProcedure)
	stream, err := client.CallServerStream(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		t.Fatalf("connect server stream CallServerStream() error = %v", err)
	}
	count := 0
	for stream.Receive() {
		count++
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("connect server stream Err() = %v", err)
	}
	return count
}

func connectBidiStreamCall(t *testing.T, httpClient *http.Client, baseURL string) int {
	t.Helper()
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterChatConnectProcedure)
	stream := client.CallBidiStream(context.Background())
	if err := stream.Send(&emptypb.Empty{}); err != nil {
		t.Fatalf("connect bidi Send() error = %v", err)
	}
	count := 0
	resp, err := stream.Receive()
	if err != nil {
		t.Fatalf("connect bidi Receive() first error = %v", err)
	}
	if resp == nil {
		t.Fatal("connect bidi first response = nil")
	}
	count++
	if err := stream.CloseRequest(); err != nil {
		t.Fatalf("connect bidi CloseRequest() error = %v", err)
	}
	_, err = stream.Receive()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("connect bidi final Receive() error = %v, want io.EOF", err)
	}
	return count
}

func connectBidiOneRequestTwoResponsesCall(t *testing.T, httpClient *http.Client, baseURL string) int {
	t.Helper()
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+v1.GreeterChatConnectProcedure)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream := client.CallBidiStream(ctx)
	if err := stream.Send(&emptypb.Empty{}); err != nil {
		t.Fatalf("connect bidi two-response Send() error = %v", err)
	}
	count := 0
	resp, err := stream.Receive()
	if err != nil {
		t.Fatalf("connect bidi two-response first Receive() error = %v", err)
	}
	if resp == nil {
		t.Fatal("connect bidi two-response first response = nil")
	}
	count++
	resp, err = stream.Receive()
	if err != nil {
		t.Fatalf("connect bidi two-response second Receive() error = %v", err)
	}
	if resp == nil {
		t.Fatal("connect bidi two-response second response = nil")
	}
	count++
	if err := stream.CloseRequest(); err != nil {
		t.Fatalf("connect bidi two-response CloseRequest() error = %v", err)
	}
	_, err = stream.Receive()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("connect bidi two-response final Receive() error = %v, want io.EOF", err)
	}
	return count
}

type transportNativeServer struct{}

var (
	transportGoNativeUnaryCalls      int
	transportGoNativeUploadStarts    int
	transportGoNativeUploadSends     int
	transportGoNativeUploadFinishes  int
	transportGoNativeListStarts      int
	transportGoNativeListRecvs       int
	transportGoNativeListDones       int
	transportGoNativeChatStarts      int
	transportGoNativeChatSends       int
	transportGoNativeChatRecvs       int
	transportGoNativeChatCloseSends  int
	transportGoNativeChatCancels     int
	transportGoNativeChatResponses   int
	transportGoNativeUnaryErrEnabled bool
)

func resetTransportGoNativeCounters() {
	transportGoNativeUnaryCalls = 0
	transportGoNativeUploadStarts = 0
	transportGoNativeUploadSends = 0
	transportGoNativeUploadFinishes = 0
	transportGoNativeListStarts = 0
	transportGoNativeListRecvs = 0
	transportGoNativeListDones = 0
	transportGoNativeChatStarts = 0
	transportGoNativeChatSends = 0
	transportGoNativeChatRecvs = 0
	transportGoNativeChatCloseSends = 0
	transportGoNativeChatCancels = 0
	transportGoNativeChatResponses = 1
	transportGoNativeUnaryErrEnabled = false
}

func setTransportGoNativeUnaryError(enabled bool) {
	transportGoNativeUnaryErrEnabled = enabled
}

func setTransportGoNativeChatResponses(count int) {
	transportGoNativeChatResponses = count
}

func assertGoNativeTransportCounters(t *testing.T) {
	t.Helper()
	if got := transportGoNativeUnaryCalls; got != 1 {
		t.Fatalf("transport native unary calls = %d, want 1", got)
	}
	if got := transportGoNativeUploadStarts; got != 1 {
		t.Fatalf("transport native upload starts = %d, want 1", got)
	}
	if got := transportGoNativeUploadSends; got != 1 {
		t.Fatalf("transport native upload sends = %d, want 1", got)
	}
	if got := transportGoNativeUploadFinishes; got != 1 {
		t.Fatalf("transport native upload finishes = %d, want 1", got)
	}
	if got := transportGoNativeListStarts; got != 1 {
		t.Fatalf("transport native list starts = %d, want 1", got)
	}
	if got := transportGoNativeListRecvs; got != 2 {
		t.Fatalf("transport native list recvs = %d, want 2 including EOF", got)
	}
	if got := transportGoNativeChatStarts; got != 1 {
		t.Fatalf("transport native chat starts = %d, want 1", got)
	}
	if got := transportGoNativeChatSends; got != 1 {
		t.Fatalf("transport native chat sends = %d, want 1", got)
	}
	if got := transportGoNativeChatRecvs; got != 2 {
		t.Fatalf("transport native chat recvs = %d, want 2 including EOF", got)
	}
	if got := transportGoNativeChatCloseSends; got != 1 {
		t.Fatalf("transport native chat close sends = %d, want 1", got)
	}
}

func (transportNativeServer) Unary(context.Context) error {
	transportGoNativeUnaryCalls++
	if transportGoNativeUnaryErrEnabled {
		return errors.New("transport native unary boom")
	}
	return nil
}

func (transportNativeServer) Upload(ctx context.Context, stream v1.GreeterUploadNativeClientStream) error {
	transportGoNativeUploadStarts++
	for {
		err := stream.Recv(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		transportGoNativeUploadSends++
	}
	transportGoNativeUploadFinishes++
	return nil
}

func (transportNativeServer) List(ctx context.Context, stream v1.GreeterListNativeServerStream) error {
	transportGoNativeListStarts++
	transportGoNativeListRecvs++
	if err := stream.Send(ctx); err != nil {
		return err
	}
	transportGoNativeListRecvs++
	transportGoNativeListDones++
	return nil
}

func (transportNativeServer) Chat(ctx context.Context, stream v1.GreeterChatNativeBidiStream) error {
	transportGoNativeChatStarts++
	for {
		err := stream.Recv(ctx)
		if err == io.EOF {
			transportGoNativeChatRecvs++
			transportGoNativeChatCloseSends++
			return nil
		}
		if err != nil {
			transportGoNativeChatCancels++
			return err
		}
		transportGoNativeChatSends++
		for i := 0; i < transportGoNativeChatResponses; i++ {
			if err := stream.Send(ctx); err != nil {
				return err
			}
			transportGoNativeChatRecvs++
		}
	}
}
`
