package generator

import "testing"

func TestRenderMessageClientCGODefinesUnaryExportSurface(t *testing.T) {
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

	const cgoClientFile = "test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go"
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
		"func CallGreeterSayHelloMessageUnary(ctx context.Context, requestPtr uintptr, requestLen int32, output *GreeterMessageOutput) int32 {",
		"ctx = context.Background()",
		`return int32(rpcruntime.StoreError(errors.New("rpccgo: message unary client output is nil")))`,
		"req, err := decodeGreeterSayHelloMessageRequestBytes(requestPtr, requestLen)",
		`rpccgo: message request protobuf unmarshal failed`,
		"resp, err := v1.NewGreeterCGOMessageClientBridge().SayHello(ctx, req)",
		`rpccgo: message response protobuf unmarshal failed`,
		"ptr, length, err := encodeGreeterSayHelloMessageResponseBytes(resp)",
		"output.DataPtr = ptr",
		"output.DataLen = length",
		"func decodeGreeterSayHelloMessageRequestBytes(ptr uintptr, length int32) ([]byte, error) {",
		`return nil, errors.New("rpccgo: message request length is negative")`,
		"if ptr == 0 || length == 0 {",
		"unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))",
		"func encodeGreeterSayHelloMessageResponseBytes(data []byte) (uintptr, int32, error) {",
		"length, err := rpcruntime.LengthToInt32(len(data))",
		"ptr, err := rpcruntime.PinBytes(data)",
		"return 0",
	} {
		assertGeneratedContentContains(t, plugin, cgoClientFile, fragment)
	}
}
