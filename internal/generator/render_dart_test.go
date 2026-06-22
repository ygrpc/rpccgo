package generator

import (
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGenerateDartEmitsMessageFFIClient(t *testing.T) {
	plugin := newTestDartPlugin(t, "paths=source_relative,dart_package=rpccgo_test", simpleTestFile())

	if _, err := GenerateDartWithOptions(plugin); err != nil {
		t.Fatalf("GenerateDartWithOptions() error = %v", err)
	}

	assertDartGeneratedFilenames(t, plugin, []string{
		"rpccgo.dart",
		"rpccgo.lifecycle.dart",
		"test/v1/greeter.greeter.rpccgo.dart",
	})
	assertGeneratedContentContains(t, plugin, "rpccgo.dart", "export 'test/v1/greeter.greeter.rpccgo.dart';")
	assertGeneratedContentContains(t, plugin, "rpccgo.dart", "export 'test/v1/greeter.pb.dart';")
	assertGeneratedContentContains(t, plugin, "rpccgo.dart", "export 'rpccgo.lifecycle.dart';")
	assertGeneratedContentContains(t, plugin, "rpccgo.lifecycle.dart", "class RpccgoStreamLifecycle {")
	assertGeneratedContentContains(t, plugin, "rpccgo.lifecycle.dart", "static T withLifecycle<T>(RpccgoStreamLifecycle lifecycle, T Function() fn) => runZoned(")
	assertGeneratedContentContains(t, plugin, "rpccgo.lifecycle.dart", "static void registerGlobalLifecycle(RpccgoStreamLifecycle? lifecycle) {")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "@ffi.DefaultAsset('package:rpccgo_test/gen/rpccgo.dart')")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "import '../../rpccgo.lifecycle.dart';")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "class GreeterRpccgoClient {")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "const GreeterRpccgoClient();")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "class GreeterRpccgoClient {\n  const GreeterRpccgoClient();")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "  ({pb.HelloReply? value, String? error}) SayHello(pb.HelloRequest request) {\n    final responsePtr = pkg_ffi.calloc<ffi.UintPtr>();")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "    try {\n      final errID = _sayHelloRaw(")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "import 'greeter.pb.dart' as pb;")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "({pb.HelloReply? value, String? error}) SayHello(pb.HelloRequest request) {")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "@ffi.Native<_RpccgoMessageUnaryCAbi>(symbol: 'rpccgoMsgTestv1GreeterSayHello')")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "request.writeToBuffer()")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.rpccgo.dart", "pb.HelloReply.fromBuffer(responseBytes.value!)")
	assertGeneratedFileContentDoesNotContain(t, plugin, "test/v1/greeter.greeter.rpccgo.dart",
		"RpccgoResult",
		"RpccgoVoidResult",
		"throw StateError",
		"_throwIfError",
		"ffi.DynamicLibrary library",
		"lookupFunction<",
		"typedef _RpccgoMessageUnaryNative",
		"typedef _RpccgoStreamSendNative",
		"int _sayHello(",
		"int _release(",
		"int _takeErrorText(",
	)
}

