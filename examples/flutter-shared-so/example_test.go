package fluttersharedso

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	backend "example.com/rpccgo-flutter-shared-so/internal/backend"
	fluttersharedv1 "example.com/rpccgo-flutter-shared-so/proto"
)

func TestSharedSoDemoInvokeMessageContract(t *testing.T) {
	if err := fluttersharedv1.RegisterSharedSoDemoConnectHandler(backend.NewSharedSoDemoServer()); err != nil {
		t.Fatalf("register shared so demo server: %v", err)
	}
	defer func() {
		if err := fluttersharedv1.ClearSharedSoDemoServer(); err != nil {
			t.Fatalf("clear shared so demo server: %v", err)
		}
	}()

	resp, err := fluttersharedv1.InvokeSharedSoDemoMessageComposeGreeting(context.Background(), &fluttersharedv1.ComposeGreetingRequest{
		Name:   "Ada",
		Caller: "go-test",
	})
	if err != nil {
		t.Fatalf("invoke shared so demo message contract: %v", err)
	}
	if got, want := resp.GetMessage(), "hello Ada via go-test"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
	if got, want := resp.GetServedBy(), "go-connect-handler"; got != want {
		t.Fatalf("served_by = %q, want %q", got, want)
	}
	if got, want := resp.GetLibrary(), "librpccgo_flutter_shared.so"; got != want {
		t.Fatalf("library = %q, want %q", got, want)
	}
}

func TestSharedSoDemoSharesMutableRuntimeState(t *testing.T) {
	server := backend.NewSharedSoDemoServer()
	if err := fluttersharedv1.RegisterSharedSoDemoConnectHandler(server); err != nil {
		t.Fatalf("register shared so demo server: %v", err)
	}
	defer func() {
		if err := fluttersharedv1.ClearSharedSoDemoServer(); err != nil {
			t.Fatalf("clear shared so demo server: %v", err)
		}
	}()

	updated, err := fluttersharedv1.InvokeSharedSoDemoMessageIncrementRuntimeState(context.Background(), &fluttersharedv1.IncrementRuntimeStateRequest{
		Delta:  7,
		Caller: "flutter-ffi-test",
	})
	if err != nil {
		t.Fatalf("increment runtime state: %v", err)
	}
	observed, err := fluttersharedv1.InvokeSharedSoDemoMessageReadRuntimeState(context.Background(), &fluttersharedv1.ReadRuntimeStateRequest{
		Caller: "kotlin-jni-test",
	})
	if err != nil {
		t.Fatalf("read runtime state: %v", err)
	}
	if got, want := observed.GetValue(), updated.GetValue(); got != want {
		t.Fatalf("observed value = %d, want %d", got, want)
	}
	if got, want := observed.GetRevision(), updated.GetRevision(); got != want {
		t.Fatalf("observed revision = %d, want %d", got, want)
	}
	if observed.GetInstanceAddress() == "" || observed.GetInstanceAddress() != updated.GetInstanceAddress() {
		t.Fatalf("instance addresses differ: updated=%q observed=%q", updated.GetInstanceAddress(), observed.GetInstanceAddress())
	}
	if observed.GetPid() <= 0 || observed.GetPid() != updated.GetPid() {
		t.Fatalf("PIDs differ or invalid: updated=%d observed=%d", updated.GetPid(), observed.GetPid())
	}
}

