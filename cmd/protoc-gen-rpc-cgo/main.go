package main

import (
	"github.com/ygrpc/rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	generator.ProtogenOptions().Run(run)
}

func run(plugin *protogen.Plugin) error {
	plan, err := generator.Generate(plugin)
	if err != nil {
		return err
	}
	return generator.RenderGeneratedFiles(plugin, plan)
}
