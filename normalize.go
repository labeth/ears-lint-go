package earslint

import "strings"

func normalizeForParsing(text string) string {
	return strings.TrimSpace(text)
}

func normalizeKey(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	parts := strings.Fields(strings.ToLower(text))
	for i := range parts {
		parts[i] = strings.Trim(parts[i], ".,;:!?\"'`")
	}
	return strings.Join(parts, " ")
}
