package rpcruntime

import (
	"strings"
	"testing"

	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestEncodeMessageRejectsNil(t *testing.T) {
	var message *anypb.Any
	if _, _, err := EncodeMessage(message); err == nil || err.Error() != "message is nil" {
		t.Fatalf("EncodeMessage(nil) error = %v, want message is nil", err)
	}
}

func TestEncodeMessageReturnsZeroPointerForEmptyPayload(t *testing.T) {
	ptr, length, err := EncodeMessage(&emptypb.Empty{})
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	if ptr != 0 || length != 0 {
		t.Fatalf("EncodeMessage() = (%d, %d), want (0, 0)", ptr, length)
	}
}

func TestEncodeMessageAndDecodeMessageRoundTrip(t *testing.T) {
	src := &anypb.Any{TypeUrl: "type.googleapis.com/test.Payload", Value: []byte("hello")}
	ptr, length, err := EncodeMessage(src)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	if ptr == 0 || length == 0 {
		t.Fatalf("EncodeMessage() = (%d, %d), want non-zero ptr/len", ptr, length)
	}
	t.Cleanup(func() {
		if !Release(ptr) {
			t.Fatalf("Release(%d) = false, want true", ptr)
		}
	})

	got := &anypb.Any{}
	if err := DecodeMessage(ptr, length, got); err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if !protobuf.Equal(got, src) {
		t.Fatalf("DecodeMessage() = %v, want %v", got, src)
	}
}

func TestDecodeMessageRejectsNilTarget(t *testing.T) {
	var message *anypb.Any
	if err := DecodeMessage(0, 0, message); err == nil || err.Error() != "message is nil" {
		t.Fatalf("DecodeMessage(nil target) error = %v, want message is nil", err)
	}
}

func TestDecodeMessageRejectsNegativeLength(t *testing.T) {
	if err := DecodeMessage(0, -1, &anypb.Any{}); err == nil || err.Error() != "message length is negative" {
		t.Fatalf("DecodeMessage(negative) error = %v, want message length is negative", err)
	}
}

func TestDecodeMessageRejectsNilPointerWithPayload(t *testing.T) {
	if err := DecodeMessage(0, 1, &anypb.Any{}); err == nil || err.Error() != "message pointer is nil" {
		t.Fatalf("DecodeMessage(nil pointer) error = %v, want message pointer is nil", err)
	}
}

func TestDecodeMessageZeroLengthReturnsEmptyMessage(t *testing.T) {
	got := &anypb.Any{TypeUrl: "type.googleapis.com/test.Payload", Value: []byte("stale")}
	if err := DecodeMessage(0, 0, got); err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if got.TypeUrl != "" || len(got.Value) != 0 {
		t.Fatalf("DecodeMessage() = %v, want reset empty message", got)
	}
}

func TestDecodeMessageReportsUnmarshalError(t *testing.T) {
	ptr, err := PinBytes([]byte("not-protobuf"))
	if err != nil {
		t.Fatalf("PinBytes() error = %v", err)
	}
	t.Cleanup(func() {
		if !Release(ptr) {
			t.Fatalf("Release(%d) = false, want true", ptr)
		}
	})

	decodeErr := DecodeMessage(ptr, int32(len("not-protobuf")), &anypb.Any{})
	if decodeErr == nil {
		t.Fatal("DecodeMessage() error = nil, want non-nil")
	}
	if !strings.Contains(decodeErr.Error(), "protobuf unmarshal failed") {
		t.Fatalf("DecodeMessage() error = %v, want protobuf unmarshal failed", decodeErr)
	}
}
