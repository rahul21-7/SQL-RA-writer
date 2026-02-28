package agent

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func PredictRA(question string, dbID string) (string, error) {
	ctx := context.Background()

	llm, err := openai.New(
		openai.WithBaseURL("http://localhost:8000/v1"),
		openai.WithToken("no-token-needed"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect to local AI server: %v", err)
	}

	// This template MUST match your train_ra.json "instruction" and "input" keys
	promptTemplate := `Below is an instruction that describes a task. Write a response that appropriately completes the request.

### Instruction:
Convert the SQL query to Relational Algebra.

### Input:
Question: %s
Database: %s

### Response (RA):`

	fullPrompt := fmt.Sprintf(promptTemplate, question, dbID)

	response, err := llms.GenerateFromSinglePrompt(ctx, llm, fullPrompt,
		llms.WithTemperature(0.1), 
		llms.WithMaxTokens(128),
	)

	if err != nil {
		return "", fmt.Errorf("AI generation failed: %v", err)
	}

	return response, nil
}