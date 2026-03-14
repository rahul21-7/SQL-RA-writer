package main

import (
	"fmt"
	"log"
	"strings"

	"ra-sql-agent/internal/agent"
	"ra-sql-agent/internal/algebra"
	"ra-sql-agent/internal/db"
	"ra-sql-agent/internal/schema"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	// 1. Load Spider metadata (used for schema info passed to the LLM)
	allSchemas, err := schema.LoadMetadata("./data/tables.json")
	if err != nil {
		log.Fatalf("Error loading metadata: %v", err)
	}
	fmt.Printf("📂 Loaded %d database schemas\n", len(allSchemas))

	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		log.Fatalf("Error loading questions: %v", err)
	}

	// 2. Pick the test question
	testQ := questions[0]
	fmt.Printf("\n🚀 STARTING AGENT\nQuestion: %s\nTarget DB: %s\n", testQ.Question, testQ.DBID)

	// 3. Build schema info string for the LLM prompt
	dbSchema := schema.GetSchemaForDB(testQ.DBID, allSchemas)
	schemaInfo := schema.FormatSchema(dbSchema)

	// 4. Call the LLM
	fmt.Println("\n🤖 Calling LLM...")
	RAString, err := agent.PredictRA(testQ.Question, testQ.DBID, schemaInfo)
	if err != nil {
		log.Fatalf("AI error: %v", err)
	}

	RAString = strings.TrimSpace(RAString)
	fmt.Printf("AI Predicted RA: %s\n", RAString)

	// 5. Parse RA → SQL
	parsedRA := algebra.ParseRA(RAString)
	raDisplay := parsedRA.String()
	generatedSQL := parsedRA.ToSQL()

	fmt.Println("\n📐 RELATIONAL ALGEBRA (AI LOGIC):")
	fmt.Printf("   %s\n", raDisplay)

	fmt.Println("\n🛠️  GENERATED SQL (DATABASE CODE):")
	fmt.Printf("   %s\n", generatedSQL)

	// 6. Execute against the configured database.
	//    If config.json is present, it takes priority.
	//    Otherwise falls back to the Spider SQLite file.
	sqlitePath := fmt.Sprintf("./data/database/%s/%s.sqlite", testQ.DBID, testQ.DBID)

	fmt.Println("\n💾 Querying Database...")
	err = db.ExecuteAndPrint(sqlitePath, generatedSQL)
	if err != nil {
		log.Fatalf("SQL Execution Error: %v", err)
	}

	fmt.Println("\n------------------------------------")
	fmt.Printf("🎯 GOLD SQL WAS: %s\n", testQ.SQL)
	fmt.Println("------------------------------------")
}