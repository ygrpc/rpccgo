package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderNativeSourceSessionInterfaces(g *protogen.GeneratedFile, methods []runtimeMethodProjection) {
	for _, method := range methods {
		nativeName := method.Symbols.NativeSourceSessionType
		g.P("type ", nativeName, " interface {")
		if method.Stream.CanSend {
			g.P("Send(ctx context.Context", method.Native.Args, ") error")
		}
		if method.Stream.CanRecv {
			g.P("Recv(ctx context.Context) (", method.Native.Returns, ")")
		}
		if method.Stream.CanCloseSend {
			g.P("CloseSend(ctx context.Context) error")
		}
		if method.Stream.FinishReturnsResponse {
			g.P("Finish(ctx context.Context) (", method.Native.Returns, ")")
		} else {
			g.P("Finish(ctx context.Context) error")
		}
		g.P("Cancel(ctx context.Context) error")
		g.P("}")
		g.P()
	}
}

func renderMessageSourceSessionInterfaces(g *protogen.GeneratedFile, methods []runtimeMethodProjection) {
	for _, method := range methods {
		messageName := method.Symbols.MessageSourceSessionType
		g.P("type ", messageName, " interface {")
		if method.Stream.CanSend {
			g.P("Send(ctx context.Context, req []byte) error")
		}
		if method.Stream.CanRecv {
			g.P("Recv(ctx context.Context) ([]byte, error)")
		}
		if method.Stream.CanCloseSend {
			g.P("CloseSend(ctx context.Context) error")
		}
		if method.Stream.FinishReturnsResponse {
			g.P("Finish(ctx context.Context) ([]byte, error)")
		} else {
			g.P("Finish(ctx context.Context) error")
		}
		g.P("Cancel(ctx context.Context) error")
		g.P("}")
		g.P()
	}
}
