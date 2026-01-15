package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Parser parses .api files into AST
type Parser struct {
	tokens   []Token
	pos      int
	filename string
}

// Parse parses an .api file and returns the AST
func Parse(filename string) (*ApiSpec, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return ParseContent(string(content), filename)
}

// ParseContent parses .api content and returns the AST
func ParseContent(content, filename string) (*ApiSpec, error) {
	lexer := NewLexer(content, filename)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	// Filter out comments (keep only doc comments for now)
	var filteredTokens []Token
	for _, tok := range tokens {
		if tok.Type != TokenComment {
			filteredTokens = append(filteredTokens, tok)
		}
	}

	p := &Parser{
		tokens:   filteredTokens,
		filename: filename,
	}

	return p.parse()
}

func (p *Parser) parse() (*ApiSpec, error) {
	spec := &ApiSpec{
		Filename: p.filename,
	}

	for !p.isEOF() {
		tok := p.current()

		switch tok.Type {
		case TokenSyntax:
			syntax, err := p.parseSyntax()
			if err != nil {
				return nil, err
			}
			spec.Syntax = syntax

		case TokenInfo:
			info, err := p.parseInfo()
			if err != nil {
				return nil, err
			}
			spec.Info = info

		case TokenImport:
			imports, err := p.parseImport()
			if err != nil {
				return nil, err
			}
			spec.Imports = append(spec.Imports, imports...)

		case TokenTypeKw:
			types, err := p.parseType()
			if err != nil {
				return nil, err
			}
			spec.Types = append(spec.Types, types...)

		case TokenAtServer:
			service, err := p.parseServiceWithAnnotation()
			if err != nil {
				return nil, err
			}
			spec.Services = append(spec.Services, service)

		case TokenService:
			service, err := p.parseService(nil)
			if err != nil {
				return nil, err
			}
			spec.Services = append(spec.Services, service)

		case TokenDoc, TokenEOF:
			p.advance()

		default:
			return nil, p.errorf("unexpected token %s", tok)
		}
	}

	return spec, nil
}

// parseSyntax parses: syntax = "v1"
func (p *Parser) parseSyntax() (*SyntaxExpr, error) {
	pos := p.current().Position
	if err := p.expect(TokenSyntax); err != nil {
		return nil, err
	}

	if err := p.expect(TokenEqual); err != nil {
		return nil, err
	}

	version, err := p.expectString()
	if err != nil {
		return nil, err
	}

	return &SyntaxExpr{
		Position: pos,
		Version:  version,
	}, nil
}

// parseInfo parses: info ( key: value ... )
func (p *Parser) parseInfo() (*InfoExpr, error) {
	pos := p.current().Position
	if err := p.expect(TokenInfo); err != nil {
		return nil, err
	}

	if err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	info := &InfoExpr{
		Position:   pos,
		Properties: make(map[string]string),
	}

	for !p.check(TokenRParen) && !p.isEOF() {
		key, keyErr := p.expectIdent()
		if keyErr != nil {
			return nil, keyErr
		}

		if colonErr := p.expect(TokenColon); colonErr != nil {
			return nil, colonErr
		}

		value, err := p.expectString()
		if err != nil {
			return nil, err
		}

		info.Properties[key] = value

		// Map to specific fields
		switch key {
		case "title":
			info.Title = value
		case "desc", "description":
			info.Desc = value
		case "author":
			info.Author = value
		case "email":
			info.Email = value
		case "version":
			info.Version = value
		}
	}

	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	return info, nil
}

// parseImport parses: import "file.api" or import ( "file1.api" "file2.api" )
func (p *Parser) parseImport() ([]*ImportExpr, error) {
	if err := p.expect(TokenImport); err != nil {
		return nil, err
	}

	var imports []*ImportExpr

	if p.check(TokenLParen) {
		p.advance() // skip (

		for !p.check(TokenRParen) && !p.isEOF() {
			pos := p.current().Position
			value, err := p.expectString()
			if err != nil {
				return nil, err
			}
			imports = append(imports, &ImportExpr{
				Position: pos,
				Value:    value,
			})
		}

		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
	} else {
		pos := p.current().Position
		value, err := p.expectString()
		if err != nil {
			return nil, err
		}
		imports = append(imports, &ImportExpr{
			Position: pos,
			Value:    value,
		})
	}

	return imports, nil
}

