package generator

import "google.golang.org/protobuf/compiler/protogen"

func serviceHasUnaryMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindUnary {
			return true
		}
	}
	return false
}

func serviceHasStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming != StreamingKindUnary {
			return true
		}
	}
	return false
}

func serviceHasClientStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindClientStreaming {
			return true
		}
	}
	return false
}

func serviceHasBidiStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindBidiStreaming {
			return true
		}
	}
	return false
}

func qualifiedMethodType(g *protogen.GeneratedFile, message MethodIOPlan) string {
	return g.QualifiedGoIdent(protogen.GoIdent{
		GoName:       message.GoName,
		GoImportPath: protogen.GoImportPath(message.GoImportPath),
	})
}
