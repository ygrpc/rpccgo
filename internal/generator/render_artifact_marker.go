package generator

func nativeStageMarker(service ServicePlan, file GeneratedArtifactPlan) string {
	switch file.Kind {
	case GeneratedArtifactKindRuntime:
		return "rpccgo service runtime generated file for " + service.GoName
	case GeneratedArtifactKindNativeServer:
		return "rpccgo native generated file for " + service.GoName + " go native server"
	case GeneratedArtifactKindCGONativeServer:
		return "rpccgo native generated file for " + service.GoName + " cgo native server"
	case GeneratedArtifactKindCGONativeClient:
		return "rpccgo native generated file for " + service.GoName + " cgo native client"
	default:
		return "rpccgo service generated file for " + service.GoName + " unknown"
	}
}

func messageStageMarker(service ServicePlan, file GeneratedArtifactPlan) string {
	switch file.Kind {
	case GeneratedArtifactKindRuntime:
		return "rpccgo message direct generated file for " + service.GoName + " runtime"
	case GeneratedArtifactKindMessageServer:
		return "rpccgo message direct generated file for " + service.GoName + " cgo message server contract"
	case GeneratedArtifactKindCGOMessageServer:
		return "rpccgo message direct generated file for " + service.GoName + " cgo message server callbacks"
	case GeneratedArtifactKindCGOMessageClient:
		return "rpccgo message direct generated file for " + service.GoName + " cgo message client"
	default:
		return "rpccgo message direct generated file for " + service.GoName + " unknown"
	}
}
