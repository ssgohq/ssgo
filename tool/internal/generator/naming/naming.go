package naming

import (
	"strings"
	"unicode"
)

// ToPascalCase converts snake_case or other formats to PascalCase
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle common abbreviations
	s = handleAbbreviations(s)

	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			capitalizeNext = true
			continue
		}

		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

// ToCamelCase converts snake_case or other formats to camelCase
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if pascal == "" {
		return ""
	}

	// Find the first run of uppercase letters (for abbreviations like ID, URL)
	runes := []rune(pascal)
	for i := 0; i < len(runes); i++ {
		if i == 0 {
			runes[i] = unicode.ToLower(runes[i])
		} else if unicode.IsUpper(runes[i]) && (i+1 >= len(runes) || unicode.IsUpper(runes[i+1])) {
			// Part of an abbreviation at the start
			if i == 1 {
				runes[i] = unicode.ToLower(runes[i])
			} else {
				break
			}
		} else {
			break
		}
	}

	return string(runes)
}

// ToSnakeCase converts PascalCase or camelCase to snake_case
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				// Don't add underscore between consecutive uppercase (like ID)
				prevIsUpper := unicode.IsUpper(runes[i-1])
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

				if !prevIsUpper || nextIsLower {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Singularize converts a plural word to singular
func Singularize(s string) string {
	if s == "" {
		return ""
	}

	lower := strings.ToLower(s)

	// Common irregular plurals
	irregulars := map[string]string{
		"people":   "person",
		"men":      "man",
		"women":    "woman",
		"children": "child",
		"feet":     "foot",
		"teeth":    "tooth",
		"geese":    "goose",
		"mice":     "mouse",
		"data":     "datum",
		"media":    "medium",
		"indices":  "index",
		"vertices": "vertex",
		"matrices": "matrix",
	}

	if singular, ok := irregulars[lower]; ok {
		return preserveCase(s, singular)
	}

	// Common patterns
	switch {
	case strings.HasSuffix(lower, "ies") && len(s) > 3:
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(lower, "ves"):
		return s[:len(s)-3] + "f"
	case strings.HasSuffix(lower, "oes"):
		return s[:len(s)-2]
	case strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") ||
		strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "shes"):
		return s[:len(s)-2]
	case strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss"):
		return s[:len(s)-1]
	}

	return s
}

// Pluralize converts a singular word to plural
func Pluralize(s string) string {
	if s == "" {
		return ""
	}

	lower := strings.ToLower(s)

	// Common irregular plurals
	irregulars := map[string]string{
		"person": "people",
		"man":    "men",
		"woman":  "women",
		"child":  "children",
		"foot":   "feet",
		"tooth":  "teeth",
		"goose":  "geese",
		"mouse":  "mice",
		"datum":  "data",
		"medium": "media",
		"index":  "indices",
		"vertex": "vertices",
		"matrix": "matrices",
	}

	if plural, ok := irregulars[lower]; ok {
		return preserveCase(s, plural)
	}

	// Common patterns
	switch {
	case strings.HasSuffix(lower, "y") && len(s) > 1 && !isVowel(rune(lower[len(lower)-2])):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(lower, "f"):
		return s[:len(s)-1] + "ves"
	case strings.HasSuffix(lower, "fe"):
		return s[:len(s)-2] + "ves"
	case strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") ||
		strings.HasSuffix(lower, "o"):
		return s + "es"
	}

	return s + "s"
}

func isVowel(r rune) bool {
	return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u'
}

func preserveCase(original, replacement string) string {
	if len(original) == 0 || len(replacement) == 0 {
		return replacement
	}

	// If original starts with uppercase, capitalize replacement
	if unicode.IsUpper(rune(original[0])) {
		runes := []rune(replacement)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}

	return replacement
}

// handleAbbreviations handles common abbreviations like ID, URL, API
func handleAbbreviations(s string) string {
	// Replace common abbreviations with properly cased versions
	replacements := map[string]string{
		"_id":   "_ID",
		"_url":  "_URL",
		"_api":  "_API",
		"_uuid": "_UUID",
		"_ip":   "_IP",
		"_http": "_HTTP",
		"_json": "_JSON",
		"_xml":  "_XML",
		"_sql":  "_SQL",
		"_html": "_HTML",
		"_css":  "_CSS",
	}

	lower := strings.ToLower(s)
	for pattern, replacement := range replacements {
		if strings.HasSuffix(lower, pattern) {
			return s[:len(s)-len(pattern)] + replacement
		}
	}

	// Handle standalone abbreviations
	standalone := map[string]string{
		"id":   "ID",
		"url":  "URL",
		"api":  "API",
		"uuid": "UUID",
		"ip":   "IP",
	}

	if replacement, ok := standalone[lower]; ok {
		return replacement
	}

	return s
}
