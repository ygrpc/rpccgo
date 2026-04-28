package rpcruntime

import (
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

type freeCallbackRecorder struct {
	calls atomic.Int32
	ptr   atomic.Uintptr
}

func registerFreeCallbackRecorder() *freeCallbackRecorder {
	recorder := &freeCallbackRecorder{}
	RegisterFreeCallback(func(ptr unsafe.Pointer) {
		recorder.ptr.Store(uintptr(ptr))
		recorder.calls.Add(1)
	})
	return recorder
}

func expectPanic(t *testing.T, name string, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatalf("expected %s to panic", name)
		}
	}()
	fn()
}

func TestEmptyRpcBytesGetterReturnsCanonicalEmptyWrapper(t *testing.T) {
	got := EmptyRpcBytes()
	if got == nil {
		t.Fatal("EmptyRpcBytes() should not be nil")
	}
	if got != EmptyRpcBytes() {
		t.Fatal("expected EmptyRpcBytes() to return the same instance across calls")
	}
	if got != emptyRpcBytes {
		t.Fatal("expected EmptyRpcBytes() to return the package-private singleton")
	}
	if unsafeBytes := got.UnsafeBytes(); unsafeBytes != nil {
		t.Fatalf("expected nil unsafe bytes, got %v", unsafeBytes)
	}
	if safeBytes := got.SafeBytes(); safeBytes != nil {
		t.Fatalf("expected nil safe bytes, got %v", safeBytes)
	}
	if safeBytes := got.SafeBytes(); safeBytes != nil {
		t.Fatalf("expected repeated SafeBytes() calls to stay nil, got %v", safeBytes)
	}
	if err := got.Release(); err != nil {
		t.Fatalf("expected EmptyRpcBytes().Release() to be a no-op, got %v", err)
	}
}

func TestEmptyRpcStringGetterReturnsCanonicalEmptyWrapper(t *testing.T) {
	got := EmptyRpcString()
	if got == nil {
		t.Fatal("EmptyRpcString() should not be nil")
	}
	if got != EmptyRpcString() {
		t.Fatal("expected EmptyRpcString() to return the same instance across calls")
	}
	if got != emptyRpcString {
		t.Fatal("expected EmptyRpcString() to return the package-private singleton")
	}
	if unsafeString := got.UnsafeString(); unsafeString != "" {
		t.Fatalf("expected empty unsafe string, got %q", unsafeString)
	}
	if safeString := got.SafeString(); safeString != "" {
		t.Fatalf("expected empty safe string, got %q", safeString)
	}
	if safeString := got.SafeString(); safeString != "" {
		t.Fatalf("expected repeated SafeString() calls to stay empty, got %q", safeString)
	}
	if err := got.Release(); err != nil {
		t.Fatalf("expected EmptyRpcString().Release() to be a no-op, got %v", err)
	}
}

func TestRpcBytesUnsafeBytesZeroCopy(t *testing.T) {
	src := []byte("unsafe-bytes")
	rpc := NewRpcBytes(&src[0], int32(len(src)), false)

	got := rpc.UnsafeBytes()
	if string(got) != string(src) {
		t.Fatalf("unexpected unsafe bytes: got %q want %q", string(got), string(src))
	}
	if unsafe.SliceData(got) != &src[0] {
		t.Fatal("expected UnsafeBytes to return a view over the original backing pointer")
	}
}

func TestRpcBytesUnsafeBytesNilAndEmpty(t *testing.T) {
	if got := NewRpcBytes(nil, 0, false).UnsafeBytes(); got != nil {
		t.Fatalf("expected nil bytes for nil input, got %v", got)
	}

	src := []byte("ignored")
	if got := NewRpcBytes(&src[0], 0, false).UnsafeBytes(); got != nil {
		t.Fatalf("expected nil bytes for zero-length input, got %v", got)
	}
}

func TestNewRpcBytesRejectsNegativeLength(t *testing.T) {
	src := []byte("x")
	expectPanic(t, "NewRpcBytes", func() {
		_ = NewRpcBytes(&src[0], -1, false)
	})
}

func TestNewRpcStringRejectsNegativeLength(t *testing.T) {
	src := []byte("x")
	expectPanic(t, "NewRpcString", func() {
		_ = NewRpcString(&src[0], -1, false)
	})
}

