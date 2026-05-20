package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderDispatcherStreamCancelIfPresent(g *protogen.GeneratedFile, serviceName, sessionType, handleExpr, ctxExpr string) {
	g.P("_ = rpcruntime.DispatcherStreamCancel[", serviceName, "ActiveAdapter, ", sessionType, "](", serviceName, "DispatcherForRuntime(), rpcruntime.StreamHandle(", handleExpr, "), func(session ", sessionType, ") error {")
	g.P("return session.Cancel(", ctxExpr, ")")
	g.P("})")
}
