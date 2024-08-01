package grouter

import "strings"

func NewLineStrings(text ...string) string {
	return strings.Join(text, "\n")
}

func IsEmptyText(text string) bool {
	return len(strings.TrimSpace(text)) == 0
}
