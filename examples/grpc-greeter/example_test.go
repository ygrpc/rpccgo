package grpcgreeter

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGRPCGreeterGenerate(t *testing.T) {
	binDir := installProtocPlugins(t)
	generateGRPCGreeter(t, binDir)

	for _, path := range []string{
		"gen/greeter/v1/greeter.pb.go",
		"gen/greeter/v1/greeter_grpc.pb.go",
		"gen/greeter/v1/greeter.greeter.runtime.rpccgo.go",
		"gen/greeter/v1/greeter.greeter.server.native.rpccgo.go",
		"cmd/rpc/rpccgo.exports.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.native.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s missing: %v", path, err)
		}
	}

	assertFileContains(t, "cmd/rpc/rpccgo.exports.cgo.rpccgo.go", "//export rpccgoRegisterFree")
	assertFileContains(t, "cmd/rpc/rpccgo.exports.cgo.rpccgo.go", "//export rpccgoStoreErrorText")
	assertFileContains(t, "cmd/rpc/rpccgo.exports.cgo.rpccgo.go", "//export rpccgoTakeErrorText")
	assertFileContains(t, "cmd/rpc/rpccgo.exports.cgo.rpccgo.go", "//export rpccgoRelease")
	assertFileContains(t, "cmd/rpc/greeter.greeter.client.native.cgo.rpccgo.go", "//export rpccgoNativeGreeterv1GreeterSayHello")
	assertFileContains(t, "cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go", "//export rpccgoMsgGreeterv1GreeterSayHello")
	assertFileContains(t, "gen/greeter/v1/greeter.greeter.runtime.rpccgo.go", "func RegisterGreeterGRPCRemoteServer(client GreeterClient) error")
}

func TestGRPCGreeterExample(t *testing.T) {
	binDir := installProtocPlugins(t)
	generateGRPCGreeter(t, binDir)

	cmd := exec.Command("go", "test", "./cmd/rpc", "-count=1")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gRPC example failed: %v\n%s", err, out)
	}
}

func TestGRPCGreeterCSharedClientExample(t *testing.T) {
	binDir := installProtocPlugins(t)
	generateGRPCGreeter(t, binDir)

	artifactDir := t.TempDir()
	headerPath, callerPath := buildGRPCGreeterCSharedArtifacts(t, artifactDir)

	header, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatalf("read c-shared header error = %v", err)
	}
	for _, symbol := range []string{
		"rpccgoNativeGreeterv1GreeterSayHello",
		"rpccgoNativeGreeterv1GreeterCollectStart",
		"rpccgoNativeGreeterv1GreeterBroadcastStart",
		"rpccgoNativeGreeterv1GreeterChatStart",
		"rpccgoMsgGreeterv1GreeterSayHello",
		"rpccgoRegisterFree",
		"rpccgoStoreErrorText",
		"rpccgoTakeErrorText",
		"rpccgoRelease",
	} {
		if !bytes.Contains(header, []byte(symbol)) {
			t.Fatalf("c-shared header missing %q", symbol)
		}
	}

	cmd := exec.Command(callerPath)
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+artifactDir+string(os.PathListSeparator)+os.Getenv("LD_LIBRARY_PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("c client example failed: %v\n%s", err, out)
	}
	for _, marker := range []string{
		"native unary: hello ffi from c",
		"native collect: collect:ada,grace",
		"native broadcast: broadcast[0]:stream",
		"native broadcast: broadcast[1]:stream",
		"native chat c->server: ada",
		"native chat server->c: chat:ada",
		"native chat c->server: grace",
		"native chat server->c: chat:grace",
	} {
		if !bytes.Contains(out, []byte(marker)) {
			t.Fatalf("c client output missing %q\n%s", marker, out)
		}
	}
}

func TestGRPCGreeterMageRunNoPanic(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/magefile/mage", "run")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mage run error = %v\n%s", err, out)
	}
	if bytes.Contains(out, []byte("panic:")) {
		t.Fatalf("mage run output contains panic:\n%s", out)
	}
	for _, marker := range []string{
		"route: grpc server registered server",
		"route: grpc remote registered server",
		"route: go native registered server",
	} {
		if !bytes.Contains(out, []byte(marker)) {
			t.Fatalf("mage run output missing %q\n%s", marker, out)
		}
	}
}

func installProtocPlugins(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	for _, pkg := range []string{
		"google.golang.org/protobuf/cmd/protoc-gen-go",
		"google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1",
		"../../cmd/protoc-gen-rpc-cgo",
	} {
		args := []string{"install", pkg}
		if pkg == "../../cmd/protoc-gen-rpc-cgo" {
			args = []string{"build", "-o", filepath.Join(binDir, "protoc-gen-rpc-cgo"), pkg}
		}
		cmd := exec.Command("go", args...)
		cmd.Env = append(os.Environ(), "GOBIN="+binDir, "GOFLAGS=-mod=mod")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s %s failed: %v\n%s", args[0], pkg, err, out)
		}
	}
	return binDir
}

func generateGRPCGreeter(t *testing.T, binDir string) {
	t.Helper()

	args := []string{
		"--unsafe_allow_out_dir_escape",
		"-I", ".",
		"--plugin=protoc-gen-go=" + filepath.Join(binDir, "protoc-gen-go"),
		"--plugin=protoc-gen-go-grpc=" + filepath.Join(binDir, "protoc-gen-go-grpc"),
		"--plugin=protoc-gen-rpc-cgo=" + filepath.Join(binDir, "protoc-gen-rpc-cgo"),
		"--go_out=.",
		"--go_opt=module=example.com/rpccgo-grpc",
		"--go-grpc_out=.",
		"--go-grpc_opt=module=example.com/rpccgo-grpc",
		"--rpc-cgo_out=.",
		"--rpc-cgo_opt=module=example.com/rpccgo-grpc",
		"--rpc-cgo_opt=cgo_dir=../../../cmd/rpc",
		"proto/greeter.proto",
	}
	cmd := exec.Command("protoc", args...)
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("protoc generate failed: %v\n%s", err, out)
	}
}

func buildGRPCGreeterCSharedArtifacts(t *testing.T, artifactDir string) (string, string) {
	t.Helper()

	libPath := filepath.Join(artifactDir, "librpccgo_grpc_greeter.so")
	headerPath := filepath.Join(artifactDir, "librpccgo_grpc_greeter.h")
	callerPath := filepath.Join(artifactDir, "grpc-greeter-caller")

	build := exec.Command("go", "build", "-buildmode=c-shared", "-o", libPath, "./cmd/rpc")
	build.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build c-shared library failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(headerPath); err != nil {
		t.Fatalf("c-shared header missing: %v", err)
	}

	compile := exec.Command(
		"cc",
		"-std=c11",
		"-Wall",
		"-Wextra",
		"-o", callerPath,
		"./c/main.c",
		"-I"+artifactDir,
		"-L"+artifactDir,
		"-lrpccgo_grpc_greeter",
		"-Wl,-rpath,$ORIGIN",
	)
	compile.Env = os.Environ()
	out, err = compile.CombinedOutput()
	if err != nil {
		t.Fatalf("compile c client failed: %v\n%s", err, out)
	}

	return headerPath, callerPath
}

func assertFileContains(t *testing.T, path, fragment string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s error = %v", path, err)
	}
	if !bytes.Contains(data, []byte(fragment)) {
		t.Fatalf("%s missing %q", path, fragment)
	}
}

func testEnvWithBinDir(binDir string) []string {
	return append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
}
