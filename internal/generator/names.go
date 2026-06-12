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
