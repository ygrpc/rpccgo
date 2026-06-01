package generator

import "testing"

func TestRenderMessageClientCGODefinesUnaryExportSurface(t *testing.T) {
	file := messageCgoTestFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	AttachMessageFileFamilyPlan(&plans[0])

	if err := RenderMessageStageFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderMessageStageFiles() error = %v", err)
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
		`v1.NewGreeterUploadMessageStream(rpcruntime.StreamHandle(handle)).Send(ctx, req)`,
		`resp, err = v1.NewGreeterUploadMessageStream(rpcruntime.StreamHandle(handle)).Finish(ctx)`,
		"CloseSendGreeterChatMessageBidiStream",
		`err = v1.NewGreeterChatMessageStream(rpcruntime.StreamHandle(handle)).CloseSend(ctx)`,
		"ctx = context.Background()",
		`return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))`,
		"req, err := decodeGreeterUnaryMessageRequestBytes(requestPtr, requestLen)",
		`rpccgo: message request protobuf unmarshal failed`,
		"resp, err := v1.InvokeGreeterMessageUnary(ctx, req)",
		`rpccgo: message response protobuf unmarshal failed`,
		"ptr, length, err := encodeGreeterUnaryMessageResponseBytes(resp)",
		"output.DataPtr = ptr",
		"output.DataLen = length",
		"func decodeGreeterUnaryMessageRequestBytes(ptr uintptr, length int32) ([]byte, error) {",
		`return nil, errors.New("rpccgo: message request length is negative")`,
		"if ptr == 0 || length == 0 {",
		"unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))",
		"func encodeGreeterUnaryMessageResponseBytes(data []byte) (uintptr, int32, error) {",
		"length, err := rpcruntime.LengthToInt32(len(data))",
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