func TestRpcBytesSafeBytesCopiesOnlyOnce(t *testing.T) {
	src := []byte("safe-bytes")
	rpc := NewRpcBytes(&src[0], int32(len(src)), false)

	first := rpc.SafeBytes()
	if string(first) != "safe-bytes" {
		t.Fatalf("unexpected first safe bytes: %q", string(first))
	}

	src[0] = 'X'

	second := rpc.SafeBytes()
	if string(second) != "safe-bytes" {
		t.Fatalf("expected cached safe bytes to remain stable, got %q", string(second))
	}
	if unsafe.SliceData(first) != unsafe.SliceData(second) {
		t.Fatal("expected SafeBytes to reuse the cached copy")
	}
}

func TestRpcBytesSafeBytesConcurrentUsesSingleStableCopy(t *testing.T) {
	src := []byte("concurrent-safe-bytes")
	rpc := NewRpcBytes(&src[0], int32(len(src)), false)
	want := string(src)

	const goroutines = 16
	results := make([][]byte, goroutines)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = rpc.SafeBytes()
		}(i)
	}
	close(start)
	wg.Wait()

	src[0] = 'X'
	final := rpc.SafeBytes()

	for i, got := range results {
		if string(got) != want {
			t.Fatalf("goroutine %d returned unstable safe bytes: got %q want %q", i, string(got), want)
		}
		if unsafe.SliceData(got) != unsafe.SliceData(final) {
			t.Fatalf("goroutine %d did not reuse cached safe bytes", i)
		}
	}
	if string(final) != want {
		t.Fatalf("expected cached safe bytes to remain stable after source mutation, got %q want %q", string(final), want)
	}
}

func TestRpcBytesReleaseOwnershipFalseIsNoOp(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("borrowed")
	rpc := NewRpcBytes(&src[0], int32(len(src)), false)
	if err := rpc.Release(); err != nil {
		t.Fatalf("expected borrowed release to be a no-op, got %v", err)
	}
	if got := recorder.calls.Load(); got != 0 {
		t.Fatalf("expected borrowed release not to call free callback, got %d calls", got)
	}
}

