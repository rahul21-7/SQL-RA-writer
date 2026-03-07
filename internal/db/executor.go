package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

// ExecuteAndPrint runs a SQL query and prints the result as a formatted table.
func ExecuteAndPrint(dbPath string, query string) error{
	db, err := sql.Open("sqlite", dbPath)

	if err != nil{
		return err
	}

	defer db.Close()

	rows, err := db.Query(query)
	if err != nil{
		return err
	}
	defer rows.Close()

	cols, _ := rows.Columns()

	// Print the header
	fmt.Printf("\n%-20s\n", "--- DATABASE RESULT ---")
	for _, col := range cols {
		fmt.Printf("%-20s", strings.ToUpper(col))
	}
	fmt.Println("\n" + strings.Repeat("-", len(cols)*20))

	for rows.Next(){
		columns := make([]interface{}, len(cols))

		columnsPointers := make([]interface{}, len(cols))

		for i := range columns{
			columnsPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnsPointers...);err != nil{
			return err
		}

		for _, val  := range columns{
			if val == nil{
				fmt.Printf("%-20s", "NULL")
			}else{
				fmt.Printf("%-20v", val)
			}
		}

		fmt.Println()
	}

	return nil
}