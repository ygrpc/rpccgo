package rpcruntime

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

type foundationAdapter struct {
	name string
}

type foundationStreamSession struct {
	snapshot AdapterSnapshot[*foundationAdapter]
}

func TestDispatcherFoundationRegistersAndReplacesActiveServerForUnary(t *testing.T) {
	var dispatcher Dispatcher[*foundationAdapter]
	firstAdapter := &foundationAdapter{name: "first"}
	secondAdapter := &foundationAdapter{name: "second"}

	firstSnapshot, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("register first active server: %v", err)
	}
	if !firstSnapshot.HasAdapter() {
		t.Fatal("registered snapshot should have an adapter")
	}

	var firstUnary AdapterSnapshot[*foundationAdapter]
	if err := dispatcher.Invoke(context.Background(), func(_ context.Context, snapshot AdapterSnapshot[*foundationAdapter]) error {
		firstUnary = snapshot
		return nil
	}); err != nil {
		t.Fatalf("invoke first unary: %v", err)
	}
	if firstUnary.Adapter != firstAdapter || firstUnary.Version != firstSnapshot.Version {
		t.Fatalf("first unary used snapshot %#v, want adapter %#v version %d", firstUnary, firstAdapter, firstSnapshot.Version)
	}

	secondSnapshot, err := dispatcher.Register(ServerKindConnectHandler, ServerContractMessage, secondAdapter)
	if err != nil {
		t.Fatalf("register replacement active server: %v", err)
	}

	var secondUnary AdapterSnapshot[*foundationAdapter]
	if err := dispatcher.Invoke(context.Background(), func(_ context.Context, snapshot AdapterSnapshot[*foundationAdapter]) error {
		secondUnary = snapshot
		return nil
	}); err != nil {
		t.Fatalf("invoke second unary: %v", err)
	}
	if secondUnary.Adapter != secondAdapter || secondUnary.Version != secondSnapshot.Version {
		t.Fatalf("second unary used snapshot %#v, want adapter %#v version %d", secondUnary, secondAdapter, secondSnapshot.Version)
	}
	if secondUnary.Adapter == firstUnary.Adapter || secondUnary.Version <= firstUnary.Version {
		t.Fatalf("replacement did not affect later unary calls: first %#v second %#v", firstUnary, secondUnary)
	}
}

func TestDispatcherFoundationStartedStreamKeepsSnapshot(t *testing.T) {
	var dispatcher Dispatcher[*foundationAdapter]
	firstAdapter := &foundationAdapter{name: "first"}
	secondAdapter := &foundationAdapter{name: "second"}

	firstSnapshot, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, firstAdapter)
	if err != nil {
		t.Fatalf("register first active server: %v", err)
	}

	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}
	if handle == 0 {
		t.Fatal("StartStream returned zero handle")
	}

	secondSnapshot, err := dispatcher.Register(ServerKindGRPCServer, ServerContractMessage, secondAdapter)
	if err != nil {
		t.Fatalf("register replacement active server: %v", err)
	}

	var session *foundationStreamSession
	if err := DispatcherStreamReceive(&dispatcher, handle, func(got *foundationStreamSession) error {
		session = got
		return nil
	}); err != nil {
		t.Fatalf("load stream handle %d: %v", handle, err)
	}
	if session.snapshot.Adapter != firstAdapter || session.snapshot.Version != firstSnapshot.Version {
		t.Fatalf("stream snapshot changed: got %#v, want adapter %#v version %d", session.snapshot, firstAdapter, firstSnapshot.Version)
	}
	if session.snapshot.Adapter == secondAdapter || session.snapshot.Version == secondSnapshot.Version {
		t.Fatalf("stream used replacement snapshot %#v", secondSnapshot)
	}
}

func TestDispatcherFoundationStreamTerminalOperationsAreStable(t *testing.T) {
	var dispatcher Dispatcher[*foundationAdapter]
	if _, err := dispatcher.Register(ServerKindGoNative, ServerContractNative, &foundationAdapter{name: "server"}); err != nil {
		t.Fatalf("register active server: %v", err)
	}

	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	if err := DispatcherStreamReceive(&dispatcher, handle, func(session *foundationStreamSession) error {
		if session.snapshot.Adapter == nil {
			t.Fatal("receive callback saw nil adapter")
		}
		return nil
	}); err != nil {
		t.Fatalf("receive stream: %v", err)
	}
	if err := DispatcherStreamReceive[*foundationAdapter, *foundationStreamSession](&dispatcher, StreamHandle(123), nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("receive unknown handle returned %v, want invalid handle", err)
	}

	handle, err = dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start send stream: %v", err)
	}
	if err := DispatcherStreamSend(&dispatcher, handle, func(*foundationStreamSession) error { return nil }); err != nil {
		t.Fatalf("send stream: %v", err)
	}
	if err := DispatcherStreamCloseSend(&dispatcher, handle, func(*foundationStreamSession) error { return nil }); err != nil {
		t.Fatalf("close send stream: %v", err)
	}
	if err := DispatcherStreamSend[*foundationAdapter, *foundationStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamSendClosed) {
		t.Fatalf("send after close returned %v, want %v", err, ErrStreamSendClosed)
	}

	handle, err = dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start finish stream: %v", err)
	}
	if err := DispatcherStreamFinish(&dispatcher, handle, func(*foundationStreamSession) error { return nil }); err != nil {
		t.Fatalf("finish stream: %v", err)
	}
	if err := DispatcherStreamFinish[*foundationAdapter, *foundationStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("finish after consume returned %v, want invalid handle", err)
	}

	handle, err = dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start cancel stream: %v", err)
	}
	cancelCalls := 0
	if err := DispatcherStreamCancel(&dispatcher, handle, func(*foundationStreamSession) error {
		cancelCalls++
		return nil
	}); err != nil {
		t.Fatalf("cancel stream: %v", err)
	}
	if cancelCalls != 1 {
		t.Fatalf("cancel callback called %d times, want 1", cancelCalls)
	}
	if err := DispatcherStreamCancel[*foundationAdapter, *foundationStreamSession](&dispatcher, handle, nil); !errors.Is(err, ErrStreamInvalidHandle) {
		t.Fatalf("second cancel returned %v, want invalid handle", err)
	}
}

func TestRuntimeFoundationPackageHasNoRPCFrameworkImports(t *testing.T) {
	forbiddenRoots := []string{
		"google.golang.org/protobuf",
		"connectrpc.com/connect",
		"google.golang.org/grpc",
	}

	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list runtime Go files: %v", err)
	}
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports from %s: %v", path, err)
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			for _, root := range forbiddenRoots {
				if importPath == root || strings.HasPrefix(importPath, root+"/") {
					t.Fatalf("%s imports forbidden runtime foundation dependency %q", path, importPath)
				}
			}
			if importsInternalGenerator(importPath) {
				t.Fatalf("%s imports forbidden runtime foundation dependency %q", path, importPath)
			}
		}
	}
}

func importsInternalGenerator(importPath string) bool {
	return importPath == "internal/generator" ||
		strings.HasPrefix(importPath, "internal/generator/") ||
		strings.Contains(importPath, "/internal/generator/") ||
		strings.HasSuffix(importPath, "/internal/generator")
}
