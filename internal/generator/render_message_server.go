package generator

import (
	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := newGeneratedFile(plugin, plan, file, protogen.GoImportPath(plan.GoImportPath))
	runtimeMethods, err := buildRuntimeAdapterMethods(g, service)
	if err != nil {
		return err
	}

	serverName := service.GoName + "CGOMessageServer"

	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("import (")
	g.P(`context "context"`)
	g.P(`errors "errors"`)
	g.P(`rpcruntime "rpccgo/rpcruntime"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()
	g.P("type ", serverName, " interface {")
	for _, method := range runtimeMethods {
		switch method.SessionKind {
		case SessionKindNone:
			g.P(method.MethodGoName, "(ctx context.Context, req []byte) ([]byte, error)")
		case SessionKindClient:
			g.P("Start", method.MethodGoName, "(ctx context.Context) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		case SessionKindServer:
			g.P("Start", method.MethodGoName, "(ctx context.Context, req []byte) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		case SessionKindBidi:
			g.P("Start", method.MethodGoName, "(ctx context.Context) (", service.GoName, method.MethodGoName, "MessageStreamSession, error)")
		}
	}
	g.P("}")
	g.P()
	g.P("func Register", service.GoName, "CGOMessageServer(server ", serverName, ") (rpcruntime.AdapterSnapshot[", serverName, "], error) {")
	g.P("if server == nil {")
	g.P(`return rpcruntime.AdapterSnapshot[`, serverName, `]{}, errors.New("rpccgo: `, service.GoName, ` cgo message server is nil")`)
	g.P("}")
	g.P("return register", service.GoName, "CGOMessageServer(server)")
	g.P("}")
	g.P()
	return nil
}
