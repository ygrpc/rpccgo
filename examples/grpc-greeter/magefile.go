//go:build mage

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

var protocPluginPackages = []string{
	"google.golang.org/protobuf/cmd/protoc-gen-go",
	"google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1",
	"../../cmd/protoc-gen-rpc-cgo",
}

// Generate refreshes all generated files for the gRPC greeter example.
func Generate() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return generateGRPCProtos(binDir)
}

// Test verifies the full gRPC transport matrix and the real c-shared C client demo.
func Test() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := generateGRPCProtos(binDir); err != nil {
		return err
	}
	if err := runWithBinDir(binDir, "go", "test", "./cmd/rpc", "-run", "^TestGRPCGreeterTransportAndStreamingMatrix$", "-count=1"); err != nil {
		return err
	}
	return buildAndRunCClient()
}

// Run regenerates the example, then demonstrates switching among supported gRPC example servers.
func Run() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := generateGRPCProtos(binDir); err != nil {
		return err
	}
	artifactDir, callerPath, cleanupArtifacts, err := buildCSharedArtifacts()
	if err != nil {
		return err
	}
	defer cleanupArtifacts()

	addr, err := reserveTCPAddr()
	if err != nil {
		return err
	}
	serverBin := filepath.Join(os.TempDir(), "rpccgo-grpc-server-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "build", "-o", serverBin, "./cmd/server"); err != nil {
		return err
	}
	defer func() { _ = os.Remove(serverBin) }()

	server := exec.Command(serverBin, "--addr", addr)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	server.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if err := server.Start(); err != nil {
		return err
	}
	defer func() {
		if server.Process != nil {
			_ = server.Process.Kill()
			_ = server.Wait()
		}
	}()
	if err := waitForTCP(addr); err != nil {
		return err
	}

	for _, step := range []struct {
		title string
		args  []string
	}{
		{
			title: "grpc server registered server",
			args:  []string{"--server=grpc_server", "--route=grpc server registered server"},
		},
		{
			title: "grpc remote registered server",
			args:  []string{"--server=grpc_remote", "--grpc-target=" + addr, "--route=grpc remote registered server"},
		},
		{
			title: "go native registered server",
			args:  []string{"--server=go_native", "--route=go native registered server"},
		},
	} {
		fmt.Println("== switch to " + step.title + " ==")
		if err := runCClient(artifactDir, callerPath, step.args...); err != nil {
			return err
		}
	}
	return nil
}

// Server starts the gRPC greeter example server.
func Server() error {
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "run", "./cmd/server")
}

func installProtocPlugins() (string, func(), error) {
	binDir, err := os.MkdirTemp("", "rpccgo-grpc-example-bin-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(binDir) }
	for _, pkg := range protocPluginPackages {
		if pkg == "../../cmd/protoc-gen-rpc-cgo" {
			if err := runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "build", "-o", filepath.Join(binDir, "protoc-gen-rpc-cgo"), pkg); err != nil {
				cleanup()
				return "", nil, fmt.Errorf("build %s: %w", pkg, err)
			}
			continue
		}
		if err := runWithEnv(map[string]string{"GOBIN": binDir, "GOFLAGS": "-mod=mod"}, "go", "install", pkg); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("install %s: %w", pkg, err)
		}
	}
	return binDir, cleanup, nil
}

func generateGRPCProtos(binDir string) error {
	args := []string{
		"--unsafe_allow_out_dir_escape",
		"-I", ".",
		"--plugin=protoc-gen-go=" + pluginPath(binDir, "protoc-gen-go"),
		"--plugin=protoc-gen-go-grpc=" + pluginPath(binDir, "protoc-gen-go-grpc"),
		"--plugin=protoc-gen-rpc-cgo=" + pluginPath(binDir, "protoc-gen-rpc-cgo"),
		"--go_out=.",
		"--go_opt=module=example.com/rpccgo-grpc",
		"--go-grpc_out=.",
		"--go-grpc_opt=module=example.com/rpccgo-grpc",
		"--rpc-cgo_out=.",
		"--rpc-cgo_opt=module=example.com/rpccgo-grpc",
		"--rpc-cgo_opt=cgo_dir=../../../cmd/rpc",
		"proto/greeter.proto",
	}
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "protoc", args...)
}

func pluginPath(binDir, name string) string {
	return filepath.Join(binDir, name)
}

func buildAndRunCClient() error {
	artifactDir, callerPath, cleanup, err := buildCSharedArtifacts()
	if err != nil {
		return err
	}
	defer cleanup()

	return runCClient(artifactDir, callerPath)
}

func buildCSharedArtifacts() (string, string, func(), error) {
	artifactDir, err := os.MkdirTemp("", "rpccgo-grpc-c-shared-*")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(artifactDir) }

	libPath := filepath.Join(artifactDir, "librpccgo_grpc_greeter.so")
	headerPath := filepath.Join(artifactDir, "librpccgo_grpc_greeter.h")
	callerPath := filepath.Join(artifactDir, "grpc-greeter-caller")

	if err := runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "build", "-buildmode=c-shared", "-o", libPath, "./cmd/rpc"); err != nil {
		cleanup()
		return "", "", nil, err
	}
	if _, err := os.Stat(headerPath); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("c-shared header missing: %w", err)
	}
	if err := runWithEnv(nil,
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
	); err != nil {
		cleanup()
		return "", "", nil, err
	}
	return artifactDir, callerPath, cleanup, nil
}

func runCClient(artifactDir, callerPath string, args ...string) error {
	env := map[string]string{
		"LD_LIBRARY_PATH": artifactDir + string(os.PathListSeparator) + os.Getenv("LD_LIBRARY_PATH"),
	}
	return runWithEnv(env, callerPath, args...)
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

func reserveTCPAddr() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", err
	}
	return addr, nil
}

func waitForTCP(addr string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server did not start listening on %s", addr)
}
