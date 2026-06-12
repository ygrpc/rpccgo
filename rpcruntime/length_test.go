package rpcruntime

import (
	"math"
	"testing"
)

func TestLengthToInt32AcceptsSmallPositiveValue(t *testing.T) {
	got, err := LengthToInt32(7)
	if err != nil {
		t.Fatalf("expected small positive length to convert successfully, got %v", err)
	}
	if got != 7 {
		t.Fatalf("unexpected converted length: got %d want 7", got)
	}
}

func TestLengthToInt32AcceptsZero(t *testing.T) {
	got, err := LengthToInt32(0)
	if err != nil {
		t.Fatalf("expected zero length to convert successfully, got %v", err)
	}
	if got != 0 {
		t.Fatalf("unexpected converted zero length: got %d", got)
	}
}

func TestLengthToInt32AcceptsMaxInt32(t *testing.T) {
	got, err := LengthToInt32(int(math.MaxInt32))
	if err != nil {
		t.Fatalf("expected math.MaxInt32 to convert successfully, got %v", err)
	}
	if got != math.MaxInt32 {
		t.Fatalf("unexpected converted max length: got %d want %d", got, int32(math.MaxInt32))
	}
}

func TestLengthToInt32RejectsNegativeValues(t *testing.T) {
	_, err := LengthToInt32(-1)
	if err == nil {
		t.Fatal("expected negative outbound length to be rejected")
	}
}

func TestLengthToInt32RejectsValuesAboveMaxInt32(t *testing.T) {
	_, err := LengthToInt32(int(math.MaxInt32) + 1)
	if err == nil {
		t.Fatal("expected values above math.MaxInt32 to be rejected")
	}
}

func TestLengthFromInt32AcceptsZero(t *testing.T) {
	got, err := LengthFromInt32(0)
	if err != nil {
		t.Fatalf("expected zero inbound length to convert successfully, got %v", err)
	}
	if got != 0 {
		t.Fatalf("unexpected inbound zero length: got %d", got)
	}
}

func TestLengthFromInt32AcceptsMaxInt32(t *testing.T) {
	got, err := LengthFromInt32(math.MaxInt32)
	if err != nil {
		t.Fatalf("expected math.MaxInt32 inbound length to convert successfully, got %v", err)
	}
	if got != int(math.MaxInt32) {
		t.Fatalf("unexpected inbound max length: got %d want %d", got, int(math.MaxInt32))
	}
}

func TestLengthFromInt32RejectsNegativeValues(t *testing.T) {
	_, err := LengthFromInt32(-1)
	if err == nil {
		t.Fatal("expected negative inbound length to be rejected")
	}
}
