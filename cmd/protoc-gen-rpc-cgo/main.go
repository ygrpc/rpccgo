package main

import (
	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	generator.ProtogenOptions().Run(run)
}

func run(plugin *protogen.Plugin) error {
	_, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true})
	return err
}
