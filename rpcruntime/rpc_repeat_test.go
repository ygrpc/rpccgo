package rpcruntime

import (
	"slices"
	"testing"
	"unsafe"
)

type testRpcRepeatEnum int32

func TestEmptyRpcBoolRepeatReturnsCanonicalEmptyWrapper(t *testing.T) {
	got := EmptyRpcBoolRepeat()
	if got == nil {
		t.Fatal("EmptyRpcBoolRepeat() should not be nil")
	}
	if got != EmptyRpcBoolRepeat() {
		t.Fatal("expected EmptyRpcBoolRepeat() to return the same instance across calls")
	}
	if got != emptyRpcBoolRepeat {
		t.Fatal("expected EmptyRpcBoolRepeat() to return the package-private singleton")
	}
	if got.Len() != 0 {
		t.Fatalf("expected empty bool repeat length to be 0, got %d", got.Len())
	}
	if safeSlice := got.SafeSlice(); safeSlice != nil {
		t.Fatalf("expected nil safe slice for empty bool repeat, got %v", safeSlice)
	}
	if safeSlice := got.SafeSlice(); safeSlice != nil {
		t.Fatalf("expected repeated SafeSlice() calls to stay nil, got %v", safeSlice)
	}
	if err := got.Release(); err != nil {
		t.Fatalf("expected EmptyRpcBoolRepeat().Release() to be a no-op, got %v", err)
	}
}

func TestEmptyRpcRepeatReturnsCanonicalEmptyWrapperPerType(t *testing.T) {
	got := EmptyRpcRepeat[int32]()
	if got == nil {
		t.Fatal("EmptyRpcRepeat[int32]() should not be nil")
	}
	if got != EmptyRpcRepeat[int32]() {
		t.Fatal("expected EmptyRpcRepeat[int32]() to return the same instance across calls")
	}
	if unsafe.Pointer(got) == unsafe.Pointer(EmptyRpcRepeat[int64]()) {
		t.Fatal("expected EmptyRpcRepeat to keep separate canonical wrappers per type")
	}
	if unsafeSlice := got.UnsafeSlice(); unsafeSlice != nil {
		t.Fatalf("expected nil unsafe slice for empty repeat, got %v", unsafeSlice)
	}
	if safeSlice := got.SafeSlice(); safeSlice != nil {
		t.Fatalf("expected nil safe slice for empty repeat, got %v", safeSlice)
	}
	if safeSlice := got.SafeSlice(); safeSlice != nil {
		t.Fatalf("expected repeated SafeSlice() calls to stay nil, got %v", safeSlice)
	}
	if err := got.Release(); err != nil {
		t.Fatalf("expected EmptyRpcRepeat[int32]().Release() to be a no-op, got %v", err)
	}
}

func TestEmptyRpcRepeatDifferentTypesStillBehaveAsEmptyWrapper(t *testing.T) {
	got := EmptyRpcRepeat[testRpcRepeatEnum]()
	if got == nil {
		t.Fatal("EmptyRpcRepeat[testRpcRepeatEnum]() should not be nil")
	}
	if unsafeSlice := got.UnsafeSlice(); unsafeSlice != nil {
		t.Fatalf("expected nil unsafe slice for empty enum repeat, got %v", unsafeSlice)
	}
	if safeSlice := got.SafeSlice(); safeSlice != nil {
		t.Fatalf("expected nil safe slice for empty enum repeat, got %v", safeSlice)
	}
	if err := got.Release(); err != nil {
		t.Fatalf("expected EmptyRpcRepeat[testRpcRepeatEnum]().Release() to be a no-op, got %v", err)
	}
}

