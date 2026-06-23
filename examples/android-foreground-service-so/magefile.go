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
	"../../cmd/protoc-gen-rpc-cgo-jni",
}

type protocPluginInstall struct {
	goBinDir string
}

// Generate refreshes generated Go and Android JNI files.
func Generate() error {
	plugins, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return generateWithPlugins(plugins)
}

// Test verifies the Go contract and Android project files.
func Test() error {
	if err := Generate(); err != nil {
		return err
	}
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "test", "./...", "-count=1")
}

func installProtocPlugins() (protocPluginInstall, func(), error) {
	root, err := os.MkdirTemp("", "rpccgo-android-foreground-service-tools-*")
	if err != nil {
		return protocPluginInstall{}, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	plugins := protocPluginInstall{goBinDir: filepath.Join(root, "go-bin")}
	if err := os.MkdirAll(plugins.goBinDir, 0o755); err != nil {
		cleanup()
		return protocPluginInstall{}, nil, err
	}
	for _, pkg := range protocPluginPackages {
		if err := runWithEnv(map[string]string{"GOBIN": plugins.goBinDir, "GOFLAGS": "-mod=mod"}, "go", "install", pkg); err != nil {
			cleanup()
			return protocPluginInstall{}, nil, fmt.Errorf("install %s: %w", pkg, err)
		}
	}
	return plugins, cleanup, nil
}

func generateWithPlugins(plugins protocPluginInstall) error {
	_ = os.RemoveAll(filepath.Join("android_app", "app", "src", "main", "java", "examples", "android", "foregroundservice", "v1"))
	if err := os.MkdirAll(filepath.Join("android_app", "app", "src", "main", "java"), 0o755); err != nil {
		return err
	}
	args := []string{
		"--plugin=protoc-gen-go=" + pluginPath(plugins.goBinDir, "protoc-gen-go"),
		"--plugin=protoc-gen-connect-go=" + pluginPath(plugins.goBinDir, "protoc-gen-connect-go"),
		"--plugin=protoc-gen-rpc-cgo=" + pluginPath(plugins.goBinDir, "protoc-gen-rpc-cgo"),
		"--plugin=protoc-gen-rpc-cgo-jni=" + pluginPath(plugins.goBinDir, "protoc-gen-rpc-cgo-jni"),
		"--unsafe_allow_out_dir_escape",
		"-I", "proto",
		"--go_out=proto",
		"--go_opt=paths=source_relative",
		"--connect-go_out=proto",
		"--connect-go_opt=paths=source_relative",
		"--connect-go_opt=package_suffix=",
		"--connect-go_opt=simple=true",
		"--java_out=lite:android_app/app/src/main/java",
		"--rpc-cgo_out=proto",
		"--rpc-cgo_opt=paths=source_relative",
		"--rpc-cgo_opt=cgo_dir=../cmd/rpc",
		"--rpc-cgo-jni_out=android_app/app/src/main",
		"--rpc-cgo-jni_opt=paths=source_relative",
		"--rpc-cgo-jni_opt=jni_class=com.ygrpc.examples.rpccgoandroidforegroundservice.ForegroundServiceDemoJni",
		"--rpc-cgo-jni_opt=rpccgo_header=librpccgo_android_foreground_service.h",
		"proto/foreground_service.proto",
	}
	return runWithEnv(map[string]string{
		"GOFLAGS": "-mod=mod",
		"PATH":    plugins.goBinDir + string(os.PathListSeparator) + os.Getenv("PATH"),
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

// BuildApk compiles the Android debug APK.
func BuildApk() error {
	if err := Generate(); err != nil {
		return err
	}
	if err := runWithEnv(nil, "bash", "android_app/tool/build_android_so.sh"); err != nil {
		return err
	}
	return runWithEnv(nil, "./android_app/gradlew", "-p", "android_app", "assembleDebug")
}
