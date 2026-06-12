package generator

import (
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGenerateDartEmitsMessageFFIClient(t *testing.T) {
	plugin := newTestDartPlugin(t, "paths=source_relative", simpleTestFile())

	if _, err := GenerateDartWithOptions(plugin); err != nil {
		t.Fatalf("GenerateDartWithOptions() error = %v", err)
	}

	assertDartGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.rpccgo.dart",
	})
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "class GreeterRpccgoClient {")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "import 'greeter.pb.dart' as pb;")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "pb.HelloReply SayHello(pb.HelloRequest request) {")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "'rpccgo_msg_testv1_Greeter_SayHello'")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "request.writeToBuffer()")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "pb.HelloReply.fromBuffer(responseBytes)")
}

func TestGenerateDartEmitsStreamingMessageFFIClient(t *testing.T) {
	plugin := newTestDartPlugin(t, "paths=source_relative", messageContractTestFile())

	if _, err := GenerateDartWithOptions(plugin); err != nil {
		t.Fatalf("GenerateDartWithOptions() error = %v", err)
	}

	const file = "test/v1/message_contract.greeter.rpccgo.dart"
	assertDartGeneratedFilenames(t, plugin, []string{file})
	for _, fragment := range []string{
		"GreeterUploadStream Upload() {",
		"void send(pb.MessageRequest request) {",
		"pb.MessageReply finish() {",
		"GreeterListStream List(pb.MessageRequest request) {",
		"final errID = _listStart(requestPtr.address, requestBytes.length, handlePtr);",
		"pb.MessageReply read() {",
		"void finish() {",
		"GreeterChatStream Chat() {",
		"void closeSend() {",
		"'rpccgo_msg_testv1_Greeter_Upload_start'",
		"'rpccgo_msg_testv1_Greeter_List_read'",
		"'rpccgo_msg_testv1_Greeter_Chat_close_send'",
	} {
		assertGeneratedContentContains(t, plugin, file, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, file,
		"final _RpccgoStreamFinishVoid _uploadCloseSend;",
		"final _RpccgoStreamFinishVoid _listCloseSend;",
	)
}

func TestDartProtogenOptionsRejectUnknownParameter(t *testing.T) {
	request := newTestCodeGeneratorRequest("mode=message", simpleTestFile())

	_, err := DartProtogenOptions().New(request)
	if err == nil {
		t.Fatal("DartProtogenOptions().New() error = nil, want unknown parameter error")
	}
	if !strings.Contains(err.Error(), `unknown rpccgo dart parameter "mode"`) {
		t.Fatalf("DartProtogenOptions().New() error = %q, want unknown mode parameter", err.Error())
	}
}

func newTestDartPlugin(t *testing.T, parameter string, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()

	request := newTestCodeGeneratorRequest(parameter, files...)
	plugin, err := DartProtogenOptions().New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func assertDartGeneratedFilenames(t *testing.T, plugin *protogen.Plugin, want []string) {
	t.Helper()

	got := generatedFilenames(plugin)
	if len(got) != len(want) {
		t.Fatalf("generated files = %v, want %v; response error: %q", got, want, plugin.Response().GetError())
	}
	sort.Strings(got)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	for i, file := range got {
		if file != sortedWant[i] {
			t.Fatalf("generated file %d = %q, want %q; all files: %v", i, file, sortedWant[i], got)
		}
	}
}
