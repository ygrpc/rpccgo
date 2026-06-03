package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func protoDocComment(comments string) string {
	if comments == "" {
		return ""
	}

	lines := strings.Split(comments, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		if strings.HasPrefix(trimmed, serviceRPCCGODirective) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n")
}

func renderDocLine(g *protogen.GeneratedFile, comments string, args ...any) {
	if comments == "" {
		g.P(args...)
		return
	}

	line := make([]any, 0, len(args)+1)
	line = append(line, protogen.Comments(comments))
	line = append(line, args...)
	g.P(line...)
}
