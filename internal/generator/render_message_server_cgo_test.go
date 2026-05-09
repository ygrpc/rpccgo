package generator

import "testing"

func TestRenderMessageServerCGODefinesUnaryCallbackTableAndRegistration(t *testing.T) {
	file := simpleTestFile()
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
		"package main",
		`import "C"`,
		`errors "errors"`,
		`fmt "fmt"`,
		`protobuf "google.golang.org/protobuf/proto"`,
		`rpcruntime "rpccgo/rpcruntime"`,
		`rpccgo: message request protobuf unmarshal failed`,
		`rpccgo: message response protobuf unmarshal failed`,
		"typedef int32_t (*GreeterSayHelloCGOMessageUnaryCallback)(uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len);",
		"typedef struct GreeterCGOMessageServerCallbacks {",
		"GreeterSayHelloCGOMessageUnaryCallback SayHello;",
		"static inline int32_t callGreeterSayHelloCGOMessageUnary(GreeterSayHelloCGOMessageUnaryCallback callback, uintptr_t request_ptr, int32_t request_len, uintptr_t* response_ptr, int32_t* response_len) {",
		"type greeterCGOMessageAdapter struct {",
		"callbacks C.GreeterCGOMessageServerCallbacks",
		"func (a *greeterCGOMessageAdapter) SayHelloMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"requestLen, err := rpcruntime.LengthToInt32(len(req))",
		"errID := int32(C.callGreeterSayHelloCGOMessageUnary(callback, C.uintptr_t(requestPtr), C.int32_t(requestLen), &responsePtr, &responseLen))",
		"resp, err := decodeGreeterSayHelloCGOMessageResponseBytes(responsePtr, responseLen)",
		"if err := protobuf.Unmarshal(resp, &v1.HelloReply{}); err != nil {",
		"func RegisterGreeterCGOMessageServer(callbacks *C.GreeterCGOMessageServerCallbacks) (rpcruntime.AdapterSnapshot[v1.GreeterMessageAdapter], error) {",
		"return v1.RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindCGOMessage, &greeterCGOMessageAdapter{callbacks: callbacksCopy})",
		"callbacksCopy := *callbacks",
	} {
		assertGeneratedContentContains(t, plugin, cgoServerFile, fragment)
	}
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
