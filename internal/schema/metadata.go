package schema

import (
	"encoding/json"
	"os"
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