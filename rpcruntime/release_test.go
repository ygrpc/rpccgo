package rpcruntime

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPinBytesAndRelease(t *testing.T) {
	data := []byte("hello")
	ptr, err := PinBytes(data)
	if err != nil {
		t.Fatalf("unexpected pin error: %v", err)
	}
	if ptr == 0 {
		t.Fatal("expected non-zero pointer")
	}
	if !Release(ptr) {
		t.Fatal("expected first release to succeed")
	}
	if Release(ptr) {
		t.Fatal("expected second release to fail")
	}
}

func TestPinStringAndRelease(t *testing.T) {
	data, ptr, err := PinString("world")
	if err != nil {
		t.Fatalf("unexpected pin error: %v", err)
	}
	if string(data) != "world" {
		t.Fatalf("unexpected string bytes: %q", string(data))
	}
	if ptr == 0 {
		t.Fatal("expected non-zero pointer")
	}
	if !Release(ptr) {
		t.Fatal("expected release to succeed")
	}
}

func TestPinSliceAndRelease(t *testing.T) {
	data := []int32{1, 2, 3}
	ptr, err := PinSlice(data)
	if err != nil {
		t.Fatalf("unexpected pin error: %v", err)
	}
	if ptr == 0 {
		t.Fatal("expected non-zero pointer")
	}
	if !Release(ptr) {
		t.Fatal("expected slice release to succeed")
	}
}

func TestPinBoolSliceDoesNotCompile(t *testing.T) {
	dir := t.TempDir()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	repoRoot := filepath.Dir(filepath.Dir(testFile))

	goMod := `module boolfixture

go 1.24.4

require rpccgo v0.0.0

replace rpccgo => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write compile fixture go.mod: %v", err)
	}

	source := `package main

import "rpccgo/rpcruntime"

func main() {
	_, _ = rpcruntime.PinSlice([]bool{true, false})
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatalf("write compile fixture: %v", err)
	}

	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(goBinary, "build", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected PinSlice([]bool) compile failure, got success")
	}
	if !strings.Contains(string(output), "bool does not satisfy") &&
		!strings.Contains(string(output), "does not satisfy") {
		t.Fatalf("unexpected compile failure:\n%s", output)
	}
}

func TestDuplicatePinSharedBackingStoreReturnsErrorAndKeepsOldRecord(t *testing.T) {
	data := []byte("shared")

	ptr1, err := PinBytes(data)
	if err != nil {
		t.Fatalf("unexpected first pin error: %v", err)
	}

	ptr2, err := PinBytes(data)
	if err == nil {
		t.Fatal("expected duplicate pin to return error")
	}
	if !strings.Contains(err.Error(), "already pinned") {
		t.Fatalf("duplicate pin error = %q", err)
	}
	if ptr2 != ptr1 {
		t.Fatal("expected duplicate pin to return the original pointer")
	}

	if !Release(ptr1) {
		t.Fatal("expected original record to remain releasable")
	}
}

func TestReleaseUnknownPointer(t *testing.T) {
	if Release(9999) {
		t.Fatal("expected unknown pointer release to fail")
	}
}

func TestPinEmptyValuesReturnZeroPointerAndDoNotRegister(t *testing.T) {
	if ptr, err := PinBytes(nil); err != nil || ptr != 0 {
		t.Fatalf("PinBytes(nil) = (%d, %v), want (0, nil)", ptr, err)
	}
	if ptr, err := PinBytes([]byte{}); err != nil || ptr != 0 {
		t.Fatalf("PinBytes(empty) = (%d, %v), want (0, nil)", ptr, err)
	}

	data, ptr, err := PinString("")
	if err != nil || ptr != 0 || data != nil {
		t.Fatalf("PinString(empty) = (%v, %d, %v), want (nil, 0, nil)", data, ptr, err)
	}

	if ptr, err := PinSlice([]int32{}); err != nil || ptr != 0 {
		t.Fatalf("PinSlice(empty) = (%d, %v), want (0, nil)", ptr, err)
	}

	if Release(0) {
		t.Fatal("expected zero pointer release to fail")
	}
}

func TestReleaseHandlesBytesStringAndSlicePointers(t *testing.T) {
	bytesPtr, err := PinBytes([]byte("bytes"))
	if err != nil {
		t.Fatalf("pin bytes: %v", err)
	}
	if !Release(bytesPtr) {
		t.Fatal("expected bytes pointer release to succeed")
	}

	_, stringPtr, err := PinString("string")
	if err != nil {
		t.Fatalf("pin string: %v", err)
	}
	if !Release(stringPtr) {
		t.Fatal("expected string pointer release to succeed")
	}

	slicePtr, err := PinSlice([]int32{1, 2, 3})
	if err != nil {
		t.Fatalf("pin slice: %v", err)
	}
	if !Release(slicePtr) {
		t.Fatal("expected slice pointer release to succeed")
	}
}

func TestReleaseAfterGCWithoutExternalBytesReference(t *testing.T) {
	ptr, err := PinBytes([]byte("survives gc"))
	if err != nil {
		t.Fatalf("pin bytes: %v", err)
	}

	runtime.GC()

	if !Release(ptr) {
		t.Fatal("expected bytes release after GC to succeed")
	}
}

func TestReleaseAfterGCWithoutExternalStringReference(t *testing.T) {
	_, ptr, err := PinString(strings.Repeat("x", 64))
	if err != nil {
		t.Fatalf("pin string: %v", err)
	}

	runtime.GC()

	if !Release(ptr) {
		t.Fatal("expected string release after GC to succeed")
	}
}
