package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestStage6RemoteTransportAcceptance(t *testing.T) {
	tmp := t.TempDir()
	remotePlugin := newRemoteTransportStage6TestPlugin(t, "remote/v1/stage6.proto", "example.com/stage6/remote/v1;remotev1")
	localPlugin := newRemoteTransportStage6TestPlugin(t, "local/v1/stage6.proto", "example.com/stage6/local/v1;localv1")
	if _, err := generator.GenerateWithOptions(remotePlugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(remote) error = %v", err)
	}
	if _, err := generator.GenerateWithOptions(localPlugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(local) error = %v", err)
	}

	writeStage6RemoteGeneratedModule(t, tmp, remotePlugin, localPlugin)
	writeFile(t, filepath.Join(tmp, "remote/v1/message_integration_reset.go"), strings.ReplaceAll(messageDirectPathResetSource, "package testv1", "package remotev1"))
	writeFile(t, filepath.Join(tmp, "local/v1/message_integration_reset.go"), strings.ReplaceAll(messageDirectPathResetSource, "package testv1", "package localv1"))
	writeFile(t, filepath.Join(tmp, "remote/v1/cgo/message_direct_path_callbacks.go"), strings.ReplaceAll(messageDirectPathFixtureCallbackSource, "example.com/messagedirect/test/v1", "example.com/stage6/remote/v1"))
	writeFile(t, filepath.Join(tmp, "local/v1/cgo/message_direct_path_callbacks.go"), strings.ReplaceAll(messageDirectPathFixtureCallbackSource, "example.com/messagedirect/test/v1", "example.com/stage6/local/v1"))
	writeFile(t, filepath.Join(tmp, "remote/v1/cgo/remote_server_main.go"), stage6RemoteServerMainSource)
	writeFile(t, filepath.Join(tmp, "local/v1/cgo/remote_transport_stage6_test.go"), stage6LocalFixtureTestSource)

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmp, "stage6-remote-server"), "./remote/v1/cgo")
	buildCmd.Dir = tmp
	buildCmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build remote server fixture failed: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./local/v1/cgo", "-run", "^TestRemoteTransportStage6Acceptance$", "-count=1", "-timeout", "12s", "-v")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "RPCCGO_STAGE6_MODULE_ROOT="+tmp)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("remote transport fixture timed out\n%s", out)
	}
	if err != nil {
		t.Fatalf("remote transport fixture failed: %v\n%s", err, out)
	}
}

