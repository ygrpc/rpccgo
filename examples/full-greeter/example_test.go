package full

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFullGreeterGenerate(t *testing.T) {
	binDir := installProtocPlugins(t)

	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	for _, path := range []string{
		"proto/greeter.pb.go",
		"proto/greeter.greeter.runtime.rpccgo.go",
		"proto/greeter.greeter.server.native.rpccgo.go",
		"proto/greeter.greeter.server.connect.rpccgo.go",
		"proto/greeter.greeter.server.grpc.rpccgo.go",
		"proto/greeter.greeter.remote.connect.rpccgo.go",
		"proto/greeter.greeter.remote.grpc.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s missing: %v", path, err)
		}
	}
}

func TestFullGreeterExample(t *testing.T) {
	binDir := installProtocPlugins(t)

	generate := exec.Command("go", "generate", "./...")
	generate.Env = testEnvWithBinDir(binDir)
	if out, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	cmd := exec.Command("go", "test", "./cmd/rpc", "-run", "^TestFullGreeterTransportAndStreamingMatrix$", "-count=1")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("full example failed: %v\n%s", err, out)
	}
}

func TestFullGreeterMageRunNoPanic(t *testing.T) {
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

func testEnvWithBinDir(binDir string) []string {
	return append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
}

func repoPath(elem ...string) string {
	parts := append([]string{"..", ".."}, elem...)
	return filepath.Join(parts...)
}
