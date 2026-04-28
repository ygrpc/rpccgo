package main

import (
	"rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	generator.ProtogenOptions().Run(func(plugin *protogen.Plugin) error {
		_, err := generator.Generate(plugin)
		return err
	})
}
