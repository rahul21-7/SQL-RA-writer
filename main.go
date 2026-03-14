package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"ra-sql-agent/internal/agent"
	"ra-sql-agent/internal/algebra"
	"ra-sql-agent/internal/db"
	"ra-sql-agent/internal/schema"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	// 1. Load metadata
	allSchemas, err := schema.LoadMetadata("./data/tables.json")
	if err != nil {
		log.Fatalf("Error loading metadata: %v", err)
	}
	fmt.Printf("📂 Loaded %d database schemas\n", len(allSchemas))

	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		log.Fatalf("Error loading questions: %v", err)
	}

	// 2. Pick the test question (index 0 by default)
	testQ := questions[0]
	fmt.Printf("\n🚀 STARTING AGENT\nQuestion: %s\nTarget DB: %s\n", testQ.Question, testQ.DBID)

	// 3. Build the schema info string for the LLM prompt
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

	// 5. Parse & translate RA → SQL
	parsedRA := algebra.ParseRA(RAString)
	raDisplay := parsedRA.String()
	generatedSQL := parsedRA.ToSQL()

	fmt.Println("\n📐 RELATIONAL ALGEBRA (AI LOGIC):")
	fmt.Printf("   %s\n", raDisplay)

	fmt.Println("\n🛠️  GENERATED SQL (DATABASE CODE):")
	fmt.Printf("   %s\n", generatedSQL)

	// 6. Determine database path.
	//    To connect to a real database, set the DB_DRIVER and DB_DSN env vars:
	//      Postgres: export DB_DRIVER=postgres DB_DSN="host=... user=... password=... dbname=... sslmode=disable"
	//      MySQL:    export DB_DRIVER=mysql    DB_DSN="user:pass@tcp(host:3306)/dbname"
	//    When those vars are set, sqlitePath is ignored.
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = fmt.Sprintf("./data/database/%s/%s.sqlite", testQ.DBID, testQ.DBID)
	}

	// 7. Execute and print results
	fmt.Println("\n💾 Querying Database...")
	err = db.ExecuteAndPrint(sqlitePath, generatedSQL)
	if err != nil {
		log.Fatalf("SQL Execution Error: %v\n\nMake sure the database file exists at: %s\n"+
			"Or set DB_DRIVER + DB_DSN environment variables to use a real database.", err, sqlitePath)
	}

	fmt.Println("\n------------------------------------")
	fmt.Printf("🎯 GOLD SQL WAS: %s\n", testQ.SQL)
	fmt.Println("------------------------------------")
}