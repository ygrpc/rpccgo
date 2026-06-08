package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ygrpc/rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestRemoteTransportAcceptance(t *testing.T) {
	tmp := t.TempDir()
	remotePlugin := newRemoteTransportTestPlugin(t, "remote/v1/remote_transport.proto", "example.com/remotetransport/remote/v1;remotev1")
	localPlugin := newRemoteTransportTestPlugin(t, "local/v1/remote_transport.proto", "example.com/remotetransport/local/v1;localv1")
	if _, err := generator.GenerateWithOptions(remotePlugin); err != nil {
		t.Fatalf("GenerateWithOptions(remote) error = %v", err)
	}
	if _, err := generator.GenerateWithOptions(localPlugin); err != nil {
		t.Fatalf("GenerateWithOptions(local) error = %v", err)
	}

	writeRemoteTransportGeneratedModule(t, tmp, remotePlugin, localPlugin)
	writeFile(t, filepath.Join(tmp, "remote/v1/message_integration_stubs.go"), strings.ReplaceAll(messageDirectPathHandlerStubSource, "package testv1", "package remotev1"))
	writeFile(t, filepath.Join(tmp, "local/v1/message_integration_stubs.go"), strings.ReplaceAll(messageDirectPathHandlerStubSource, "package testv1", "package localv1"))
	writeFile(t, filepath.Join(tmp, "remote/v1/message_integration_reset.go"), strings.ReplaceAll(messageDirectPathResetSource, "package testv1", "package remotev1"))
	writeFile(t, filepath.Join(tmp, "local/v1/message_integration_reset.go"), strings.ReplaceAll(messageDirectPathResetSource, "package testv1", "package localv1"))
	writeFile(t, filepath.Join(tmp, "remote/v1/remote_transport.connect.go"), strings.ReplaceAll(remoteTransportConnectClientSource, "package testv1", "package remotev1"))
	writeFile(t, filepath.Join(tmp, "local/v1/remote_transport.connect.go"), strings.ReplaceAll(remoteTransportConnectClientSource, "package testv1", "package localv1"))
	writeFile(t, filepath.Join(tmp, "remote/v1/cgo/message_direct_path_callbacks.go"), strings.ReplaceAll(strings.ReplaceAll(messageDirectPathFixtureCallbackSource, "example.com/messagedirect/test/v1", "example.com/remotetransport/remote/v1"), "testv1", "remotev1"))
	writeFile(t, filepath.Join(tmp, "local/v1/cgo/message_direct_path_callbacks.go"), strings.ReplaceAll(strings.ReplaceAll(messageDirectPathFixtureCallbackSource, "example.com/messagedirect/test/v1", "example.com/remotetransport/local/v1"), "testv1", "localv1"))
	writeFile(t, filepath.Join(tmp, "local/v1/cgo/message_direct_path_cgo_client_bridge.go"), strings.ReplaceAll(strings.ReplaceAll(messageDirectPathCGOClientBridgeSource, "example.com/messagedirect/test/v1", "example.com/remotetransport/local/v1"), "testv1", "localv1"))
	writeFile(t, filepath.Join(tmp, "remote/v1/cgo/remote_server_boot.go"), remoteTransportServerBootSource)
	writeFile(t, filepath.Join(tmp, "local/v1/cgo/remote_transport_test.go"), remoteTransportLocalFixtureTestSource)

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmp, "remote-transport-server"), "./remote/v1/cgo")
	buildCmd.Dir = tmp
	buildCmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build remote server fixture failed: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./local/v1/cgo", "-run", "^TestRemoteTransportAcceptance$", "-count=1", "-timeout", "12s", "-v")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "RPCCGO_REMOTE_TRANSPORT_MODULE_ROOT="+tmp)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("remote transport fixture timed out\n%s", out)
	}
	if err != nil {
		t.Fatalf("remote transport fixture failed: %v\n%s", err, out)
	}
}

func newRemoteTransportTestPlugin(t *testing.T, protoPath, goPackage string) *protogen.Plugin {
	t.Helper()
	emptyFile := protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto)
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{protoPath},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			emptyFile,
			{
				Name:       proto.String(protoPath),
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

func writeRemoteTransportGeneratedModule(t *testing.T, root string, plugins ...*protogen.Plugin) {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/remotetransport\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\tgoogle.golang.org/protobuf v1.36.11\n\tgithub.com/ygrpc/rpccgo v0.0.0\n)\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n")
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	writeFile(t, filepath.Join(root, "go.sum"), string(goSum))
	for _, plugin := range plugins {
		for _, generated := range plugin.Response().GetFile() {
			writeFile(t, filepath.Join(root, generated.GetName()), generated.GetContent())
		}
	}
}

