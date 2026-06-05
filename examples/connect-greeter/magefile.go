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
	"connectrpc.com/connect/cmd/protoc-gen-connect-go",
	"../../cmd/protoc-gen-rpc-cgo",
}

// Generate refreshes all generated files for the connect greeter example.
func Generate() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return runWithBinDir(binDir, "go", "generate", "./...")
}

// Test verifies the connect transport matrix and the real c-shared C client demo.
func Test() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runWithBinDir(binDir, "go", "generate", "./..."); err != nil {
		return err
	}
	if err := runWithBinDir(binDir, "go", "test", "./cmd/rpc", "-run", "^TestConnectGreeterTransportAndStreamingMatrix$", "-count=1"); err != nil {
		return err
	}
	return buildAndRunCClient()
}

// Run regenerates the example, then demonstrates switching among supported Connect example servers.
func Run() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runWithBinDir(binDir, "go", "generate", "./..."); err != nil {
		return err
	}
	artifactDir, callerPath, cleanupArtifacts, err := buildCSharedArtifacts()
	if err != nil {
		return err
	}
	defer cleanupArtifacts()

	connectAddr, err := reserveTCPAddr()
	if err != nil {
		return err
	}
	serverBin := filepath.Join(os.TempDir(), "rpccgo-connect-server-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "build", "-o", serverBin, "./cmd/server"); err != nil {
		return err
	}
	defer func() { _ = os.Remove(serverBin) }()

	server := exec.Command(serverBin, "--addr", connectAddr)
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
	if err := waitForTCP(connectAddr); err != nil {
		return err
	}

	for _, step := range []struct {
		title string
		args  []string
	}{
		{
			title: "connect handler registered server",
			args:  []string{"--server=connect_handler", "--route=connect handler registered server"},
		},
		{
			title: "connect remote registered server",
			args:  []string{"--server=connect_remote", "--connect-url=http://" + connectAddr, "--route=connect remote registered server"},
		},
		{
			title: "cgo message registered server",
			args:  []string{"--server=cgo_message", "--route=cgo message registered server"},
		},
		{
			title: "cgo native registered server",
			args:  []string{"--server=cgo_native", "--route=cgo native registered server"},
		},
	} {
		fmt.Println("== switch to " + step.title + " ==")
		if err := runCClient(artifactDir, callerPath, step.args...); err != nil {
			return err
		}
	}
	return nil
}

// Server starts the connect example Connect h2c server.
func Server() error {
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "run", "./cmd/server")
}

func installProtocPlugins() (string, func(), error) {
	binDir, err := os.MkdirTemp("", "rpccgo-connect-example-bin-*")
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

func buildAndRunCClient() error {
	artifactDir, callerPath, cleanup, err := buildCSharedArtifacts()
	if err != nil {
		return err
	}
	defer cleanup()

	return runCClient(artifactDir, callerPath)
}

func buildCSharedArtifacts() (string, string, func(), error) {
	artifactDir, err := os.MkdirTemp("", "rpccgo-connect-c-shared-*")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(artifactDir) }

	libPath := filepath.Join(artifactDir, "librpccgo_connect_greeter.so")
	headerPath := filepath.Join(artifactDir, "librpccgo_connect_greeter.h")
	callerPath := filepath.Join(artifactDir, "connect-greeter-caller")

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
		"-lrpccgo_connect_greeter",
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
