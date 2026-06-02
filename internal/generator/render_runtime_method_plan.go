package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

type runtimeAdapterMethod struct {
	SourceFullName        string
	AdapterName           string
	AdapterArgs           string
	AdapterResult         string
	MethodGoName          string
	SessionName           string
	NativeArgs            string
	NativeReturns         string
	NativeZero            string
	NativeErrZero         string
	NativeNoActiveZero    string
	NativeConverterZero   string
	NativeInvalidZero     string
	NativeArgNames        string
	NativeNames           string
	NativeVarDecls        []string
	Streaming             bool
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
}

func buildRuntimeAdapterMethods(g *protogen.GeneratedFile, service ServicePlan) ([]runtimeAdapterMethod, error) {
	if len(service.Methods) == 0 {
		return []runtimeAdapterMethod{
			{AdapterName: "DispatchUnary", AdapterResult: " error", MethodGoName: "DispatchUnary", SessionName: service.GoName + "DispatchUnaryNativeStreamSession"},
			{AdapterName: "StartClientStream", AdapterResult: " (" + service.GoName + "ClientStreamNativeStreamSession, error)", MethodGoName: "ClientStream", SessionName: service.GoName + "ClientStreamNativeStreamSession", Streaming: true, CanSend: true, FinishReturnsResponse: true},
			{AdapterName: "StartServerStream", AdapterResult: " (" + service.GoName + "ServerStreamNativeStreamSession, error)", MethodGoName: "ServerStream", SessionName: service.GoName + "ServerStreamNativeStreamSession", Streaming: true, CanRecv: true},
			{AdapterName: "StartBidiStream", AdapterResult: " (" + service.GoName + "BidiStreamNativeStreamSession, error)", MethodGoName: "BidiStream", SessionName: service.GoName + "BidiStreamNativeStreamSession", Streaming: true, CanSend: true, CanRecv: true, CanCloseSend: true},
		}, nil
	}

	methods := make([]runtimeAdapterMethod, 0, len(service.Methods))
	seen := make(map[string]string, len(service.Methods))
	for _, method := range service.Methods {
		rendered, err := runtimeAdapterMethodFor(g, method)
		if err != nil {
			return nil, err
		}
		if previous, exists := seen[rendered.AdapterName]; exists {
			return nil, fmt.Errorf("runtime adapter method %s for %s collides with %s", rendered.AdapterName, method.FullName, previous)
		}
		seen[rendered.AdapterName] = method.FullName
		methods = append(methods, rendered)
	}
	return methods, nil
}

func runtimeAdapterMethodFor(g *protogen.GeneratedFile, method MethodPlan) (runtimeAdapterMethod, error) {
	if err := ValidateMethodRenderPlan(method); err != nil {
		return runtimeAdapterMethod{}, err
	}
	shape := method.RenderPlan
	nativeFields := method.Contract.Native.RequestFields
	responseFields := method.Contract.Native.ResponseFields
	sessionName := shape.Symbols.NativeSessionType
	nativeArgs := nativeGoRequestParams(g, nativeFields)
	nativeReturns := nativeGoResponseReturns(g, responseFields)
	nativeZero := nativeGoZeroReturns(responseFields, `errors.New("rpccgo native server method is not implemented")`)
	nativeErrZero := nativeGoZeroReturns(responseFields, "err")
	nativeNoActiveZero := nativeGoZeroReturns(responseFields, "rpcruntime.ErrNoActiveServer")
	nativeConverterZero := nativeGoZeroReturns(responseFields, "err")
	nativeInvalidZero := nativeGoZeroReturns(responseFields, "rpcruntime.ErrStreamInvalidHandle")
	nativeArgNames := nativeGoRequestArgNames(nativeFields)
	nativeResultNames := nativeGoResponseResultNames(responseFields)
	nativeVarDecls := nativeGoResponseResultVarDecls(g, responseFields)
	rendered := runtimeAdapterMethod{
		SourceFullName:        method.FullName,
		MethodGoName:          method.GoName,
		AdapterName:           shape.Symbols.NativeAdapterMethod,
		SessionName:           sessionName,
		NativeArgs:            nativeArgs,
		NativeReturns:         nativeReturns,
		NativeZero:            nativeZero,
		NativeErrZero:         nativeErrZero,
		NativeNoActiveZero:    nativeNoActiveZero,
		NativeConverterZero:   nativeConverterZero,
		NativeInvalidZero:     nativeInvalidZero,
		NativeArgNames:        nativeArgNames,
		NativeNames:           nativeResultNames,
		NativeVarDecls:        nativeVarDecls,
		Streaming:             shape.Lifecycle.Streaming,
		CanSend:               shape.Lifecycle.CanSend,
		CanRecv:               shape.Lifecycle.CanRecv,
		CanCloseSend:          shape.Lifecycle.CanCloseSend,
		FinishReturnsResponse: shape.Lifecycle.FinishReturnsResponse,
	}
	if !rendered.Streaming {
		rendered.AdapterArgs = nativeArgs
		rendered.AdapterResult = " (" + nativeReturns + ")"
		return rendered, nil
	}
	rendered.AdapterResult = " (" + sessionName + ", error)"
	if rendered.CanRecv && !rendered.CanSend {
		rendered.AdapterArgs = nativeArgs
	}
	return rendered, nil
}

type runtimeStreamShape int

const (
	runtimeStreamInvalid runtimeStreamShape = iota
	runtimeStreamUnary
	runtimeStreamClient
	runtimeStreamServer
	runtimeStreamBidi
)

func runtimeStreamShapeFor(method runtimeAdapterMethod) runtimeStreamShape {
	switch {
	case !method.Streaming:
		return runtimeStreamUnary
	case method.CanSend && method.FinishReturnsResponse:
		return runtimeStreamClient
	case method.CanRecv && !method.CanSend:
		return runtimeStreamServer
	case method.CanSend && method.CanRecv && method.CanCloseSend:
		return runtimeStreamBidi
	default:
		return runtimeStreamInvalid
	}
}

func nativeRuntimeMessageType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}

func runtimeStreamingMethods(methods []runtimeAdapterMethod) []runtimeAdapterMethod {
	streaming := make([]runtimeAdapterMethod, 0, len(methods))
	for _, method := range methods {
		if method.Streaming {
			streaming = append(streaming, method)
		}
	}
	return streaming
}

func methodForRuntimeService(service ServicePlan, method runtimeAdapterMethod) MethodPlan {
	for _, candidate := range service.Methods {
		if candidate.GoName == method.MethodGoName {
			return candidate
		}
	}
	return MethodPlan{GoName: method.MethodGoName}
}
