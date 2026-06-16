//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
)

var protocPluginPackages = []string{
	"google.golang.org/protobuf/cmd/protoc-gen-go",
	"connectrpc.com/connect/cmd/protoc-gen-connect-go",
	"../../cmd/protoc-gen-rpc-cgo",
	"../../cmd/protoc-gen-rpc-cgo-dart",
}

// Generate refreshes all generated files for the flutter shared-so example.
func Generate() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return runWithBinDir(binDir, "go", "generate", "./...")
}

// Test verifies the shared-so E2E contracts and build process.
func Test() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runWithBinDir(binDir, "go", "generate", "./..."); err != nil {
		return err
	}
	return runWithBinDir(binDir, "go", "test", "./...", "-count=1", "-skip", "^TestSharedSoDemoMageTestNoPanic$")
}

func installProtocPlugins() (string, func(), error) {
	binDir, err := os.MkdirTemp("", "rpccgo-flutter-example-bin-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(binDir) }
	for _, pkg := range protocPluginPackages {
		if err := runWithEnv(map[string]string{"GOBIN": binDir, "GOFLAGS": "-mod=mod"}, "go", "install", pkg); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("install %s: %w", pkg, err)
		}
	}
	return binDir, cleanup, nil
}

func runWithBinDir(binDir string, name string, args ...string) error {
	return runWithEnv(map[string]string{
		"GOFLAGS": "-mod=mod",
		"PATH":    binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}, name, args...)
}

func runWithEnv(extra map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for key, value := range extra {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	return cmd.Run()
}

// BuildApk compiles the Flutter application into a release APK.
func BuildApk() error {
	return runInFlutterApp("flutter", "build", "apk")
}

// Install deploys the built APK to a connected Android device or emulator.
func Install() error {
	return runInFlutterApp("flutter", "install")
}

func runInFlutterApp(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = "flutter_app"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
