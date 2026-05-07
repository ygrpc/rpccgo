//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
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

// Run regenerates the example and then runs its verification path.
func Run() error {
	return Test()
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
