package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderDoc(g *protogen.GeneratedFile, name, text string) {
	g.P("// ", name, " ", text)
}
