package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config holds database connection settings from config.json.
type Config struct {
	Driver  string `json:"driver"`  // "postgres", "mysql", or "sqlite"
	DSN     string `json:"dsn"`     // connection string
	Dialect string `json:"dialect"` // "postgres", "mysql", or "sqlite"
}

// LoadConfig reads config.json from the project root.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		driver := os.Getenv("DB_DRIVER")
		dsn := os.Getenv("DB_DSN")
		if driver != "" && dsn != "" {
			return &Config{Driver: driver, DSN: dsn, Dialect: driver}, nil
		}
		return nil, fmt.Errorf("no config.json and DB_DRIVER/DB_DSN not set")
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %v", err)
	}
	if cfg.Dialect == "" {
		cfg.Dialect = cfg.Driver
	}
	return &cfg, nil
}

// Connect opens and verifies a database connection.
// Returns the connection, config, and any error.
func Connect() (*sql.DB, *Config, error) {
	cfg, err := LoadConfig("./config.json")
	if err != nil {
		return nil, nil, err
	}
	driverName := cfg.Driver
	if driverName == "postgres" {
		driverName = "pgx"
	}
	conn, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s: %v", cfg.Driver, err)
	}
	if err := conn.Ping(); err != nil {
		return nil, nil, fmt.Errorf("cannot reach %s database: %v", cfg.Driver, err)
	}
	return conn, cfg, nil
}

// OpenDB opens a connection using config.json, falling back to a Spider SQLite file.
func OpenDB(sqlitePath string) (*sql.DB, *Config, error) {
	conn, cfg, err := Connect()
	if err != nil {
		// Fallback to Spider SQLite file
		conn2, err2 := sql.Open("sqlite", sqlitePath)
		if err2 != nil {
			return nil, nil, fmt.Errorf("failed to open SQLite fallback %s: %v", sqlitePath, err2)
		}
		return conn2, &Config{Driver: "sqlite", Dialect: "sqlite"}, nil
	}
	return conn, cfg, nil
}

// DiscoverSchema introspects the connected database and returns a
// human-readable schema string identical in format to the Spider schema strings
// the LLM was trained on. Works for postgres, mysql, and sqlite.
func DiscoverSchema(conn *sql.DB, cfg *Config) (string, error) {
	switch strings.ToLower(cfg.Dialect) {
	case "postgres":
		return discoverPostgres(conn)
	case "mysql":
		return discoverMySQL(conn)
	case "sqlite":
		return discoverSQLite(conn)
	default:
		return "", fmt.Errorf("unknown dialect: %s", cfg.Dialect)
	}
}

func discoverPostgres(conn *sql.DB) (string, error) {
	// Get all tables in the public schema
	tableRows, err := conn.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return "", err
	}
	defer tableRows.Close()

	var tables []string
	for tableRows.Next() {
		var t string
		tableRows.Scan(&t)
		tables = append(tables, t)
	}

	var sb strings.Builder
	for _, table := range tables {
		colRows, err := conn.Query(`
			SELECT column_name FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position`, table)
		if err != nil {
			continue
		}
		var cols []string
		for colRows.Next() {
			var c string
			colRows.Scan(&c)
			cols = append(cols, c)
		}
		colRows.Close()
		sb.WriteString(fmt.Sprintf("Table %s: (%s)\n", table, strings.Join(cols, ", ")))
	}
	if sb.Len() == 0 {
		return "No tables found in public schema. Please seed the database first.", nil
	}
	return sb.String(), nil
}

func discoverMySQL(conn *sql.DB) (string, error) {
	tableRows, err := conn.Query(`SHOW TABLES`)
	if err != nil {
		return "", err
	}
	defer tableRows.Close()

	var tables []string
	for tableRows.Next() {
		var t string
		tableRows.Scan(&t)
		tables = append(tables, t)
	}

	var sb strings.Builder
	for _, table := range tables {
		colRows, err := conn.Query(fmt.Sprintf("SHOW COLUMNS FROM `%s`", table))
		if err != nil {
			continue
		}
		var cols []string
		for colRows.Next() {
			var field, typ, null, key, def, extra sql.NullString
			colRows.Scan(&field, &typ, &null, &key, &def, &extra)
			cols = append(cols, field.String)
		}
		colRows.Close()
		sb.WriteString(fmt.Sprintf("Table %s: (%s)\n", table, strings.Join(cols, ", ")))
	}
	return sb.String(), nil
}

func discoverSQLite(conn *sql.DB) (string, error) {
	tableRows, err := conn.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		return "", err
	}
	defer tableRows.Close()

	var tables []string
	for tableRows.Next() {
		var t string
		tableRows.Scan(&t)
		tables = append(tables, t)
	}

	var sb strings.Builder
	for _, table := range tables {
		colRows, err := conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			continue
		}
		var cols []string
		for colRows.Next() {
			var cid int
			var name, typ string
			var notNull int
			var dflt sql.NullString
			var pk int
			colRows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk)
			cols = append(cols, name)
		}
		colRows.Close()
		sb.WriteString(fmt.Sprintf("Table %s: (%s)\n", table, strings.Join(cols, ", ")))
	}
	return sb.String(), nil
}

// ExecuteQuery runs a SQL query and returns rows as a slice of maps.
func ExecuteQuery(conn *sql.DB, cfg *Config, query string) ([]map[string]interface{}, []string, error) {
	query = NormalizeSQL(query, cfg.Dialect)
	rows, err := conn.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range columns {
			pointers[i] = &columns[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = columns[i]
		}
		results = append(results, row)
	}
	return results, cols, rows.Err()
}

// PrintResults prints query results as a formatted table.
func PrintResults(results []map[string]interface{}, cols []string) {
	if len(results) == 0 {
		fmt.Println("  (no rows returned)")
		return
	}
	fmt.Printf("\n%-20s\n", "--- RESULT ---")
	for _, col := range cols {
		fmt.Printf("%-20s", strings.ToUpper(col))
	}
	fmt.Println("\n" + strings.Repeat("-", len(cols)*20))
	for _, row := range results {
		for _, col := range cols {
			val := row[col]
			if val == nil {
				fmt.Printf("%-20s", "NULL")
			} else {
				fmt.Printf("%-20v", val)
			}
		}
		fmt.Println()
	}
	fmt.Printf("  (%d rows)\n", len(results))
}

// ExecuteAndPrint runs a SQL query and prints results — kept for Spider fallback.
func ExecuteAndPrint(sqlitePath string, query string) error {
	conn, cfg, err := OpenDB(sqlitePath)
	if err != nil {
		return err
	}
	defer conn.Close()
	results, cols, err := ExecuteQuery(conn, cfg, query)
	if err != nil {
		return err
	}
	PrintResults(results, cols)
	return nil
}

func NormalizeSQL(query string, dialect string) string {
	switch strings.ToLower(dialect) {
	case "mysql", "postgres":
		return deduplicateAliases(query)
	default:
		return query
	}
}

func deduplicateAliases(q string) string {
	count := 0
	re := regexp.MustCompile(`(?i)\bAS\s+sub\b`)
	return re.ReplaceAllStringFunc(q, func(_ string) string {
		count++
		if count == 1 {
			return "AS sub"
		}
		return fmt.Sprintf("AS sub%d", count)
	})
}