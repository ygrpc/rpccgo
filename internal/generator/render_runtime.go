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
		if serviceHasClientStreamingMethod(service) || serviceHasBidiStreamingMethod(service) || nativeServerHasClientInputStreamingMethod(service) {
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
	messageAdapterName := service.GoName + "CGOMessageServer"
	bindingName := lowerInitial(service.GoName) + "Binding"
	currentBindingName := lowerInitial(service.GoName) + "CurrentBinding"
	streamRegistryName := lowerInitial(service.GoName) + "StreamRegistry"

	if !service.HasArtifact(GeneratedArtifactKindNativeServer) {
		renderGoNativeServerInterface(g, service, adapterName)
		renderGoNativeStreamInterfaces(g, service)
	}
	errorNames := nativeServerErrorNamesFor(service)
	g.P("var (")
	g.P(errorNames.RequestBridgeNotImplemented, ` = errors.New("rpccgo: native request bridge is not implemented")`)
	g.P(errorNames.StreamBridgeNotImplemented, ` = errors.New("rpccgo: native stream bridge is not implemented")`)
	g.P(errorNames.StreamIsNil, ` = errors.New("rpccgo: native stream is nil")`)
	g.P(errorNames.StreamClosed, ` = errors.New("rpccgo: native stream is closed")`)
	g.P(")")
	g.P()
	nativeBindingName := lowerInitial(service.GoName) + "NativeBinding"
	renderGoNativeAdapter(g, service, runtimeMethods, service.GoName+"NativeServer", nativeBindingName, bindingName, errorNames)
	messageBindingName := lowerInitial(service.GoName) + "MessageBinding"
	renderRuntimeSourceSessionInterfaces(g, service.GoName, streamingMethods)
	renderMessageBinding(g, service, runtimeMethods, messageAdapterName, messageBindingName, bindingName)

	renderRuntimeBindingType(g, service, runtimeMethods)
	for _, method := range streamingMethods {
		renderRuntimeStreamSessions(g, service.GoName, method)
		renderRuntimeNativeStreamFacade(g, service.GoName, streamRegistryName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, streamRegistryName, method)
	}

	g.P("// ", currentBindingName, " stores the binding used by new calls and stream starts.")
	g.P("// Existing stream handles keep using the binding captured by Start.")
	g.P("var ", currentBindingName, " atomic.Pointer[", bindingName, "]")
	g.P("var ", streamRegistryName, " rpcruntime.StreamRegistry")
	g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P()

	if err := renderRuntimeRegistrations(g, service, runtimeMethods, currentBindingName, bindingName, nativeBindingName, messageBindingName); err != nil {
		return err
	}
	renderRuntimeTransportMessageSessions(g, service, streamingMethods)
	renderRuntimeEntrypoints(g, service.GoName, adapterName, currentBindingName, streamRegistryName, runtimeMethods)

	return nil
}
