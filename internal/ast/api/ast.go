// Package ast provides parser for .api files.
// The .api format is used to define HTTP service interfaces.
package ast

import (
	"fmt"
)

// Position represents a position in source code
type Position struct {
	Filename string
	Line     int
	Column   int
}

// String returns a string representation of the position
func (p Position) String() string {
	if p.Filename != "" {
		return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Node is the base interface for all AST nodes
type Node interface {
	Pos() Position
}

// SyntaxExpr represents syntax = "v1"
type SyntaxExpr struct {
	Position Position
	Version  string
}

func (s *SyntaxExpr) Pos() Position { return s.Position }

// InfoExpr represents info block
type InfoExpr struct {
	Position   Position
	Properties map[string]string
	Title      string
	Desc       string
	Author     string
	Email      string
	Version    string
}

func (i *InfoExpr) Pos() Position { return i.Position }

// Member represents a field in a type definition
type Member struct {
	Position Position
	Name     string   // Field name (empty for anonymous/inline fields)
	Type     TypeExpr // Type expression
	Tag      string   // struct tag: `json:"name" validate:"required"`
	Comment  string   // Inline comment
	Docs     []string // Doc comments before the field
	IsInline bool     // true if it's an anonymous/inline field
}

func (m *Member) Pos() Position { return m.Position }

// TypeExpr is the interface for type expressions
type TypeExpr interface {
	Node
	TypeName() string
}

// IdentType represents a simple type identifier (e.g., int64, string, User)
type IdentType struct {
	Position Position
	Name     string
}

func (t *IdentType) Pos() Position    { return t.Position }
func (t *IdentType) TypeName() string { return t.Name }

// ArrayType represents []Type
type ArrayType struct {
	Position Position
	Element  TypeExpr
}

func (t *ArrayType) Pos() Position    { return t.Position }
func (t *ArrayType) TypeName() string { return "[]" + t.Element.TypeName() }

// MapType represents map[Key]Value
type MapType struct {
	Position Position
	Key      TypeExpr
	Value    TypeExpr
}

func (t *MapType) Pos() Position { return t.Position }
func (t *MapType) TypeName() string {
	return fmt.Sprintf("map[%s]%s", t.Key.TypeName(), t.Value.TypeName())
}

// PointerType represents *Type
type PointerType struct {
	Position Position
	Element  TypeExpr
}

func (t *PointerType) Pos() Position    { return t.Position }
func (t *PointerType) TypeName() string { return "*" + t.Element.TypeName() }

// StructType represents inline struct definition
type StructType struct {
	Position Position
	Members  []Member
}

func (t *StructType) Pos() Position    { return t.Position }
func (t *StructType) TypeName() string { return "struct{...}" }

// InterfaceType represents interface{}
type InterfaceType struct {
	Position Position
}

func (t *InterfaceType) Pos() Position    { return t.Position }
func (t *InterfaceType) TypeName() string { return "interface{}" }

// TypeDefine represents a named type definition
type TypeDefine struct {
	Position Position
	Name     string
	Members  []Member
	Docs     []string
	IsAlias  bool     // true if this is a type alias
	AliasOf  TypeExpr // the aliased type (only set if IsAlias is true)
}

func (t *TypeDefine) Pos() Position { return t.Position }

// AnnotationExpr represents @server annotations
type AnnotationExpr struct {
	Position   Position
	Properties map[string]string
	Prefix     string   // prefix: /api/v1
	Group      string   // group: user
	JWT        string   // jwt: Auth
	Middleware []string // middleware: M1,M2
	Timeout    string   // timeout: 30s
	MaxBytes   string   // maxBytes: 10485760
}

func (a *AnnotationExpr) Pos() Position { return a.Position }

// DocExpr represents @doc annotation
type DocExpr struct {
	Position    Position
	Text        string            // Single line @doc "text"
	Properties  map[string]string // Multi-line @doc (...)
	Summary     string
	Description string
}

func (d *DocExpr) Pos() Position { return d.Position }

// Route represents a single route definition
type Route struct {
	Position     Position
	Doc          *DocExpr
	Handler      string   // Handler name from @handler
	HandlerDoc   []string // Doc comments before @handler
	Method       string   // GET, POST, PUT, DELETE, etc.
	Path         string   // /users/:id
	RequestType  TypeExpr // Request type (can be nil)
	ResponseType TypeExpr // Response type (can be nil)
	Comment      string   // Inline comment
}

func (r *Route) Pos() Position { return r.Position }

// ServiceExpr represents a service definition
type ServiceExpr struct {
	Position   Position
	Name       string
	Annotation *AnnotationExpr
	Routes     []Route
}

func (s *ServiceExpr) Pos() Position { return s.Position }

// ImportExpr represents an import statement
type ImportExpr struct {
	Position Position
	Value    string
	Docs     []string
}

func (i *ImportExpr) Pos() Position { return i.Position }

// ApiSpec represents the complete parsed API file
type ApiSpec struct {
	Position Position
	Filename string
	Syntax   *SyntaxExpr
	Info     *InfoExpr
	Imports  []*ImportExpr
	Types    []*TypeDefine
	Services []*ServiceExpr
}

func (a *ApiSpec) Pos() Position { return a.Position }
