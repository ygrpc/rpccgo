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

func TestLengthToInt32RejectsValuesAboveMaxInt32(t *testing.T) {
	_, err := LengthToInt32(int(math.MaxInt32) + 1)
	if err == nil {
		t.Fatal("expected values above math.MaxInt32 to be rejected")
	}
}

func TestLengthFromInt32RejectsNegativeValues(t *testing.T) {
	_, err := LengthFromInt32(-1)
	if err == nil {
		t.Fatal("expected negative inbound length to be rejected")
	}
}
