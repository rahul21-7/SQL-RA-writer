package main

import (
	"fmt"
	"log"
	"strings"

	"ra-sql-agent/internal/agent"
	"ra-sql-agent/internal/algebra"
	"ra-sql-agent/internal/db"
	"ra-sql-agent/internal/schema"

	_ "github.com/glebarez/go-sqlite" // Registers driver for internal/db
)

func main() {
	// 1. Load Data
	allSchemas, err := schema.LoadMetadata("./data/tables.json")
	if err != nil {
		log.Fatalf("Error loading metadata: %v", err)
	}
	fmt.Printf("📂 Loaded %d database schemas\n", len(allSchemas))

	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		log.Fatalf("Error loading questions: %v", err)
	}

	// 2. Setup Test Case
	testQ := questions[0]
	fmt.Printf("\n🚀 STARTING AGENT\nQuestion: %s\nTarget DB: %s\n", testQ.Question, testQ.DBID)

	// 3. Get AI Response
	fmt.Println("\nCalling LLM...")
	RAString, err := agent.PredictRA(testQ.Question, testQ.DBID)
	if err != nil {
		log.Fatalf("AI error: %v", err)
	}

	RAString = strings.TrimSpace(RAString)
	fmt.Printf("AI Predicted RA: %s\n", RAString)

	// 4. PARSE & TRANSLATE
	// This uses the algebra.ParseRA function to build the expression tree
	parsedRA := algebra.ParseRA(RAString)

	// Use methods on the parsed object
	raDisplay := parsedRA.String()
	generatedSQL := parsedRA.ToSQL()

	fmt.Println("\n📐 RELATIONAL ALGEBRA (AI LOGIC):")
	fmt.Printf("   %s\n", raDisplay)

	fmt.Println("\n🛠️ GENERATED SQL (DATABASE CODE):")
	fmt.Printf("   %s\n", generatedSQL)

	// 5. Build Database Path
	dbPath := fmt.Sprintf("./data/database/%s/%s.sqlite", testQ.DBID, testQ.DBID)

	// 6. Execute and Print Results
	fmt.Println("💾 Querying Database...")
	
	// ExecuteAndPrint handles the connection and prints the table automatically
	err = db.ExecuteAndPrint(dbPath, generatedSQL)
	if err != nil {
		log.Fatalf("SQL Execution Error: %v", err)
	}

	fmt.Println("\n------------------------------------")
	fmt.Printf("🎯 GOLD SQL WAS:   %s\n", testQ.SQL)
	fmt.Println("------------------------------------")
}