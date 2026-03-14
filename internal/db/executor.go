package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	// SQLite driver (pure-Go, no CGO)
	_ "github.com/glebarez/go-sqlite"

	// Postgres driver — add to go.mod: github.com/lib/pq v1.10.9
	// Uncomment the line below once you run: go get github.com/lib/pq
	// _ "github.com/lib/pq"

	// MySQL driver — add to go.mod: github.com/go-sql-driver/mysql v1.8.1
	// Uncomment the line below once you run: go get github.com/go-sql-driver/mysql
	// _ "github.com/go-sql-driver/mysql"
)

// OpenDB opens a database connection using either:
//  1. The DB_DRIVER + DB_DSN environment variables (for real databases), or
//  2. The provided sqlitePath fallback (for Spider SQLite files).
//
// Set environment variables to connect to a real database:
//
//	Postgres: DB_DRIVER=postgres  DB_DSN="host=localhost user=me password=secret dbname=mydb sslmode=disable"
//	MySQL:    DB_DRIVER=mysql     DB_DSN="me:secret@tcp(127.0.0.1:3306)/mydb"
//	SQLite:   DB_DRIVER=sqlite    DB_DSN="/path/to/file.sqlite"   (or leave unset to use sqlitePath)
func OpenDB(sqlitePath string) (*sql.DB, error) {
	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")

	if driver != "" && dsn != "" {
		// Real database via environment variables
		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s database: %v", driver, err)
		}
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("cannot reach %s database: %v", driver, err)
		}
		fmt.Printf("✅ Connected to %s database\n", driver)
		return db, nil
	}

	// Default: Spider SQLite file
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite file %s: %v", sqlitePath, err)
	}
	return db, nil
}

// ExecuteAndPrint runs a SQL query against the given database path (or env-var DB)
// and prints the results as a formatted table.
func ExecuteAndPrint(sqlitePath string, query string) error {
	db, err := OpenDB(sqlitePath)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("query failed: %v\nSQL was: %s", err, query)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Print header
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