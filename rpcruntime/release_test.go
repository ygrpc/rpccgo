package rpcruntime

import (
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

func TestPinBoolSliceAndRelease(t *testing.T) {
	ptr, err := PinSlice([]bool{true, false})
	if err != nil {
		t.Fatalf("unexpected pin error: %v", err)
	}
	if ptr == 0 {
		t.Fatal("expected non-zero pointer")
	}
	if !Release(ptr) {
		t.Fatal("expected bool slice release to succeed")
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
