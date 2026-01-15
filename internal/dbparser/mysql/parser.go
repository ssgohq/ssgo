package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/ssgohq/ssgo/internal/dbparser"
)

func init() {
	dbparser.RegisterMySQLParser(NewParser)
	dbparser.RegisterMySQLMapper(NewMapper)
}

// Parser implements dbparser.Parser for MySQL
type Parser struct {
	db      *sql.DB
	dbName  string
	verbose bool
}

// NewParser creates a new MySQL parser
func NewParser() dbparser.Parser {
	return &Parser{}
}

// DatabaseType returns the database type
func (p *Parser) DatabaseType() dbparser.DatabaseType {
	return dbparser.DatabaseTypeMySQL
}

// Connect establishes a connection to the database
func (p *Parser) Connect(ctx context.Context, dsn string) error {
	// Extract database name from DSN
	p.dbName = extractDatabaseName(dsn)
	if p.dbName == "" {
		return fmt.Errorf("could not extract database name from DSN")
	}

	// Connect to information_schema
	infoDSN := replaceDatabase(dsn, "information_schema")

	db, err := sql.Open("mysql", infoDSN)
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

	// For MySQL, schemaName is the database name
	if schemaName == "" {
		schemaName = p.dbName
	}

	tables, err := p.ParseTables(ctx, schemaName, nil, opts)
	if err != nil {
		return nil, err
	}

	return &dbparser.Schema{
		Name:     schemaName,
		Tables:   tables,
		Enums:    nil, // MySQL doesn't have separate enum types
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

	if schemaName == "" {
		schemaName = p.dbName
	}

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
	if schemaName == "" {
		schemaName = p.dbName
	}

	query := `
		SELECT TABLE_NAME 
		FROM TABLES 
		WHERE TABLE_SCHEMA = ? 
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
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

// GetEnums returns nil for MySQL (enum is a column type, not a separate type)
func (p *Parser) GetEnums(ctx context.Context, schemaName string) ([]*dbparser.Enum, error) {
	return nil, nil
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

	// Get primary key and indexes
	pk, indexes, err := p.getIndexes(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	table.PrimaryKey = pk
	table.Indexes = indexes

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

	// Get foreign keys
	fks, err := p.getForeignKeys(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	table.ForeignKeys = fks

	return table, nil
}

func (p *Parser) getTableComment(ctx context.Context, schemaName, tableName string) string {
	query := `
		SELECT TABLE_COMMENT 
		FROM TABLES 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
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
			COLUMN_NAME,
			ORDINAL_POSITION,
			DATA_TYPE,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			EXTRA,
			COLUMN_COMMENT,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			DATETIME_PRECISION
		FROM COLUMNS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
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
			extra         string
			charMaxLen    sql.NullInt64
			numPrec       sql.NullInt64
			numScale      sql.NullInt64
			dtPrec        sql.NullInt64
		)

		err := rows.Scan(
			&col.Name,
			&col.OrdinalPosition,
			&col.DataType,
			&col.ColumnType,
			&isNullable,
			&columnDefault,
			&extra,
			&col.Comment,
			&charMaxLen,
			&numPrec,
			&numScale,
			&dtPrec,
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
		}

		// Detect auto_increment from EXTRA
		col.IsAutoIncr = strings.Contains(strings.ToLower(extra), "auto_increment")

		// Detect unsigned from COLUMN_TYPE
		col.IsUnsigned = strings.Contains(strings.ToLower(col.ColumnType), "unsigned")

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

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (p *Parser) getIndexes(
	ctx context.Context,
	schemaName, tableName string,
) (*dbparser.PrimaryKey, []*dbparser.Index, error) {
	query := `
		SELECT 
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			SEQ_IN_INDEX,
			INDEX_TYPE
		FROM STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	rows, err := p.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*dbparser.Index)
	var indexOrder []string

	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique, seqInIndex int

		err := rows.Scan(&indexName, &columnName, &nonUnique, &seqInIndex, &indexType)
		if err != nil {
			return nil, nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &dbparser.Index{
				Name:      indexName,
				IsUnique:  nonUnique == 0,
				IsPrimary: indexName == "PRIMARY",
				Type:      strings.ToLower(indexType),
			}
			indexOrder = append(indexOrder, indexName)
		}

		indexMap[indexName].Columns = append(indexMap[indexName].Columns, columnName)
	}

	// Separate primary key from other indexes
	var pk *dbparser.PrimaryKey
	var indexes []*dbparser.Index

	for _, name := range indexOrder {
		idx := indexMap[name]
		if idx.IsPrimary {
			pk = &dbparser.PrimaryKey{
				Name:    name,
				Columns: idx.Columns,
			}
		} else {
			indexes = append(indexes, idx)
		}
	}

	return pk, indexes, rows.Err()
}

func (p *Parser) getForeignKeys(ctx context.Context, schemaName, tableName string) ([]*dbparser.ForeignKey, error) {
	query := `
		SELECT 
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			rc.DELETE_RULE,
			rc.UPDATE_RULE
		FROM KEY_COLUMN_USAGE kcu
		JOIN REFERENTIAL_CONSTRAINTS rc
			ON kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
			AND kcu.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE kcu.TABLE_SCHEMA = ?
			AND kcu.TABLE_NAME = ?
			AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
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

// Helper functions

func extractDatabaseName(dsn string) string {
	// Parse MySQL DSN: user:pass@tcp(host:port)/dbname?params
	idx := strings.LastIndex(dsn, "/")
	if idx == -1 {
		return ""
	}
	dbPart := dsn[idx+1:]
	if qIdx := strings.Index(dbPart, "?"); qIdx != -1 {
		return dbPart[:qIdx]
	}
	return dbPart
}

func replaceDatabase(dsn, newDB string) string {
	idx := strings.LastIndex(dsn, "/")
	if idx == -1 {
		return dsn
	}

	prefix := dsn[:idx+1]
	suffix := ""

	dbPart := dsn[idx+1:]
	if qIdx := strings.Index(dbPart, "?"); qIdx != -1 {
		suffix = dbPart[qIdx:]
	}

	return prefix + newDB + suffix
}
