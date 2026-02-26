package main

import (
	"fmt"
	"log"
	"ra-sql-agent/internal/schema"  
	"ra-sql-agent/internal/algebra" // ✅ Fixed: Must use full path from go.mod
)

func main() {
	// 1. Load data
	allSchemas, err := schema.LoadMetadata("./data/tables.json") 
	if err != nil {
		log.Fatal(err)
	}

	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		log.Fatal(err)
	}

	// Pick the first question as a test
	testQ := questions[0]
	fmt.Printf("Question : %s\n", testQ.Question)
	fmt.Printf("Target DB: %s\n", testQ.DBID)

	// 2. Load the specific schema context
	dbSchema := schema.GetSchemaForDB(testQ.DBID, allSchemas)

	if dbSchema != nil {
		fmt.Println("\nAGENT CONTEXT (What the AI sees):")
		for i, tableName := range dbSchema.TableNamesOriginal {
			fmt.Printf("Table [%s] has columns: \n", tableName)
			for _, col := range dbSchema.ColumnNamesOriginal {
				if int(col[0].(float64)) == i {
					fmt.Printf("  - %s\n", col[1])
				}
			}
		}
	}

	// 3. The "Gold" Answer
	fmt.Printf("\n🎯 GOLD SQL: %s\n", testQ.SQL)

	// 4. Test the Algebra Package
	// This creates a nested RA tree: Aggregate -> Selection -> Relation
	expr := algebra.RAExpression{
		Type:    algebra.Aggregate,
		Columns: []string{"COUNT(*)"},
		Input: &algebra.RAExpression{
			Type:      algebra.Selection,
			Condition: "age > 56",
			Input:     &algebra.RAExpression{Type: algebra.Relation, Table: "head"},
		},
	}

	fmt.Println("\n📐 RA Representation:", expr.String())
}