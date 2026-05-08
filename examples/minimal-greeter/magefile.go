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
	"../../cmd/protoc-gen-rpc-cgo",
}

// Generate refreshes all generated files for the minimal greeter example.
func Generate() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return runWithBinDir(binDir, "go", "generate", "./...")
}

// Test verifies the minimal generated server and cgo client path.
func Test() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runWithBinDir(binDir, "go", "generate", "./..."); err != nil {
		return err
	}
	return runWithBinDir(binDir, "go", "test", "./cmd/server", "./cmd/rpc", "-count=1")
}

// Run regenerates the example, starts the server, runs the client, and exits.
func Run() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runWithBinDir(binDir, "go", "generate", "./..."); err != nil {
		return err
	}
	addr, err := reserveTCPAddr()
	if err != nil {
		return err
	}
	serverBin := filepath.Join(os.TempDir(), "rpccgo-minimal-server-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "build", "-o", serverBin, "./cmd/server"); err != nil {
		return err
	}
	defer func() { _ = os.Remove(serverBin) }()

	server := exec.Command(serverBin)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	server.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "RPCCGO_MINIMAL_CONNECT_ADDR="+addr)
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
	return runWithEnv(map[string]string{
		"GOFLAGS":                    "-mod=mod",
		"RPCCGO_MINIMAL_CONNECT_URL": "http://" + addr,
	}, "go", "run", "./cmd/client")
}

// Server starts the minimal example Connect server.
func Server() error {
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "run", "./cmd/server")
}

func installProtocPlugins() (string, func(), error) {
	binDir, err := os.MkdirTemp("", "rpccgo-minimal-example-bin-*")
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
