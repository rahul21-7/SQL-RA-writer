package agent

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// PredictRA takes the user question and the DB schema and returns the RA string.
func PredictRA(question string, schemaContext string) (string, error) {
	ctx := context.Background()

	// 1. Setup connection to your LOCAL Python server
	// We use the OpenAI driver because it matches the JSON format we set up in Flask.
	llm, err := openai.New(
		openai.WithBaseURL("http://localhost:8000/v1"), // Matches Flask port 8000
		openai.WithToken("no-token-needed"),            
		openai.WithModel("llama-3.2-ra"),               
	)

	if err != nil {
		return "", fmt.Errorf("failed to connect to local AI server: %v", err)
	}

	// 2. The Prompt Template
	// IMPORTANT: This must match the format the model saw during training!
	promptTemplate := `Below is an instruction that describes a task. Write a response that appropriately completes the request.

### Instruction:
You are a database expert. Convert the following natural language question into Relational Algebra (RA).
Use standard operators: σ (select), π (project), ⨝ (join), γ (aggregate).

### Input:
Schema: %s
Question: %s

### Response (RA):`

	fullPrompt := fmt.Sprintf(promptTemplate, schemaContext, question)

	// 3. Generate the response
	// We use GenerateFromSinglePrompt which sends a POST to /v1/completions
	response, err := llms.GenerateFromSinglePrompt(ctx, llm, fullPrompt,
		llms.WithTemperature(0.1), // Low temperature for mathematical precision
		llms.WithMaxTokens(128),
	)

	if err != nil {
		return "", fmt.Errorf("AI generation failed: %v", err)
	}

	return response, nil
}