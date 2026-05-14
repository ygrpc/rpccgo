package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderDispatcherStreamLoadSession(g *protogen.GeneratedFile, serviceName, sessionType, invalidHandleReturn, handleExpr string) {
	g.P("session, ok := rpcruntime.LoadDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "))")
	g.P("if !ok {")
	g.P("return ", invalidHandleReturn)
	g.P("}")
}

func renderDispatcherStreamTakeSession(g *protogen.GeneratedFile, serviceName, sessionType, invalidHandleReturn, handleExpr, terminalName string) {
	g.P(terminalName, ", ok := rpcruntime.TakeDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "))")
	g.P("if !ok {")
	g.P("return ", invalidHandleReturn)
	g.P("}")
}

func renderDispatcherStreamCancelIfPresent(g *protogen.GeneratedFile, serviceName, sessionType, handleExpr, ctxExpr string) {
	g.P("if terminal, ok := rpcruntime.TakeDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, ")); ok {")
	g.P("_ = terminal.Cancel(", ctxExpr, ")")
	g.P("}")
}

func renderDispatcherStreamFinishOnce(g *protogen.GeneratedFile, serviceName, sessionType, invalidHandleErrExpr, handleExpr, ctxExpr string) {
	g.P("var terminalOnce sync.Once")
	g.P("finish := func(done bool) error {")
	g.P("var finishErr error")
	g.P("terminalOnce.Do(func() {")
	g.P("terminal, ok := rpcruntime.TakeDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "))")
	g.P("if !ok {")
	g.P("finishErr = ", invalidHandleErrExpr)
	g.P("return")
	g.P("}")
	g.P("if done {")
	g.P("finishErr = terminal.Done(", ctxExpr, ")")
	g.P("return")
	g.P("}")
	g.P("finishErr = terminal.Cancel(", ctxExpr, ")")
	g.P("})")
	g.P("return finishErr")
	g.P("}")
}