func TestGenerateDartEmitsStreamingMessageFFIClient(t *testing.T) {
	plugin := newTestDartPlugin(t, "paths=source_relative,dart_package=rpccgo_test", messageContractTestFile())

	if _, err := GenerateDartWithOptions(plugin); err != nil {
		t.Fatalf("GenerateDartWithOptions() error = %v", err)
	}

	const file = "test/v1/message_contract.greeter.rpccgo.dart"
	assertDartGeneratedFilenames(t, plugin, []string{"rpccgo.dart", "rpccgo.lifecycle.dart", file})
	assertGeneratedContentContains(t, plugin, "rpccgo.lifecycle.dart", "void dispose() {")
	for _, fragment := range []string{
		"({GreeterUploadStream? value, String? error}) UploadStart() {",
		"final lifecycle = Rpccgo.currentLifecycle;",
		"if (lifecycle == null) {",
		"return (value: null, error: 'rpccgo: no stream lifecycle is registered; use Rpccgo.withLifecycle(...) or Rpccgo.registerGlobalLifecycle(...) before starting a stream');",
		"String? Send(pb.MessageRequest request) {",
		"({pb.MessageReply? value, String? error}) Finish() {",
		"ListStart(",
		"final errID = _listStartRaw(",
		"({pb.MessageReply? value, String? error}) Recv() {",
		"String? Finish() {",
		"({GreeterChatStream? value, String? error}) ChatStart() {",
		"class GreeterUploadStream implements RpccgoLifecycleBoundStream {\n  GreeterUploadStream._(this._client, this._lifecycle, this._handle);",
		"  String? Send(pb.MessageRequest request) {\n    final requestBytes = request.writeToBuffer();",
		"    try {\n      final errID = _uploadSendRaw(",
		"  final RpccgoStreamLifecycle _lifecycle;",
		"  bool _releasedLifecycle = false;",
		"  void releaseLifecycle() {",
		"  @override\n  /// Cancels this stream because its lifecycle has been disposed.\n  void cancelFromLifecycle() {",
		"    if (_releasedLifecycle) {\n      return null;\n    }",
		"String? CloseSend() {",
		"typedef _RpccgoStreamRecvCAbi = ffi.Int32 Function(",
		"symbol: 'rpccgoMsgTestv1GreeterUploadStart'",
		"symbol: 'rpccgoMsgTestv1GreeterListRecv'",
		"symbol: 'rpccgoMsgTestv1GreeterChatCloseSend'",
	} {
		assertGeneratedContentContains(t, plugin, file, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, file,
		"({GreeterUploadStream? value, String? error}) Upload() {",
		"({GreeterListStream? value, String? error}) List(pb.MessageRequest request) {",
		"({GreeterChatStream? value, String? error}) Chat() {",
		"({GreeterUploadStream? value, String? error}) StartUpload() {",
		"({GreeterListStream? value, String? error}) StartList(pb.MessageRequest request) {",
		"({GreeterChatStream? value, String? error}) StartChat() {",
		"String? send(pb.MessageRequest request) {",
		"({pb.MessageReply? value, String? error}) read() {",
		"String? finish() {",
		"String? closeSend() {",
		"typedef _RpccgoStreamReadCAbi",
		"symbol: 'rpccgoMsgTestv1GreeterListRead'",
		"symbol: 'rpccgoMsgTestv1GreeterChatRead'",
		"final _RpccgoStreamFinishVoid _uploadCloseSend;",
		"final _RpccgoStreamFinishVoid _listCloseSend;",
		"int _uploadStart(",
		"int _chatCloseSend(",
		"RpccgoResult",
		"RpccgoVoidResult",
		"throw StateError",
		"_throwIfError",
	)
}

func TestGenerateDartLifecycleAttachFailureCancelsOnce(t *testing.T) {
	plugin := newTestDartPlugin(t, "paths=source_relative,dart_package=rpccgo_test", messageContractTestFile())

	if _, err := GenerateDartWithOptions(plugin); err != nil {
		t.Fatalf("GenerateDartWithOptions() error = %v", err)
	}

	const file = "test/v1/message_contract.greeter.rpccgo.dart"
	assertGeneratedContentContains(t, plugin, file, "final lifecycleError = lifecycle.attach(stream);\n      if (lifecycleError != null) {\n        stream.cancelFromLifecycle();\n        return (value: null, error: lifecycleError);\n      }")
	assertGeneratedContentContains(t, plugin, file, "String? Cancel() {\n    if (_releasedLifecycle) {\n      return null;\n    }\n    final errID = _uploadCancelRaw(_handle);")
}

func TestGenerateDartRequiresDartPackage(t *testing.T) {
	plugin := newTestDartPlugin(t, "paths=source_relative", simpleTestFile())

	_, err := GenerateDartWithOptions(plugin)
	if err == nil {
		t.Fatal("GenerateDartWithOptions() error = nil, want missing dart_package error")
	}
	if !strings.Contains(err.Error(), "dart_package parameter is required") {
		t.Fatalf("GenerateDartWithOptions() error = %q, want missing dart_package error", err.Error())
	}
}

func TestGenerateDartRejectsEmptyDartPackage(t *testing.T) {
	request := newTestCodeGeneratorRequest("paths=source_relative,dart_package=", simpleTestFile())

	_, err := DartProtogenOptions().New(request)
	if err == nil {
		t.Fatal("DartProtogenOptions().New() error = nil, want empty dart_package error")
	}
	if !strings.Contains(err.Error(), "dart_package must not be empty") {
		t.Fatalf("DartProtogenOptions().New() error = %q, want empty dart_package error", err.Error())
	}
}

func TestDartProtogenOptionsRejectUnknownParameter(t *testing.T) {
	request := newTestCodeGeneratorRequest("dart_package=rpccgo_test,mode=message", simpleTestFile())

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
