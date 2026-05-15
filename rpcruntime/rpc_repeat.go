package rpcruntime

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

const (
	rpcRepeatLabel     = "RpcRepeat"
	rpcBoolRepeatLabel = "RpcBoolRepeat"
)

type NativeRepeatElem interface {
	~int32 | ~int64 | ~float32 | ~float64
}

// RpcRepeat wraps a borrowed or owned fixed-width repeated input from the cgo boundary.
type RpcRepeat[T NativeRepeatElem] struct {
	ptr       *T
	length    int32
	ownership bool

	safeOnce  sync.Once
	safeCache []T

	cleanup    runtime.Cleanup
	hasCleanup bool
}

// RpcBoolRepeat wraps a borrowed or owned bool repeated input encoded as bytes.
type RpcBoolRepeat struct {
	ptr       *byte
	length    int32
	ownership bool

	safeOnce  sync.Once
	safeCache []bool

	cleanup    runtime.Cleanup
	hasCleanup bool
}

var (
	emptyRpcBoolRepeat   = &RpcBoolRepeat{}
	emptyRpcRepeatByType sync.Map
)

func NewRpcRepeat[T NativeRepeatElem](ptr *T, length int32, ownership bool) *RpcRepeat[T] {
	rpc, err := NewRpcRepeatChecked(ptr, length, ownership)
	if err != nil {
		panic(err)
	}
	return rpc
}

func NewRpcRepeatChecked[T NativeRepeatElem](ptr *T, length int32, ownership bool) (*RpcRepeat[T], error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcRepeat: %w", err)
	}
	return newRpcRepeatUnchecked(ptr, length, ownership), nil
}

func NewRpcRepeatView[T NativeRepeatElem](ptr *T, length int32, owner any) *RpcRepeat[T] {
	rpc := NewRpcRepeat(ptr, length, false)
	runtime.KeepAlive(owner)
	return rpc
}

func newRpcRepeatUnchecked[T NativeRepeatElem](ptr *T, length int32, ownership bool) *RpcRepeat[T] {
	rpc := &RpcRepeat[T]{
		ptr:       ptr,
		length:    length,
		ownership: ownership,
	}
	rpc.attachCleanup(rpcRepeatLabel)
	return rpc
}

func NewRpcBoolRepeat(ptr *byte, length int32, ownership bool) *RpcBoolRepeat {
	rpc, err := NewRpcBoolRepeatChecked(ptr, length, ownership)
	if err != nil {
		panic(err)
	}
	return rpc
}

func NewRpcBoolRepeatChecked(ptr *byte, length int32, ownership bool) (*RpcBoolRepeat, error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcBoolRepeat: %w", err)
	}
	return newRpcBoolRepeatUnchecked(ptr, length, ownership), nil
}

func NewRpcBoolRepeatView(ptr *byte, length int32, owner any) *RpcBoolRepeat {
	rpc := NewRpcBoolRepeat(ptr, length, false)
	runtime.KeepAlive(owner)
	return rpc
}

func newRpcBoolRepeatUnchecked(ptr *byte, length int32, ownership bool) *RpcBoolRepeat {
	rpc := &RpcBoolRepeat{
		ptr:       ptr,
		length:    length,
		ownership: ownership,
	}
	rpc.attachCleanup(rpcBoolRepeatLabel)
	return rpc
}

// EmptyRpcRepeat returns the canonical read-only empty wrapper for T.
func EmptyRpcRepeat[T NativeRepeatElem]() *RpcRepeat[T] {
	typeKey := reflect.TypeFor[T]()
	if cached, ok := emptyRpcRepeatByType.Load(typeKey); ok {
		return cached.(*RpcRepeat[T])
	}

	empty := &RpcRepeat[T]{}
	actual, _ := emptyRpcRepeatByType.LoadOrStore(typeKey, empty)
	return actual.(*RpcRepeat[T])
}

// EmptyRpcBoolRepeat returns the canonical read-only empty bool repeat wrapper.
func EmptyRpcBoolRepeat() *RpcBoolRepeat {
	return emptyRpcBoolRepeat
}

