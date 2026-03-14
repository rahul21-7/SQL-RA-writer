package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SpiderQuestion represents an entry in train_spider.json
type SpiderQuestion struct {
	Question string `json:"question"`
	SQL      string `json:"query"`
	DBID     string `json:"db_id"`
}

// SpiderSchema represents the structure of an entry in tables.json
type SpiderSchema struct {
	DBID                string          `json:"db_id"`
	TableNamesOriginal  []string        `json:"table_names_original"`
	ColumnNamesOriginal [][]interface{} `json:"column_names_original"` // [table_index, "column_name"]
}

// LoadMetadata reads the tables.json file and returns the map of all databases
func LoadMetadata(path string) ([]SpiderSchema, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var schemas []SpiderSchema
	if err := json.Unmarshal(file, &schemas); err != nil {
		return nil, err
	}

	return schemas, nil
}

// LoadQuestions reads the training/dev json files (textbook questions)
func LoadQuestions(path string) ([]SpiderQuestion, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var questions []SpiderQuestion
	err = json.Unmarshal(file, &questions)

	return questions, err
}

// GetSchemaForDB is a helper to find one specific database's info
func GetSchemaForDB(dbID string, allSchemas []SpiderSchema) *SpiderSchema {
	for _, s := range allSchemas {
		if s.DBID == dbID {
			return &s
		}
	}
	return nil
}

// FIX: FormatSchema converts a SpiderSchema into a human-readable string for the LLM prompt.
// Previously this didn't exist, so schemaInfo was never built and PredictRA was called
// with the wrong number of arguments.
func FormatSchema(s *SpiderSchema) string {
	if s == nil {
		return "No schema available."
	}

	// Group columns by table index
	tableColumns := make(map[int][]string)
	for _, col := range s.ColumnNamesOriginal {
		if len(col) < 2 {
			continue
		}
		// col[0] is table index (float64 from JSON), col[1] is column name
		idxFloat, ok := col[0].(float64)
		if !ok {
			continue
		}
		idx := int(idxFloat)
		if idx < 0 {
			// idx == -1 means a special "*" column, skip
			continue
		}
		colName, ok := col[1].(string)
		if !ok {
			continue
		}
		tableColumns[idx] = append(tableColumns[idx], colName)
	}

	var sb strings.Builder
	for i, tableName := range s.TableNamesOriginal {
		cols := tableColumns[i]
		sb.WriteString(fmt.Sprintf("Table %s: (%s)\n", tableName, strings.Join(cols, ", ")))
	}
	return sb.String()
}