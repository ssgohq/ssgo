package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/ssgohq/ssgo/internal/dbparser"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	dbparser.RegisterPostgresParser(NewParser)
	dbparser.RegisterPostgresMapper(NewMapper)
}

// Parser implements dbparser.Parser for PostgreSQL
type Parser struct {
	db      *sql.DB
	verbose bool
}

// NewParser creates a new PostgreSQL parser
func NewParser() dbparser.Parser {
	return &Parser{}
}

// DatabaseType returns the database type
func (p *Parser) DatabaseType() dbparser.DatabaseType {
	return dbparser.DatabaseTypePostgres
}

// Connect establishes a connection to the database
func (p *Parser) Connect(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	return nil
}

// Close closes the database connection
func (p *Parser) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// ParseSchema parses the entire schema
func (p *Parser) ParseSchema(
	ctx context.Context,
	schemaName string,
	opts dbparser.ParseOptions,
) (*dbparser.Schema, error) {
	p.verbose = opts.Verbose

	tables, err := p.ParseTables(ctx, schemaName, nil, opts)
	if err != nil {
		return nil, err
	}

	enums, err := p.GetEnums(ctx, schemaName)
	if err != nil {
		return nil, err
	}

	return &dbparser.Schema{
		Name:     schemaName,
		Tables:   tables,
		Enums:    enums,
		ParsedAt: time.Now(),
	}, nil
}