func (r *RpcRepeat[T]) Len() int32 {
	if r == nil {
		return 0
	}
	return r.length
}

func (r *RpcRepeat[T]) At(i int32) T {
	var zero T
	if r == nil || r.ptr == nil || i < 0 || i >= r.length {
		return zero
	}
	return r.UnsafeSlice()[mustLengthFromInt32(i, "RpcRepeat.At")]
}

// MustAt mirrors slice indexing semantics for callers that want a hard failure.
func (r *RpcRepeat[T]) MustAt(i int32) T {
	if r == nil || r.ptr == nil || i < 0 || i >= r.length {
		panic(fmt.Sprintf("RpcRepeat.MustAt: index %d out of range [0, %d)", i, r.Len()))
	}
	return r.At(i)
}

// UnsafeSlice returns a zero-copy borrowed view over the underlying input.
// Callers must keep the wrapper reachable while using the returned slice.
func (r *RpcRepeat[T]) UnsafeSlice() []T {
	if r == nil || r.ptr == nil || r.length == 0 {
		return nil
	}
	return unsafe.Slice(r.ptr, mustLengthFromInt32(r.length, "RpcRepeat.UnsafeSlice"))
}

// SafeSlice returns a cached copy that is safe to retain after the wrapper is released.
func (r *RpcRepeat[T]) SafeSlice() []T {
	if r == nil || r.ptr == nil || r.length == 0 {
		return nil
	}

	r.safeOnce.Do(func() {
		r.safeCache = append([]T(nil), r.UnsafeSlice()...)
	})
	return r.safeCache
}

// Release deterministically releases owned input memory when ownership is true.
func (r *RpcRepeat[T]) Release() error {
	if r == nil {
		return nil
	}
	if err := releaseRpcInput(unsafe.Pointer(r.ptr), r.ownership, rpcRepeatLabel); err != nil {
		return err
	}
	if r.hasCleanup {
		r.cleanup.Stop()
	}
	return nil
}

func (r *RpcBoolRepeat) Len() int32 {
	if r == nil {
		return 0
	}
	return r.length
}

func (r *RpcBoolRepeat) At(i int32) bool {
	if r == nil || r.ptr == nil || i < 0 || i >= r.length {
		return false
	}
	raw := unsafe.Slice(r.ptr, mustLengthFromInt32(r.length, "RpcBoolRepeat.At"))
	return raw[mustLengthFromInt32(i, "RpcBoolRepeat.At")] != 0
}

// MustAt mirrors slice indexing semantics for callers that want a hard failure.
func (r *RpcBoolRepeat) MustAt(i int32) bool {
	if r == nil || r.ptr == nil || i < 0 || i >= r.length {
		panic(fmt.Sprintf("RpcBoolRepeat.MustAt: index %d out of range [0, %d)", i, r.Len()))
	}
	return r.At(i)
}

// SafeSlice returns a cached bool copy that is safe to retain after the wrapper is released.
func (r *RpcBoolRepeat) SafeSlice() []bool {
	if r == nil || r.ptr == nil || r.length == 0 {
		return nil
	}

	r.safeOnce.Do(func() {
		raw := unsafe.Slice(r.ptr, mustLengthFromInt32(r.length, "RpcBoolRepeat.SafeSlice"))
		r.safeCache = make([]bool, len(raw))
		for i, value := range raw {
			r.safeCache[i] = value != 0
		}
	})
	return r.safeCache
}

// Release deterministically releases owned input memory when ownership is true.
func (r *RpcBoolRepeat) Release() error {
	if r == nil {
		return nil
	}
	if err := releaseRpcInput(unsafe.Pointer(r.ptr), r.ownership, rpcBoolRepeatLabel); err != nil {
		return err
	}
	if r.hasCleanup {
		r.cleanup.Stop()
	}
	return nil
}

func (r *RpcRepeat[T]) attachCleanup(label string) {
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

func (r *RpcBoolRepeat) attachCleanup(label string) {
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
