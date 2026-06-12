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
		`protobuf "google.golang.org/protobuf/proto"`,
		`rpcruntime "github.com/ygrpc/rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
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
		`err = v1.SendGreeterMessageUpload(ctx, rpcruntime.StreamHandle(handleValue), req)`,
		`resp, err := v1.FinishGreeterMessageUpload(ctx, rpcruntime.StreamHandle(handleValue))`,
		`err := v1.CloseSendGreeterMessageChat(ctx, rpcruntime.StreamHandle(handleValue))`,
		"ctx := context.Background()",
		`return C.int32_t(rpcruntime.StoreError(errors.New("rpccgo: message client output pointer is nil")))`,
		"req, err := decodeGreeterUnaryMessageRequest(uintptr(requestPtr), int32(requestLen))",
		`rpccgo: message request protobuf unmarshal failed`,
		"resp, err := v1.InvokeGreeterMessageUnary(ctx, req)",
		`rpccgo: message response protobuf marshal failed`,
		"ptr, length, err := encodeGreeterUnaryMessageResponse(resp)",
		"*responsePtr = C.uintptr_t(ptr)",
		"*responseLen = C.int32_t(length)",
		"func decodeGreeterUnaryMessageRequest(ptr uintptr, length int32) (*v1.HelloRequest, error) {",
		"msg := &v1.HelloRequest{}",
		`return nil, errors.New("rpccgo: message request length is negative")`,
		"if length == 0 {",
		"return msg, nil",
		`return nil, errors.New("rpccgo: message request pointer is nil")`,
		"data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))",
		"if err := protobuf.Unmarshal(data, msg); err != nil {",
		"func encodeGreeterUnaryMessageResponse(message *v1.HelloReply) (uintptr, int32, error) {",
		`return 0, 0, errors.New("rpccgo: message response is nil")`,
		"data, err := protobuf.Marshal(message)",
		"length, err := rpcruntime.LengthToInt32(len(data))",
		"if length == 0 {",
		"ptr, err := rpcruntime.PinBytes(data)",
		"return 0",
	} {
		assertGeneratedContentContains(t, plugin, cgoClientFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, cgoClientFile,
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
