package ast

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError

	// Literals
	TokenIdent  // identifier
	TokenString // "..." or '...'
	TokenNumber // integer

	// Keywords
	TokenSyntax  // syntax
	TokenInfo    // info
	TokenTypeKw  // type (keyword)
	TokenService // service
	TokenReturns // returns
	TokenImport  // import
	TokenAny     // any (HTTP method)

	// HTTP Methods
	TokenGet     // get
	TokenPost    // post
	TokenPut     // put
	TokenDelete  // delete
	TokenPatch   // patch
	TokenHead    // head
	TokenOptions // options
	TokenConnect // connect
	TokenTrace   // trace

	// Annotations
	TokenAtServer  // @server
	TokenAtHandler // @handler
	TokenAtDoc     // @doc

	// Symbols
	TokenLParen    // (
	TokenRParen    // )
	TokenLBrace    // {
	TokenLBracket  // [
	TokenRBracket  // ]
	TokenRBrace    // }
	TokenColon     // :
	TokenSemicolon // ;
	TokenComma     // ,
	TokenEqual     // =
	TokenStar      // *
	TokenBacktick  // `
	TokenSlash     // /
	TokenMinus     // -

	// Special
	TokenTag     // struct tag `...`
	TokenComment // // comment
	TokenDoc     // /// doc comment or /* ... */
)

// Token represents a lexical token
type Token struct {
	Type     TokenType
	Value    string
	Position Position
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("%s: %q at %s", tokenNames[t.Type], t.Value, t.Position)
}

var tokenNames = map[TokenType]string{
	TokenEOF:       "EOF",
	TokenError:     "Error",
	TokenIdent:     "Ident",
	TokenString:    "String",
	TokenNumber:    "Number",
	TokenSyntax:    "syntax",
	TokenInfo:      "info",
	TokenTypeKw:    "type",
	TokenService:   "service",
	TokenReturns:   "returns",
	TokenImport:    "import",
	TokenAny:       "any",
	TokenGet:       "get",
	TokenPost:      "post",
	TokenPut:       "put",
	TokenDelete:    "delete",
	TokenPatch:     "patch",
	TokenHead:      "head",
	TokenOptions:   "options",
	TokenConnect:   "connect",
	TokenTrace:     "trace",
	TokenAtServer:  "@server",
	TokenAtHandler: "@handler",
	TokenAtDoc:     "@doc",
	TokenLParen:    "(",
	TokenRParen:    ")",
	TokenLBrace:    "{",
	TokenLBracket:  "[",
	TokenRBracket:  "]",
	TokenRBrace:    "}",
	TokenColon:     ":",
	TokenSemicolon: ";",
	TokenComma:     ",",
	TokenEqual:     "=",
	TokenStar:      "*",
	TokenBacktick:  "`",
	TokenSlash:     "/",
	TokenMinus:     "-",
	TokenTag:       "Tag",
	TokenComment:   "Comment",
	TokenDoc:       "Doc",
}

var keywords = map[string]TokenType{
	"syntax":  TokenSyntax,
	"info":    TokenInfo,
	"type":    TokenTypeKw,
	"service": TokenService,
	"returns": TokenReturns,
	"import":  TokenImport,
	"any":     TokenAny,
	"get":     TokenGet,
	"post":    TokenPost,
	"put":     TokenPut,
	"delete":  TokenDelete,
	"patch":   TokenPatch,
	"head":    TokenHead,
	"options": TokenOptions,
	"connect": TokenConnect,
	"trace":   TokenTrace,
}

// Lexer tokenizes .api file content
type Lexer struct {
	input    string
	filename string
	start    int // start position of current token
	pos      int // current position in input
	line     int // current line number (1-indexed)
	col      int // current column number (1-indexed)
	linePos  int // position of line start

	tokens []Token
	errors []error
}

// NewLexer creates a new lexer for the given input
func NewLexer(input, filename string) *Lexer {
	return &Lexer{
		input:    input,
		filename: filename,
		line:     1,
		col:      1,
		linePos:  0,
	}
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}

	if len(l.errors) > 0 {
		return l.tokens, l.errors[0]
	}

	return l.tokens, nil
}

// nextToken returns the next token
func (l *Lexer) nextToken() Token {
	l.skipWhitespace()

	if l.isEOF() {
		return l.makeToken(TokenEOF, "")
	}

	ch := l.peek()

	// Handle comments
	if ch == '/' {
		if l.peekAhead(1) == '/' {
			return l.scanLineComment()
		}
		if l.peekAhead(1) == '*' {
			return l.scanBlockComment()
		}
	}

	// Handle annotations
	if ch == '@' {
		return l.scanAnnotation()
	}

	// Handle struct tags
	if ch == '`' {
		return l.scanTag()
	}

	// Handle strings
	if ch == '"' || ch == '\'' {
		return l.scanString()
	}

	// Handle numbers
	if isDigit(ch) {
		return l.scanNumber()
	}

	// Handle identifiers and keywords
	if isIdentStart(ch) {
		return l.scanIdentifier()
	}

	// Handle symbols
	return l.scanSymbol()
}

func (l *Lexer) skipWhitespace() {
	for !l.isEOF() {
		ch := l.peek()
		switch ch {
		case ' ', '\t', '\r':
			l.advance()
		case '\n':
			l.advance()
			l.line++
			l.linePos = l.pos
			l.col = 1
		default:
			l.start = l.pos
			return
		}
	}
	l.start = l.pos
}