func TestSharedSoDemoMessageStreamContract(t *testing.T) {
	server := backend.NewSharedSoDemoServer()
	if err := fluttersharedv1.RegisterSharedSoDemoConnectHandler(server); err != nil {
		t.Fatalf("register shared so demo server: %v", err)
	}
	defer func() {
		if err := fluttersharedv1.ClearSharedSoDemoServer(); err != nil {
			t.Fatalf("clear shared so demo server: %v", err)
		}
	}()

	handle, err := fluttersharedv1.SharedSoDemoMessageWatchRuntimeStateStart(context.Background(), &fluttersharedv1.ReadRuntimeStateRequest{
		Caller: "kotlin-jni-stream-test",
	})
	if err != nil {
		t.Fatalf("start runtime state stream: %v", err)
	}
	for i := 0; i < 2; i++ {
		resp, err := fluttersharedv1.SharedSoDemoMessageWatchRuntimeStateRecv(context.Background(), handle)
		if err != nil {
			t.Fatalf("recv runtime state stream response %d: %v", i, err)
		}
		if got, want := resp.GetCaller(), "kotlin-jni-stream-test"; got != want {
			t.Fatalf("stream caller = %q, want %q", got, want)
		}
		if resp.GetInstanceAddress() == "" {
			t.Fatalf("stream instance address is empty")
		}
	}
	if _, err := fluttersharedv1.SharedSoDemoMessageWatchRuntimeStateRecv(context.Background(), handle); !errors.Is(err, io.EOF) {
		t.Fatalf("watch runtime state EOF = %v, want EOF", err)
	}

	collect, err := fluttersharedv1.SharedSoDemoMessageCollectRuntimeStateStart(context.Background())
	if err != nil {
		t.Fatalf("start collect runtime state stream: %v", err)
	}
	for _, req := range []*fluttersharedv1.IncrementRuntimeStateRequest{
		{Delta: 2, Caller: "client-stream-a"},
		{Delta: 3, Caller: "client-stream-b"},
	} {
		if err := fluttersharedv1.SharedSoDemoMessageCollectRuntimeStateSend(context.Background(), collect, req); err != nil {
			t.Fatalf("send collect runtime state request: %v", err)
		}
	}
	collected, err := fluttersharedv1.SharedSoDemoMessageCollectRuntimeStateFinish(context.Background(), collect)
	if err != nil {
		t.Fatalf("finish collect runtime state stream: %v", err)
	}
	if got, want := collected.GetValue(), int64(5); got != want {
		t.Fatalf("collected value = %d, want %d", got, want)
	}
	if got, want := collected.GetCaller(), "client-stream-b"; got != want {
		t.Fatalf("collected caller = %q, want %q", got, want)
	}

	stream, err := fluttersharedv1.SharedSoDemoMessageStreamRuntimeStateStart(context.Background(), &fluttersharedv1.ReadRuntimeStateRequest{
		Caller: "server-stream-test",
	})
	if err != nil {
		t.Fatalf("start stream runtime state: %v", err)
	}
	for i := 0; i < 3; i++ {
		resp, err := fluttersharedv1.SharedSoDemoMessageStreamRuntimeStateRecv(context.Background(), stream)
		if err != nil {
			t.Fatalf("recv stream runtime state response %d: %v", i, err)
		}
		if got, want := resp.GetCaller(), "server-stream-test"; got != want {
			t.Fatalf("server stream caller = %q, want %q", got, want)
		}
	}
	if _, err := fluttersharedv1.SharedSoDemoMessageStreamRuntimeStateRecv(context.Background(), stream); !errors.Is(err, io.EOF) {
		t.Fatalf("stream runtime state EOF = %v, want EOF", err)
	}

	chat, err := fluttersharedv1.SharedSoDemoMessageChatRuntimeStateStart(context.Background())
	if err != nil {
		t.Fatalf("start chat runtime state stream: %v", err)
	}
	for i, req := range []*fluttersharedv1.IncrementRuntimeStateRequest{
		{Delta: 4, Caller: "bidi-stream-a"},
		{Delta: 5, Caller: "bidi-stream-b"},
	} {
		if err := fluttersharedv1.SharedSoDemoMessageChatRuntimeStateSend(context.Background(), chat, req); err != nil {
			t.Fatalf("send chat runtime state request %d: %v", i, err)
		}
		resp, err := fluttersharedv1.SharedSoDemoMessageChatRuntimeStateRecv(context.Background(), chat)
		if err != nil {
			t.Fatalf("recv chat runtime state response %d: %v", i, err)
		}
		if got, want := resp.GetCaller(), req.GetCaller(); got != want {
			t.Fatalf("bidi caller = %q, want %q", got, want)
		}
	}
	if err := fluttersharedv1.SharedSoDemoMessageChatRuntimeStateCloseSend(context.Background(), chat); err != nil {
		t.Fatalf("close send chat runtime state stream: %v", err)
	}
	if err := fluttersharedv1.SharedSoDemoMessageChatRuntimeStateFinish(context.Background(), chat); err != nil {
		t.Fatalf("finish chat runtime state stream: %v", err)
	}
}

