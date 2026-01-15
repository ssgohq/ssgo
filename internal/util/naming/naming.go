// Package naming provides naming convention utilities for code generation.
// It supports conversion between common naming styles used in Go projects.
package naming

import (
	"regexp"
	"strings"
	"unicode"
)

// commonAcronyms is a map of common Go acronyms that should be all uppercase
var commonAcronyms = map[string]string{
	"id":    "ID",
	"url":   "URL",
	"uri":   "URI",
	"http":  "HTTP",
	"https": "HTTPS",
	"api":   "API",
	"json":  "JSON",
	"xml":   "XML",
	"html":  "HTML",
	"css":   "CSS",
	"sql":   "SQL",
	"ip":    "IP",
	"tcp":   "TCP",
	"udp":   "UDP",
	"rpc":   "RPC",
	"grpc":  "GRPC",
	"cpu":   "CPU",
	"ram":   "RAM",
	"os":    "OS",
	"io":    "IO",
	"uid":   "UID",
	"gid":   "GID",
	"uuid":  "UUID",
	"tls":   "TLS",
	"ssl":   "SSL",
	"ssh":   "SSH",
	"ttl":   "TTL",
	"dns":   "DNS",
	"smtp":  "SMTP",
	"imap":  "IMAP",
	"pop":   "POP",
	"ftp":   "FTP",
	"sftp":  "SFTP",
	"scp":   "SCP",
	"aws":   "AWS",
	"gcp":   "GCP",
	"cdn":   "CDN",
	"jwt":   "JWT",
	"db":    "DB",
	"eof":   "EOF",
	"acl":   "ACL",
	"oauth": "OAuth",
}