// parseType parses: type TypeName { ... } or type ( TypeName { ... } ... )
func (p *Parser) parseType() ([]*TypeDefine, error) {
	if err := p.expect(TokenTypeKw); err != nil {
		return nil, err
	}

	var types []*TypeDefine

	if p.check(TokenLParen) {
		p.advance() // skip (

		for !p.check(TokenRParen) && !p.isEOF() {
			typeDef, err := p.parseTypeDef()
			if err != nil {
				return nil, err
			}
			types = append(types, typeDef)
		}

		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
	} else {
		typeDef, err := p.parseTypeDef()
		if err != nil {
			return nil, err
		}
		types = append(types, typeDef)
	}

	return types, nil
}

// parseTypeDef parses a single type definition: TypeName { ... } or type alias: TypeName = OtherType
func (p *Parser) parseTypeDef() (*TypeDefine, error) {
	pos := p.current().Position

	name, nameErr := p.expectIdent()
	if nameErr != nil {
		return nil, nameErr
	}

	// Check for type alias: type ID = int64
	if p.check(TokenEqual) {
		p.advance() // skip =
		aliasType, err := p.parseTypeExpr()
		if err != nil {
			return nil, err
		}
		return &TypeDefine{
			Position: pos,
			Name:     name,
			IsAlias:  true,
			AliasOf:  aliasType,
		}, nil
	}

	// Check for struct definition
	if !p.check(TokenLBrace) {
		return nil, p.errorf("expected '{' or '=' after type name %s", name)
	}

	if braceErr := p.expect(TokenLBrace); braceErr != nil {
		return nil, braceErr
	}

	members, err := p.parseMembers()
	if err != nil {
		return nil, err
	}

	if err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return &TypeDefine{
		Position: pos,
		Name:     name,
		Members:  members,
		IsAlias:  false,
	}, nil
}