const remoteTransportConnectClientSource = `package testv1

import (
	context "context"
	strings "strings"

	connect "connectrpc.com/connect"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GreeterClient interface {
	Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	Upload(context.Context) (*connect.ClientStreamForClientSimple[emptypb.Empty, emptypb.Empty], error)
	List(context.Context, *emptypb.Empty) (*connect.ServerStreamForClient[emptypb.Empty], error)
	Chat(context.Context) (*connect.BidiStreamForClientSimple[emptypb.Empty, emptypb.Empty], error)
}

type greeterClient struct {
	unary  *connect.Client[emptypb.Empty, emptypb.Empty]
	upload *connect.Client[emptypb.Empty, emptypb.Empty]
	list   *connect.Client[emptypb.Empty, emptypb.Empty]
	chat   *connect.Client[emptypb.Empty, emptypb.Empty]
}

func NewGreeterClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) GreeterClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &greeterClient{
		unary:  connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+GreeterUnaryConnectProcedure, opts...),
		upload: connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+GreeterUploadConnectProcedure, opts...),
		list:   connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+GreeterListConnectProcedure, opts...),
		chat:   connect.NewClient[emptypb.Empty, emptypb.Empty](httpClient, baseURL+GreeterChatConnectProcedure, opts...),
	}
}

func (c *greeterClient) Unary(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
	response, err := c.unary.CallUnary(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (c *greeterClient) Upload(ctx context.Context) (*connect.ClientStreamForClientSimple[emptypb.Empty, emptypb.Empty], error) {
	return c.upload.CallClientStreamSimple(ctx)
}

func (c *greeterClient) List(ctx context.Context, request *emptypb.Empty) (*connect.ServerStreamForClient[emptypb.Empty], error) {
	stream, err := c.list.CallServerStream(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (c *greeterClient) Chat(ctx context.Context) (*connect.BidiStreamForClientSimple[emptypb.Empty, emptypb.Empty], error) {
	return c.chat.CallBidiStreamSimple(ctx)
}
`

const remoteTransportServerBootSource = `package main

import (
	context "context"
	errors "errors"
	flag "flag"
	fmt "fmt"
	net "net"
	http "net/http"
	os "os"
	osignal "os/signal"
	strings "strings"
	syscall "syscall"
	time "time"

	remotev1 "example.com/remotetransport/remote/v1"
)

func init() {
	transport := flag.String("transport", "connect", "remote transport")
	unaryError := flag.Bool("unary-error", false, "enable unary error")
	cancelObserver := flag.Bool("cancel-observer", false, "enable remote cancel observer mode")
	cancelSignalFile := flag.String("cancel-signal-file", "", "cancel observer signal file")
	flag.Parse()

	if *cancelObserver {
		if err := remotev1.RegisterGreeterGoNativeServer(cancelObserverGreeter{signalFile: *cancelSignalFile}); err != nil {
			fatal(err)
		}
	} else {
		if err := registerGreeterMessageCallbacksForIntegration(); err != nil {
			fatal(err)
		}
		setGreeterMessageStreamEOFModeForIntegration(true)
		if *unaryError {
			setGreeterMessageUnaryErrorForIntegration(true)
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal(err)
	}

	stop := make(chan os.Signal, 1)
	osignal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	switch *transport {
	case "connect":
		_, handler := remotev1.NewGreeterHandler(remotev1.GreeterEntryForIntegrationTest())
		if *cancelObserver {
			handler = wrapConnectCancelObserver(handler, *cancelSignalFile)
		}
		server := &http.Server{Handler: handler}
		fmt.Println(listener.Addr().String())
		go func() {
			_ = server.Serve(listener)
		}()
		<-stop
		_ = server.Close()
	default:
		fatal(fmt.Errorf("unknown transport: %s", *transport))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

type cancelObserverGreeter struct {
	signalFile string
}

func (s cancelObserverGreeter) Unary(context.Context) error {
	return nil
}

func (s cancelObserverGreeter) Upload(ctx context.Context, stream remotev1.GreeterUploadNativeClientStream) error {
	<-ctx.Done()
	writeCancelSignal(s.signalFile, "upload")
	return ctx.Err()
}

func (s cancelObserverGreeter) List(ctx context.Context, stream remotev1.GreeterListNativeServerStream) error {
	return errors.New("cancel observer list stream is not used")
}

func (s cancelObserverGreeter) Chat(ctx context.Context, stream remotev1.GreeterChatNativeBidiStream) error {
	select {
	case <-ctx.Done():
		writeCancelSignal(s.signalFile, "chat")
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return errors.New("cancel observer chat stream timed out waiting for cancellation")
	}
}

func writeCancelSignal(path, signal string) {
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(signal + "\n")
}

func wrapConnectCancelObserver(next http.Handler, signalFile string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Upload") {
			go func() {
				<-r.Context().Done()
				writeCancelSignal(signalFile, "upload")
			}()
		}
		if strings.Contains(r.URL.Path, "/Chat") {
			go func() {
				<-r.Context().Done()
				writeCancelSignal(signalFile, "chat")
			}()
		}
		next.ServeHTTP(w, r)
	})
}
`

