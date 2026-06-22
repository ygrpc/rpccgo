package generator

import (
	"strconv"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeMethodProjections(g, service)
	if err != nil {
		return err
	}
	streamingMethods := runtimeStreamingMethodProjections(runtimeMethods)
	directConnectStreaming := service.Generation.MessageTransport == MessageTransportConnect && serviceHasStreamingMethod(service)
	directGRPCStreaming := service.Generation.MessageTransport == MessageTransportGRPC && serviceHasStreamingMethod(service)
	directUnary := serviceHasUnaryMethod(service)
	directFmt := directUnary || directConnectStreaming || directGRPCStreaming

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	if runtimeNeedsGoRuntime(service) {
		g.P(`goruntime "runtime"`)
	}
	if directFmt {
		g.P(`fmt "fmt"`)
	}
	runtimeNeedsIO := serviceHasServerStreamingMethod(service) ||
		directConnectStreaming && serviceHasServerStreamingMethod(service) ||
		directGRPCStreaming && (serviceHasServerStreamingMethod(service) || serviceHasBidiStreamingMethod(service))
	if runtimeNeedsIO {
		g.P(`io "io"`)
	}
	if directConnectStreaming {
		g.P(`connect "connectrpc.com/connect"`)
		if serviceHasClientStreamingMethod(service) {
			g.P(`time "time"`)
		}
	}
	if directGRPCStreaming {
		g.P(`grpc "google.golang.org/grpc"`)
	}
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	serviceIDName := lowerInitial(service.GoName) + "ServiceID"

	g.P("const ", serviceIDName, " rpcruntime.ServiceID = ", strconv.Quote(service.FullName))
	if service.Generation.NativeEnabled {
		renderDoc(g, service.GoName+"NativeServerUnavailableErr", "is returned when a native server registration is missing or invalid.")
		g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	}
	renderDoc(g, service.GoName+"MessageServerUnavailableErr", "is returned when a message server registration is missing or invalid.")
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P()

	renderDoc(g, "Clear"+service.GoName+"Server", "clears the current registered server for this service.")
	g.P("func Clear", service.GoName, "Server() error {")
	g.P("return rpcruntime.ClearServer(", serviceIDName, ")")
	g.P("}")
	g.P()
	renderDoc(g, "Load"+service.GoName+"RegisteredServer", "loads the current registered server record for this service.")
	g.P("func Load", service.GoName, "RegisteredServer() (rpcruntime.RegisteredServer, error) {")
	g.P("return rpcruntime.LoadServer(", serviceIDName, ")")
	g.P("}")
	g.P()

	if err := renderRuntimeRegistrations(g, service, serviceIDName); err != nil {
		return err
	}
	renderRuntimeTransportMessageSessions(g, service, streamingMethods)
	if err := renderRuntimeEntrypoints(g, service, serviceIDName, runtimeMethods); err != nil {
		return err
	}

	return nil
}

func runtimeNeedsGoRuntime(service ServicePlan) bool {
	if !service.Generation.NativeEnabled {
		return false
	}
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindUnary || method.Streaming == StreamingKindServerStreaming {
			return true
		}
	}
	return false
}