func TestRpcBytesReleaseOwnershipTrueOnlyFreesOnce(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("owned")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], int32(len(src)), true)
	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := rpc.Release(); err != nil {
		t.Fatalf("expected repeated release to stay a no-op, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected owned release to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected owned release to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcBytesReleaseConcurrentStillFreesOnce(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("owned-concurrent")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], int32(len(src)), true)

	const goroutines = 16
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			errs <- rpc.Release()
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("expected concurrent release to stay successful, got %v", err)
		}
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected concurrent release to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected concurrent release to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcBytesReleaseAfterSafeBytesStillFreesOriginalPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("owned-safe-bytes")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], int32(len(src)), true)

	safe := rpc.SafeBytes()
	if len(safe) == 0 {
		t.Fatal("expected SafeBytes to return copied content before release")
	}
	if unsafe.SliceData(safe) == &src[0] {
		t.Fatal("expected SafeBytes to return copied storage distinct from the original pointer")
	}

	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error after SafeBytes: %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected release after SafeBytes to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected release after SafeBytes to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcBytesReleaseZeroLengthOwnedStillFreesPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte{0}
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], 0, true)

	if err := rpc.Release(); err != nil {
		t.Fatalf("expected zero-length owned bytes to release successfully, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected zero-length owned bytes to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected zero-length owned bytes to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcBytesReleaseWithoutRegisteredCallbackCanRetry(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	src := []byte("owned-without-callback")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcBytes(&src[0], int32(len(src)), true)

	if err := rpc.Release(); err == nil {
		t.Fatal("expected release without registered free callback to fail")
	}

	recorder := registerFreeCallbackRecorder()
	if err := rpc.Release(); err != nil {
		t.Fatalf("expected release to succeed after registering callback, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected retry release to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected retry release to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcStringUnsafeStringZeroCopy(t *testing.T) {
	src := []byte("unsafe-string")
	rpc := NewRpcString(&src[0], int32(len(src)), false)

	got := rpc.UnsafeString()
	if got != string(src) {
		t.Fatalf("unexpected unsafe string: got %q want %q", got, string(src))
	}
	if unsafe.StringData(got) != &src[0] {
		t.Fatal("expected UnsafeString to return a view over the original backing pointer")
	}
}

func TestRpcStringUnsafeStringNilAndEmpty(t *testing.T) {
	if got := NewRpcString(nil, 0, false).UnsafeString(); got != "" {
		t.Fatalf("expected empty string for nil input, got %q", got)
	}

	src := []byte("ignored")
	if got := NewRpcString(&src[0], 0, false).UnsafeString(); got != "" {
		t.Fatalf("expected empty string for zero-length input, got %q", got)
	}
}

func TestRpcStringSafeStringCopiesOnlyOnce(t *testing.T) {
	src := []byte("safe-string")
	rpc := NewRpcString(&src[0], int32(len(src)), false)

	first := rpc.SafeString()
	if first != "safe-string" {
		t.Fatalf("unexpected first safe string: %q", first)
	}

	src[0] = 'X'

	second := rpc.SafeString()
	if second != "safe-string" {
		t.Fatalf("expected cached safe string to remain stable, got %q", second)
	}
	if unsafe.StringData(first) != unsafe.StringData(second) {
		t.Fatal("expected SafeString to reuse the cached copy")
	}
}

func TestRpcStringSafeStringConcurrentUsesSingleStableCopy(t *testing.T) {
	src := []byte("concurrent-safe-string")
	rpc := NewRpcString(&src[0], int32(len(src)), false)
	want := string(src)

	const goroutines = 16
	results := make([]string, goroutines)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = rpc.SafeString()
		}(i)
	}
	close(start)
	wg.Wait()

	src[0] = 'X'
	final := rpc.SafeString()

	for i, got := range results {
		if got != want {
			t.Fatalf("goroutine %d returned unstable safe string: got %q want %q", i, got, want)
		}
		if unsafe.StringData(got) != unsafe.StringData(final) {
			t.Fatalf("goroutine %d did not reuse cached safe string", i)
		}
	}
	if final != want {
		t.Fatalf("expected cached safe string to remain stable after source mutation, got %q want %q", final, want)
	}
}

func TestRpcStringReleaseOwnershipFalseIsNoOp(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("borrowed")
	rpc := NewRpcString(&src[0], int32(len(src)), false)
	if err := rpc.Release(); err != nil {
		t.Fatalf("expected borrowed release to be a no-op, got %v", err)
	}
	if got := recorder.calls.Load(); got != 0 {
		t.Fatalf("expected borrowed release not to call free callback, got %d calls", got)
	}
}

func TestRpcStringReleaseOwnershipTrueOnlyFreesOnce(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("owned")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcString(&src[0], int32(len(src)), true)
	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := rpc.Release(); err != nil {
		t.Fatalf("expected repeated release to stay a no-op, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected owned release to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected owned release to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcStringReleaseConcurrentStillFreesOnce(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("owned-concurrent")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcString(&src[0], int32(len(src)), true)

	const goroutines = 16
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			errs <- rpc.Release()
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("expected concurrent release to stay successful, got %v", err)
		}
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected concurrent release to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected concurrent release to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcStringReleaseAfterSafeStringStillFreesOriginalPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte("owned-safe-string")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcString(&src[0], int32(len(src)), true)

	safe := rpc.SafeString()
	if safe == "" {
		t.Fatal("expected SafeString to return copied content before release")
	}
	if unsafe.StringData(safe) == &src[0] {
		t.Fatal("expected SafeString to return copied storage distinct from the original pointer")
	}

	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error after SafeString: %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected release after SafeString to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected release after SafeString to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcStringReleaseZeroLengthOwnedStillFreesPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()

	src := []byte{0}
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcString(&src[0], 0, true)

	if err := rpc.Release(); err != nil {
		t.Fatalf("expected zero-length owned string to release successfully, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected zero-length owned string to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected zero-length owned string to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcStringReleaseWithoutRegisteredCallbackCanRetry(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	src := []byte("owned-string-without-callback")
	wantPtr := uintptr(unsafe.Pointer(&src[0]))
	rpc := NewRpcString(&src[0], int32(len(src)), true)

	if err := rpc.Release(); err == nil {
		t.Fatal("expected release without registered free callback to fail")
	}

	recorder := registerFreeCallbackRecorder()
	if err := rpc.Release(); err != nil {
		t.Fatalf("expected release to succeed after registering callback, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected retry release to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected retry release to free original pointer %#x, got %#x", wantPtr, got)
	}
}
