package generator

import "testing"

func TestRenderMessageClientCGODefinesUnaryExportSurface(t *testing.T) {
	file := messageCgoTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	const cgoClientFile = "test/v1/cgo/message_cgo.greeter.client.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		"package main",
		`context "context"`,
		`errors "errors"`,
		`fmt "fmt"`,
		`rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"`,
		"//export rpccgo_msg_testv1_Greeter_Unary",
		"func rpccgo_msg_testv1_Greeter_Unary(requestPtr C.uintptr_t, requestLen C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {",
		"//export rpccgo_msg_testv1_Greeter_Upload_start",
		"func rpccgo_msg_testv1_Greeter_Upload_start(handle *C.int32_t) C.int32_t {",
		"//export rpccgo_msg_testv1_Greeter_Upload_send",
		"func rpccgo_msg_testv1_Greeter_Upload_send(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {",
		"//export rpccgo_msg_testv1_Greeter_Upload_finish",
		"func rpccgo_msg_testv1_Greeter_Upload_finish(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {",
		"//export rpccgo_msg_testv1_Greeter_List_start",
		"func rpccgo_msg_testv1_Greeter_List_start(requestPtr C.uintptr_t, requestLen C.int32_t, handle *C.int32_t) C.int32_t {",
		"//export rpccgo_msg_testv1_Greeter_List_read",
		"func rpccgo_msg_testv1_Greeter_List_read(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {",
		"//export rpccgo_msg_testv1_Greeter_Chat_close_send",
		"func rpccgo_msg_testv1_Greeter_Chat_close_send(handle C.int32_t) C.int32_t {",
		`if err := v1.SendGreeterMessageUpload(ctx, rpcruntime.StreamHandle(handleValue), req); err != nil {`,
		`resp, err := v1.FinishGreeterMessageUpload(ctx, rpcruntime.StreamHandle(handleValue))`,
		`err := v1.CloseSendGreeterMessageChat(ctx, rpcruntime.StreamHandle(handleValue))`,
		"ctx := context.Background()",
		`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))`,
		"req := &v1.HelloRequest{}",
		`if err := rpcruntime.DecodeMessage(uintptr(requestPtr), int32(requestLen), req); err != nil {`,
		"resp, err := v1.InvokeGreeterMessageUnary(ctx, req)",
		"ptr, length, err := rpcruntime.EncodeMessage(resp)",
		`return C.int32_t(rpcruntime.StoreError(fmt.Errorf("rpccgo: message response encode failed: %w", err)))`,
		"*responsePtr = C.uintptr_t(ptr)",
		"*responseLen = C.int32_t(length)",
		"return 0",
	} {
		assertGeneratedContentContains(t, plugin, cgoClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, cgoClientFile,
		"func decodeGreeterUnaryMessageRequest(",
		"func encodeGreeterUnaryMessageResponse(",
		"func decodeGreeterUploadMessageRequest(",
		"func encodeGreeterUploadMessageResponse(",
		"func decodeGreeterListMessageRequest(",
		"func encodeGreeterListMessageResponse(",
		"func decodeGreeterChatMessageRequest(",
		"func encodeGreeterChatMessageResponse(",
		"type GreeterMessageOutput struct {",
		"func CallGreeterUnaryMessageUnary(",
		"func StartGreeterUploadMessageClientStream(",
		"func SendGreeterUploadMessageClientStream(",
		"func FinishGreeterUploadMessageClientStream(",
		"func StartGreeterListMessageServerStream(",
		"func ReadGreeterListMessageServerStream(",
		"func FinishGreeterListMessageServerStream(",
		"func StartGreeterChatMessageBidiStream(",
		"func SendGreeterChatMessageBidiStream(",
		"func ReadGreeterChatMessageBidiStream(",
		"func CloseSendGreeterChatMessageBidiStream(",
		"func FinishGreeterChatMessageBidiStream(",
		"LoadUploadMessageStream",
		"TakeUploadMessageStream",
		"LoadListMessageStream",
		"TakeListMessageStream",
		"LoadChatMessageStream",
		"TakeChatMessageStream",
		"NewGreeterUploadMessage"+"Stream",
		"NewGreeterChatMessage"+"Stream",
		"rpccgo_msg_go_Greeter_Unary",
		"rpcruntime.DispatcherStreamSend[",
		"rpcruntime.DispatcherStreamReceive[",
		"rpcruntime.DispatcherStreamFinish[",
		"rpcruntime.DispatcherStreamCancel[",
		"rpcruntime.DispatcherStreamCloseSend[",
		"DoneGreeterListMessageServerStream",
		"DoneGreeterChatMessageBidiStream",
	)
}