// ParseTables parses specific tables
func (p *Parser) ParseTables(
	ctx context.Context,
	schemaName string,
	tableNames []string,
	opts dbparser.ParseOptions,
) ([]*dbparser.Table, error) {
	p.verbose = opts.Verbose

	// If no specific tables, get all
	if len(tableNames) == 0 {
		var err error
		tableNames, err = p.ListTables(ctx, schemaName)
		if err != nil {
			return nil, err
		}
	}

	// Filter tables based on options
	tableNames = dbparser.FilterTables(tableNames, opts)

	var tables []*dbparser.Table
	for _, tableName := range tableNames {
		if p.verbose {
			fmt.Printf("  Parsing table: %s\n", tableName)
		}

		table, err := p.parseTable(ctx, schemaName, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse table %s: %w", tableName, err)
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// ListTables returns all table names in the schema
func (p *Parser) ListTables(ctx context.Context, schemaName string) ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

// GetEnums returns all enum types in the schema
func (p *Parser) GetEnums(ctx context.Context, schemaName string) ([]*dbparser.Enum, error) {
	query := `
		SELECT 
			t.typname AS enum_name,
			array_agg(e.enumlabel ORDER BY e.enumsortorder) AS enum_values
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1
		GROUP BY t.typname
		ORDER BY t.typname
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get enums: %w", err)
	}
	defer rows.Close()

	var enums []*dbparser.Enum
	for rows.Next() {
		enum := &dbparser.Enum{}
		var values pq.StringArray

		if err := rows.Scan(&enum.Name, &values); err != nil {
			return nil, err
		}
		enum.Values = values
		enums = append(enums, enum)
	}

	return enums, rows.Err()
}

// parseTable parses a single table
func (p *Parser) parseTable(ctx context.Context, schemaName, tableName string) (*dbparser.Table, error) {
	table := &dbparser.Table{
		Name: tableName,
	}

	// Get table comment
	table.Comment = p.getTableComment(ctx, schemaName, tableName)

	// Get columns
	columns, err := p.getColumns(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	table.Columns = columns

	// Get primary key
	pk, err := p.getPrimaryKey(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary key: %w", err)
	}
	table.PrimaryKey = pk

	// Mark PK columns
	if pk != nil {
		pkSet := make(map[string]bool)
		for _, col := range pk.Columns {
			pkSet[col] = true
		}
		for _, col := range table.Columns {
			col.IsPrimaryKey = pkSet[col.Name]
		}
	}

	// Get foreign keys
	fks, err := p.getForeignKeys(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	table.ForeignKeys = fks

	// Get indexes
	indexes, err := p.getIndexes(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	table.Indexes = indexes

	// Mark unique columns from single-column unique indexes
	for _, idx := range indexes {
		if idx.IsUnique && len(idx.Columns) == 1 {
			for _, col := range table.Columns {
				if col.Name == idx.Columns[0] {
					col.IsUnique = true
				}
			}
		}
	}

	return table, nil
}

func (p *Parser) getTableComment(ctx context.Context, schemaName, tableName string) string {
	query := `
		SELECT COALESCE(obj_description(
			(quote_ident($1) || '.' || quote_ident($2))::regclass, 
			'pg_class'
		), '')
	`
	var comment string
	// Ignore scan errors - comment is optional
	if err := p.db.QueryRowContext(ctx, query, schemaName, tableName).Scan(&comment); err != nil {
		return ""
	}
	return comment
}

func (p *Parser) getColumns(ctx context.Context, schemaName, tableName string) ([]*dbparser.Column, error) {
	query := `
		SELECT 
			c.column_name,
			c.ordinal_position,
			c.data_type,
			c.udt_name,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			c.datetime_precision,
			COALESCE(c.is_identity, 'NO') as is_identity,
			COALESCE(pgd.description, '') as column_comment
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st
			ON c.table_schema = st.schemaname AND c.table_name = st.relname
		LEFT JOIN pg_catalog.pg_description pgd
			ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*dbparser.Column
	for rows.Next() {
		col := &dbparser.Column{}
		var (
			isNullable    string
			columnDefault sql.NullString
			charMaxLen    sql.NullInt64
			numPrec       sql.NullInt64
			numScale      sql.NullInt64
			dtPrec        sql.NullInt64
			isIdentity    string
			udtName       string
		)

		err := rows.Scan(
			&col.Name,
			&col.OrdinalPosition,
			&col.DataType,
			&udtName,
			&isNullable,
			&columnDefault,
			&charMaxLen,
			&numPrec,
			&numScale,
			&dtPrec,
			&isIdentity,
			&col.Comment,
		)
		if err != nil {
			return nil, err
		}

		// Process nullable
		col.IsNullable = isNullable == "YES"

		// Process default
		if columnDefault.Valid {
			col.HasDefault = true
			col.Default = columnDefault.String
			// Detect auto increment from default (nextval)
			if strings.Contains(col.Default, "nextval(") {
				col.IsAutoIncr = true
			}
		}

		// Process identity
		if isIdentity == "YES" {
			col.IsAutoIncr = true
		}

		// Process lengths
		if charMaxLen.Valid {
			v := int(charMaxLen.Int64)
			col.CharMaxLength = &v
		}
		if numPrec.Valid {
			v := int(numPrec.Int64)
			col.NumPrecision = &v
		}
		if numScale.Valid {
			v := int(numScale.Int64)
			col.NumScale = &v
		}
		if dtPrec.Valid {
			v := int(dtPrec.Int64)
			col.DatetimePrecision = &v
		}

		// Build full column type
		col.ColumnType = p.buildColumnType(col, udtName)

		// Detect array type (PostgreSQL prefixes array types with _)
		if strings.HasPrefix(udtName, "_") {
			col.IsArray = true
			col.DataType = strings.TrimPrefix(udtName, "_")
		}

		// Store enum name if it's a user-defined type
		if col.DataType == "USER-DEFINED" {
			col.EnumName = udtName
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (p *Parser) buildColumnType(col *dbparser.Column, udtName string) string {
	switch col.DataType {
	case "character varying":
		if col.CharMaxLength != nil {
			return fmt.Sprintf("varchar(%d)", *col.CharMaxLength)
		}
		return "varchar"
	case "character":
		if col.CharMaxLength != nil {
			return fmt.Sprintf("char(%d)", *col.CharMaxLength)
		}
		return "char"
	case "numeric":
		if col.NumPrecision != nil && col.NumScale != nil {
			return fmt.Sprintf("numeric(%d,%d)", *col.NumPrecision, *col.NumScale)
		}
		if col.NumPrecision != nil {
			return fmt.Sprintf("numeric(%d)", *col.NumPrecision)
		}
		return "numeric"
	case "ARRAY":
		return udtName + "[]"
	case "USER-DEFINED":
		return udtName
	default:
		return col.DataType
	}
}

func (p *Parser) getPrimaryKey(ctx context.Context, schemaName, tableName string) (*dbparser.PrimaryKey, error) {
	query := `
		SELECT 
			tc.constraint_name,
			kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
		ORDER BY kcu.ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pk *dbparser.PrimaryKey
	for rows.Next() {
		var constraintName, columnName string
		if err := rows.Scan(&constraintName, &columnName); err != nil {
			return nil, err
		}

		if pk == nil {
			pk = &dbparser.PrimaryKey{Name: constraintName}
		}
		pk.Columns = append(pk.Columns, columnName)
	}

	return pk, rows.Err()
}

func (p *Parser) getForeignKeys(ctx context.Context, schemaName, tableName string) ([]*dbparser.ForeignKey, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
		ORDER BY tc.constraint_name, kcu.ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]*dbparser.ForeignKey)
	var fkOrder []string

	for rows.Next() {
		var constraintName, colName, refTable, refCol, onDelete, onUpdate string
		if err := rows.Scan(&constraintName, &colName, &refTable, &refCol, &onDelete, &onUpdate); err != nil {
			return nil, err
		}

		if _, exists := fkMap[constraintName]; !exists {
			fkMap[constraintName] = &dbparser.ForeignKey{
				Name:     constraintName,
				RefTable: refTable,
				OnDelete: onDelete,
				OnUpdate: onUpdate,
			}
			fkOrder = append(fkOrder, constraintName)
		}

		fk := fkMap[constraintName]
		fk.Columns = append(fk.Columns, colName)
		fk.RefColumns = append(fk.RefColumns, refCol)
	}

	// Preserve order
	var fks []*dbparser.ForeignKey
	for _, name := range fkOrder {
		fks = append(fks, fkMap[name])
	}

	return fks, rows.Err()
}

func (p *Parser) getIndexes(ctx context.Context, schemaName, tableName string) ([]*dbparser.Index, error) {
	query := `
		SELECT
			i.relname AS index_name,
			ix.indisunique AS is_unique,
			ix.indisprimary AS is_primary,
			am.amname AS index_type,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS columns
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1
		  AND t.relname = $2
		GROUP BY i.relname, ix.indisunique, ix.indisprimary, am.amname
		ORDER BY i.relname
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*dbparser.Index
	for rows.Next() {
		idx := &dbparser.Index{}
		var columns pq.StringArray

		err := rows.Scan(
			&idx.Name,
			&idx.IsUnique,
			&idx.IsPrimary,
			&idx.Type,
			&columns,
		)
		if err != nil {
			return nil, err
		}

		idx.Columns = columns

		// Skip primary key index (tracked separately)
		if !idx.IsPrimary {
			indexes = append(indexes, idx)
		}
	}

	return indexes, rows.Err()
}
