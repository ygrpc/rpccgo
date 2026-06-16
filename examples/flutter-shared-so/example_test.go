package fluttersharedso

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
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

func TestSharedSoDemoFlutterProjectContracts(t *testing.T) {
	assertFileContains(t, "flutter_app/hook/build.dart", "DynamicLoadingSystem(")
	assertFileContains(t, "flutter_app/hook/build.dart", "Uri.file('librpccgo_flutter_shared.so')")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "System.loadLibrary(\"rpccgo_flutter_shared\")")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "nativeComposeGreeting")
	assertFileContains(t, "flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt", "nativeReadRuntimeState")
	assertFileContains(t, "flutter_app/lib/main.dart", "IncrementRuntimeState")
	assertFileContains(t, "flutter_app/lib/main.dart", "Latest Activity")
	assertFileContains(t, "flutter_app/lib/main.dart", "_latestActivityBody")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "dependsOn(buildSharedSoForAndroid)")
	assertFileContains(t, "flutter_app/android/app/build.gradle.kts", "abiFilters += listOf(\"arm64-v8a\", \"x86_64\")")
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
		"rpccgoTakeErrorText",
		"rpccgoRelease",
	} {
		if !bytes.Contains(header, []byte(fragment)) {
			t.Fatalf("header missing %q", fragment)
		}
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
