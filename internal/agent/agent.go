package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// completionRequest matches the OpenAI /v1/completions request body.
type completionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Stop        string  `json:"stop,omitempty"`
}

// completionResponse matches the OpenAI /v1/completions response shape.
type completionResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// PredictRA calls the local LLM server and returns a Relational Algebra string.
func PredictRA(question string, dbID string, schemaInfo string, model string) (string, error) {
	// EXACT MATCH to prompt_style in train.py — newlines and headers must be identical.
	promptTemplate := `### Instruction:
Convert the natural language question to Relational Algebra.
IMPORTANT: ONLY use tables and columns listed in the Schema below. If no schema is available, do not hallucinate tables; instead, explain that the database is empty.

### Input:
Question: %s
Database: %s
Schema: %s

### Response:
RA: `

	fullPrompt := fmt.Sprintf(promptTemplate, question, dbID, schemaInfo)

	reqBody := completionRequest{
		Model:       model,
		Prompt:      fullPrompt,
		MaxTokens:   128,
		Temperature: 0.0,
		Stop:        "\n",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(
		"http://localhost:8000/v1/completions",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return "", fmt.Errorf("failed to reach local AI server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var completionResp completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("server returned no choices")
	}

	raw := completionResp.Choices[0].Text

	// --- CLEANUP ---

	// 1. Standardize line endings and trim whitespace
	clean := strings.ReplaceAll(raw, "\r\n", "\n")
	clean = strings.TrimSpace(clean)

	// 2. Strip the "RA:" label if the model echoed it
	clean = strings.TrimPrefix(clean, "RA:")
	clean = strings.TrimSpace(clean)

	// 3. Find the line that contains actual RA operators (γ, σ, π, ⨝)
	lines := strings.Split(clean, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.ContainsAny(trimmed, "γσπ⨝") {
			return trimmed, nil
		}
	}

	// 4. Fallback: return the first non-empty line (e.g. plain table name)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed, nil
		}
	}

	return clean, nil
}