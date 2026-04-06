package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type completionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Stop        string  `json:"stop,omitempty"`
}

type completionResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// PredictRA communicates with the local AI server to translate natural language into Relational Algebra.
func PredictRA(question, dbID, schemaInfo, mode string) (string, error) {
	// Specialized instructions for different forge intensities
	instruction := "Convert the natural language question to Relational Algebra."
	if mode == "frozen" {
		instruction = "[CORE_STABILIZED] Convert the question to high-precision Relational Algebra logic."
	}

	promptTemplate := `### Instruction:
%s
IMPORTANT: ONLY use tables and columns listed in the Schema. Do not hallucinate.

### Input:
Question: %s
Database: %s
Schema: %s

### Response:
RA: `

	fullPrompt := fmt.Sprintf(promptTemplate, instruction, question, dbID, schemaInfo)

	reqBody := completionRequest{
		Model:       "ra-sql-model",
		Prompt:      fullPrompt,
		MaxTokens:   128,
		Temperature: 0.1,
		Stop:        "\n",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:8000/v1/completions", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("AI server unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI server error (Status %d)", resp.StatusCode)
	}

	var completionResp completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil || len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("invalid AI response")
	}

	raw := completionResp.Choices[0].Text
	
	// Post-processing to extract valid RA logic
	clean := strings.ReplaceAll(raw, "\r\n", "\n")
	clean = strings.TrimPrefix(strings.TrimSpace(clean), "RA:")
	clean = strings.TrimSpace(clean)

	lines := strings.Split(clean, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.ContainsAny(trimmed, "γσπ⨝") {
			return trimmed, nil
		}
	}

	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return clean, nil
}