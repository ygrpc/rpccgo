package main

import (
	"github.com/ygrpc/rpccgo/internal/generator"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	generator.JNIProtogenOptions().Run(run)
}

func run(plugin *protogen.Plugin) error {
	_, err := generator.GenerateJNIWithOptions(plugin)
	return err
}