func TestRpcRepeatSignedNumericEnumAndFloatValues(t *testing.T) {
	ints := []int32{-10, 20, -30}
	intRepeat := NewRpcRepeat(&ints[0], int32(len(ints)), false)
	if got, want := intRepeat.Len(), int32(len(ints)); got != want {
		t.Fatalf("int length mismatch: got %d want %d", got, want)
	}
	if got, want := intRepeat.At(1), int32(20); got != want {
		t.Fatalf("int At mismatch: got %d want %d", got, want)
	}
	if got, want := intRepeat.MustAt(2), int32(-30); got != want {
		t.Fatalf("int MustAt mismatch: got %d want %d", got, want)
	}

	largeInts := []int64{-1, 1 << 40}
	int64Repeat := NewRpcRepeat(&largeInts[0], int32(len(largeInts)), false)
	if got, want := int64Repeat.SafeSlice(), largeInts; !slices.Equal(got, want) {
		t.Fatalf("int64 safe slice mismatch: got %v want %v", got, want)
	}

	enums := []testRpcRepeatEnum{1, -2, 3}
	enumRepeat := NewRpcRepeat(&enums[0], int32(len(enums)), false)
	if got, want := enumRepeat.SafeSlice(), enums; !slices.Equal(got, want) {
		t.Fatalf("enum safe slice mismatch: got %v want %v", got, want)
	}

	floats := []float32{1.25, -2.5}
	floatRepeat := NewRpcRepeat(&floats[0], int32(len(floats)), false)
	if got, want := floatRepeat.At(0), float32(1.25); got != want {
		t.Fatalf("float32 At mismatch: got %v want %v", got, want)
	}

	wideFloats := []float64{-1.5, 2.75}
	float64Repeat := NewRpcRepeat(&wideFloats[0], int32(len(wideFloats)), false)
	if got, want := float64Repeat.SafeSlice(), wideFloats; !slices.Equal(got, want) {
		t.Fatalf("float64 safe slice mismatch: got %v want %v", got, want)
	}
}

func TestRpcRepeatUnsafeSliceSharesBackingMemory(t *testing.T) {
	values := []int32{10, 20, 30}

	rpc := NewRpcRepeat(&values[0], int32(len(values)), false)
	got := rpc.UnsafeSlice()

	if len(got) != len(values) {
		t.Fatalf("length mismatch: got %d want %d", len(got), len(values))
	}
	if unsafe.SliceData(got) != &values[0] {
		t.Fatal("expected UnsafeSlice to share the original backing memory")
	}
}

func TestRpcRepeatSafeSliceCopiesOnlyOnce(t *testing.T) {
	values := []int32{7, 8, 9}

	rpc := NewRpcRepeat(&values[0], int32(len(values)), false)
	first := rpc.SafeSlice()
	values[0] = 99
	second := rpc.SafeSlice()

	if got, want := first[0], int32(7); got != want {
		t.Fatalf("first safe slice mismatch: got %d want %d", got, want)
	}
	if got, want := second[0], int32(7); got != want {
		t.Fatalf("cached safe slice mismatch: got %d want %d", got, want)
	}
	if unsafe.SliceData(first) != unsafe.SliceData(second) {
		t.Fatal("expected SafeSlice to reuse the cached copy")
	}
}

func TestRpcRepeatNilAndEmptyInputsAreAllowed(t *testing.T) {
	if got := NewRpcRepeat[int32](nil, 0, false).Len(); got != 0 {
		t.Fatalf("expected length 0 for nil input, got %d", got)
	}
	if got := NewRpcRepeat[int32](nil, 0, false).At(0); got != 0 {
		t.Fatalf("expected zero value At for nil input, got %d", got)
	}
	if got := NewRpcRepeat[int32](nil, 0, false).UnsafeSlice(); got != nil {
		t.Fatalf("expected nil UnsafeSlice for nil input, got %v", got)
	}
	if got := NewRpcRepeat[int32](nil, 0, false).SafeSlice(); got != nil {
		t.Fatalf("expected nil SafeSlice for nil input, got %v", got)
	}

	values := []int32{1}
	if got := NewRpcRepeat(&values[0], 0, false).UnsafeSlice(); got != nil {
		t.Fatalf("expected nil UnsafeSlice for zero-length input, got %v", got)
	}
	if got := NewRpcRepeat(&values[0], 0, false).SafeSlice(); got != nil {
		t.Fatalf("expected nil SafeSlice for zero-length input, got %v", got)
	}
}

