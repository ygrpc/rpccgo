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
	currentNativeBindingName := lowerInitial(service.GoName) + "CurrentNativeBinding"
	currentMessageBindingName := lowerInitial(service.GoName) + "CurrentMessageBinding"
	streamRegistryName := lowerInitial(service.GoName) + "StreamRegistry"

	if service.Generation.NativeEnabled && !service.HasArtifact(GeneratedArtifactKindNativeServer) {
		renderGoNativeServerInterface(g, service, adapterName)
		renderGoNativeStreamInterfaces(g, service)
	}
	nativeBindingName := lowerInitial(service.GoName) + "NativeBinding"
	messageBindingName := lowerInitial(service.GoName) + "MessageBinding"
	nativeActiveBindingName := lowerInitial(service.GoName) + "NativeActiveBinding"
	messageActiveBindingName := lowerInitial(service.GoName) + "MessageActiveBinding"
	renderRuntimeSourceSessionInterfaces(g, service.GoName, streamingMethods)

	renderRuntimeBindingTypes(g, service, runtimeMethods)
	for _, method := range streamingMethods {
		renderRuntimeStreamSessions(g, service.GoName, method)
		renderRuntimeNativeStreamFacade(g, service.GoName, streamRegistryName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, streamRegistryName, method)
	}

	g.P("// ", currentNativeBindingName, " stores the native binding used by new native calls and stream starts.")
	g.P("// Existing stream handles keep using the binding captured by Start.")
	g.P("var ", currentNativeBindingName, " atomic.Pointer[", nativeActiveBindingName, "]")
	g.P("// ", currentMessageBindingName, " stores the message binding used by new message calls and stream starts.")
	g.P("// Existing stream handles keep using the binding captured by Start.")
	g.P("var ", currentMessageBindingName, " atomic.Pointer[", messageActiveBindingName, "]")
	g.P("var ", streamRegistryName, " rpcruntime.StreamRegistry")
	if service.Generation.NativeEnabled {
		g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	}
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P()

	if err := renderRuntimeRegistrations(g, service, runtimeMethods, currentNativeBindingName, currentMessageBindingName, nativeBindingName, messageBindingName, nativeActiveBindingName, messageActiveBindingName); err != nil {
		return err
	}
	renderRuntimeTransportMessageSessions(g, service, streamingMethods)
	renderRuntimeEntrypoints(g, service.GoName, adapterName, currentNativeBindingName, currentMessageBindingName, streamRegistryName, runtimeMethods)

	return nil
}
