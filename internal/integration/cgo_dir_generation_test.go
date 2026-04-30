package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func TestCGODirGeneration(t *testing.T) {
	tests := []struct {
		name       string
		parameter  string
		cgoPackage string
		cgoFiles   []string
	}{
		{
			name:       "default cgo subdir",
			parameter:  "paths=source_relative",
			cgoPackage: "./test/v1/cgo",
			cgoFiles: []string{
				"test/v1/cgo/native_unary.greeter.server.cgo.rpccgo.go",
				"test/v1/cgo/native_unary.greeter.client.cgo.rpccgo.go",
			},
		},
		{
			name:       "external cgo dir",
			parameter:  "paths=source_relative,cgo_dir=../cmd/rpc",
			cgoPackage: "./test/cmd/rpc",
			cgoFiles: []string{
				"test/cmd/rpc/native_unary.greeter.server.cgo.rpccgo.go",
				"test/cmd/rpc/native_unary.greeter.client.cgo.rpccgo.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			plugin := newNativeUnaryTestPluginWithParameter(t, tt.parameter)
			if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderNativeStageFiles: true}); err != nil {
				t.Fatalf("GenerateWithOptions() error = %v", err)
			}

			for _, name := range []string{
				"test/v1/native_unary.greeter.runtime.rpccgo.go",
				"test/v1/native_unary.greeter.server.native.rpccgo.go",
			} {
				assertGeneratedFileExists(t, plugin, name)
			}
			for _, name := range tt.cgoFiles {
				assertGeneratedFileExists(t, plugin, name)
				assertGeneratedFileContains(t, plugin, name, "package main")
				assertGeneratedFileContains(t, plugin, name, `v1 "example.com/nativeunary/test/v1"`)
			}

			writeNativeUnaryGeneratedModule(t, tmp, plugin)
			writeFile(t, filepath.Join(tmp, "test/v1/native_unary_stubs.go"), nativeUnaryStubSource)

			cmd := exec.Command("go", "test", "./test/v1", tt.cgoPackage, "-count=1")
			cmd.Dir = tmp
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("cgo_dir fixture failed: %v\n%s", err, out)
			}
		})
	}
}

func newNativeUnaryTestPluginWithParameter(t *testing.T, parameter string) *protogen.Plugin {
	t.Helper()

	plugin := newNativeUnaryTestPlugin(t)
	plugin.Request.Parameter = &parameter
	plugin, err := generator.ProtogenOptions().New(plugin.Request)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func writeNativeUnaryGeneratedModule(t *testing.T, root string, plugin *protogen.Plugin) {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/nativeunary\n\ngo 1.24.4\n\nrequire rpccgo v0.0.0\n\nreplace rpccgo => "+repoRoot+"\n")
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		if !strings.Contains(name, ".runtime.rpccgo.go") &&
			!strings.Contains(name, ".server.native.rpccgo.go") &&
			!strings.Contains(name, ".server.cgo.rpccgo.go") &&
			!strings.Contains(name, ".client.cgo.rpccgo.go") {
			continue
		}
		writeFile(t, filepath.Join(root, name), generated.GetContent())
	}
}

func assertGeneratedFileExists(t *testing.T, plugin *protogen.Plugin, filename string) {
	t.Helper()
	for _, file := range plugin.Response().GetFile() {
		if file.GetName() == filename {
			return
		}
	}
	t.Fatalf("generated file %q not found", filename)
}

func assertGeneratedFileContains(t *testing.T, plugin *protogen.Plugin, filename, fragment string) {
	t.Helper()
	for _, file := range plugin.Response().GetFile() {
		if file.GetName() != filename {
			continue
		}
		if !strings.Contains(file.GetContent(), fragment) {
			t.Fatalf("generated file %q missing %q: %q", filename, fragment, file.GetContent())
		}
		return
	}
	t.Fatalf("generated file %q not found", filename)
}