func TestRpcRepeatMustAtPanicsOnOutOfRange(t *testing.T) {
	values := []int32{1}
	rpc := NewRpcRepeat(&values[0], int32(len(values)), false)

	defer func() {
		if recover() == nil {
			t.Fatal("expected MustAt to panic on out-of-range index")
		}
	}()
	_ = rpc.MustAt(2)
}

func TestNewRpcRepeatRejectsNegativeLength(t *testing.T) {
	values := []int32{1}
	expectPanic(t, "NewRpcRepeat", func() {
		_ = NewRpcRepeat(&values[0], -1, false)
	})
}

func TestNewRpcBoolRepeatRejectsNegativeLength(t *testing.T) {
	values := []byte{1}
	expectPanic(t, "NewRpcBoolRepeat", func() {
		_ = NewRpcBoolRepeat(&values[0], -1, false)
	})
}

func TestRpcRepeatReleaseOwnershipTrueOnlyFreesOnce(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	values := []int32{1, 2, 3}
	wantPtr := uintptr(unsafe.Pointer(&values[0]))
	rpc := NewRpcRepeat(&values[0], int32(len(values)), true)

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

func TestRpcRepeatReleaseZeroLengthOwnedStillFreesPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	values := []int32{42}
	wantPtr := uintptr(unsafe.Pointer(&values[0]))
	rpc := NewRpcRepeat(&values[0], 0, true)

	if err := rpc.Release(); err != nil {
		t.Fatalf("expected zero-length owned repeat to release successfully, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected zero-length owned repeat to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected zero-length owned repeat to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcRepeatReleaseWithoutRegisteredCallbackCanRetry(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	values := []int32{3, 4, 5}
	wantPtr := uintptr(unsafe.Pointer(&values[0]))
	rpc := NewRpcRepeat(&values[0], int32(len(values)), true)

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

func TestRpcRepeatCleanupReleasesOwnedPointerAfterGC(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var wantPtr uintptr

	func() {
		values := []int32{11, 22, 33}
		wantPtr = uintptr(unsafe.Pointer(&values[0]))
		_ = NewRpcRepeat(&values[0], int32(len(values)), true)
	}()

	waitForRpcInputCleanupOnce(t, recorder, wantPtr)
}

func TestRpcRepeatReleasePreventsCleanupDoubleFree(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var rpc *RpcRepeat[int32]
	var wantPtr uintptr

	func() {
		values := []int32{5, 6, 7}
		wantPtr = uintptr(unsafe.Pointer(&values[0]))
		rpc = NewRpcRepeat(&values[0], int32(len(values)), true)
	}()

	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected explicit release to free exactly once before GC, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected explicit release to free original pointer %#x, got %#x", wantPtr, got)
	}

	rpc = nil
	assertRpcInputCleanupCallCountStays(t, recorder, 1)
}

func TestRpcBoolRepeatLenAtMustAtAndSafeSlice(t *testing.T) {
	values := []byte{1, 0, 2}

	rpc := NewRpcBoolRepeat(&values[0], int32(len(values)), false)
	if got, want := rpc.Len(), int32(len(values)); got != want {
		t.Fatalf("length mismatch: got %d want %d", got, want)
	}
	if !rpc.At(0) {
		t.Fatal("expected first bool element to be true")
	}
	if rpc.At(1) {
		t.Fatal("expected second bool element to be false")
	}
	if !rpc.MustAt(2) {
		t.Fatal("expected third bool element to be true")
	}
	first := rpc.SafeSlice()
	values[0] = 0
	second := rpc.SafeSlice()
	want := []bool{true, false, true}
	if !slices.Equal(first, want) {
		t.Fatalf("first safe slice mismatch: got %v want %v", first, want)
	}
	if !slices.Equal(second, want) {
		t.Fatalf("cached safe slice mismatch: got %v want %v", second, want)
	}
	if unsafe.SliceData(first) != unsafe.SliceData(second) {
		t.Fatal("expected RpcBoolRepeat.SafeSlice to reuse the cached copy")
	}
}

func TestRpcBoolRepeatAtOutOfRangeReturnsFalse(t *testing.T) {
	values := []byte{1, 0}
	rpc := NewRpcBoolRepeat(&values[0], int32(len(values)), false)

	if got := rpc.At(-1); got {
		t.Fatal("expected negative index to return false")
	}
	if got := rpc.At(int32(len(values))); got {
		t.Fatal("expected out-of-range index to return false")
	}
	if got := (*RpcBoolRepeat)(nil).At(0); got {
		t.Fatal("expected nil receiver to return false")
	}
}

func TestRpcBoolRepeatMustAtPanicsOnOutOfRange(t *testing.T) {
	values := []byte{1}
	rpc := NewRpcBoolRepeat(&values[0], int32(len(values)), false)

	defer func() {
		if recover() == nil {
			t.Fatal("expected MustAt to panic on out-of-range index")
		}
	}()
	_ = rpc.MustAt(2)
}

func TestRpcBoolRepeatNilAndEmptyInputsAreAllowed(t *testing.T) {
	if got := NewRpcBoolRepeat(nil, 0, false).SafeSlice(); got != nil {
		t.Fatalf("expected nil SafeSlice for nil input, got %v", got)
	}

	values := []byte{1}
	rpc := NewRpcBoolRepeat(&values[0], 0, false)
	if got, want := rpc.Len(), int32(0); got != want {
		t.Fatalf("length mismatch for zero-length input: got %d want %d", got, want)
	}
	if got := rpc.SafeSlice(); got != nil {
		t.Fatalf("expected nil SafeSlice for zero-length input, got %v", got)
	}
}

func TestRpcBoolRepeatReleaseOwnershipTrueOnlyFreesOnce(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	values := []byte{1, 0, 1}
	wantPtr := uintptr(unsafe.Pointer(&values[0]))
	rpc := NewRpcBoolRepeat(&values[0], int32(len(values)), true)

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

func TestRpcBoolRepeatReleaseZeroLengthOwnedStillFreesPointer(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	values := []byte{1}
	wantPtr := uintptr(unsafe.Pointer(&values[0]))
	rpc := NewRpcBoolRepeat(&values[0], 0, true)

	if err := rpc.Release(); err != nil {
		t.Fatalf("expected zero-length owned bool repeat to release successfully, got %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected zero-length owned bool repeat to free exactly once, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected zero-length owned bool repeat to free original pointer %#x, got %#x", wantPtr, got)
	}
}

func TestRpcBoolRepeatReleaseWithoutRegisteredCallbackCanRetry(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	values := []byte{1, 1, 0}
	wantPtr := uintptr(unsafe.Pointer(&values[0]))
	rpc := NewRpcBoolRepeat(&values[0], int32(len(values)), true)

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

func TestRpcBoolRepeatCleanupReleasesOwnedPointerAfterGC(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var wantPtr uintptr

	func() {
		values := []byte{1, 0, 1}
		wantPtr = uintptr(unsafe.Pointer(&values[0]))
		_ = NewRpcBoolRepeat(&values[0], int32(len(values)), true)
	}()

	waitForRpcInputCleanupOnce(t, recorder, wantPtr)
}

func TestRpcBoolRepeatReleasePreventsCleanupDoubleFree(t *testing.T) {
	ResetFreeCallbackForTesting()
	t.Cleanup(ResetFreeCallbackForTesting)

	recorder := registerFreeCallbackRecorder()
	var rpc *RpcBoolRepeat
	var wantPtr uintptr

	func() {
		values := []byte{1, 0, 1}
		wantPtr = uintptr(unsafe.Pointer(&values[0]))
		rpc = NewRpcBoolRepeat(&values[0], int32(len(values)), true)
	}()

	if err := rpc.Release(); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if got := recorder.calls.Load(); got != 1 {
		t.Fatalf("expected explicit release to free exactly once before GC, got %d calls", got)
	}
	if got := recorder.ptr.Load(); got != wantPtr {
		t.Fatalf("expected explicit release to free original pointer %#x, got %#x", wantPtr, got)
	}

	rpc = nil
	assertRpcInputCleanupCallCountStays(t, recorder, 1)
}
