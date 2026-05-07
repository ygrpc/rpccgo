package minimal

import (
	"os"
	"os/exec"
	"testing"
)

func TestMinimalGreeterGenerate(t *testing.T) {
	binDir := installProtocPlugins(t)

	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	for _, path := range []string{
		"gen/greeter/v1/greeter.pb.go",
		"gen/greeter/v1/greeter.greeter.runtime.rpccgo.go",
		"gen/greeter/v1/greeter.greeter.server.native.rpccgo.go",
		"gen/greeter/v1/greeter.greeter.server.connect.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s missing: %v", path, err)
		}
	}
}

func TestMinimalGreeterExample(t *testing.T) {
	binDir := installProtocPlugins(t)

	generate := exec.Command("go", "generate", "./...")
	generate.Env = testEnvWithBinDir(binDir)
	out, err := generate.CombinedOutput()
	if err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}

	cmd := exec.Command("go", "test", "./cmd/server", "./cmd/rpc", "-count=1")
	cmd.Env = testEnvWithBinDir(binDir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("minimal example failed: %v\n%s", err, out)
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
