package generator

import (
	"strings"
	"testing"
)

func TestRenderMessageServerCGODefinesUnaryCallbackTableAndRegistration(t *testing.T) {
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
		"typedef struct GreeterCGOMessageServerCallbacks {",
		"GreeterUnaryCGOMessageUnaryCallback Unary;",
		"static inline int32_t callGreeterUnaryCGOMessageUnary(GreeterUnaryCGOMessageUnaryCallback callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {",
		"type greeterCGOMessageAdapter struct {",
		"callbacks C.GreeterCGOMessageServerCallbacks",
		"func (a *greeterCGOMessageAdapter) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"func (a *greeterCGOMessageAdapter) StartUploadMessage(ctx context.Context) (v1.GreeterUploadMessageStreamSession, error) {",
		"func (a *greeterCGOMessageAdapter) StartListMessage(ctx context.Context, req []byte) (v1.GreeterListMessageStreamSession, error) {",
		"func (a *greeterCGOMessageAdapter) StartChatMessage(ctx context.Context) (v1.GreeterChatMessageStreamSession, error) {",
		"requestLen, err := rpcruntime.LengthToInt32(len(req))",
		"errID := int32(C.callGreeterUnaryCGOMessageUnary(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))",
		"resp, err := decodeGreeterUnaryCGOMessageResponseBytes(responsePtr, responseLen)",
		"decodeGreeterUploadCGOMessageResponseBytes",
		"decodeGreeterListCGOMessageResponseBytes",
		"decodeGreeterChatCGOMessageResponseBytes",
		"if err := protobuf.Unmarshal(resp, &v1.HelloReply{}); err != nil {",
		"func RegisterGreeterCGOMessageServer(callbacks *C.GreeterCGOMessageServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterMessageAdapter], error) {",
		"return v1.RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindCGOMessage, &greeterCGOMessageAdapter{callbacks: callbacksCopy})",
		"callbacksCopy := *callbacks",
		"func greeterCGOMessageServerError(errID int32) error {",
		"if ok {",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}

	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != cgoServerFile {
			continue
		}
		content := file.GetContent()
		closeSend := "errID := int32(C.callGreeterChatCGOMessageBidiStreamCloseSend(s.callbacks.ChatCloseSend, C.int32_t(s.stream)))"
		markClosed := "s.lifecycle.MarkSendClosed()"
		if closeSendIndex, markClosedIndex := strings.Index(content, closeSend), strings.Index(content, markClosed); closeSendIndex < 0 || markClosedIndex < 0 || markClosedIndex < closeSendIndex {
			t.Fatalf("generated CloseSend lifecycle order invalid: CloseSend index=%d MarkSendClosed index=%d", closeSendIndex, markClosedIndex)
		}
		return
	}
	t.Fatalf("generated file %q not found", cgoServerFile)
}

func TestRenderMessageServerCGOFileEmitsStreamEOFHelper(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	AttachMessageFileFamilyPlan(&plans[0])

	if err := RenderMessageStageFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderMessageStageFiles() error = %v", err)
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