// ToSnakeCase converts a string to snake_case.
//
// Examples:
//
//	ToSnakeCase("UserID") -> "user_id"
//	ToSnakeCase("HTTPServer") -> "http_server"
//	ToSnakeCase("getUser") -> "get_user"
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				// Don't add underscore if previous is uppercase and next is uppercase or end
				if !unicode.IsUpper(prev) {
					result.WriteRune('_')
				} else if i+1 < len(s) && !unicode.IsUpper(rune(s[i+1])) {
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

// ToCamelCase converts a string to camelCase.
//
// Examples:
//
//	ToCamelCase("user_id") -> "userID"
//	ToCamelCase("http_server") -> "httpServer"
//	ToCamelCase("GetUser") -> "getUser"
func ToCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	for i, word := range words {
		lower := strings.ToLower(word)
		if i == 0 {
			// First word is always lowercase
			result.WriteString(lower)
		} else if acronym, ok := commonAcronyms[lower]; ok {
			result.WriteString(acronym)
		} else if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

// ToPascalCase converts a string to PascalCase with proper Go acronym handling.
//
// Examples:
//
//	ToPascalCase("user_id") -> "UserID"
//	ToPascalCase("http-server") -> "HTTPServer"
//	ToPascalCase("get user") -> "GetUser"
func ToPascalCase(s string) string {
	words := splitWords(s)

	var result strings.Builder
	for _, word := range words {
		lower := strings.ToLower(word)
		if acronym, ok := commonAcronyms[lower]; ok {
			result.WriteString(acronym)
		} else if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

// ToKebabCase converts a string to kebab-case.
//
// Examples:
//
//	ToKebabCase("UserID") -> "user-id"
//	ToKebabCase("HTTPServer") -> "http-server"
//	ToKebabCase("getUser") -> "get-user"
func ToKebabCase(s string) string {
	snake := ToSnakeCase(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// splitWords splits a string into words by common delimiters (hyphen, underscore, space, camelCase)
func splitWords(s string) []string {
	var words []string
	var word strings.Builder

	for i, r := range s {
		if r == '-' || r == '_' || r == ' ' {
			if word.Len() > 0 {
				words = append(words, word.String())
				word.Reset()
			}
			continue
		}
		// Split on uppercase letters (camelCase input)
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(s[i-1])
			if !unicode.IsUpper(prev) && prev != '-' && prev != '_' && prev != ' ' {
				if word.Len() > 0 {
					words = append(words, word.String())
					word.Reset()
				}
			}
		}
		word.WriteRune(r)
	}
	if word.Len() > 0 {
		words = append(words, word.String())
	}
	return words
}

// SanitizePackageName converts a string to a valid Go package name.
//
// Examples:
//
//	SanitizePackageName("my-package") -> "mypackage"
//	SanitizePackageName("123pkg") -> "pkg123pkg"
func SanitizePackageName(name string) string {
	result := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, name)

	// Ensure it doesn't start with a digit
	if len(result) > 0 && unicode.IsDigit(rune(result[0])) {
		result = "pkg" + result
	}

	return result
}

// CleanTypeName removes pointer and slice prefixes from type name.
//
// Examples:
//
//	CleanTypeName("*User") -> "User"
//	CleanTypeName("[]User") -> "User"
//	CleanTypeName("[]*User") -> "User"
func CleanTypeName(name string) string {
	name = strings.TrimPrefix(name, "*")
	name = strings.TrimPrefix(name, "[]")
	name = strings.TrimPrefix(name, "*")
	return name
}

// IsPointerType checks if type is a pointer type.
func IsPointerType(name string) bool {
	return strings.HasPrefix(name, "*")
}

// IsSliceType checks if type is a slice type.
func IsSliceType(name string) bool {
	return strings.HasPrefix(name, "[]")
}

// AddAcronym adds a custom acronym to the common acronyms map.
// This allows plugins to extend the default acronym handling.
//
// Example:
//
//	AddAcronym("pdf", "PDF")
//	ToPascalCase("pdf_reader") -> "PDFReader"
func AddAcronym(lower, upper string) {
	commonAcronyms[strings.ToLower(lower)] = upper
}

// GetAcronyms returns a copy of the current acronyms map.
func GetAcronyms() map[string]string {
	result := make(map[string]string, len(commonAcronyms))
	for k, v := range commonAcronyms {
		result[k] = v
	}
	return result
}

// HandlerName generates handler function name from route and method.
//
// Example:
//
//	HandlerName("GetUser") -> "GetUserHandler"
func HandlerName(handler string) string {
	return ToPascalCase(handler) + "Handler"
}

// LogicName generates logic struct name from handler.
//
// Example:
//
//	LogicName("Login") -> "LoginLogic"
func LogicName(handler string) string {
	return ToPascalCase(handler) + "Logic"
}

// FileNameFromHandler generates file name from handler.
//
// Example:
//
//	FileNameFromHandler("GetUser") -> "get_user"
func FileNameFromHandler(handler string) string {
	return ToSnakeCase(handler)
}

// BaseHandlerName returns handler name without "Handler" suffix for file naming.
//
// Example:
//
//	BaseHandlerName("CreateTodoHandler") -> "create_todo"
func BaseHandlerName(handler string) string {
	name := ToPascalCase(handler)
	name = strings.TrimSuffix(name, "Handler")
	return ToSnakeCase(name)
}

// GroupVarName generates variable name for route group.
//
// Example:
//
//	GroupVarName("user") -> "userGroup"
//	GroupVarName("") -> "r"
func GroupVarName(group string) string {
	if group == "" {
		return "r"
	}
	return ToCamelCase(group) + "Group"
}

// goVersionRegex matches Go version strings like "go1.21.5"
var goVersionRegex = regexp.MustCompile(`go(\d+\.\d+)`)

// ParseGoVersion extracts the major.minor version from a Go version string.
//
// Example:
//
//	ParseGoVersion("go1.21.5") -> "1.21"
//	ParseGoVersion("go version go1.22.0 darwin/arm64") -> "1.22"
func ParseGoVersion(version string) string {
	matches := goVersionRegex.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
