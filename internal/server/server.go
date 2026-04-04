package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"ra-sql-agent/internal/agent"
	"ra-sql-agent/internal/algebra"
	"ra-sql-agent/internal/db"
	"time"
)

type QueryRequest struct {
	Question string `json:"question"`
	DBID     string `json:"db_id"`
	Model    string `json:"model"`
}

type QueryResponse struct {
	RA  string `json:"ra"`
	SQL string `json:"sql"`
}

type ExecuteRequest struct {
	SQL string `json:"sql"`
}

type ExecuteResponse struct {
	Results []map[string]interface{} `json:"results"`
	Columns []string                 `json:"columns"`
	Error   string                   `json:"error,omitempty"`
}

type StatsResponse struct {
	Total   int            `json:"total"`
	Ratings map[string]int `json:"ratings"`
}

type SchemaResponse struct {
	Schema string `json:"schema"`
}

func Start(port int) {
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	// API Endpoints
	mux.HandleFunc("POST /api/query", handleQuery)
	mux.HandleFunc("POST /api/execute", handleExecute)
	mux.HandleFunc("POST /api/feedback", handleFeedback)
	mux.HandleFunc("POST /api/seed", handleSeed)
	mux.HandleFunc("GET /api/stats", handleStats)
	mux.HandleFunc("GET /api/schema", handleGetSchema)
	mux.HandleFunc("GET /api/benchmarks", handleBenchmarks)

	fmt.Printf("🚀 The SQL Forge is heating up at http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mux))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Load schema information
	// For simplicity, we'll try to discover it from the local DB first
	conn, cfg, err := db.Connect()
	var schemaInfo string
	if err == nil {
		defer conn.Close()
		schemaInfo, _ = db.DiscoverSchema(conn, cfg)
	}

	// Predict RA
	raString, err := agent.PredictRA(req.Question, req.DBID, schemaInfo, req.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI error: %v", err), http.StatusInternalServerError)
		return
	}

	parsedRA := algebra.ParseRA(raString)
	generatedSQL := parsedRA.ToSQL()

	json.NewEncoder(w).Encode(QueryResponse{
		RA:  parsedRA.String(),
		SQL: generatedSQL,
	})
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn, cfg, err := db.Connect()
	if err != nil {
		json.NewEncoder(w).Encode(ExecuteResponse{Error: fmt.Sprintf("DB Connection error: %v", err)})
		return
	}
	defer conn.Close()

	results, cols, err := db.ExecuteQuery(conn, cfg, req.SQL)
	if err != nil {
		json.NewEncoder(w).Encode(ExecuteResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(ExecuteResponse{
		Results: results,
		Columns: cols,
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open("feedback_log.jsonl")
	if err != nil {
		json.NewEncoder(w).Encode(StatsResponse{Total: 0, Ratings: make(map[string]int)})
		return
	}
	defer f.Close()

	ratings := map[string]int{"2": 0, "1": 0, "0": 0}
	total := 0
	dec := json.NewDecoder(f)
	for dec.More() {
		var entry struct {
			Rating string `json:"rating"`
		}
		if err := dec.Decode(&entry); err == nil {
			ratings[entry.Rating]++
			total++
		}
	}

	json.NewEncoder(w).Encode(StatsResponse{
		Total:   total,
		Ratings: ratings,
	})
}

func handleFeedback(w http.ResponseWriter, r *http.Request) {
	var entry struct {
		Timestamp    string `json:"timestamp"`
		Question     string `json:"question"`
		DBID         string `json:"db_id"`
		SchemaInfo   string `json:"schema_info"`
		PredictedRA  string `json:"predicted_ra"`
		GeneratedSQL string `json:"generated_sql"`
		Rating       string `json:"rating"`
		Comment      string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	entry.Timestamp = time.Now().Format(time.RFC3339)

	f, err := os.OpenFile("feedback_log.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, "Could not open log", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		http.Error(w, "Could not save feedback", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleSeed(w http.ResponseWriter, r *http.Request) {
	conn, cfg, err := db.Connect()
	if err != nil {
		http.Error(w, "Could not connect to DB", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	if err := db.SeedSampleData(conn, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Seeding failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Seed successful")
}

func handleBenchmarks(w http.ResponseWriter, r *http.Request) {
	qwenData, _ := os.ReadFile("results_qwen.json")
	llamaData, _ := os.ReadFile("results_llama.json")

	var qwen, llama interface{}
	json.Unmarshal(qwenData, &qwen)
	json.Unmarshal(llamaData, &llama)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"qwen":  qwen,
		"llama": llama,
	})
}

func handleGetSchema(w http.ResponseWriter, r *http.Request) {
	conn, cfg, err := db.Connect()
	if err != nil {
		http.Error(w, "Could not connect to DB", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	schemaInfo, err := db.DiscoverSchema(conn, cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(SchemaResponse{Schema: schemaInfo})
}
