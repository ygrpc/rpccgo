package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderCGOSharedFilesBuildCSharedWithoutUserMain(t *testing.T) {
	plugin := newGeneratedLayoutPlugin(t)

	tmp := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/cgomain\n\ngo 1.24.4\n\nrequire github.com/ygrpc/rpccgo v0.0.0\n\nreplace github.com/ygrpc/rpccgo => "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	goSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.sum"), goSum, 0o644); err != nil {
		t.Fatalf("write go.sum: %v", err)
	}

	var cgoPackageDir string
	var wroteMain bool
	var wroteExports bool
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		isExports := strings.HasSuffix(name, "/rpccgo.exports.cgo.rpccgo.go")
		isMain := strings.HasSuffix(name, "/main.go")
		if !isExports && !isMain {
			continue
		}
		target := filepath.Join(tmp, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir generated dir: %v", err)
		}
		if err := os.WriteFile(target, []byte(generated.GetContent()), 0o644); err != nil {
			t.Fatalf("write generated file %s: %v", name, err)
		}
		cgoPackageDir = filepath.Dir(name)
		wroteExports = wroteExports || isExports
		wroteMain = wroteMain || isMain
	}
	if !wroteExports {
		t.Fatal("shared cgo exports artifact was not generated")
	}
	if !wroteMain {
		t.Fatal("shared cgo main artifact was not generated")
	}

	libPath := filepath.Join(tmp, "librpccgo_cgo_main.so")
	cmd := exec.Command("go", "build", "-mod=mod", "-buildmode=c-shared", "-o", libPath, "./"+filepath.ToSlash(cgoPackageDir))
	cmd.Dir = tmp
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated cgo exports c-shared build failed: %v\n%s", err, output)
	}
}
