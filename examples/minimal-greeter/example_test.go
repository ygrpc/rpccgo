package minimal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMinimalGreeterGenerate(t *testing.T) {
	binDir := installProtocPlugins(t)
	generateMinimalGreeter(t, binDir)

	for _, path := range []string{
		"gen/greeter/v1/greeter.pb.go",
		"gen/greeter/v1/greeter.greeter.runtime.rpccgo.go",
		"gen/greeter/v1/greeter.greeter.server.native.rpccgo.go",
		"gen/greeter/v1/greeter.greeter.server.connect.rpccgo.go",
		"gen/greeter/v1/greeter.greeter.remote.connect.rpccgo.go",
		"cmd/rpc/greeter.exports.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s missing: %v", path, err)
		}
	}

	assertFileContains(t, "cmd/rpc/greeter.exports.cgo.rpccgo.go", "//export rpccgo_take_error_text")
	assertFileContains(t, "cmd/rpc/greeter.exports.cgo.rpccgo.go", "//export rpccgo_release")
	assertFileContains(t, "cmd/rpc/greeter.greeter.client.cgo.rpccgo.go", "//export rpccgo_native_go_Greeter_SayHello")
	assertFileContains(t, "cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go", "//export rpccgo_msg_go_Greeter_SayHello")
}

func TestMinimalGreeterExample(t *testing.T) {
	binDir := installProtocPlugins(t)
	generateMinimalGreeter(t, binDir)

	cmd := exec.Command("go", "test", "./cmd/rpc", "-count=1")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("minimal example failed: %v\n%s", err, out)
	}
}

func TestMinimalGreeterCSharedClientExample(t *testing.T) {
	binDir := installProtocPlugins(t)
	generateMinimalGreeter(t, binDir)

	artifactDir := t.TempDir()
	headerPath, callerPath := buildMinimalGreeterCSharedArtifacts(t, artifactDir)

	header, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatalf("read c-shared header error = %v", err)
	}
	for _, symbol := range []string{
		"rpccgo_native_go_Greeter_SayHello",
		"rpccgo_msg_go_Greeter_SayHello",
		"rpccgo_take_error_text",
		"rpccgo_release",
		"rpccgo_minimal_greeter_register_native_server",
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
		"native unary: hello, minimal-c",
		"native output error: rpccgo: native client output pointer is nil",
	} {
		if !bytes.Contains(out, []byte(marker)) {
			t.Fatalf("c client output missing %q\n%s", marker, out)
		}
	}
}

func TestMinimalGreeterMageRunNoPanic(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/magefile/mage", "run")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mage run error = %v\n%s", err, out)
	}
	if bytes.Contains(out, []byte("panic:")) {
		t.Fatalf("mage run output contains panic:\n%s", out)
	}
}

func installProtocPlugins(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	for _, pkg := range []string{
		"google.golang.org/protobuf/cmd/protoc-gen-go",
		"../../cmd/protoc-gen-rpc-cgo",
	} {
		cmd := exec.Command("go", "install", pkg)
		cmd.Env = append(os.Environ(), "GOBIN="+binDir, "GOFLAGS=-mod=mod")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go install %s failed: %v\n%s", pkg, err, out)
		}
	}
	return binDir
}

func generateMinimalGreeter(t *testing.T, binDir string) {
	t.Helper()

	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}
}

func buildMinimalGreeterCSharedArtifacts(t *testing.T, artifactDir string) (string, string) {
	t.Helper()

	libPath := filepath.Join(artifactDir, "librpccgo_minimal_greeter.so")
	headerPath := filepath.Join(artifactDir, "librpccgo_minimal_greeter.h")
	callerPath := filepath.Join(artifactDir, "minimal-greeter-caller")

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
		"-lrpccgo_minimal_greeter",
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
