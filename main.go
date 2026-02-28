package main

import (
	"database/sql"
	"fmt"
	"log"

	"ra-sql-agent/internal/algebra"
	"ra-sql-agent/internal/schema"

	_ "github.com/glebarez/go-sqlite" // NO CGO REQUIRED
)

func main() {
	// 1. Load Data
	// 1. Load Data
	allSchemas, err := schema.LoadMetadata("./data/tables.json")
	if err != nil {
		log.Fatalf("Error loading metadata: %v", err)
	}
	fmt.Printf("📂 Loaded %d database schemas\n", len(allSchemas)) // Now it's used!

	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		log.Fatalf("Error loading questions: %v", err)
	}

	// 2. Setup Test Case
	testQ := questions[0]
	fmt.Printf("🚀 STARTING AGENT\nQuestion: %s\nTarget DB: %s\n", testQ.Question, testQ.DBID)

	// 3. MOCK RA (The goal for your next fine-tuning)
	mockRA := &algebra.RAExpression{
		Type:    algebra.Aggregate,
		Columns: []string{"COUNT(*)"},
		Input: &algebra.RAExpression{
			Type:      algebra.Selection,
			Condition: "age > 56",
			Input:     &algebra.RAExpression{Type: algebra.Relation, Table: "head"},
		},
	}

	// 4. Translate to SQL using your new translator
	generatedSQL := mockRA.ToSQL()
	fmt.Printf("\n🛠️ Generated SQL: %s\n", generatedSQL)

	// 5. Connect to Database (Pure Go Driver)
	dbPath := fmt.Sprintf("./data/database/%s/%s.sqlite", testQ.DBID, testQ.DBID)

	// Use "sqlite" here, NOT "sqlite3"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// 6. Execute and Print Results
	fmt.Println("💾 Querying Database...")
	var count int
	err = db.QueryRow(generatedSQL).Scan(&count)
	if err != nil {
		log.Fatalf("SQL Execution Error: %v\nCheck: Does the file exist at %s?", err, dbPath)
	}

	fmt.Println("\n------------------------------------")
	fmt.Printf("✅ DATABASE RESULT: %d\n", count)
	fmt.Printf("🎯 GOLD SQL WAS:   %s\n", testQ.SQL)
	fmt.Println("------------------------------------")
}
