package generator

import (
	"path"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderCGOExportSupportFileOnce(plugin *protogen.Plugin, plan FilePlan, rendered map[string]bool) error {
	file := cgoExportSupportFilePlan(plan)
	if !file.Enabled {
		return nil
	}
	if rendered != nil && rendered[file.Filename] {
		return nil
	}
	renderCGOExportSupportFile(plugin, plan, file)
	if rendered != nil {
		rendered[file.Filename] = true
	}
	return nil
}

func cgoExportSupportFilePlan(plan FilePlan) GeneratedFilePlan {
	return GeneratedFilePlan{
		Filename: cgoExportSupportFilename(plan),
		Enabled:  plan.GeneratedFilenamePrefix != "" && planHasCGOPackageFiles(plan),
	}
}

func cgoExportSupportFilename(plan FilePlan) string {
	prefix := plan.GeneratedFilenamePrefix
	cgoPrefix := path.Join(path.Dir(prefix), cgoDirForFilePlan(plan), path.Base(prefix))
	return cgoPrefix + ".exports.cgo.rpccgo.go"
}

func planHasCGOPackageFiles(plan FilePlan) bool {
	for _, service := range plan.Services {
		if service.NativeFileFamily.CGONativeServer.Enabled || service.NativeFileFamily.CGONativeClient.Enabled {
			return true
		}
		if service.MessageFileFamily.CGOMessageServer.Enabled || service.MessageFileFamily.CGOMessageClient.Enabled {
			return true
		}
	}
	return false
}

func renderCGOExportSupportFile(plugin *protogen.Plugin, plan FilePlan, file GeneratedFilePlan) {
	g := newGeneratedSharedFile(plugin, file, protogen.GoImportPath(cgoGoImportPath(plan)), "rpccgo cgo export support")
	g.P("package main")
	g.P()
	g.P("/*")
	g.P("#include <stdint.h>")
	g.P("*/")
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// rpccgo cgo support generated file for shared exports")
	g.P()
	g.P("//export rpccgo_take_error_text")
	g.P("func rpccgo_take_error_text(errID C.int32_t, textPtr *C.uintptr_t, textLen *C.int32_t) C.int32_t {")
	g.P("if textPtr != nil {")
	g.P("*textPtr = 0")
	g.P("}")
	g.P("if textLen != nil {")
	g.P("*textLen = 0")
	g.P("}")
	g.P("if textPtr == nil || textLen == nil {")
	g.P("return 1")
	g.P("}")
	g.P("var goPtr uintptr")
	g.P("var goLen int32")
	g.P("status := rpcruntime.TakeErrorTextForExport(int32(errID), &goPtr, &goLen)")
	g.P("if status != 0 {")
	g.P("return C.int32_t(status)")
	g.P("}")
	g.P("*textPtr = C.uintptr_t(goPtr)")
	g.P("*textLen = C.int32_t(goLen)")
	g.P("return 0")
	g.P("}")
	g.P()
	g.P("//export rpccgo_release")
	g.P("func rpccgo_release(ptr C.uintptr_t) C.int32_t {")
	g.P("if !rpcruntime.Release(uintptr(ptr)) {")
	g.P("return 1")
	g.P("}")
	g.P("return 0")
	g.P("}")
}
