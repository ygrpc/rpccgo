package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderDispatcherStreamLoadSession(g *protogen.GeneratedFile, serviceName, sessionType, invalidHandleReturn, handleExpr string) {
	g.P("session, err := rpcruntime.RequireDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "), errors.New(\"rpccgo: stream handle is invalid\"))")
	g.P("if err != nil {")
	g.P("return ", invalidHandleReturn)
	g.P("}")
}

func renderDispatcherStreamTakeSession(g *protogen.GeneratedFile, serviceName, sessionType, invalidHandleReturn, handleExpr, terminalName string) {
	g.P(terminalName, ", err := rpcruntime.TakeRequiredDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "), errors.New(\"rpccgo: stream handle is invalid\"))")
	g.P("if err != nil {")
	g.P("return ", invalidHandleReturn)
	g.P("}")
}

func renderDispatcherStreamCancelIfPresent(g *protogen.GeneratedFile, serviceName, sessionType, handleExpr, ctxExpr string) {
	g.P("_ = rpcruntime.EndDispatcherStream[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "), errors.New(\"rpccgo: stream handle is invalid\"), func(terminal ", sessionType, ") error {")
	g.P("return terminal.Cancel(", ctxExpr, ")")
	g.P("})")
}
