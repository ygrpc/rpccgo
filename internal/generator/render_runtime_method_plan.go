package generator

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

type runtimeMethodProjection struct {
	Identity runtimeMethodIdentityProjection
	Native   runtimeNativeProjection
	Message  runtimeMessageProjection
	Stream   runtimeStreamProjection
	Symbols  runtimeMethodSymbolsProjection
	Codec    runtimeCodecProjection
}

type runtimeMethodIdentityProjection struct {
	SourceFullName   string
	GoName           string
	DocComment       string
	MessageMethodRef string
}

type runtimeNativeProjection struct {
	EntryArgs   string
	EntryResult string

	Args             string
	Returns          string
	Zero             string
	ErrZero          string
	NoRegisteredZero string
	ConverterZero    string
	InvalidZero      string
	ArgNames         string
	ResultNames      string
	ResultVarDecls   []string
}

type runtimeMessageProjection struct {
	RequestType  string
	ResponseType string
}

type runtimeStreamProjection struct {
	Shape                 runtimeStreamShape
	Streaming             bool
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
	StartAcceptsRequest   bool
}

type runtimeMethodSymbolsProjection struct {
	NativeEntryMethod        string
	MessageEntryMethod       string
	NativeStreamRequestType  string
	NativeStreamResponseType string
}

type runtimeCodecProjection struct {
	MessageToNativeRequest            string
	MessageToNativeRequestAssignNames string
	NativeRequestToMessage            string
	MessageToNativeResponse           string
	NativeResponseToMessage           string
}

func buildRuntimeMethodProjections(g *protogen.GeneratedFile, service ServicePlan) ([]runtimeMethodProjection, error) {
	return buildRuntimeMethodProjectionsWithMessageTypes(g, service, true)
}

func buildRuntimeMethodProjectionsWithMessageTypes(g *protogen.GeneratedFile, service ServicePlan, includeMessageTypes bool) ([]runtimeMethodProjection, error) {
	if len(service.Methods) == 0 {
		return []runtimeMethodProjection{
			runtimePlaceholderMethodProjection(service.GoName, "DispatchUnary", runtimeStreamUnary),
			runtimePlaceholderMethodProjection(service.GoName, "ClientStream", runtimeStreamClient),
			runtimePlaceholderMethodProjection(service.GoName, "ServerStream", runtimeStreamServer),
			runtimePlaceholderMethodProjection(service.GoName, "BidiStream", runtimeStreamBidi),
		}, nil
	}

	methods := make([]runtimeMethodProjection, 0, len(service.Methods))
	seen := make(map[string]string, len(service.Methods))
	for _, method := range service.Methods {
		projected, err := projectRuntimeMethod(g, service, method, includeMessageTypes)
		if err != nil {
			return nil, err
		}
		if previous, exists := seen[projected.Symbols.NativeEntryMethod]; exists {
			return nil, fmt.Errorf("runtime adapter method symbol %s for %s collides with %s", projected.Symbols.NativeEntryMethod, method.FullName, previous)
		}
		seen[projected.Symbols.NativeEntryMethod] = method.FullName
		methods = append(methods, projected)
	}
	return methods, nil
}

func runtimePlaceholderMethodProjection(serviceName, methodName string, shape runtimeStreamShape) runtimeMethodProjection {
	projected := runtimeMethodProjection{
		Identity: runtimeMethodIdentityProjection{
			GoName:           methodName,
			MessageMethodRef: methodName,
		},
		Stream: runtimeStreamProjection{
			Shape: shape,
		},
		Symbols: runtimeMethodSymbolsProjection{
			NativeEntryMethod:        methodName,
			MessageEntryMethod:       methodName,
			NativeStreamRequestType:  serviceName + methodName + "NativeStreamRequest",
			NativeStreamResponseType: serviceName + methodName + "NativeStreamResponse",
		},
	}
	switch shape {
	case runtimeStreamUnary:
		projected.Native.EntryResult = " error"
	case runtimeStreamClient:
		projected.Stream.Streaming = true
		projected.Stream.CanSend = true
		projected.Stream.FinishReturnsResponse = true
		projected.Native.EntryResult = " (" + runtimeNativeStreamingClientInterface(projected) + ", error)"
	case runtimeStreamServer:
		projected.Stream.Streaming = true
		projected.Stream.Shape = runtimeStreamServer
		projected.Stream.CanRecv = true
		projected.Stream.StartAcceptsRequest = true
		projected.Native.EntryResult = " (" + runtimeNativeStreamingClientInterface(projected) + ", error)"
	case runtimeStreamBidi:
		projected.Stream.Streaming = true
		projected.Stream.CanSend = true
		projected.Stream.CanRecv = true
		projected.Stream.CanCloseSend = true
		projected.Native.EntryResult = " (" + runtimeNativeStreamingClientInterface(projected) + ", error)"
	}
	return projected
}

func projectRuntimeMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, includeMessageTypes bool) (runtimeMethodProjection, error) {
	if err := ValidateMethodRenderPlan(method); err != nil {
		return runtimeMethodProjection{}, err
	}

	stream, err := projectRuntimeStream(method)
	if err != nil {
		return runtimeMethodProjection{}, err
	}

	nativeFields := method.Contract.Native.RequestFields
	responseFields := method.Contract.Native.ResponseFields
	nativeArgs := nativeGoRequestParams(g, nativeFields)
	nativeReturns := nativeGoResponseReturns(g, responseFields)
	symbols := runtimeMethodSymbolsProjection{
		NativeEntryMethod:        method.RenderPlan.Symbols.NativeEntryMethod,
		MessageEntryMethod:       method.RenderPlan.Symbols.MessageEntryMethod,
		NativeStreamRequestType:  method.RenderPlan.Symbols.NativeStreamRequestType,
		NativeStreamResponseType: method.RenderPlan.Symbols.NativeStreamResponseType,
	}
	requestType := ""
	responseType := ""
	if includeMessageTypes {
		requestType = qualifiedMethodType(g, method.Contract.Message.RequestType)
		responseType = qualifiedMethodType(g, method.Contract.Message.ResponseType)
	}
	projected := runtimeMethodProjection{
		Identity: runtimeMethodIdentityProjection{
			SourceFullName:   method.FullName,
			GoName:           method.GoName,
			DocComment:       method.DocComment,
			MessageMethodRef: method.GoName,
		},
		Native: runtimeNativeProjection{
			Args:             nativeArgs,
			Returns:          nativeReturns,
			Zero:             nativeGoZeroReturns(responseFields, `errors.New("rpccgo native server method is not implemented")`),
			ErrZero:          nativeGoZeroReturns(responseFields, "err"),
			NoRegisteredZero: nativeGoZeroReturns(responseFields, "rpcruntime.ErrNoRegisteredServer"),
			ConverterZero:    nativeGoZeroReturns(responseFields, "err"),
			InvalidZero:      nativeGoZeroReturns(responseFields, "rpcruntime.ErrStreamInvalidHandle"),
			ArgNames:         nativeGoRequestArgNames(nativeFields),
			ResultNames:      nativeGoResponseResultNames(responseFields),
			ResultVarDecls:   nativeGoResponseResultVarDecls(g, responseFields),
		},
		Message: runtimeMessageProjection{
			RequestType:  requestType,
			ResponseType: responseType,
		},
		Stream:  stream,
		Symbols: symbols,
		Codec: runtimeCodecProjection{
			MessageToNativeRequest:            codecMessageToNativeRequestName(service, method),
			MessageToNativeRequestAssignNames: codecMessageToNativeRequestAssignNames(nativeFields, "reqOwner", "err"),
			NativeRequestToMessage:            codecNativeRequestToMessageName(service, method),
			MessageToNativeResponse:           codecMessageToNativeResponseName(service, method),
			NativeResponseToMessage:           codecNativeResponseToMessageName(service, method),
		},
	}
	if !stream.Streaming {
		projected.Native.EntryArgs = nativeArgs
		projected.Native.EntryResult = " (" + nativeReturns + ")"
		return projected, nil
	}
	projected.Native.EntryResult = " (" + runtimeNativeStreamingClientInterface(projected) + ", error)"
	if stream.StartAcceptsRequest {
		projected.Native.EntryArgs = nativeArgs
	}
	return projected, nil
}

type runtimeStreamShape int

const (
	runtimeStreamInvalid runtimeStreamShape = iota
	runtimeStreamUnary
	runtimeStreamClient
	runtimeStreamServer
	runtimeStreamBidi
)

func projectRuntimeStream(method MethodPlan) (runtimeStreamProjection, error) {
	capability := method.RenderPlan.Stream
	projected := runtimeStreamProjection{
		Streaming:             capability.Streaming,
		CanSend:               capability.CanSend,
		CanRecv:               capability.CanRecv,
		CanCloseSend:          capability.CanCloseSend,
		FinishReturnsResponse: capability.FinishReturnsResponse,
	}

	switch {
	case !projected.Streaming:
		projected.Shape = runtimeStreamUnary
	case projected.CanSend && projected.FinishReturnsResponse:
		projected.Shape = runtimeStreamClient
	case projected.CanRecv && !projected.CanSend:
		projected.Shape = runtimeStreamServer
		projected.StartAcceptsRequest = true
	case projected.CanSend && projected.CanRecv && projected.CanCloseSend:
		projected.Shape = runtimeStreamBidi
	default:
		return runtimeStreamProjection{}, fmt.Errorf("method %s runtime stream shape is invalid", methodPlanName(method))
	}
	return projected, nil
}

func nativeRuntimeMessageType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return "*" + g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}

func codecMessageToNativeRequestAssignNames(fields []FieldPlan, ownerName, errName string) string {
	names := make([]string, 0, len(fields)+2)
	for _, field := range fields {
		names = append(names, lowerInitial(field.GoName))
	}
	names = append(names, ownerName, errName)
	return strings.Join(names, ", ")
}

func runtimeStreamingMethodProjections(methods []runtimeMethodProjection) []runtimeMethodProjection {
	streaming := make([]runtimeMethodProjection, 0, len(methods))
	for _, method := range methods {
		if method.Stream.Streaming {
			streaming = append(streaming, method)
		}
	}
	return streaming
}