func newRemoteTransportStage6TestPlugin(t *testing.T, protoPath, goPackage string) *protogen.Plugin {
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
					LeadingComments: proto.String("@rpccgo: msg-connect|msg-grpc|native\n"),
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

func writeStage6RemoteGeneratedModule(t *testing.T, root string, plugins ...*protogen.Plugin) {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/stage6\n\ngo 1.24.4\n\nrequire (\n\tconnectrpc.com/connect v1.19.1\n\tgoogle.golang.org/grpc v1.79.3\n\tgoogle.golang.org/protobuf v1.36.11\n\trpccgo v0.0.0\n)\n\nreplace rpccgo => "+repoRoot+"\n")
	writeFile(t, filepath.Join(root, "go.sum"), "google.golang.org/protobuf v1.36.11 h1:fV6ZwhNocDyBLK0dj+fg8ektcVegBBuEolpbTQyBNVE=\ngoogle.golang.org/protobuf v1.36.11/go.mod h1:HTf+CrKn2C3g5S8VImy6tdcUvCska2kB7j23XfzDpco=\n")
	for _, plugin := range plugins {
		for _, generated := range plugin.Response().GetFile() {
			writeFile(t, filepath.Join(root, generated.GetName()), generated.GetContent())
		}
	}
}

const stage6RemoteServerMainSource = `package main

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

	remotev1 "example.com/stage6/remote/v1"
	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	transport := flag.String("transport", "connect", "remote transport")
	unaryError := flag.Bool("unary-error", false, "enable unary error")
	cancelObserver := flag.Bool("cancel-observer", false, "enable remote cancel observer mode")
	cancelSignalFile := flag.String("cancel-signal-file", "", "cancel observer signal file")
	flag.Parse()

	if *cancelObserver {
		if _, err := remotev1.RegisterGreeterGoNativeServer(cancelObserverGreeter{signalFile: *cancelSignalFile}); err != nil {
			panic(err)
		}
	} else {
		if err := registerGreeterMessageCallbacksForIntegration(); err != nil {
			panic(err)
		}
		setGreeterMessageStreamEOFModeForIntegration(true)
		if *unaryError {
			setGreeterMessageUnaryErrorForIntegration(true)
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	stop := make(chan os.Signal, 1)
	osignal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	switch *transport {
	case "connect":
		_, handler := remotev1.NewGreeterConnectHandler()
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
	case "grpc":
		server := grpc.NewServer()
		if err := remotev1.RegisterGreeterGRPCServer(server); err != nil {
			panic(err)
		}
		fmt.Println(listener.Addr().String())
		go func() {
			_ = server.Serve(listener)
		}()
		<-stop
		server.Stop()
	default:
		panic("unknown transport: " + *transport)
	}
}

type cancelObserverGreeter struct {
	signalFile string
}

func (s cancelObserverGreeter) Unary(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s cancelObserverGreeter) Upload(ctx context.Context) (remotev1.GreeterUploadNativeClientStream, error) {
	stream := &cancelObserverUploadStream{ctx: ctx, signalFile: s.signalFile}
	go stream.watchCancel()
	return stream, nil
}

func (s cancelObserverGreeter) List(context.Context, *emptypb.Empty) (remotev1.GreeterListNativeServerStream, error) {
	return &cancelObserverListStream{}, nil
}

func (s cancelObserverGreeter) Chat(ctx context.Context) (remotev1.GreeterChatNativeBidiStream, error) {
	stream := &cancelObserverChatStream{ctx: ctx, signalFile: s.signalFile}
	go stream.watchCancel()
	return stream, nil
}

type cancelObserverUploadStream struct {
	ctx        context.Context
	signalFile string
}

func (s *cancelObserverUploadStream) watchCancel() {
	<-s.ctx.Done()
	writeCancelSignal(s.signalFile, "upload")
}

func (s *cancelObserverUploadStream) Send(context.Context, *emptypb.Empty) error {
	return nil
}

func (s *cancelObserverUploadStream) Finish(context.Context) (*emptypb.Empty, error) {
	select {
	case <-s.ctx.Done():
		writeCancelSignal(s.signalFile, "upload")
		return nil, s.ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, errors.New("cancel observer upload stream timed out waiting for cancellation")
	}
}

func (s *cancelObserverUploadStream) Cancel(context.Context) error {
	writeCancelSignal(s.signalFile, "upload")
	return nil
}

type cancelObserverListStream struct{}

func (*cancelObserverListStream) Recv(context.Context) (*emptypb.Empty, error) {
	return nil, errors.New("cancel observer list stream is not used")
}

func (*cancelObserverListStream) Done(context.Context) error {
	return nil
}

func (*cancelObserverListStream) Cancel(context.Context) error {
	return nil
}

type cancelObserverChatStream struct {
	ctx        context.Context
	signalFile string
}

func (s *cancelObserverChatStream) watchCancel() {
	<-s.ctx.Done()
	writeCancelSignal(s.signalFile, "chat")
}

func (*cancelObserverChatStream) Send(context.Context, *emptypb.Empty) error {
	return nil
}

func (s *cancelObserverChatStream) Recv(context.Context) (*emptypb.Empty, error) {
	select {
	case <-s.ctx.Done():
		writeCancelSignal(s.signalFile, "chat")
		return nil, s.ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, errors.New("cancel observer chat stream timed out waiting for cancellation")
	}
}

func (*cancelObserverChatStream) CloseSend(context.Context) error {
	return nil
}

func (s *cancelObserverChatStream) Done(context.Context) error {
	writeCancelSignal(s.signalFile, "chat")
	return nil
}

func (s *cancelObserverChatStream) Cancel(context.Context) error {
	writeCancelSignal(s.signalFile, "chat")
	return nil
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

const stage6LocalFixtureTestSource = `package main

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

	localv1 "example.com/stage6/local/v1"
	grpc "google.golang.org/grpc"
	insecure "google.golang.org/grpc/credentials/insecure"
	rpcruntime "rpccgo/rpcruntime"
)

