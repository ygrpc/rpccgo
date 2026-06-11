package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedExportedDeclarationsHaveDocComments(t *testing.T) {
	file := messageContractTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect|native\n")
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", file)
	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		assertGoSourceExportedDeclarationsHaveDocComments(t, generated.GetName(), generated.GetContent())
	}
}

func TestGeneratedCGOExportWrappersHaveDocComments(t *testing.T) {
	file := messageContractTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect|native\n")
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", file)
	if _, err := GenerateWithOptions(plugin); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	for _, generated := range plugin.Response().GetFile() {
		if !strings.HasSuffix(generated.GetName(), ".cgo.rpccgo.go") {
			continue
		}
		assertCGOExportWrappersHaveDocComments(t, generated.GetName(), generated.GetContent())
	}
}

func TestGeneratorExportedDeclarationsHaveDocComments(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob generator files: %v", err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		assertGoSourceExportedDeclarationsHaveDocComments(t, name, string(content))
	}
}

func assertCGOExportWrappersHaveDocComments(t *testing.T, name, content string) {
	t.Helper()

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "//export ") {
			continue
		}
		if i == 0 {
			t.Errorf("%s:%d: cgo export wrapper lacks doc comment", name, i+1)
			continue
		}
		previous := ""
		for j := i - 1; j >= 0; j-- {
			previous = strings.TrimSpace(lines[j])
			if previous != "" {
				break
			}
		}
		if !strings.HasPrefix(previous, "//") || strings.HasPrefix(previous, "//export ") {
			t.Errorf("%s:%d: cgo export wrapper lacks doc comment", name, i+1)
		}
	}
}

func assertGoSourceExportedDeclarationsHaveDocComments(t *testing.T, name, content string) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, content, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(s.Name.Name) && !hasPublicDoc(d.Doc, s.Doc) {
						t.Errorf("%s:%d: exported type %s lacks doc comment", name, fset.Position(s.Pos()).Line, s.Name.Name)
					}
				case *ast.ValueSpec:
					for _, ident := range s.Names {
						if ast.IsExported(ident.Name) && !hasPublicDoc(d.Doc, s.Doc) {
							t.Errorf("%s:%d: exported value %s lacks doc comment", name, fset.Position(ident.Pos()).Line, ident.Name)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if ast.IsExported(d.Name.Name) && exportedReceiver(d.Recv) && !hasPublicDoc(d.Doc) {
				t.Errorf("%s:%d: exported function %s lacks doc comment", name, fset.Position(d.Pos()).Line, d.Name.Name)
			}
		}
	}
}

func hasPublicDoc(groups ...*ast.CommentGroup) bool {
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if text == "" || strings.HasPrefix(text, "export ") {
				continue
			}
			return true
		}
	}
	return false
}

func exportedReceiver(recv *ast.FieldList) bool {
	if recv == nil || len(recv.List) == 0 {
		return true
	}
	return receiverTypeExported(recv.List[0].Type)
}

func receiverTypeExported(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return ast.IsExported(t.Name)
	case *ast.StarExpr:
		return receiverTypeExported(t.X)
	case *ast.IndexExpr:
		return receiverTypeExported(t.X)
	case *ast.IndexListExpr:
		return receiverTypeExported(t.X)
	default:
		return false
	}
}
