package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderGRPCServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// ", messageStageMarker(service, file))
	return nil
}
