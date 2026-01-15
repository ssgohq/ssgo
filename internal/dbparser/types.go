// Package dbparser provides database introspection for model generation.
// It reads schema metadata from PostgreSQL and MySQL databases to generate Go models.
package dbparser

import "time"

// Schema represents the complete database schema
type Schema struct {
	Name     string    // Schema/database name
	Tables   []*Table  // All tables
	Enums    []*Enum   // PostgreSQL enums
	ParsedAt time.Time // When schema was parsed
}

// Table represents a database table
type Table struct {
	Name        string        // Table name (e.g., "users")
	Comment     string        // Table comment
	Columns     []*Column     // All columns (ordered by OrdinalPosition)
	PrimaryKey  *PrimaryKey   // Primary key constraint
	ForeignKeys []*ForeignKey // Foreign key constraints
	Indexes     []*Index      // All indexes (excluding PK)
}

// Column represents a table column with full metadata
type Column struct {
	// Basic info
	Name            string // Column name (e.g., "user_id")
	OrdinalPosition int    // Position in table (1-based)
	Comment         string // Column comment

	// Type info
	DataType   string // Base type: varchar, int4, timestamp
	ColumnType string // Full type: varchar(255), int unsigned
	GoType     string // Mapped Go type (filled by mapper)
	ImportPath string // Import path for GoType (e.g., "time", "github.com/google/uuid")

	// Constraints
	IsNullable   bool // NULL allowed
	IsPrimaryKey bool // Part of primary key
	IsUnique     bool // Has unique constraint
	IsAutoIncr   bool // Auto increment / SERIAL / IDENTITY
	IsUnsigned   bool // MySQL unsigned

	// Default value
	HasDefault bool   // Has default value
	Default    string // Default expression (e.g., "now()", "'active'")

	// PostgreSQL specific
	IsArray  bool   // Array type (e.g., varchar[])
	EnumName string // Enum type name if applicable

	// Numeric precision
	CharMaxLength     *int // varchar(N) - the N
	NumPrecision      *int // numeric(P,S) - the P
	NumScale          *int // numeric(P,S) - the S
	DatetimePrecision *int // timestamp(N) - the N
}

// PrimaryKey represents a primary key constraint
type PrimaryKey struct {
	Name    string   // Constraint name
	Columns []string // Column names (ordered)
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name       string   // Constraint name
	Columns    []string // Local column names
	RefTable   string   // Referenced table name
	RefColumns []string // Referenced column names
	OnDelete   string   // CASCADE, SET NULL, RESTRICT, NO ACTION, SET DEFAULT
	OnUpdate   string   // CASCADE, SET NULL, RESTRICT, NO ACTION, SET DEFAULT
}

// Index represents a database index
type Index struct {
	Name      string   // Index name
	Columns   []string // Column names (ordered)
	IsUnique  bool     // Unique index
	IsPrimary bool     // Primary key index
	Type      string   // btree, hash, gin, gist (PostgreSQL)
	Comment   string   // Index comment
}

// Enum represents a PostgreSQL enum type
type Enum struct {
	Name   string   // Enum type name
	Values []string // Enum values (ordered)
}

// GetColumn returns a column by name, or nil if not found
func (t *Table) GetColumn(name string) *Column {
	for _, col := range t.Columns {
		if col.Name == name {
			return col
		}
	}
	return nil
}

// GetPrimaryKeyColumns returns the primary key columns
func (t *Table) GetPrimaryKeyColumns() []*Column {
	if t.PrimaryKey == nil {
		return nil
	}

	var cols []*Column
	for _, name := range t.PrimaryKey.Columns {
		if col := t.GetColumn(name); col != nil {
			cols = append(cols, col)
		}
	}
	return cols
}

// HasAutoIncrementPK returns true if the primary key is auto-increment
func (t *Table) HasAutoIncrementPK() bool {
	pkCols := t.GetPrimaryKeyColumns()
	if len(pkCols) == 1 {
		return pkCols[0].IsAutoIncr
	}
	return false
}

// GetForeignKeyForColumn returns the foreign key that includes the given column
func (t *Table) GetForeignKeyForColumn(colName string) *ForeignKey {
	for _, fk := range t.ForeignKeys {
		for _, col := range fk.Columns {
			if col == colName {
				return fk
			}
		}
	}
	return nil
}

// IsNullableType returns true if the Go type should be a pointer/nullable type
func (c *Column) IsNullableType() bool {
	return c.IsNullable && !c.IsPrimaryKey
}
