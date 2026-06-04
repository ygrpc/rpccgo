package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeStreamSessions(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	renderRuntimeNativeStreamSession(g, serviceName, method)
	renderRuntimeMessageStreamSession(g, serviceName, method)
}

func renderRuntimeNativeStreamSession(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	renderRuntimeEntrySession(g, runtimeStreamNativeSessionName(serviceName, method))
}

func renderRuntimeMessageStreamSession(g *protogen.GeneratedFile, serviceName string, method runtimeMethodProjection) {
	renderRuntimeEntrySession(g, runtimeStreamMessageSessionName(serviceName, method))
}

func renderRuntimeEntrySession(g *protogen.GeneratedFile, sessionName string) {
	g.P("type ", sessionName, " struct {")
	g.P("kind rpcruntime.ServerKind")
	g.P("session any")
	g.P("}")
	g.P()
}

func runtimeStreamNativeSessionName(serviceName string, method runtimeMethodProjection) string {
	return lowerInitial(serviceName) + method.Identity.GoName + "NativeStreamSession"
}

func runtimeStreamMessageSessionName(serviceName string, method runtimeMethodProjection) string {
	return lowerInitial(serviceName) + method.Identity.GoName + "MessageStreamSession"
}
