package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderCGOMainFile(plugin *protogen.Plugin, pkg PackagePlan, file GeneratedArtifactPlan) {
	g := newGeneratedSharedFile(plugin, file, protogen.GoImportPath(packageCGOImportPath(pkg)), "rpccgo cgo main")
	g.P("package main")
	g.P()
	g.P("// rpccgo cgo main generated file")
	g.P()
	g.P("func main() {}")
}
