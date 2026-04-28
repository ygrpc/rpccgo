package rpcruntime

import (
	"context"
	"go/ast"
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
	snapshot  AdapterSnapshot[*foundationAdapter]
	lifecycle StreamLifecycle
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

	session, ok := LoadDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, handle)
	if !ok {
		t.Fatalf("load stream handle %d", handle)
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

	if _, ok := LoadDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, StreamHandle(123)); ok {
		t.Fatal("unknown stream handle loaded a session")
	}
	if _, ok := TakeDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, StreamHandle(123)); ok {
		t.Fatal("unknown stream handle was taken")
	}
	if DeleteDispatcherStream(&dispatcher, StreamHandle(123)) {
		t.Fatal("unknown stream handle delete returned true")
	}

	handle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start stream: %v", err)
	}

	session, ok := TakeDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, handle)
	if !ok {
		t.Fatalf("take stream handle %d", handle)
	}
	if _, ok := LoadDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, handle); ok {
		t.Fatal("stream handle remained loadable after terminal take")
	}
	if _, ok := TakeDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, handle); ok {
		t.Fatal("second terminal take returned true")
	}

	if !session.lifecycle.Finalize() {
		t.Fatal("first finalize returned false")
	}
	if session.lifecycle.Finalize() {
		t.Fatal("second finalize returned true")
	}
	if err := session.lifecycle.Cancel(nil); err == nil {
		t.Fatal("cancel after finalize returned nil")
	} else if !strings.Contains(err.Error(), "finalized") {
		t.Fatalf("cancel after finalize returned %q, want it to mention finalized", err.Error())
	}

	cancelHandle, err := dispatcher.StartStream(func(snapshot AdapterSnapshot[*foundationAdapter]) (any, error) {
		return &foundationStreamSession{snapshot: snapshot}, nil
	})
	if err != nil {
		t.Fatalf("start cancel stream: %v", err)
	}
	cancelSession, ok := TakeDispatcherStream[*foundationAdapter, *foundationStreamSession](&dispatcher, cancelHandle)
	if !ok {
		t.Fatalf("take cancel stream handle %d", cancelHandle)
	}
	cancelCalls := 0
	if err := cancelSession.lifecycle.Cancel(func() error {
		cancelCalls++
		return nil
	}); err != nil {
		t.Fatalf("cancel stream: %v", err)
	}
	if cancelCalls != 1 {
		t.Fatalf("cancel callback called %d times, want 1", cancelCalls)
	}
	if !cancelSession.lifecycle.Finalized() || !cancelSession.lifecycle.Canceled() {
		t.Fatal("cancel did not finalize and mark the stream canceled")
	}
	if cancelSession.lifecycle.Finalize() {
		t.Fatal("finalize after cancel returned true")
	}
	if err := cancelSession.lifecycle.Cancel(func() error {
		cancelCalls++
		return nil
	}); err == nil {
		t.Fatal("second cancel returned nil")
	} else if !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("second cancel returned %q, want it to mention canceled", err.Error())
	}
	if cancelCalls != 1 {
		t.Fatalf("second cancel called callback; calls=%d, want 1", cancelCalls)
	}
}

func TestRuntimeFoundationPackageHasNoRPCFrameworkImports(t *testing.T) {
	forbidden := []string{
		"google.golang.org/protobuf",
		"connectrpc.com/connect",
		"google.golang.org/grpc",
		"internal/generator",
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
			for _, forbiddenImport := range forbidden {
				if importPath == forbiddenImport || strings.Contains(importPath, "/"+forbiddenImport) {
					t.Fatalf("%s imports forbidden runtime foundation dependency %q", path, importPath)
				}
			}
		}
		if hasRelativeInternalGeneratorImport(file) {
			t.Fatalf("%s imports internal/generator through a relative path", path)
		}
	}
}

func hasRelativeInternalGeneratorImport(file *ast.File) bool {
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		if strings.Contains(importPath, "internal/generator") {
			return true
		}
	}
	return false
}
