package main

import (
	"strings"
	"unicode"
)

func normalizeSpace(text string) string {
	if text == "" {
		return ""
	}
	text = strings.TrimSpace(text)
	var buf strings.Builder
	var space int
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			space++
		default:
			if space > 1 {
				buf.WriteRune(' ')
				buf.WriteRune(r)
				space = 0
			} else if space == 1 {
				buf.WriteRune(' ')
				buf.WriteRune(r)
				space = 0
			} else {
				buf.WriteRune(r)
			}
		}
	}

	return buf.String()
}