func TestRemoteTransportStage6Acceptance(t *testing.T) {
	t.Run("connect remote routes message client to remote cgo message server", func(t *testing.T) {
		remote := startStage6RemoteServer(t, "connect", false)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		assertMessageNoErr(t, CallGreeterUnaryMessageUnary(ctx, 0, 0, &GreeterMessageOutput{}))
	})

	t.Run("grpc remote routes message client to remote cgo message server", func(t *testing.T) {
		remote := startStage6RemoteServer(t, "grpc", false)
		defer remote.close()
		closeRemoteClient := registerGRPCRemote(t, remote)
		defer closeRemoteClient()

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		assertMessageNoErr(t, CallGreeterUnaryMessageUnary(ctx, 0, 0, &GreeterMessageOutput{}))
	})

	t.Run("connect remote reuses converter for native client", func(t *testing.T) {
		remote := startStage6RemoteServer(t, "connect", false)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		assertNativeUnaryNoErr(t, CallGreeterUnaryNativeUnary(ctx, &GreeterUnaryNativeUnaryInput{}, &GreeterUnaryNativeUnaryOutput{}))
	})

	t.Run("grpc remote reuses converter for native client", func(t *testing.T) {
		remote := startStage6RemoteServer(t, "grpc", false)
		defer remote.close()
		closeRemoteClient := registerGRPCRemote(t, remote)
		defer closeRemoteClient()

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		assertNativeUnaryNoErr(t, CallGreeterUnaryNativeUnary(ctx, &GreeterUnaryNativeUnaryInput{}, &GreeterUnaryNativeUnaryOutput{}))
	})

	t.Run("grpc remote client stream captures adapter snapshot", func(t *testing.T) {
		remote := startStage6RemoteServer(t, "grpc", false)
		defer remote.close()
		closeRemoteClient := registerGRPCRemote(t, remote)
		defer closeRemoteClient()

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		handle, errID := StartGreeterUploadMessageClientStream(ctx)
		assertMessageNoErr(t, errID)
		if err := registerGreeterMessageCallbacksWithoutResetForIntegration(); err != nil {
			t.Fatalf("registerGreeterMessageCallbacksWithoutResetForIntegration() error = %v", err)
		}

		assertMessageNoErr(t, SendGreeterUploadMessageClientStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, FinishGreeterUploadMessageClientStream(ctx, handle, &GreeterMessageOutput{}))
		if got := greeterMessageUploadSendsForIntegration(); got != 0 {
			t.Fatalf("local message upload sends = %d, want 0 for remote snapshot", got)
		}
	})

	t.Run("connect remote surfaces downstream errors", func(t *testing.T) {
		remote := startStage6RemoteServer(t, "connect", true)
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		errID := CallGreeterUnaryMessageUnary(ctx, 0, 0, &GreeterMessageOutput{})
		assertMessageErrContains(t, errID, "unknown error id 99999")
	})

	t.Run("connect remote client stream cancel notifies remote context", func(t *testing.T) {
		remote := startStage6RemoteCancelObserverServer(t, "connect")
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		handle, errID := StartGreeterUploadMessageClientStream(ctx)
		assertMessageNoErr(t, errID)
		assertMessageNoErr(t, SendGreeterUploadMessageClientStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, CancelGreeterUploadMessageClientStream(ctx, handle))
		remote.waitForCancelSignal(t, "upload")
	})

	t.Run("connect remote bidi cancel notifies remote context", func(t *testing.T) {
		remote := startStage6RemoteCancelObserverServer(t, "connect")
		defer remote.close()
		registerConnectRemote(t, remote)

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		handle, errID := StartGreeterChatMessageBidiStream(ctx)
		assertMessageNoErr(t, errID)
		assertMessageNoErr(t, SendGreeterChatMessageBidiStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, CancelGreeterChatMessageBidiStream(ctx, handle))
		remote.waitForCancelSignal(t, "chat")
	})

	t.Run("grpc remote client stream cancel notifies remote context", func(t *testing.T) {
		remote := startStage6RemoteCancelObserverServer(t, "grpc")
		defer remote.close()
		closeRemoteClient := registerGRPCRemote(t, remote)
		defer closeRemoteClient()

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		handle, errID := StartGreeterUploadMessageClientStream(ctx)
		assertMessageNoErr(t, errID)
		assertMessageNoErr(t, SendGreeterUploadMessageClientStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, CancelGreeterUploadMessageClientStream(ctx, handle))
		remote.waitForCancelSignal(t, "upload")
	})

	t.Run("grpc remote bidi cancel notifies remote context", func(t *testing.T) {
		remote := startStage6RemoteCancelObserverServer(t, "grpc")
		defer remote.close()
		closeRemoteClient := registerGRPCRemote(t, remote)
		defer closeRemoteClient()

		ctx, cancel := stage6CallContext(t)
		defer cancel()
		handle, errID := StartGreeterChatMessageBidiStream(ctx)
		assertMessageNoErr(t, errID)
		assertMessageNoErr(t, SendGreeterChatMessageBidiStream(ctx, handle, 0, 0))
		assertMessageNoErr(t, CancelGreeterChatMessageBidiStream(ctx, handle))
		remote.waitForCancelSignal(t, "chat")
	})
}

