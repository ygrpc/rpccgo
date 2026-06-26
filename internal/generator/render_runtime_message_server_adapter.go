package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderMessageStartHelpers(g *protogen.GeneratedFile, service ServicePlan, methods []runtimeMethodProjection, serverName string) {
	for _, method := range methods {
		if !method.Stream.Streaming {
			continue
		}
		switch method.Stream.Shape {
		case runtimeStreamClient:
			renderMessageServerClientStreamStartHelper(g, service.GoName, serverName, method)
		case runtimeStreamServer:
			renderMessageServerServerStreamStartHelper(g, service.GoName, serverName, method)
		case runtimeStreamBidi:
			renderMessageServerBidiStreamStartHelper(g, service.GoName, serverName, method)
		}
	}
}

func cgoMessageStartHelperName(serviceName, methodName string) string {
	return lowerInitial(serviceName) + methodName + "CGOMessageStart"
}

func renderMessageServerClientStreamStartHelper(g *protogen.GeneratedFile, serviceName, serverName string, method runtimeMethodProjection) {
	clientType := runtimeMessageStreamingClientInterface(method)
	reqType := runtimeMessageRequestType(method)
	respType := runtimeMessageResponseType(method)
	g.P("func ", cgoMessageStartHelperName(serviceName, method.Identity.GoName), "(ctx context.Context, server ", serverName, ") (", clientType, ", error) {")
	g.P("client, stream, streamCtx := rpcruntime.NewClientStreaming[", reqType, ", ", respType, "](ctx, rpcruntime.LocalStreamOptions{")
	g.P("RequestBuffer: 16,")
	g.P(`StreamClosed: errors.New("rpccgo: message stream is closed"),`)
	g.P(`NilRequest: errors.New("rpccgo: message request is nil"),`)
	g.P("})")
	g.P("go func() {")
	g.P("resp, err := server.", method.Identity.GoName, "(streamCtx, stream)")
	g.P("stream.Complete(resp, err)")
	g.P("}()")
	g.P("return client, nil")
	g.P("}")
	g.P()
}

func renderMessageServerServerStreamStartHelper(g *protogen.GeneratedFile, serviceName, serverName string, method runtimeMethodProjection) {
	clientType := runtimeMessageStreamingClientInterface(method)
	reqType := runtimeMessageRequestType(method)
	respType := runtimeMessageResponseType(method)
	g.P("func ", cgoMessageStartHelperName(serviceName, method.Identity.GoName), "(ctx context.Context, server ", serverName, ", req ", reqType, ") (", clientType, ", error) {")
	g.P("if req == nil {")
	g.P(`return nil, errors.New("rpccgo: message request is nil")`)
	g.P("}")
	g.P("if direct, ok := server.(interface{")
	g.P(method.Identity.GoName, "Start(context.Context, ", reqType, ") (", clientType, ", error)")
	g.P("}); ok {")
	g.P("return direct.", method.Identity.GoName, "Start(ctx, req)")
	g.P("}")
	g.P("client, stream, streamCtx := rpcruntime.NewServerStreaming[", respType, "](ctx, rpcruntime.LocalStreamOptions{")
	g.P("ResponseBuffer: 1,")
	g.P(`StreamClosed: errors.New("rpccgo: message stream is closed"),`)
	g.P(`NilResponse: errors.New("rpccgo: message response is nil"),`)
	g.P("})")
	g.P("go func() {")
	g.P("err := server.", method.Identity.GoName, "(streamCtx, req, stream)")
	g.P("stream.Complete(err)")
	g.P("}()")
	g.P("return client, nil")
	g.P("}")
	g.P()
}

func renderMessageServerBidiStreamStartHelper(g *protogen.GeneratedFile, serviceName, serverName string, method runtimeMethodProjection) {
	clientType := runtimeMessageStreamingClientInterface(method)
	reqType := runtimeMessageRequestType(method)
	respType := runtimeMessageResponseType(method)
	g.P("func ", cgoMessageStartHelperName(serviceName, method.Identity.GoName), "(ctx context.Context, server ", serverName, ") (", clientType, ", error) {")
	g.P("client, stream, streamCtx := rpcruntime.NewBidiStreaming[", reqType, ", ", respType, "](ctx, rpcruntime.LocalStreamOptions{")
	g.P("RequestBuffer: 16,")
	g.P("ResponseBuffer: 1,")
	g.P(`StreamClosed: errors.New("rpccgo: message stream is closed"),`)
	g.P(`NilRequest: errors.New("rpccgo: message request is nil"),`)
	g.P(`NilResponse: errors.New("rpccgo: message response is nil"),`)
	g.P("})")
	g.P("go func() {")
	g.P("err := server.", method.Identity.GoName, "(streamCtx, stream)")
	g.P("stream.Complete(err)")
	g.P("}()")
	g.P("return client, nil")
	g.P("}")
	g.P()
}
