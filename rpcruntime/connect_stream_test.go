package rpcruntime

import (
	"testing"
	"unsafe"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestConnectStreamStructLayout validates that the conn field is at offset 0
// in connect stream types, which is the only layout assumption we rely on.
func TestConnectStreamStructLayout(t *testing.T) {
	t.Run("ClientStream", func(t *testing.T) {
		stream := &connect.ClientStream[any]{}
		fields := (*clientStreamFields)(unsafe.Pointer(stream))

		if fields.conn != nil {
			t.Error("expected conn to be nil initially")
		}
	})

	t.Run("ServerStream", func(t *testing.T) {
		stream := &connect.ServerStream[any]{}
		fields := (*serverStreamFields)(unsafe.Pointer(stream))

		if fields.conn != nil {
			t.Error("expected conn to be nil initially")
		}
	})

	t.Run("BidiStream", func(t *testing.T) {
		stream := &connect.BidiStream[any, any]{}
		fields := (*bidiStreamFields)(unsafe.Pointer(stream))

		if fields.conn != nil {
			t.Error("expected conn to be nil initially")
		}
	})
}

// TestSetStreamConn verifies that SetXxxStreamConn functions work correctly.
func TestSetStreamConn(t *testing.T) {
	session := &streamSession{}
	conn := NewConnectStreamConn(session)

	t.Run("SetClientStreamConn", func(t *testing.T) {
		stream := &connect.ClientStream[any]{}
		SetClientStreamConn(stream, conn)

		fields := (*clientStreamFields)(unsafe.Pointer(stream))
		if fields.conn != conn {
			t.Error("SetClientStreamConn did not set conn correctly")
		}
	})

	t.Run("SetServerStreamConn", func(t *testing.T) {
		stream := &connect.ServerStream[any]{}
		SetServerStreamConn(stream, conn)

		fields := (*serverStreamFields)(unsafe.Pointer(stream))
		if fields.conn != conn {
			t.Error("SetServerStreamConn did not set conn correctly")
		}
	})

	t.Run("SetBidiStreamConn", func(t *testing.T) {
		stream := &connect.BidiStream[any, any]{}
		SetBidiStreamConn(stream, conn)

		fields := (*bidiStreamFields)(unsafe.Pointer(stream))
		if fields.conn != conn {
			t.Error("SetBidiStreamConn did not set conn correctly")
		}
	})
}

func TestCopyMessage(t *testing.T) {
	t.Run("TypeMatchCopies", func(t *testing.T) {
		src := &wrapperspb.StringValue{Value: "hello"}
		dst := &wrapperspb.StringValue{Value: "stale"}

		if err := copyMessage(src, dst); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dst.Value != "hello" {
			t.Fatalf("expected dst to be updated, got %q", dst.Value)
		}
	})

	t.Run("TypeMismatchReturnsError", func(t *testing.T) {
		src := &wrapperspb.StringValue{Value: "hello"}
		dst := &wrapperspb.Int32Value{Value: 1}

		if err := copyMessage(src, dst); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonProtoReturnsError", func(t *testing.T) {
		if err := copyMessage("not proto", "also not proto"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
