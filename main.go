package main

import (
	"fmt"
	"log"
	"strings"

	"ra-sql-agent/internal/agent"
	"ra-sql-agent/internal/schema"
)

func main() {
	// 1. Load Data
	allSchemas, err := schema.LoadMetadata("./data/tables.json")
	if err != nil {
		log.Fatalf("Failed to load tables: %v", err)
	}

	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		log.Fatalf("Failed to load questions: %v", err)
	}

	// 2. Select a test case (The "Department Management" example)
	testQ := questions[0]
	dbSchema := schema.GetSchemaForDB(testQ.DBID, allSchemas)

	if dbSchema == nil {
		log.Fatal("DB Schema not found")
	}

	// 3. Build the Schema Context string for the AI
	var sb strings.Builder
	for i, tableName := range dbSchema.TableNamesOriginal {
		sb.WriteString(fmt.Sprintf("Table %s(", tableName) )
		cols := []string{}
		for _, col := range dbSchema.ColumnNamesOriginal {
			// Column format in Spider: [table_index, "column_name"]
			if int(col[0].(float64)) == i {
				cols = append(cols, col[1].(string))
			}
		}
		sb.WriteString(strings.Join(cols, ", ") + "); ")
	}
	schemaContext := sb.String()

	fmt.Println("🚀 --- STARTING AI AGENT ---")
	fmt.Printf("Question: %s\n", testQ.Question)
	fmt.Printf("Context:  %s\n", schemaContext)

	// 4. Call the Python AI Server
	fmt.Println("\n🤖 AI is generating Relational Algebra...")
	prediction, err := agent.PredictRA(testQ.Question, schemaContext)
	if err != nil {
		fmt.Printf("❌ AI Error: %v\n", err)
		fmt.Println("Check: Is python/server.py running on port 8000?")
		return
	}

	// 5. Display Results
	fmt.Println("\n------------------------------------")
	fmt.Printf("🎯 GOLD SQL:  %s\n", testQ.SQL)
	fmt.Printf("📐 AI OUTPUT: %s\n", prediction)
	fmt.Println("------------------------------------")
}