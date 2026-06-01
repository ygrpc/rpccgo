package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderRuntimeFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))

	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}
	streamingMethods := runtimeStreamingMethods(runtimeMethods)
	codecEnabled := service.CodecEnabled
	directConnectStreaming := service.Adapters.Has(AdapterTokenMessageConnect) && serviceHasStreamingMethod(service)
	directGRPCStreaming := service.Adapters.Has(AdapterTokenMessageGRPC) && serviceHasStreamingMethod(service)
	directUnary := (service.Adapters.Has(AdapterTokenMessageConnect) || service.Adapters.Has(AdapterTokenMessageGRPC)) && serviceHasUnaryMethod(service)
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
	activeName := lowerInitial(service.GoName) + "ActiveServer"
	streamRegistryName := lowerInitial(service.GoName) + "StreamRegistry"

	if !service.NativeFileFamily.NativeServer.Enabled {
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
	nativeServerAdapterName := lowerInitial(service.GoName) + "NativeServerAdapter"
	renderGoNativeAdapter(g, service, runtimeMethods, service.GoName+"NativeServer", nativeServerAdapterName, errorNames)
	messageServerAdapterName := lowerInitial(service.GoName) + "MessageServerAdapter"
	renderRuntimeSourceSessionInterfaces(g, service.GoName, streamingMethods)
	renderMessageServerAdapter(g, service, runtimeMethods, messageAdapterName, messageServerAdapterName)

	renderRuntimeActiveServerRecord(g, service, runtimeMethods)
	for _, method := range streamingMethods {
		renderRuntimeFinalSessions(g, service.GoName, method)
		renderRuntimeNativeStreamFacade(g, service.GoName, streamRegistryName, method)
		renderRuntimeMessageStreamFacade(g, service.GoName, streamRegistryName, method)
	}

	g.P("var ", activeName, " atomic.Pointer[", lowerInitial(service.GoName), "ActiveServerRecord]")
	g.P("var ", streamRegistryName, " rpcruntime.StreamRegistry")
	g.P("var ", service.GoName, `NativeServerUnavailableErr = errors.New("rpccgo: native server is unavailable")`)
	g.P("var ", service.GoName, `MessageServerUnavailableErr = errors.New("rpccgo: message server is unavailable")`)
	g.P("var ", service.GoName, `NativeMessageConverterUnavailableErr = errors.New("rpccgo: native/message converter is not enabled")`)
	g.P()

	renderRuntimeRegistrations(g, service, adapterName, messageAdapterName, runtimeMethods, codecEnabled, activeName)
	renderRuntimeTransportMessageSessions(g, service, streamingMethods)
	renderRuntimeEntrypoints(g, service.GoName, adapterName, activeName, streamRegistryName, runtimeMethods)

	return nil
}