func (l *Lexer) scanLineComment() Token {
	l.advance() // skip first /
	l.advance() // skip second /

	// Check for doc comment ///
	isDoc := l.peek() == '/'

	start := l.pos
	for !l.isEOF() && l.peek() != '\n' {
		l.advance()
	}

	value := strings.TrimSpace(l.input[start:l.pos])

	if isDoc {
		return l.makeToken(TokenDoc, value)
	}
	return l.makeToken(TokenComment, value)
}

func (l *Lexer) scanBlockComment() Token {
	l.advance() // skip /
	l.advance() // skip *

	start := l.pos
	for !l.isEOF() {
		if l.peek() == '*' && l.peekAhead(1) == '/' {
			value := strings.TrimSpace(l.input[start:l.pos])
			l.advance() // skip *
			l.advance() // skip /
			return l.makeToken(TokenDoc, value)
		}
		if l.peek() == '\n' {
			l.line++
			l.col = 0
		}
		l.advance()
	}

	return l.makeErrorToken("unterminated block comment")
}

func (l *Lexer) scanAnnotation() Token {
	l.advance() // skip @
	start := l.pos

	// Read annotation name
	for !l.isEOF() && isIdentChar(l.peek()) {
		l.advance()
	}

	name := l.input[start:l.pos]

	switch name {
	case "server":
		return l.makeToken(TokenAtServer, "@server")
	case "handler":
		return l.makeToken(TokenAtHandler, "@handler")
	case "doc":
		return l.makeToken(TokenAtDoc, "@doc")
	default:
		return l.makeToken(TokenIdent, "@"+name)
	}
}

func (l *Lexer) scanTag() Token {
	l.advance() // skip opening `
	start := l.pos

	for !l.isEOF() {
		ch := l.peek()
		if ch == '`' {
			value := l.input[start:l.pos]
			l.advance() // skip closing `
			return l.makeToken(TokenTag, value)
		}
		if ch == '\n' {
			l.line++
			l.col = 0
		}
		l.advance()
	}

	return l.makeErrorToken("unterminated struct tag")
}

func (l *Lexer) scanString() Token {
	quote := l.peek()
	l.advance() // skip opening quote
	start := l.pos

	for !l.isEOF() {
		ch := l.peek()
		if ch == quote {
			value := l.input[start:l.pos]
			l.advance() // skip closing quote
			return l.makeToken(TokenString, value)
		}
		if ch == '\\' {
			l.advance() // skip backslash
			if !l.isEOF() {
				l.advance() // skip escaped char
			}
			continue
		}
		if ch == '\n' {
			return l.makeErrorToken("unterminated string")
		}
		l.advance()
	}

	return l.makeErrorToken("unterminated string")
}

func (l *Lexer) scanNumber() Token {
	start := l.pos

	for !l.isEOF() && isDigit(l.peek()) {
		l.advance()
	}

	return l.makeToken(TokenNumber, l.input[start:l.pos])
}

func (l *Lexer) scanIdentifier() Token {
	start := l.pos

	for !l.isEOF() && isIdentChar(l.peek()) {
		l.advance()
	}

	value := l.input[start:l.pos]

	// Check for keywords
	if tokType, ok := keywords[strings.ToLower(value)]; ok {
		return l.makeToken(tokType, value)
	}

	return l.makeToken(TokenIdent, value)
}

func (l *Lexer) scanSymbol() Token {
	ch := l.peek()
	l.advance()

	switch ch {
	case '(':
		return l.makeToken(TokenLParen, "(")
	case ')':
		return l.makeToken(TokenRParen, ")")
	case '{':
		return l.makeToken(TokenLBrace, "{")
	case '}':
		return l.makeToken(TokenRBrace, "}")
	case '[':
		return l.makeToken(TokenLBracket, "[")
	case ']':
		return l.makeToken(TokenRBracket, "]")
	case ':':
		return l.makeToken(TokenColon, ":")
	case ';':
		return l.makeToken(TokenSemicolon, ";")
	case ',':
		return l.makeToken(TokenComma, ",")
	case '=':
		return l.makeToken(TokenEqual, "=")
	case '*':
		return l.makeToken(TokenStar, "*")
	case '/':
		return l.makeToken(TokenSlash, "/")
	case '-':
		return l.makeToken(TokenMinus, "-")
	default:
		return l.makeErrorToken(fmt.Sprintf("unexpected character: %c", ch))
	}
}

func (l *Lexer) makeToken(typ TokenType, value string) Token {
	return Token{
		Type:  typ,
		Value: value,
		Position: Position{
			Filename: l.filename,
			Line:     l.line,
			Column:   l.start - l.linePos + 1,
		},
	}
}

func (l *Lexer) makeErrorToken(msg string) Token {
	l.errors = append(l.errors, fmt.Errorf("%s at %s:%d:%d", msg, l.filename, l.line, l.col))
	return Token{
		Type:  TokenError,
		Value: msg,
		Position: Position{
			Filename: l.filename,
			Line:     l.line,
			Column:   l.col,
		},
	}
}

func (l *Lexer) peek() rune {
	if l.isEOF() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) peekAhead(n int) rune {
	pos := l.pos
	for i := 0; i < n; i++ {
		if pos >= len(l.input) {
			return 0
		}
		_, size := utf8.DecodeRuneInString(l.input[pos:])
		pos += size
	}
	if pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[pos:])
	return r
}

func (l *Lexer) advance() {
	if l.isEOF() {
		return
	}
	_, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	l.col++
}

func (l *Lexer) isEOF() bool {
	return l.pos >= len(l.input)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentChar(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}