func stage6CallContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 3*time.Second)
}

type stage6RemoteProcess struct {
	addr             string
	cmd              *exec.Cmd
	done             chan error
	cancelSignalFile string
}

func startStage6RemoteServer(t *testing.T, transport string, unaryError bool) stage6RemoteProcess {
	t.Helper()
	moduleRoot := os.Getenv("RPCCGO_STAGE6_MODULE_ROOT")
	if moduleRoot == "" {
		t.Fatal("RPCCGO_STAGE6_MODULE_ROOT is empty")
	}
		args := []string{"-transport", transport}
	if unaryError {
		args = append(args, "-unary-error")
	}
	cmd := exec.Command(filepath.Join(moduleRoot, "stage6-remote-server"), args...)
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
		waitForStage6RemotePort(t, addr)
		return stage6RemoteProcess{addr: addr, cmd: cmd, done: done}
	case err := <-errCh:
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("remote server exited before address: %v\n%s", err, stderr.String())
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("remote server did not print address\n%s", stderr.String())
	}
	return stage6RemoteProcess{}
}

func startStage6RemoteCancelObserverServer(t *testing.T, transport string) stage6RemoteProcess {
	t.Helper()
	moduleRoot := os.Getenv("RPCCGO_STAGE6_MODULE_ROOT")
	if moduleRoot == "" {
		t.Fatal("RPCCGO_STAGE6_MODULE_ROOT is empty")
	}
	signalFile := filepath.Join(moduleRoot, "stage6-cancel-observer-"+transport+"-"+time.Now().Format("20060102150405.000000000"))
	args := []string{
		"-transport", transport,
		"-cancel-observer",
		"-cancel-signal-file", signalFile,
	}
	cmd := exec.Command(filepath.Join(moduleRoot, "stage6-remote-server"), args...)
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
		waitForStage6RemotePort(t, addr)
		return stage6RemoteProcess{addr: addr, cmd: cmd, done: done, cancelSignalFile: signalFile}
	case err := <-errCh:
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("cancel-observer remote server exited before address: %v\n%s", err, stderr.String())
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("cancel-observer remote server did not print address\n%s", stderr.String())
	}
	return stage6RemoteProcess{}
}

func (p stage6RemoteProcess) close() {
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

func (p stage6RemoteProcess) waitForCancelSignal(t *testing.T, signal string) {
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

func waitForStage6RemotePort(t *testing.T, addr string) {
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

func registerConnectRemote(t *testing.T, remote stage6RemoteProcess) {
	t.Helper()
	localv1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := localv1.RegisterGreeterConnectRemoteServer(http.DefaultClient, "http://"+remote.addr); err != nil {
		t.Fatalf("RegisterGreeterConnectRemoteServer() error = %v", err)
	}
}

func registerGRPCRemote(t *testing.T, remote stage6RemoteProcess) func() {
	t.Helper()
	localv1.ResetGreeterDispatcherForIntegrationTest()
	conn, err := grpc.NewClient(
		"passthrough:///"+remote.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return net.Dial("tcp", remote.addr)
		}),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	if _, err := localv1.RegisterGreeterGRPCRemoteServer(conn); err != nil {
		_ = conn.Close()
		t.Fatalf("RegisterGreeterGRPCRemoteServer() error = %v", err)
	}
	return func() { _ = conn.Close() }
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
