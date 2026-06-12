package main

import (
	"github.com/ygrpc/rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	generator.DartProtogenOptions().Run(run)
}

func run(plugin *protogen.Plugin) error {
	_, err := generator.GenerateDartWithOptions(plugin)
	return err
}
