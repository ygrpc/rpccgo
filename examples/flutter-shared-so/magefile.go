//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var protocPluginPackages = []string{
	"google.golang.org/protobuf/cmd/protoc-gen-go",
	"connectrpc.com/connect/cmd/protoc-gen-connect-go",
	"../../cmd/protoc-gen-rpc-cgo",
	"../../cmd/protoc-gen-rpc-cgo-dart",
	"../../cmd/protoc-gen-rpc-cgo-jni",
}

// Generate refreshes all generated files for the flutter shared-so example.
func Generate() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return generateWithBinDir(binDir)
}

// Test verifies the shared-so E2E contracts and build process.
func Test() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := generateWithBinDir(binDir); err != nil {
		return err
	}
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "test", "./...", "-count=1", "-skip", "^TestSharedSoDemoMageTestNoPanic$")
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

func generateWithBinDir(binDir string) error {
	args := []string{
		"--plugin=protoc-gen-go=" + pluginPath(binDir, "protoc-gen-go"),
		"--plugin=protoc-gen-connect-go=" + pluginPath(binDir, "protoc-gen-connect-go"),
		"--plugin=protoc-gen-rpc-cgo=" + pluginPath(binDir, "protoc-gen-rpc-cgo"),
		"--plugin=protoc-gen-rpc-cgo-dart=" + pluginPath(binDir, "protoc-gen-rpc-cgo-dart"),
		"--plugin=protoc-gen-rpc-cgo-jni=" + pluginPath(binDir, "protoc-gen-rpc-cgo-jni"),
		"--unsafe_allow_out_dir_escape",
		"-I", "proto",
		"--go_out=proto",
		"--go_opt=paths=source_relative",
		"--connect-go_out=proto",
		"--connect-go_opt=paths=source_relative",
		"--connect-go_opt=package_suffix=",
		"--connect-go_opt=simple=true",
		"--rpc-cgo_out=proto",
		"--rpc-cgo_opt=paths=source_relative",
		"--rpc-cgo_opt=cgo_dir=../cmd/rpc",
		"--dart_out=flutter_app/lib/gen",
		"--rpc-cgo-dart_out=flutter_app/lib/gen",
		"--rpc-cgo-dart_opt=paths=source_relative,dart_package=rpccgofluttersharedso",
		"--rpc-cgo-jni_out=flutter_app/android/app/src/main",
		"--rpc-cgo-jni_opt=paths=source_relative",
		"--rpc-cgo-jni_opt=jni_class=com.ygrpc.examples.rpccgofluttersharedso.SharedSoDemoJni",
		"--rpc-cgo-jni_opt=rpccgo_header=librpccgo_flutter_shared.h",
		"proto/shared_so.proto",
	}
	return runWithEnv(map[string]string{
		"GOFLAGS": "-mod=mod",
		"PATH":    binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}, "protoc", args...)
}

func pluginPath(binDir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir, name)
}

func runWithEnv(extra map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = mergedEnv(os.Environ(), extra)
	return cmd.Run()
}

func mergedEnv(base []string, extra map[string]string) []string {
	env := make([]string, 0, len(base)+len(extra))
	seen := make(map[string]bool, len(extra))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			env = append(env, entry)
			continue
		}
		value, exists := extra[key]
		if !exists {
			env = append(env, entry)
			continue
		}
		env = append(env, key+"="+value)
		seen[key] = true
	}
	for key, value := range extra {
		if !seen[key] {
			env = append(env, key+"="+value)
		}
	}
	return env
}

// BuildApk compiles the Flutter application into a release APK.
func BuildApk() error {
	if err := Generate(); err != nil {
		return err
	}
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
