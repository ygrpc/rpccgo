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
		`rpcruntime "rpccgo/rpcruntime"`,
		`unsafe "unsafe"`,
		"type GreeterMessageOutput struct {",
		"DataPtr uintptr",
		"DataLen int32",
		"func CallGreeterUnaryMessageUnary(ctx context.Context, requestPtr uintptr, requestLen int32, output *GreeterMessageOutput) int32 {",
		"//export rpccgo_msg_testv1_Greeter_Unary",
		"func StartGreeterUploadMessageClientStream(ctx context.Context) (int32, int32) {",
		"func SendGreeterUploadMessageClientStream(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {",
		"func FinishGreeterUploadMessageClientStream(ctx context.Context, handle int32, output *GreeterMessageOutput) int32 {",
		"func StartGreeterListMessageServerStream(ctx context.Context, requestPtr uintptr, requestLen int32) (int32, int32) {",
		"func ReadGreeterListMessageServerStream(ctx context.Context, handle int32, output *GreeterMessageOutput) int32 {",
		"func FinishGreeterListMessageServerStream(ctx context.Context, handle int32) int32 {",
		"func StartGreeterChatMessageBidiStream(ctx context.Context) (int32, int32) {",
		"func SendGreeterChatMessageBidiStream(ctx context.Context, handle int32, requestPtr uintptr, requestLen int32) int32 {",
		"func ReadGreeterChatMessageBidiStream(ctx context.Context, handle int32, output *GreeterMessageOutput) int32 {",
		"func FinishGreeterChatMessageBidiStream(ctx context.Context, handle int32) int32 {",
		`err = v1.SendGreeterMessageUpload(ctx, rpcruntime.StreamHandle(handle), req)`,
		`resp, err = v1.FinishGreeterMessageUpload(ctx, rpcruntime.StreamHandle(handle))`,
		"CloseSendGreeterChatMessageBidiStream",
		`err = v1.CloseSendGreeterMessageChat(ctx, rpcruntime.StreamHandle(handle))`,
		"ctx = context.Background()",
		`return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))`,
		"req, err := decodeGreeterUnaryMessageRequest(requestPtr, requestLen)",
		`rpccgo: message request protobuf unmarshal failed`,
		"resp, err := v1.InvokeGreeterMessageUnary(ctx, req)",
		`rpccgo: message response protobuf marshal failed`,
		"ptr, length, err := encodeGreeterUnaryMessageResponse(resp)",
		"output.DataPtr = ptr",
		"output.DataLen = length",
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
