package generator

import (
	"strings"
	"testing"
)

func TestRenderMessageServerCGODefinesFlatServiceRegistration(t *testing.T) {
	file := messageCgoTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/cgo/message_cgo.greeter.server.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		"package main",
		`import "C"`,
		`errors "errors"`,
		`fmt "fmt"`,
		`protobuf "google.golang.org/protobuf/proto"`,
		`rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"`,
		`rpccgo: message request protobuf marshal failed`,
		`rpccgo: message response protobuf unmarshal failed`,
		"rpcruntime.TakeErrorText",
		"unknown error id",
		"typedef int32_t (*GreeterUnaryCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);",
		"static inline int32_t callGreeterUnaryCGOMessageUnary(GreeterUnaryCGOMessageUnaryCallback callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {",
		"type greeterCGOMessageAdapter struct {",
		"C.GreeterUnaryCGOMessageUnaryCallback",
		"func (a *greeterCGOMessageAdapter) Unary(ctx context.Context, req *v1.HelloRequest) (*v1.HelloReply, error) {",
		"func (a *greeterCGOMessageAdapter) StartUpload(ctx context.Context) (rpcruntime.ClientStreamingClient[*v1.HelloRequest, *v1.HelloReply], error) {",
		"func (a *greeterCGOMessageAdapter) Upload(ctx context.Context, stream rpcruntime.ClientStreamingServer[*v1.HelloRequest]) (*v1.HelloReply, error) {",
		"func (a *greeterCGOMessageAdapter) StartList(ctx context.Context, req *v1.HelloRequest) (rpcruntime.ServerStreamingClient[*v1.HelloReply], error) {",
		"func (a *greeterCGOMessageAdapter) List(ctx context.Context, req *v1.HelloRequest, stream rpcruntime.ServerStreamingServer[*v1.HelloReply]) error {",
		"func (a *greeterCGOMessageAdapter) StartChat(ctx context.Context) (rpcruntime.BidiStreamingClient[*v1.HelloRequest, *v1.HelloReply], error) {",
		"func (a *greeterCGOMessageAdapter) Chat(ctx context.Context, stream rpcruntime.BidiStreamingServer[*v1.HelloRequest, *v1.HelloReply]) error {",
		"// greeterCGOMessageRecvResult carries the result of a blocking cgo message Recv callback.",
		"type greeterCGOMessageRecvResult[T any] struct {",
		"// greeterAwaitCGOMessageRecv waits for a blocking cgo message Recv callback while allowing Finish or Cancel to interrupt the wait.",
		"func greeterAwaitCGOMessageRecv[T any](ctx context.Context, finishRequested <-chan struct{}, recv func() (T, error), finish func() error, cancel func() error) (T, error, bool) {",
		"select {\n\tcase <-finishRequested:\n\t\tvar zero T\n\t\treturn zero, finish(), true\n\tcase <-ctx.Done():",
		"case <-finishRequested:",
		"return zero, finish(), true",
		"case <-ctx.Done():",
		"return zero, errors.Join(ctx.Err(), cancel()), true",
		"resp, err, stopped := greeterAwaitCGOMessageRecv(ctx, stream.FinishRequested(), func() (*v1.HelloReply, error) { return session.Recv(ctx) }, func() error { return session.Finish(ctx) }, func() error { return session.Cancel(ctx) })",
		"bridgeCtx, cancelBridge := context.WithCancel(ctx)",
		"cancelSession := sync.OnceValue(func() error { return session.Cancel(bridgeCtx) })",
		"var resultErr error",
		"resultErr = errors.Join(resultErr, err)",
		"return resultErr",
		"reqBytes, err := protobuf.Marshal(req)",
		"requestLen, err := rpcruntime.LengthToInt32(len(reqBytes))",
		"errID := int32(C.callGreeterUnaryCGOMessageUnary(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))",
		"return decodeGreeterUnaryCGOMessageResponse(responsePtr, responseLen)",
		"decodeGreeterUploadCGOMessageResponse",
		"decodeGreeterListCGOMessageResponse",
		"decodeGreeterChatCGOMessageResponse",
		"func decodeGreeterUnaryCGOMessageResponse(responsePtr C.uintptr_t, responseLen C.int32_t) (*v1.HelloReply, error) {",
		`return nil, errors.New("rpccgo: message server response pointer is nil")`,
		"if err := protobuf.Unmarshal(data, resp); err != nil {",
		"//export rpccgo_msg_testv1_Greeter_register",
		"func rpccgo_msg_testv1_Greeter_register(unaryCallback C.GreeterUnaryCGOMessageUnaryCallback, uploadStart C.GreeterUploadCGOMessageClientStreamStartCallback, uploadSend C.GreeterUploadCGOMessageClientStreamSendCallback, uploadFinish C.GreeterUploadCGOMessageClientStreamFinishCallback, uploadCancel C.GreeterUploadCGOMessageClientStreamCancelCallback, listStart C.GreeterListCGOMessageServerStreamStartCallback, listRecv C.GreeterListCGOMessageServerStreamRecvCallback, listFinish C.GreeterListCGOMessageServerStreamFinishCallback, listCancel C.GreeterListCGOMessageServerStreamCancelCallback, chatStart C.GreeterChatCGOMessageBidiStreamStartCallback, chatSend C.GreeterChatCGOMessageBidiStreamSendCallback, chatRecv C.GreeterChatCGOMessageBidiStreamRecvCallback, chatCloseSend C.GreeterChatCGOMessageBidiStreamCloseSendCallback, chatFinish C.GreeterChatCGOMessageBidiStreamFinishCallback, chatCancel C.GreeterChatCGOMessageBidiStreamCancelCallback) C.int32_t {",
		"next := &greeterCGOMessageAdapter{}",
		"var registerErr error",
		"next.UnaryCallback = unaryCallback",
		"next.ListFinish = listFinish",
		"next.ChatFinish = chatFinish",
		"if err := v1.RegisterGreeterCGOMessageServer(next); err != nil {",
		"greeterCGOMessageServerAdapter = next",
		"//export rpccgo_msg_testv1_Greeter_register_Unary",
		"func rpccgo_msg_testv1_Greeter_register_Unary(unaryCallback C.GreeterUnaryCGOMessageUnaryCallback) C.int32_t {",
		"next := greeterCGOMessageServerAdapterForRegister()",
		"//export rpccgo_msg_testv1_Greeter_register_Upload",
		"if uploadStart == nil && uploadSend == nil && uploadFinish == nil && uploadCancel == nil {",
		"if registerErr == nil {",
		"registerErr = greeterCGOMessageServerStreamPartiallyRegistered",
		"func greeterCGOMessageServerAdapterForRegister() *greeterCGOMessageAdapter {",
		"registered, err := v1.LoadGreeterRegisteredServer()",
		"if err == nil && registered.Kind == rpcruntime.ServerKindCGOMessage {",
		"next := *current",
		`errors.New("rpccgo: Greeter.Unary cgo message server method is not implemented")`,
		"func greeterCGOMessageServerError(errID int32) error {",
		"if ok {",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}

	assertGeneratedFileContentDoesNotContain(t, plugin, cgoServerFile,
		"typedef struct GreeterCGOMessageServerCallbacks",
		"callbacks C.GreeterCGOMessageServerCallbacks",
		"rpccgo_msg_testv1_Greeter_Unary_register",
		"rpcruntime.StreamLifecycle",
		"s.capability.EnsureCanSend()",
		"s.capability.MarkSendClosed()",
	)

	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != cgoServerFile {
			continue
		}
		content := file.GetContent()
		register := "if err := v1.RegisterGreeterCGOMessageServer(next); err != nil {"
		commit := "greeterCGOMessageServerAdapter = next"
		if registerIndex, commitIndex := strings.Index(content, register), strings.Index(content, commit); registerIndex < 0 || commitIndex < 0 || commitIndex < registerIndex {
			t.Fatalf("generated registration side-effect order invalid: register index=%d commit index=%d", registerIndex, commitIndex)
		}
		return
	}
	t.Fatalf("generated file %q not found", cgoServerFile)
}

func TestRenderMessageServerCGOFileEmitsStreamEOFHelper(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoServerFile = "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		`io "io"`,
		"func GreeterCGOMessageStreamEOFErrorID() int32 {",
		"return int32(rpcruntime.StoreError(io.EOF))",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}
}
