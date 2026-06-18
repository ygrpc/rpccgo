//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

var dartVersionPattern = regexp.MustCompile(`Dart SDK version: ([0-9]+\.[0-9]+\.[0-9]+)`)

type protocPluginInstall struct {
	goBinDir string
}

// Generate refreshes all generated files for the flutter shared-so example.
func Generate() error {
	plugins, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return generateWithPlugins(plugins)
}

// Test verifies the shared-so E2E contracts and build process.
func Test() error {
	plugins, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := generateWithPlugins(plugins); err != nil {
		return err
	}
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "test", "./...", "-count=1", "-skip", "^TestSharedSoDemoMageTestNoPanic$")
}

func installProtocPlugins() (protocPluginInstall, func(), error) {
	root, err := os.MkdirTemp("", "rpccgo-flutter-example-tools-*")
	if err != nil {
		return protocPluginInstall{}, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	plugins := protocPluginInstall{
		goBinDir: filepath.Join(root, "go-bin"),
	}
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
	if err := installDartProtocPlugin(plugins.goBinDir); err != nil {
		cleanup()
		return protocPluginInstall{}, nil, err
	}
	return plugins, cleanup, nil
}

func generateWithPlugins(plugins protocPluginInstall) error {
	args := []string{
		"--plugin=protoc-gen-go=" + pluginPath(plugins.goBinDir, "protoc-gen-go"),
		"--plugin=protoc-gen-connect-go=" + pluginPath(plugins.goBinDir, "protoc-gen-connect-go"),
		"--plugin=protoc-gen-rpc-cgo=" + pluginPath(plugins.goBinDir, "protoc-gen-rpc-cgo"),
		"--plugin=protoc-gen-rpc-cgo-dart=" + pluginPath(plugins.goBinDir, "protoc-gen-rpc-cgo-dart"),
		"--plugin=protoc-gen-rpc-cgo-jni=" + pluginPath(plugins.goBinDir, "protoc-gen-rpc-cgo-jni"),
		"--plugin=protoc-gen-dart=" + pluginPath(plugins.goBinDir, "protoc-gen-dart"),
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
		"PATH":    plugins.goBinDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}, "protoc", args...)
}

func pluginPath(binDir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir, name)
}

func installDartProtocPlugin(binDir string) error {
	version, err := currentDartVersion()
	if err != nil {
		return err
	}
	snapshot := filepath.Join(pubCacheDir(), "global_packages", "protoc_plugin", "bin", "protoc_plugin.dart-"+version+".snapshot")
	target := pluginPath(binDir, "protoc-gen-dart")
	if runtime.GOOS == "windows" {
		script := "@echo off\r\n" +
			"if exist \"" + snapshot + "\" (\r\n" +
			"  dart \"" + snapshot + "\" %*\r\n" +
			") else (\r\n" +
			"  dart pub global run protoc_plugin:protoc_plugin %*\r\n" +
			")\r\n"
		return os.WriteFile(target, []byte(script), 0o755)
	}
	script := "#!/usr/bin/env sh\n" +
		"if [ -f " + shellQuote(snapshot) + " ]; then\n" +
		"  exec dart " + shellQuote(snapshot) + " \"$@\"\n" +
		"fi\n" +
		"exec dart pub global run protoc_plugin:protoc_plugin \"$@\"\n"
	return os.WriteFile(target, []byte(script), 0o755)
}

func currentDartVersion() (string, error) {
	output, err := exec.Command("dart", "--version").CombinedOutput()
	if err != nil {
		return "", err
	}
	matches := dartVersionPattern.FindStringSubmatch(string(output))
	if len(matches) != 2 {
		return "", fmt.Errorf("could not parse dart version from %q", strings.TrimSpace(string(output)))
	}
	return matches[1], nil
}

func pubCacheDir() string {
	if value := os.Getenv("PUB_CACHE"); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".pub-cache")
	}
	return filepath.Join(home, ".pub-cache")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
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
