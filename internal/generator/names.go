package generator

import (
	"strings"
	"unicode"
)

func lowerInitial(name string) string {
	if name == "" {
		return ""
	}

	runes := []rune(name)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func upperCamelFromSnake(value string) string {
	parts := strings.Split(value, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}

func upperCamelIdentifier(value string) string {
	if value == "" {
		return ""
	}
	if value == strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value) {
		return upperInitial(value)
	}
	return upperCamelFromSnake(lowerSnakeCase(value))
}

func cgoExportName(parts ...string) string {
	var b strings.Builder
	b.WriteString("rpccgo")
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(upperCamelIdentifier(part))
	}
	return b.String()
}

func cgoSharedExportName(name string) string {
	return cgoExportName(name)
}

func cgoServiceExportName(contract string, plan FilePlan, service ServicePlan, parts ...string) string {
	exportParts := []string{contract, plan.GoPackageName, service.GoName}
	exportParts = append(exportParts, parts...)
	return cgoExportName(exportParts...)
}

func lowerSnakeCase(name string) string {
	var b strings.Builder
	runes := []rune(name)
	b.Grow(len(name))

	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if shouldInsertSnakeSeparator(runes, i) {
				writeSnakeSeparator(&b)
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		writeSnakeSeparator(&b)
	}

	return strings.Trim(b.String(), "_")
}

func shouldInsertSnakeSeparator(runes []rune, i int) bool {
	if i == 0 {
		return false
	}

	prev := runes[i-1]
	curr := runes[i]
	if !(unicode.IsLetter(prev) || unicode.IsDigit(prev)) {
		return false
	}
	if !unicode.IsUpper(curr) {
		return false
	}
	if unicode.IsLower(prev) || unicode.IsDigit(prev) {
		return true
	}
	if i+1 >= len(runes) {
		return false
	}
	if i == 1 {
		return false
	}
	return unicode.IsLower(runes[i+1])
}

func writeSnakeSeparator(b *strings.Builder) {
	if b.Len() == 0 {
		return
	}

	s := b.String()
	if s[len(s)-1] == '_' {
		return
	}
	b.WriteByte('_')
}