func TestSharedSoDemoFlutterProjectContracts(t *testing.T) {
	assertFileContains(t, "flutter_app/hook/build.dart", "DynamicLoadingSystem(")
	assertFileContains(t, "flutter_app/hook/build.dart", "Uri.file('librpccgo_flutter_shared.so')")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "System.loadLibrary(\"rpccgo_flutter_shared\")")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "System.loadLibrary(\"rpccgo_flutter_shared_jni\")")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "SharedSoDemoJni.ComposeGreeting")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "SharedSoDemoJni.ReadRuntimeState")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "object : SharedSoDemoJni.SharedSoDemoStreamRuntimeStateListener")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/SharedSoDemoJni.kt", "fun WatchRuntimeStateStart")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/SharedSoDemoJni.kt", "fun CollectRuntimeStateStart")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/SharedSoDemoJni.kt", "fun StreamRuntimeStateStart")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/SharedSoDemoJni.kt", "fun ChatRuntimeStateStart")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "rpccgoMsgFluttersharedv1SharedSoDemoWatchRuntimeStateStart")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "rpccgoMsgFluttersharedv1SharedSoDemoCollectRuntimeStateFinish")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "rpccgoMsgFluttersharedv1SharedSoDemoChatRuntimeStateCloseSend")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "JNIEXPORT jint JNICALL JNI_OnLoad")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "AttachCurrentThread")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "void* rawEnv = nullptr;")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "env = static_cast<JNIEnv*>(rawEnv);")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/rpccgo/shared_so.shared_so_demo.jni.cpp", "JNIEnv* attachedEnv = nullptr;")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/CMakeLists.txt", "rpccgo_flutter_shared_jni")
	assertFileContains(t, "flutter_app/android/app/src/main/cpp/CMakeLists.txt", "-l:librpccgo_flutter_shared.so")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "ComposeGreetingRequest.newBuilder()")
	assertFileContains(t, "flutter_app/lib/main.dart", "IncrementRuntimeState")
	assertFileContains(t, "flutter_app/lib/main.dart", "CollectRuntimeState")
	assertFileContains(t, "flutter_app/lib/main.dart", "StreamRuntimeStateStartCallback")
	assertFileContains(t, "flutter_app/lib/main.dart", "onRecv:")
	assertFileContains(t, "flutter_app/lib/main.dart", "onDone:")
	assertFileContains(t, "flutter_app/lib/main.dart", "ChatRuntimeState")
	assertFileContains(t, "flutter_app/lib/main.dart", "Latest Activity")
	assertFileContains(t, "flutter_app/lib/main.dart", "_latestActivityBody")
	assertFileContains(t, "flutter_app/lib/main.dart", "Same Go counter")
	assertFileContains(t, "flutter_app/lib/main.dart", "continues after Flutter FFI")
	assertFileContains(t, "flutter_app/lib/main.dart", "+2+3 ->")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "+4+5 ->")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "dependsOn(buildSharedSoForAndroid)")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "externalNativeBuild")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "abiFilters.addAll(listOf(\"arm64-v8a\", \"armeabi-v7a\", \"x86_64\"))")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "protobuf-javalite")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "proguard-rules.pro")
	assertFileContains(t, "flutter_app/android/app/proguard-rules.pro", "GeneratedMessageLite")
	assertFileContains(t, "flutter_app/android/app/proguard-rules.pro", "<fields>")
	assertFileContains(t, "flutter_app/lib/gen/shared_so.shared_so_demo.rpccgo.dart", "@ffi.DefaultAsset('package:rpccgofluttersharedso/gen/rpccgo.dart')")
}