const remoteTransportLocalFixtureTestSource = `package main

import (
	bufio "bufio"
	context "context"
	errors "errors"
	io "io"
	net "net"
	http "net/http"
	os "os"
	exec "os/exec"
	filepath "path/filepath"
	strings "strings"
	"testing"
	time "time"

	localv1 "example.com/remotetransport/local/v1"
	rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"
)

func TestRemoteTransportAcceptance(t *testing.T) {
	t.Run("connect remote routes message client to remote cgo message server", func(t *testing.T) {
		remote := startRemoteTransportServer(t, "connect", false)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := remoteTransportCallContext(t)
		defer cancel()
		assertMessageNoErr(t, callGreeterUnaryMessageUnary(ctx, 0, 0, &greeterMessageOutput{}))
	})

	t.Run("connect remote client stream captures adapter snapshot", func(t *testing.T) {
		remote := startRemoteTransportServer(t, "connect", false)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := remoteTransportCallContext(t)
		defer cancel()
		handle, errID := startGreeterUploadMessageClientStream(ctx)
		assertMessageNoErr(t, errID)
		if err := registerGreeterMessageCallbacksWithoutResetForIntegration(); err != nil {
			t.Fatalf("registerGreeterMessageCallbacksWithoutResetForIntegration() error = %v", err)
		}

		assertMessageNoErr(t, sendGreeterUploadMessageClientStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, finishGreeterUploadMessageClientStream(ctx, handle, &greeterMessageOutput{}))
		if got := greeterMessageUploadSendsForIntegration(); got != 0 {
			t.Fatalf("local message upload sends = %d, want 0 for remote snapshot", got)
		}
	})

	t.Run("connect remote surfaces downstream errors", func(t *testing.T) {
		remote := startRemoteTransportServer(t, "connect", true)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := remoteTransportCallContext(t)
		defer cancel()
		errID := callGreeterUnaryMessageUnary(ctx, 0, 0, &greeterMessageOutput{})
		assertMessageErrContains(t, errID, "unknown error id 99999")
	})

	t.Run("connect remote client stream cancel closes local session", func(t *testing.T) {
		remote := startRemoteTransportServer(t, "connect", false)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := remoteTransportCallContext(t)
		defer cancel()
		handle, errID := startGreeterUploadMessageClientStream(ctx)
		assertMessageNoErr(t, errID)
		assertMessageNoErr(t, sendGreeterUploadMessageClientStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, cancelGreeterUploadMessageClientStream(ctx, handle))
		assertMessageErrContains(t, sendGreeterUploadMessageClientStream(ctx, handle, 0, 0), "stream handle is invalid")
		assertMessageErrContains(t, cancelGreeterUploadMessageClientStream(ctx, handle), "stream handle is invalid")
	})

	t.Run("connect remote bidi cancel notifies remote context", func(t *testing.T) {
		remote := startRemoteTransportCancelObserverServer(t, "connect")
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := remoteTransportCallContext(t)
		defer cancel()
		handle, errID := startGreeterChatMessageBidiStream(ctx)
		assertMessageNoErr(t, errID)
		assertMessageNoErr(t, sendGreeterChatMessageBidiStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, cancelGreeterChatMessageBidiStream(ctx, handle))
		remote.waitForCancelSignal(t, "chat")
	})
}

func remoteTransportCallContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 3*time.Second)
}

type remoteTransportProcess struct {
	addr             string
	cmd              *exec.Cmd
	done             chan error
	cancelSignalFile string
}

func startRemoteTransportServer(t *testing.T, transport string, unaryError bool) remoteTransportProcess {
	t.Helper()
	moduleRoot := os.Getenv("RPCCGO_REMOTE_TRANSPORT_MODULE_ROOT")
	if moduleRoot == "" {
		t.Fatal("RPCCGO_REMOTE_TRANSPORT_MODULE_ROOT is empty")
	}
		args := []string{"-transport", transport}
	if unaryError {
		args = append(args, "-unary-error")
	}
	cmd := exec.Command(filepath.Join(moduleRoot, "remote-transport-server"), args...)
	cmd.Dir = moduleRoot
	cmd.Env = os.Environ()
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start remote server: %v\n%s", err, stderr.String())
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			lineCh <- strings.TrimSpace(scanner.Text())
			return
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		errCh <- io.EOF
	}()

	select {
	case addr := <-lineCh:
		waitForRemoteTransportPort(t, addr)
		return remoteTransportProcess{addr: addr, cmd: cmd, done: done}
	case err := <-errCh:
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("remote server exited before address: %v\n%s", err, stderr.String())
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("remote server did not print address\n%s", stderr.String())
	}
	return remoteTransportProcess{}
}

func startRemoteTransportCancelObserverServer(t *testing.T, transport string) remoteTransportProcess {
	t.Helper()
	moduleRoot := os.Getenv("RPCCGO_REMOTE_TRANSPORT_MODULE_ROOT")
	if moduleRoot == "" {
		t.Fatal("RPCCGO_REMOTE_TRANSPORT_MODULE_ROOT is empty")
	}
	signalFile := filepath.Join(moduleRoot, "remote-transport-cancel-observer-"+transport+"-"+time.Now().Format("20060102150405.000000000"))
	args := []string{
		"-transport", transport,
		"-cancel-observer",
		"-cancel-signal-file", signalFile,
	}
	cmd := exec.Command(filepath.Join(moduleRoot, "remote-transport-server"), args...)
	cmd.Dir = moduleRoot
	cmd.Env = os.Environ()
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start cancel-observer remote server: %v\n%s", err, stderr.String())
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			lineCh <- strings.TrimSpace(scanner.Text())
			return
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		errCh <- io.EOF
	}()

	select {
	case addr := <-lineCh:
		waitForRemoteTransportPort(t, addr)
		return remoteTransportProcess{addr: addr, cmd: cmd, done: done, cancelSignalFile: signalFile}
	case err := <-errCh:
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("cancel-observer remote server exited before address: %v\n%s", err, stderr.String())
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("cancel-observer remote server did not print address\n%s", stderr.String())
	}
	return remoteTransportProcess{}
}

func (p remoteTransportProcess) close() {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(os.Interrupt)
	}
	if p.done == nil {
		if p.cancelSignalFile != "" {
			_ = os.Remove(p.cancelSignalFile)
		}
		return
	}
	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
		if p.cmd != nil && p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		<-p.done
	}
	if p.cancelSignalFile != "" {
		_ = os.Remove(p.cancelSignalFile)
	}
}

func (p remoteTransportProcess) waitForCancelSignal(t *testing.T, signal string) {
	t.Helper()
	if p.cancelSignalFile == "" {
		t.Fatalf("cancel signal file is empty, want %q", signal)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(p.cancelSignalFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.TrimSpace(line) == signal {
					return
				}
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read cancel signal file: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	content, _ := os.ReadFile(p.cancelSignalFile)
	t.Fatalf("cancel signal %q not observed within timeout, file=%q", signal, strings.TrimSpace(string(content)))
}

func waitForRemoteTransportPort(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("remote server %s did not accept TCP connections", addr)
}

func registerConnectRemote(t *testing.T, remote remoteTransportProcess) {
	t.Helper()
	localv1.ResetGreeterServerForIntegrationTest()
	client := localv1.NewGreeterClient(http.DefaultClient, "http://"+remote.addr)
	if err := localv1.RegisterGreeterConnectRemoteServer(client); err != nil {
		t.Fatalf("RegisterGreeterConnectRemoteServer() error = %v", err)
	}
}

func assertMessageNoErr(t *testing.T, errID int32) {
	t.Helper()
	if errID != 0 {
		text, _, _ := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("errID = %d, text = %q", errID, text)
	}
}

func assertNativeUnaryNoErr(t *testing.T, errID int32) {
	t.Helper()
	if errID != 0 {
		text, _, _ := rpcruntime.TakeErrorText(rpcruntime.ErrorID(errID))
		t.Fatalf("native unary errID = %d, text = %q", errID, text)
	}
}

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

var _ = errors.Is
`
