package gen

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emicklei/proto"
)

// RPC represents an RPC method extracted from proto file
type RPC struct {
	Name           string // e.g., "GetUser"
	RequestType    string // e.g., "GetUserRequest"
	ResponseType   string // e.g., "GetUserResponse"
	StreamsRequest bool   // true if request is streaming
	StreamsReturns bool   // true if response is streaming
	Comment        string // Comment from proto file
}

// Service represents a service extracted from proto file
type Service struct {
	Name string // e.g., "UserService"
	RPCs []RPC
}

// Proto represents parsed proto file
type Proto struct {
	Package      string
	GoPackage    string // Cleaned package name (after stripping path and alias)
	RawGoPackage string // Original go_package value from proto (full path with ;alias)
	Services     []Service
}

// ParseProto parses a proto file and extracts service information
func ParseProto(path string) (*Proto, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	r, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to open proto file: %w", err)
	}
	defer r.Close()

	parser := proto.NewParser(r)
	set, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto file: %w", err)
	}

	result := &Proto{}

	proto.Walk(
		set,
		proto.WithPackage(func(p *proto.Package) {
			result.Package = p.Name
		}),
		proto.WithOption(func(option *proto.Option) {
			if option.Name == "go_package" {
				result.RawGoPackage = option.Constant.Source
				result.GoPackage = option.Constant.Source
			}
		}),
		proto.WithService(func(service *proto.Service) {
			svc := Service{
				Name: service.Name,
			}

			for _, el := range service.Elements {
				rpc, ok := el.(*proto.RPC)
				if !ok {
					continue
				}

				method := RPC{
					Name:           rpc.Name,
					RequestType:    rpc.RequestType,
					ResponseType:   rpc.ReturnsType,
					StreamsRequest: rpc.StreamsRequest,
					StreamsReturns: rpc.StreamsReturns,
				}

				// Extract comment if present
				if rpc.Comment != nil {
					method.Comment = extractComment(rpc.Comment)
				}

				svc.RPCs = append(svc.RPCs, method)
			}

			result.Services = append(result.Services, svc)
		}),
	)

	// If GoPackage is empty, use Package
	if result.GoPackage == "" {
		result.GoPackage = result.Package
	}

	// Clean up go_package (remove paths, keep just the package name)
	if idx := strings.LastIndex(result.GoPackage, "/"); idx >= 0 {
		result.GoPackage = result.GoPackage[idx+1:]
	}
	// Remove any semicolon suffix (e.g., "user;user")
	if idx := strings.Index(result.GoPackage, ";"); idx >= 0 {
		result.GoPackage = result.GoPackage[idx+1:]
	}

	return result, nil
}

// extractComment extracts comment text from proto Comment
func extractComment(c *proto.Comment) string {
	if c == nil || len(c.Lines) == 0 {
		return ""
	}

	var lines []string
	for _, line := range c.Lines {
		// Clean up comment prefixes
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, " ")
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	return CamelCase(s)
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	var result []byte
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(unicode.ToLower(r)))
		} else {
			result = append(result, byte(r))
		}
	}
	return string(result)
}

// CamelCase converts a string to CamelCase
// Adapted from protobuf's CamelCase
func CamelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// GoSanitized sanitizes a string for use as a Go identifier
// Adapted from protobuf's GoSanitized
func GoSanitized(s string) string {
	// Sanitize the input to the set of valid characters,
	// which must be '_' or be in the Unicode L or N categories.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, s)

	// Prepend '_' in the event of a Go keyword conflict or if
	// the identifier is invalid (does not start in the Unicode L category).
	r, _ := utf8.DecodeRuneInString(s)
	if token.Lookup(s).IsKeyword() || !unicode.IsLetter(r) {
		return "_" + s
	}
	return s
}

func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
