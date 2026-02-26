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

	// 1. Setup the connection to your LOCAL fine-tuned model.
	// We use the OpenAI driver because almost all local servers (vLLM/Ollama) 
	// use this API format.

	llm, err := openai.New(
		openai.WithBaseURL("http://localhost:8000/v1"), // Point this to your Python server later
		openai.WithToken("no-token-needed"),            // Local servers usually don't need keys
		openai.WithModel("fine-tuned-ra-model"),        // The name you give your model
	)

	if err != nil{
		return "", fmt.Errorf("failed to connect to local LLM : %v", err)
	}

	// 2. The Fine-Tuned Prompt Template
	// Note: When you fine-tune, you must use THIS EXACT format for the model to work.
	promptTemplate := `### Instruction:
	You are a database expert. Convert the following natural language question into Relational Algebra (RA). 
	Use standard operators: σ (select), π (project), ⨝ (join), γ (aggregate).

	### Schema Context:
	%s

	### Question:
	%s

	### Response (RA):`

	fullPrompt := fmt.Sprintf(promptTemplate, schemaContext, question)

	// 3. Call the model
	response, err := llms.GenerateFromSinglePrompt(ctx, llm, fullPrompt, 
		llms.WithTemperature(0.1), // Keep it low for math-like accuracy
	)
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %v", err)
	}

	return response, nil
}