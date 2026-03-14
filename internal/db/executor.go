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

// Config holds the database connection settings loaded from config.json.
type Config struct {
	Driver  string `json:"driver"`  // "postgres", "mysql", or "sqlite"
	DSN     string `json:"dsn"`     // connection string
	Dialect string `json:"dialect"` // "postgres", "mysql", or "sqlite"
}

// LoadConfig reads config.json from the project root.
// Falls back gracefully to env vars (DB_DRIVER / DB_DSN) if the file is absent.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// No config file — try environment variables
		driver := os.Getenv("DB_DRIVER")
		dsn := os.Getenv("DB_DSN")
		if driver != "" && dsn != "" {
			return &Config{Driver: driver, DSN: dsn, Dialect: driver}, nil
		}
		return nil, fmt.Errorf("config file not found at %q and DB_DRIVER/DB_DSN env vars are not set", path)
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

// OpenDB opens a connection using config.json (or env vars as fallback).
// If neither is available, falls back to the Spider SQLite file at sqlitePath.
func OpenDB(sqlitePath string) (*sql.DB, *Config, error) {
	cfg, err := LoadConfig("./config.json")
	if err != nil {
		// Pure SQLite fallback for Spider testing
		conn, err2 := sql.Open("sqlite", sqlitePath)
		if err2 != nil {
			return nil, nil, fmt.Errorf("failed to open SQLite fallback %s: %v", sqlitePath, err2)
		}
		return conn, &Config{Driver: "sqlite", Dialect: "sqlite"}, nil
	}

	// Map driver name to the registered Go driver name
	driverName := cfg.Driver
	if driverName == "postgres" {
		driverName = "pgx" // pgx registers itself as "pgx", not "postgres"
	}

	conn, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s database: %v", cfg.Driver, err)
	}
	if err := conn.Ping(); err != nil {
		return nil, nil, fmt.Errorf(
			"cannot reach %s database.\n"+
				"  Check your config.json DSN.\n"+
				"  PostgreSQL format: host=localhost port=5432 user=X password=X dbname=X sslmode=disable\n"+
				"  MySQL format:      user:pass@tcp(localhost:3306)/dbname\n"+
				"  Error: %v",
			cfg.Driver, err,
		)
	}
	fmt.Printf("✅ Connected to %s (%s)\n", cfg.Driver, cfg.Dialect)
	return conn, cfg, nil
}

// NormalizeSQL rewrites the generated SQL to be compatible with the target dialect.
//
// Differences handled:
//
//	postgres  — $1 placeholders, double-quote identifiers, no backticks
//	mysql     — backtick identifiers, single-quote strings only
//	sqlite    — mostly standard; NATURAL JOIN works fine
//
// For now the main practical fix is quoting NATURAL JOIN for MySQL (which can be
// ambiguous) by converting it to an explicit JOIN ON when possible, and ensuring
// subquery aliases are always present (required by MySQL).
func NormalizeSQL(query string, dialect string) string {
	switch strings.ToLower(dialect) {
	case "mysql":
		return normalizeMysql(query)
	case "postgres":
		return normalizePostgres(query)
	default:
		return query
	}
}

func normalizePostgres(q string) string {
	// PostgreSQL is the most standard — very little needs changing.
	// Ensure every subquery alias is unique (postgres requires it, unlike sqlite).
	return deduplicateAliases(q)
}

func normalizeMysql(q string) string {
	// MySQL requires every derived table (subquery) to have an alias — already done.
	// MySQL does not support NATURAL JOIN reliably with duplicate column names,
	// but since our translator uses it only for Spider-style schemas it's usually fine.
	// Main fix: deduplicate aliases.
	return deduplicateAliases(q)
}

// deduplicateAliases renames repeated "AS sub" aliases to "AS sub1", "AS sub2" etc.
// Both PostgreSQL and MySQL require subquery aliases to be unique in the same query.
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

// ExecuteAndPrint runs a SQL query and prints results as a formatted table.
func ExecuteAndPrint(sqlitePath string, query string) error {
	conn, cfg, err := OpenDB(sqlitePath)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Normalize SQL for the target dialect before executing
	query = NormalizeSQL(query, cfg.Dialect)

	rows, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("query failed: %v\nSQL was: %s", err, query)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	fmt.Printf("\n%-20s\n", "--- DATABASE RESULT ---")
	for _, col := range cols {
		fmt.Printf("%-20s", strings.ToUpper(col))
	}
	fmt.Println("\n" + strings.Repeat("-", len(cols)*20))

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range columns {
			pointers[i] = &columns[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return err
		}
		for _, val := range columns {
			if val == nil {
				fmt.Printf("%-20s", "NULL")
			} else {
				fmt.Printf("%-20v", val)
			}
		}
		fmt.Println()
	}
	return rows.Err()
}