package rpcruntime

import (
	"fmt"
	"math"
)

// LengthToInt32 converts a Go length/count into the shared int32 ABI/runtime form.
func LengthToInt32(length int) (int32, error) {
	if length < 0 {
		return 0, fmt.Errorf("rpc length %d cannot be negative", length)
	}
	if length > math.MaxInt32 {
		return 0, fmt.Errorf("rpc length %d exceeds int32 limit", length)
	}
	return int32(length), nil
}

// LengthFromInt32 converts an inbound int32 ABI/runtime length into a Go int.
func LengthFromInt32(length int32) (int, error) {
	if length < 0 {
		return 0, fmt.Errorf("rpc length %d cannot be negative", length)
	}
	return int(length), nil
}

func mustLengthFromInt32(length int32, label string) int {
	value, err := LengthFromInt32(length)
	if err != nil {
		panic(fmt.Sprintf("%s: %v", label, err))
	}
	return value
}
