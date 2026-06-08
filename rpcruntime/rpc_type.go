package rpcruntime

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

const (
	rpcBytesLabel  = "RpcBytes"
	rpcStringLabel = "RpcString"
)

type rpcInputCleanupArg struct {
	ptr   unsafe.Pointer
	label string
}

func rpcInputCleanup(arg rpcInputCleanupArg) {
	_ = releaseRpcInput(arg.ptr, true, arg.label)
}

// RpcBytes wraps a borrowed or owned byte input from the cgo boundary.
type RpcBytes struct {
	ptr       *byte
	length    int32
	ownership bool

	safeOnce  sync.Once
	safeCache []byte

	cleanup    runtime.Cleanup
	hasCleanup bool
}

// RpcString wraps a borrowed or owned string input from the cgo boundary.
type RpcString struct {
	ptr       *byte
	length    int32
	ownership bool

	safeOnce  sync.Once
	safeCache string

	cleanup    runtime.Cleanup
	hasCleanup bool
}

var (
	emptyRpcBytes  = &RpcBytes{}
	emptyRpcString = &RpcString{}
)

// NewRpcBytes returns an RpcBytes wrapper for ptr and length.
// Invalid lengths return nil; callers that need error details should use NewRpcBytesChecked.
func NewRpcBytes(ptr *byte, length int32, ownership bool) *RpcBytes {
	rpc, err := NewRpcBytesChecked(ptr, length, ownership)
	if err != nil {
		return nil
	}
	return rpc
}

// NewRpcBytesChecked returns an RpcBytes wrapper after validating length.
func NewRpcBytesChecked(ptr *byte, length int32, ownership bool) (*RpcBytes, error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcBytes: %w", err)
	}
	return newRpcBytesUnchecked(ptr, length, ownership), nil
}

func newRpcBytesUnchecked(ptr *byte, length int32, ownership bool) *RpcBytes {
	rpc := &RpcBytes{
		ptr:       ptr,
		length:    length,
		ownership: ownership,
	}
	rpc.attachCleanup(rpcBytesLabel)
	return rpc
}

// NewRpcString returns an RpcString wrapper for ptr and length.
// Invalid lengths return nil; callers that need error details should use NewRpcStringChecked.
func NewRpcString(ptr *byte, length int32, ownership bool) *RpcString {
	rpc, err := NewRpcStringChecked(ptr, length, ownership)
	if err != nil {
		return nil
	}
	return rpc
}

// NewRpcStringChecked returns an RpcString wrapper after validating length.
func NewRpcStringChecked(ptr *byte, length int32, ownership bool) (*RpcString, error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcString: %w", err)
	}
	return newRpcStringUnchecked(ptr, length, ownership), nil
}

func newRpcStringUnchecked(ptr *byte, length int32, ownership bool) *RpcString {
	rpc := &RpcString{
		ptr:       ptr,
		length:    length,
		ownership: ownership,
	}
	rpc.attachCleanup(rpcStringLabel)
	return rpc
}

// EmptyRpcBytes returns the canonical read-only empty bytes wrapper.
func EmptyRpcBytes() *RpcBytes {
	return emptyRpcBytes
}

// EmptyRpcString returns the canonical read-only empty string wrapper.
func EmptyRpcString() *RpcString {
	return emptyRpcString
}

// UnsafeBytes returns a zero-copy borrowed view over the underlying input.
// Callers must keep the wrapper reachable while using the returned slice.
func (r *RpcBytes) UnsafeBytes() []byte {
	if r == nil || r.ptr == nil || r.length == 0 {
		return nil
	}
	return unsafe.Slice(r.ptr, lengthFromInt32OrZero(r.length))
}

// SafeBytes returns a cached copy that is safe to retain after the wrapper is released.
func (r *RpcBytes) SafeBytes() []byte {
	if r == nil || r.ptr == nil || r.length == 0 {
		return nil
	}

	r.safeOnce.Do(func() {
		r.safeCache = bytes.Clone(r.UnsafeBytes())
	})
	return r.safeCache
}

// Release deterministically releases owned input memory when ownership is true.
func (r *RpcBytes) Release() error {
	if r == nil {
		return nil
	}
	if err := releaseRpcInput(unsafe.Pointer(r.ptr), r.ownership, rpcBytesLabel); err != nil {
		return err
	}
	if r.hasCleanup {
		r.cleanup.Stop()
	}
	return nil
}

// UnsafeString returns a zero-copy borrowed view over the underlying input.
// Callers must keep the wrapper reachable while using the returned string.
func (r *RpcString) UnsafeString() string {
	if r == nil || r.ptr == nil || r.length == 0 {
		return ""
	}
	return unsafe.String(r.ptr, lengthFromInt32OrZero(r.length))
}

// SafeString returns a cached copy that is safe to retain after the wrapper is released.
func (r *RpcString) SafeString() string {
	if r == nil || r.ptr == nil || r.length == 0 {
		return ""
	}

	r.safeOnce.Do(func() {
		cloned := bytes.Clone(unsafe.Slice(r.ptr, lengthFromInt32OrZero(r.length)))
		r.safeCache = unsafe.String(unsafe.SliceData(cloned), len(cloned))
	})
	return r.safeCache
}

// Release deterministically releases owned input memory when ownership is true.
func (r *RpcString) Release() error {
	if r == nil {
		return nil
	}
	if err := releaseRpcInput(unsafe.Pointer(r.ptr), r.ownership, rpcStringLabel); err != nil {
		return err
	}
	if r.hasCleanup {
		r.cleanup.Stop()
	}
	return nil
}

func (r *RpcBytes) attachCleanup(label string) {
	if r == nil || !r.ownership || r.ptr == nil {
		return
	}

	registerRpcInputRelease(unsafe.Pointer(r.ptr), true, label)
	r.cleanup = runtime.AddCleanup(r, rpcInputCleanup, rpcInputCleanupArg{
		ptr:   unsafe.Pointer(r.ptr),
		label: label,
	})
	r.hasCleanup = true
}

func (r *RpcString) attachCleanup(label string) {
	if r == nil || !r.ownership || r.ptr == nil {
		return
	}

	registerRpcInputRelease(unsafe.Pointer(r.ptr), true, label)
	r.cleanup = runtime.AddCleanup(r, rpcInputCleanup, rpcInputCleanupArg{
		ptr:   unsafe.Pointer(r.ptr),
		label: label,
	})
	r.hasCleanup = true
}
