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
		"//export rpccgoMsgTestv1GreeterUnary",
		"func rpccgoMsgTestv1GreeterUnary(requestPtr C.uintptr_t, requestLen C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {",
		"//export rpccgoMsgTestv1GreeterUploadStart",
		"func rpccgoMsgTestv1GreeterUploadStart(handle *C.int32_t) C.int32_t {",
		"//export rpccgoMsgTestv1GreeterUploadSend",
		"func rpccgoMsgTestv1GreeterUploadSend(handle C.int32_t, requestPtr C.uintptr_t, requestLen C.int32_t) C.int32_t {",
		"//export rpccgoMsgTestv1GreeterUploadFinish",
		"func rpccgoMsgTestv1GreeterUploadFinish(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {",
		"//export rpccgoMsgTestv1GreeterListStart",
		"func rpccgoMsgTestv1GreeterListStart(requestPtr C.uintptr_t, requestLen C.int32_t, handle *C.int32_t, onRecv C.RpccgoMessageOnRecvCallback, onDone C.RpccgoMessageOnDoneCallback) C.int32_t {",
		"//export rpccgoMsgTestv1GreeterListRecv",
		"func rpccgoMsgTestv1GreeterListRecv(handle C.int32_t, responsePtr *C.uintptr_t, responseLen *C.int32_t) C.int32_t {",
		"callbackState, err := rpcruntime.EnableStreamCallbackReceive(rpcruntime.StreamHandle(handleValue))",
		"if rpcruntime.StreamCallbackReceiveEnabled(rpcruntime.StreamHandle(handleValue)) {",
		`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: stream receive is owned by callback receive mode")))`,
		"C.callRpccgoMessageOnRecvCallback",
		"C.callRpccgoMessageOnDoneCallback",
		"//export rpccgoMsgTestv1GreeterChatCloseSend",
		"func rpccgoMsgTestv1GreeterChatCloseSend(handle C.int32_t) C.int32_t {",
		`if err := v1.GreeterMessageUploadSend(ctx, rpcruntime.StreamHandle(handleValue), req); err != nil {`,
		`resp, err := v1.GreeterMessageUploadFinish(ctx, rpcruntime.StreamHandle(handleValue))`,
		`err := v1.GreeterMessageChatCloseSend(ctx, rpcruntime.StreamHandle(handleValue))`,
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
		"rpccgoMsgTestv1GreeterListFinish",
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
		"rpccgoMsgGoGreeterUnary",
		"rpcruntime.DispatcherStreamSend[",
		"rpcruntime.DispatcherStreamReceive[",
		"rpcruntime.DispatcherStreamFinish[",
		"rpcruntime.DispatcherStreamCancel[",
		"rpcruntime.DispatcherStreamCloseSend[",
		"DoneGreeterListMessageServerStream",
		"DoneGreeterChatMessageBidiStream",
		"rpccgoMsgTestv1GreeterListRead",
		"rpccgoMsgTestv1GreeterChatRead",
	)
}