// parseMembers parses struct members
func (p *Parser) parseMembers() ([]Member, error) {
	var members []Member

	for !p.check(TokenRBrace) && !p.isEOF() {
		member, err := p.parseMember()
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// parseMember parses a single struct member
func (p *Parser) parseMember() (Member, error) {
	pos := p.current().Position

	// First, check if this is an identifier (could be field name or inline type)
	if p.check(TokenIdent) {
		name, err := p.expectIdent()
		if err != nil {
			return Member{}, err
		}

		// Check if this is an inline struct definition (name followed by {)
		if p.check(TokenLBrace) {
			p.advance() // skip {
			subMembers, err := p.parseMembers()
			if err != nil {
				return Member{}, err
			}
			if err := p.expect(TokenRBrace); err != nil {
				return Member{}, err
			}

			tag := ""
			if p.check(TokenTag) {
				tag = p.current().Value
				p.advance()
			}

			return Member{
				Position: pos,
				Name:     name,
				Type: &StructType{
					Position: pos,
					Members:  subMembers,
				},
				Tag: tag,
			}, nil
		}

		// Check if next token is end or tag (inline field without type)
		if p.check(TokenRBrace) || p.check(TokenTag) {
			// Inline field
			tag := ""
			if p.check(TokenTag) {
				tag = p.current().Value
				p.advance()
			}
			return Member{
				Position: pos,
				Name:     "",
				Type:     &IdentType{Position: pos, Name: name},
				Tag:      tag,
				IsInline: true,
			}, nil
		}

		// Check if what follows looks like another field definition
		if p.check(TokenIdent) {
			if p.peekNext().Type == TokenIdent {
				return Member{
					Position: pos,
					Name:     "",
					Type:     &IdentType{Position: pos, Name: name},
					Tag:      "",
					IsInline: true,
				}, nil
			}
		}

		// Named field: parse the type expression
		typeExpr, err := p.parseTypeExpr()
		if err != nil {
			return Member{}, err
		}

		tag := ""
		if p.check(TokenTag) {
			tag = p.current().Value
			p.advance()
		}

		return Member{
			Position: pos,
			Name:     name,
			Type:     typeExpr,
			Tag:      tag,
		}, nil
	}

	return Member{}, p.errorf("expected field definition")
}

// parseTypeExpr parses a type expression
func (p *Parser) parseTypeExpr() (TypeExpr, error) {
	pos := p.current().Position

	// Check for pointer type
	if p.check(TokenStar) {
		p.advance()
		elem, err := p.parseTypeExpr()
		if err != nil {
			return nil, err
		}
		return &PointerType{
			Position: pos,
			Element:  elem,
		}, nil
	}

	// Check for array type
	if p.check(TokenLBracket) {
		p.advance()
		if err := p.expect(TokenRBracket); err != nil {
			return nil, err
		}
		elem, err := p.parseTypeExpr()
		if err != nil {
			return nil, err
		}
		return &ArrayType{
			Position: pos,
			Element:  elem,
		}, nil
	}

	// Check for map type
	if p.check(TokenIdent) && p.current().Value == "map" {
		p.advance()
		if bracketErr := p.expect(TokenLBracket); bracketErr != nil {
			return nil, bracketErr
		}
		key, keyErr := p.parseTypeExpr()
		if keyErr != nil {
			return nil, keyErr
		}
		if bracketErr := p.expect(TokenRBracket); bracketErr != nil {
			return nil, bracketErr
		}
		value, err := p.parseTypeExpr()
		if err != nil {
			return nil, err
		}
		return &MapType{
			Position: pos,
			Key:      key,
			Value:    value,
		}, nil
	}

	// Check for interface{}
	if p.check(TokenIdent) && p.current().Value == "interface" {
		p.advance()
		if p.check(TokenLBrace) {
			p.advance()
			if err := p.expect(TokenRBrace); err != nil {
				return nil, err
			}
		}
		return &InterfaceType{Position: pos}, nil
	}

	// Simple identifier type
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	return &IdentType{
		Position: pos,
		Name:     name,
	}, nil
}

// parseServiceWithAnnotation parses: @server (...) service name { ... }
func (p *Parser) parseServiceWithAnnotation() (*ServiceExpr, error) {
	annotation, err := p.parseAnnotation()
	if err != nil {
		return nil, err
	}

	return p.parseService(annotation)
}

// parseAnnotation parses: @server ( key: value ... )
func (p *Parser) parseAnnotation() (*AnnotationExpr, error) {
	pos := p.current().Position
	if err := p.expect(TokenAtServer); err != nil {
		return nil, err
	}

	if err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	annotation := &AnnotationExpr{
		Position:   pos,
		Properties: make(map[string]string),
	}

	for !p.check(TokenRParen) && !p.isEOF() {
		key, err := p.expectIdent()
		if err != nil {
			return nil, err
		}

		if err := p.expect(TokenColon); err != nil {
			return nil, err
		}

		// Value can be identifier or string
		var value string
		if p.check(TokenString) {
			value = p.current().Value
			p.advance()
		} else {
			// Read everything until next key: or )
			var parts []string
			for !p.isEOF() {
				tok := p.current()
				if tok.Type == TokenRParen {
					break
				}
				// Check if this looks like a new key (identifier followed by colon)
				if tok.Type == TokenIdent && p.peekNext().Type == TokenColon {
					break
				}
				parts = append(parts, tok.Value)
				p.advance()
			}
			value = strings.Join(parts, "")
		}

		annotation.Properties[key] = value

		// Map to specific fields
		switch key {
		case "prefix":
			annotation.Prefix = value
		case "group":
			annotation.Group = value
		case "jwt":
			annotation.JWT = value
		case "middleware":
			annotation.Middleware = strings.Split(value, ",")
			for i := range annotation.Middleware {
				annotation.Middleware[i] = strings.TrimSpace(annotation.Middleware[i])
			}
		case "timeout":
			annotation.Timeout = value
		case "maxBytes":
			annotation.MaxBytes = value
		}
	}

	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	return annotation, nil
}

// parseService parses: service name { ... }
func (p *Parser) parseService(annotation *AnnotationExpr) (*ServiceExpr, error) {
	pos := p.current().Position

	if err := p.expect(TokenService); err != nil {
		return nil, err
	}

	name, err := p.expectServiceName()
	if err != nil {
		return nil, err
	}

	if err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	var routes []Route

	for !p.check(TokenRBrace) && !p.isEOF() {
		route, err := p.parseRoute()
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}

	if err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return &ServiceExpr{
		Position:   pos,
		Name:       name,
		Annotation: annotation,
		Routes:     routes,
	}, nil
}

// parseRoute parses a route definition
func (p *Parser) parseRoute() (Route, error) {
	var route Route

	// Parse @doc if present
	if p.check(TokenAtDoc) {
		doc, err := p.parseDoc()
		if err != nil {
			return Route{}, err
		}
		route.Doc = doc
	}

	// Parse @handler
	if !p.check(TokenAtHandler) {
		return Route{}, p.errorf("expected @handler")
	}

	pos := p.current().Position
	p.advance()

	handler, err := p.expectIdent()
	if err != nil {
		return Route{}, err
	}
	route.Position = pos
	route.Handler = handler

	// Parse HTTP method
	method, err := p.parseHTTPMethod()
	if err != nil {
		return Route{}, err
	}
	route.Method = method

	// Parse path
	path, err := p.parsePath()
	if err != nil {
		return Route{}, err
	}
	route.Path = path

	// Parse request type (optional)
	if p.check(TokenLParen) {
		p.advance()
		if !p.check(TokenRParen) {
			reqType, err := p.parseTypeExpr()
			if err != nil {
				return Route{}, err
			}
			route.RequestType = reqType
		}
		if err := p.expect(TokenRParen); err != nil {
			return Route{}, err
		}
	}

	// Parse returns (optional)
	if p.check(TokenReturns) {
		p.advance()
		if err := p.expect(TokenLParen); err != nil {
			return Route{}, err
		}
		if !p.check(TokenRParen) {
			respType, err := p.parseTypeExpr()
			if err != nil {
				return Route{}, err
			}
			route.ResponseType = respType
		}
		if err := p.expect(TokenRParen); err != nil {
			return Route{}, err
		}
	}

	return route, nil
}

// parseDoc parses @doc annotation
func (p *Parser) parseDoc() (*DocExpr, error) {
	pos := p.current().Position
	p.advance() // skip @doc

	doc := &DocExpr{
		Position:   pos,
		Properties: make(map[string]string),
	}

	// @doc "text" - single line
	if p.check(TokenString) {
		doc.Text = p.current().Value
		doc.Summary = doc.Text
		p.advance()
		return doc, nil
	}

	// @doc ( ... ) - multi-line
	if p.check(TokenLParen) {
		p.advance()

		for !p.check(TokenRParen) && !p.isEOF() {
			key, keyErr := p.expectIdent()
			if keyErr != nil {
				return nil, keyErr
			}

			if colonErr := p.expect(TokenColon); colonErr != nil {
				return nil, colonErr
			}

			value, err := p.expectString()
			if err != nil {
				return nil, err
			}

			doc.Properties[key] = value

			switch key {
			case "summary":
				doc.Summary = value
			case "description":
				doc.Description = value
			}
		}

		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
	}

	return doc, nil
}

// httpMethodMap maps token types to HTTP method strings
var httpMethodMap = map[TokenType]string{
	TokenGet:     "GET",
	TokenPost:    "POST",
	TokenPut:     "PUT",
	TokenDelete:  "DELETE",
	TokenPatch:   "PATCH",
	TokenHead:    "HEAD",
	TokenOptions: "OPTIONS",
	TokenConnect: "CONNECT",
	TokenTrace:   "TRACE",
	TokenAny:     "ANY",
}

// parseHTTPMethod parses HTTP method keyword
func (p *Parser) parseHTTPMethod() (string, error) {
	tok := p.current()
	if method, ok := httpMethodMap[tok.Type]; ok {
		p.advance()
		return method, nil
	}
	return "", p.errorf("expected HTTP method, got %s", tok)
}

// parsePath parses URL path
func (p *Parser) parsePath() (string, error) {
	if !p.check(TokenIdent) && p.current().Value != "/" {
		return "", p.errorf("expected path")
	}

	var parts []string
	path := p.current().Value
	p.advance()

	// Handle path like /users/:id/orders
	for !p.isEOF() {
		tok := p.current()
		// Stop at common boundaries
		if tok.Type == TokenLParen || tok.Type == TokenReturns ||
			tok.Type == TokenAtHandler || tok.Type == TokenAtDoc ||
			tok.Type == TokenRBrace || tok.Type == TokenAtServer {
			break
		}
		// Stop at newline effectively (next route definition)
		if tok.Type == TokenIdent && isHTTPMethod(tok.Value) {
			break
		}
		parts = append(parts, tok.Value)
		p.advance()
	}

	if len(parts) > 0 {
		path += strings.Join(parts, "")
	}

	return path, nil
}

// Helper functions

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekNext() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) advance() {
	p.pos++
}

func (p *Parser) check(typ TokenType) bool {
	return p.current().Type == typ
}

func (p *Parser) expect(typ TokenType) error {
	if p.current().Type != typ {
		return p.errorf("expected %s, got %s", tokenNames[typ], p.current())
	}
	p.advance()
	return nil
}

func (p *Parser) expectIdent() (string, error) {
	tok := p.current()
	if tok.Type != TokenIdent {
		return "", p.errorf("expected identifier, got %s", tok)
	}
	p.advance()
	return tok.Value, nil
}

func (p *Parser) expectString() (string, error) {
	tok := p.current()
	if tok.Type != TokenString {
		return "", p.errorf("expected string, got %s", tok)
	}
	p.advance()
	return tok.Value, nil
}

func (p *Parser) expectServiceName() (string, error) {
	// Service name can be identifier with hyphens: user-api
	var parts []string
	for {
		tok := p.current()
		if tok.Type == TokenIdent {
			parts = append(parts, tok.Value)
			p.advance()
		} else if tok.Value == "-" {
			parts = append(parts, "-")
			p.advance()
		} else {
			break
		}
	}
	if len(parts) == 0 {
		return "", p.errorf("expected service name")
	}
	return strings.Join(parts, ""), nil
}

func (p *Parser) isEOF() bool {
	return p.pos >= len(p.tokens) || p.current().Type == TokenEOF
}

func (p *Parser) errorf(format string, args ...interface{}) error {
	pos := p.current().Position
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %s", pos, msg)
}

func isHTTPMethod(s string) bool {
	switch strings.ToLower(s) {
	case "get", "post", "put", "delete", "patch", "head", "options", "connect", "trace", "any":
		return true
	}
	return false
}

// ResolveImports resolves all imports and returns merged spec
func ResolveImports(spec *ApiSpec) (*ApiSpec, error) {
	if len(spec.Imports) == 0 {
		return spec, nil
	}

	baseDir := filepath.Dir(spec.Filename)
	merged := &ApiSpec{
		Filename: spec.Filename,
		Syntax:   spec.Syntax,
		Info:     spec.Info,
	}

	// First, collect all imported specs
	for _, imp := range spec.Imports {
		importPath := filepath.Join(baseDir, imp.Value)
		importedSpec, err := Parse(importPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse import %s: %w", imp.Value, err)
		}

		// Recursively resolve imports
		importedSpec, err = ResolveImports(importedSpec)
		if err != nil {
			return nil, err
		}

		// Merge types
		merged.Types = append(merged.Types, importedSpec.Types...)
		// Merge services
		merged.Services = append(merged.Services, importedSpec.Services...)
	}

	// Add current spec's types and services
	merged.Types = append(merged.Types, spec.Types...)
	merged.Services = append(merged.Services, spec.Services...)

	return merged, nil
}