func TestSharedSoDemoCSharedBuild(t *testing.T) {
	artifactDir := t.TempDir()
	libPath := filepath.Join(artifactDir, "librpccgo_flutter_shared.so")
	headerPath := filepath.Join(artifactDir, "librpccgo_flutter_shared.h")

	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", libPath, "./cmd/rpc")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build c-shared library failed: %v\n%s", err, out)
	}
	header, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatalf("read c-shared header: %v", err)
	}
	for _, fragment := range []string{
		"rpccgoMsgFluttersharedv1SharedSoDemoComposeGreeting",
		"rpccgoMsgFluttersharedv1SharedSoDemoWatchRuntimeStateStart",
		"rpccgoMsgFluttersharedv1SharedSoDemoCollectRuntimeStateStart",
		"rpccgoMsgFluttersharedv1SharedSoDemoStreamRuntimeStateRecv",
		"rpccgoMsgFluttersharedv1SharedSoDemoChatRuntimeStateCloseSend",
		"rpccgoTakeErrorText",
		"rpccgoRelease",
	} {
		if !bytes.Contains(header, []byte(fragment)) {
			t.Fatalf("header missing %q", fragment)
		}
	}
	if bytes.Contains(header, []byte("Java_com_ygrpc_examples_rpccgofluttersharedso_SharedSoDemoJni")) {
		t.Fatalf("c-shared header still contains Go-exported JNI symbols")
	}
}

func TestSharedSoDemoJNIAdapterDoesNotNeedBuildHostPath(t *testing.T) {
	patterns := []string{
		filepath.Join("flutter_app", "build", "app", "intermediates", "cxx", "*", "*", "obj", "*", "librpccgo_flutter_shared_jni.so"),
		filepath.Join("flutter_app", "build", "app", "intermediates", "cmake", "*", "obj", "*", "librpccgo_flutter_shared_jni.so"),
	}
	var soPaths []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob JNI adapter shared libraries: %v", err)
		}
		soPaths = append(soPaths, matches...)
	}
	if len(soPaths) == 0 {
		t.Skip("JNI adapter shared libraries have not been built")
	}

	readelf := findAndroidLLVMReadelf(t)
	for _, soPath := range soPaths {
		out, err := exec.Command(readelf, "-d", soPath).CombinedOutput()
		if err != nil {
			t.Fatalf("read JNI adapter dynamic section for %s: %v\n%s", soPath, err, out)
		}
		text := string(out)
		if !strings.Contains(text, "Shared library: [librpccgo_flutter_shared.so]") {
			t.Fatalf("%s missing relative Go shared library dependency:\n%s", soPath, text)
		}
		if strings.Contains(text, "/jniLibs/") || strings.Contains(text, "/home/") {
			t.Fatalf("%s dynamic dependencies contain build-host path:\n%s", soPath, text)
		}
	}
}

func findAndroidLLVMReadelf(t *testing.T) string {
	t.Helper()

	if ndk := os.Getenv("ANDROID_NDK_HOME"); ndk != "" {
		path := filepath.Join(ndk, "toolchains", "llvm", "prebuilt", androidHostTag(t), "bin", "llvm-readelf")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	sdk := os.Getenv("ANDROID_HOME")
	if sdk == "" {
		sdk = os.Getenv("ANDROID_SDK_ROOT")
	}
	if sdk == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("cannot resolve home directory: %v", err)
		}
		sdk = filepath.Join(home, "Android", "Sdk")
	}
	matches, err := filepath.Glob(filepath.Join(sdk, "ndk", "*", "toolchains", "llvm", "prebuilt", androidHostTag(t), "bin", "llvm-readelf"))
	if err != nil || len(matches) == 0 {
		t.Skip("llvm-readelf not found under Android SDK")
	}
	return matches[len(matches)-1]
}

func androidHostTag(t *testing.T) string {
	t.Helper()

	switch runtime.GOOS {
	case "linux":
		return "linux-x86_64"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "darwin-arm64"
		}
		return "darwin-x86_64"
	default:
		t.Skipf("unsupported Android NDK host OS %s", runtime.GOOS)
		return ""
	}
}

func assertFileContains(t *testing.T, path, fragment string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Contains(data, []byte(fragment)) {
		t.Fatalf("%s missing %q", path, fragment)
	}
}

func TestSharedSoDemoMageTestNoPanic(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/magefile/mage", "test")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mage test error = %v\n%s", err, out)
	}
	if bytes.Contains(out, []byte("panic:")) {
		t.Fatalf("mage test output contains panic:\n%s", out)
	}
}
