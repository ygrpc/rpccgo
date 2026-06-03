package generator

import "google.golang.org/protobuf/compiler/protogen"

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
	directProto := directUnary || directConnectStreaming || directGRPCStreaming

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`atomic "sync/atomic"`)
	if directFmt {
		g.P(`fmt "fmt"`)
	}
	if directProto {
		g.P(`proto "google.golang.org/protobuf/proto"`)
	}
	if service.Generation.NativeEnabled {
		g.P(`goruntime "runtime"`)
	}
	if directConnectStreaming || directGRPCStreaming || nativeServerHasStreamingMethod(service) || serviceHasStreamingMethod(service) {
		g.P(`io "io"`)
		if serviceHasClientStreamingMethod(service) || serviceHasBidiStreamingMethod(service) {
			g.P(`sync "sync"`)
		}
	}
	if directConnectStreaming {
		g.P(`connect "connectrpc.com/connect"`)
		if serviceHasClientStreamingMethod(service) {
			g.P(`time "time"`)
		}
	}
	if directGRPCStreaming {
		g.P(`grpc "google.golang.org/grpc"`)
		g.P(`metadata "google.golang.org/grpc/metadata"`)
	}
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", nativeStageMarker(service, file))
	g.P()

	adapterName := service.GoName + "NativeServer"
	activeServerName := lowerInitial(service.GoName) + "ActiveServer"
	currentBindingName := lowerInitial(service.GoName) + "CurrentBinding"
	streamRegistryName := lowerInitial(service.GoName) + "StreamRegistry"

	if !service.HasArtifact(GeneratedArtifactKindNativeServer) {
		renderGoNativeServerInterface(g, service, adapterName)
		renderGoNativeStreamInterfaces(g, service)
	}
	nativeBindingName := lowerInitial(service.GoName) + "NativeBinding"
	messageBindingName := lowerInitial(service.GoName) + "MessageBinding"
	nativeCallerBindingName := lowerInitial(service.GoName) + "NativeCallerBinding"
	messageCallerBindingName := lowerInitial(service.GoName) + "MessageCallerBinding"
	renderRuntimeSourceSessionInterfaces(g, service.GoName, streamingMethods)

	renderRuntimeActiveServerType(g, service, runtimeMethods)
	for _, method := range streamingMethods {
		renderRuntimeStreamSessions(g, service.GoName, method)
		renderRuntimeNativeStreamFacade(g, service.GoName, streamRegistryName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, streamRegistryName, method)
	}

	g.P("// ", currentBindingName, " stores the binding used by new calls and stream starts.")
	g.P("// Existing stream handles keep using the binding captured by Start.")
	g.P("var ", currentBindingName, " atomic.Pointer[", activeServerName, "]")
	g.P("var ", streamRegistryName, " rpcruntime.StreamRegistry")
	g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P()

	if err := renderRuntimeRegistrations(g, service, runtimeMethods, currentBindingName, activeServerName, nativeBindingName, messageBindingName, nativeCallerBindingName, messageCallerBindingName); err != nil {
		return err
	}
	renderRuntimeTransportMessageSessions(g, service, streamingMethods)
	renderRuntimeEntrypoints(g, service.GoName, adapterName, currentBindingName, streamRegistryName, runtimeMethods)

	return nil
}
