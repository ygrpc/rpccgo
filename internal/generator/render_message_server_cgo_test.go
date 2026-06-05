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
		`rpcruntime "rpccgo/rpcruntime"`,
		`rpccgo: message request protobuf unmarshal failed`,
		`rpccgo: message response protobuf unmarshal failed`,
		"rpcruntime.TakeErrorText",
		"unknown error id",
		"typedef int32_t (*GreeterUnaryCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);",
		"static inline int32_t callGreeterUnaryCGOMessageUnary(GreeterUnaryCGOMessageUnaryCallback callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {",
		"type greeterCGOMessageAdapter struct {",
		"C.GreeterUnaryCGOMessageUnaryCallback",
		"func (a *greeterCGOMessageAdapter) Unary(ctx context.Context, req []byte) ([]byte, error) {",
		"func (a *greeterCGOMessageAdapter) StartUpload(ctx context.Context) (v1.GreeterUploadMessageStreamSession, error) {",
		"func (a *greeterCGOMessageAdapter) StartList(ctx context.Context, req []byte) (v1.GreeterListMessageStreamSession, error) {",
		"func (a *greeterCGOMessageAdapter) StartChat(ctx context.Context) (v1.GreeterChatMessageStreamSession, error) {",
		"requestLen, err := rpcruntime.LengthToInt32(len(req))",
		"errID := int32(C.callGreeterUnaryCGOMessageUnary(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))",
		"resp, err := decodeGreeterUnaryCGOMessageResponseBytes(responsePtr, responseLen)",
		"decodeGreeterUploadCGOMessageResponseBytes",
		"decodeGreeterListCGOMessageResponseBytes",
		"decodeGreeterChatCGOMessageResponseBytes",
		"if err := protobuf.Unmarshal(resp, &v1.HelloReply{}); err != nil {",
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
