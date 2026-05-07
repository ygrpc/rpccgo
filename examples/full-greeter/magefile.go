//go:build mage

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var protocPluginPackages = []string{
	"google.golang.org/protobuf/cmd/protoc-gen-go",
	"../../cmd/protoc-gen-rpc-cgo",
}

// Generate refreshes all generated files for the full greeter example.
func Generate() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	return runWithBinDir(binDir, "go", "generate", "./...")
}

// Test verifies the full transport and streaming matrix.
func Test() error {
	binDir, cleanup, err := installProtocPlugins()
	if err != nil {
		return err
	}
	defer cleanup()
	if err := runWithBinDir(binDir, "go", "generate", "./..."); err != nil {
		return err
	}
	return runWithBinDir(binDir, "go", "test", "./cmd/rpc", "-run", "^TestFullGreeterTransportAndStreamingMatrix$", "-count=1")
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
	connectAddr, err := reserveTCPAddr()
	if err != nil {
		return err
	}
	grpcAddr, err := reserveTCPAddr()
	if err != nil {
		return err
	}
	server := exec.Command("go", "run", "./cmd/server")
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	server.Env = append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"RPCCGO_FULL_CONNECT_ADDR="+connectAddr,
		"RPCCGO_FULL_GRPC_ADDR="+grpcAddr,
	)
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := server.Start(); err != nil {
		return err
	}
	defer func() {
		if server.Process != nil {
			_ = syscall.Kill(-server.Process.Pid, syscall.SIGKILL)
		}
	}()
	if err := waitForTCP(connectAddr); err != nil {
		return err
	}
	if err := waitForTCP(grpcAddr); err != nil {
		return err
	}
	return runWithEnv(map[string]string{
		"GOFLAGS":                 "-mod=mod",
		"RPCCGO_FULL_CONNECT_URL": "http://" + connectAddr,
		"RPCCGO_FULL_GRPC_ADDR":   grpcAddr,
	}, "go", "run", "./cmd/client")
}

// Server starts the full example Connect h2c and gRPC server.
func Server() error {
	return runWithEnv(map[string]string{"GOFLAGS": "-mod=mod"}, "go", "run", "./cmd/server")
}

func installProtocPlugins() (string, func(), error) {
	binDir, err := os.MkdirTemp("", "rpccgo-full-example-bin-*")
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
