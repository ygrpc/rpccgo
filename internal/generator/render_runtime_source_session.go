package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderNativeSourceSessionInterfaces(g *protogen.GeneratedFile, service ServicePlan) {
	for _, method := range service.Methods {
		renderNativeStreamEnvelopeTypes(g, method)
	}
}

func renderNativeStreamEnvelopeTypes(g *protogen.GeneratedFile, method MethodPlan) {
	if method.Streaming == StreamingKindUnary {
		return
	}
	if method.Streaming == StreamingKindClientStreaming || method.Streaming == StreamingKindBidiStreaming {
		name := method.RenderPlan.Symbols.NativeStreamRequestType
		renderDoc(g, name, "carries native request values for the "+method.GoName+" stream call.")
		g.P("type ", name, " struct {")
		for _, field := range method.Contract.Native.RequestFields {
			g.P(field.GoName, " ", nativeGoRequestFieldType(g, field))
		}
		g.P("}")
		g.P()
	}
	if method.Streaming == StreamingKindClientStreaming || method.Streaming == StreamingKindServerStreaming || method.Streaming == StreamingKindBidiStreaming {
		name := method.RenderPlan.Symbols.NativeStreamResponseType
		renderDoc(g, name, "carries native response values for the "+method.GoName+" stream call.")
		g.P("type ", name, " struct {")
		for _, field := range method.Contract.Native.ResponseFields {
			g.P(field.GoName, " ", nativeGoResponseFieldType(g, field))
		}
		g.P("}")
		g.P()
	}
}
