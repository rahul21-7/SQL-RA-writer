package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"ra-sql-agent/internal/agent"
	"ra-sql-agent/internal/algebra"
	"ra-sql-agent/internal/db"
	"ra-sql-agent/internal/schema"
	"ra-sql-agent/internal/server"

	_ "github.com/glebarez/go-sqlite"
)

type FeedbackEntry struct {
	Timestamp    string `json:"timestamp"`
	Question     string `json:"question"`
	DBID         string `json:"db_id"`
	SchemaInfo   string `json:"schema_info"`
	PredictedRA  string `json:"predicted_ra"`
	GeneratedSQL string `json:"generated_sql"`
	Rating       string `json:"rating"`
	Comment      string `json:"comment"`
}

func saveFeedback(entry FeedbackEntry) error {
	f, err := os.OpenFile("feedback_log.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(entry)
}

func askFeedback(reader *bufio.Reader, question, dbID, schemaInfo, ra, sql string) {
	fmt.Println("\n  Rating: [2=correct / 1=partial / 0=wrong / s=skip]")
	fmt.Print("  > ")
	raw, _ := reader.ReadString('\n')
	rating := strings.TrimSpace(strings.ToLower(raw))
	if rating == "s" || rating == "" {
		return
	}
	fmt.Print("  Comment: ")
	commentRaw, _ := reader.ReadString('\n')
	entry := FeedbackEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		Question:     question,
		DBID:         dbID,
		SchemaInfo:   schemaInfo,
		PredictedRA:  ra,
		GeneratedSQL: sql,
		Rating:       rating,
		Comment:      strings.TrimSpace(commentRaw),
	}
	saveFeedback(entry)
}

func runAgentMode(reader *bufio.Reader) {
	conn, cfg, err := db.Connect()
	if err != nil {
		fmt.Printf("Connection error: %v\n", err)
		return
	}
	defer conn.Close()

	schemaInfo, _ := db.DiscoverSchema(conn, cfg)
	fmt.Println("\n--- Current Schema ---\n" + schemaInfo + "----------------------")

	for {
		fmt.Print("\nQuestion: ")
		question, _ := reader.ReadString('\n')
		question = strings.TrimSpace(question)
		if question == "exit" || question == "" { return }

		RAString, _ := agent.PredictRA(question, "postgres", schemaInfo, "standard")
		parsedRA := algebra.ParseRA(RAString)
		generatedSQL := parsedRA.ToSQL()

		fmt.Printf("\nRA:  %s\nSQL: %s\n", RAString, generatedSQL)

		results, cols, err := db.ExecuteQuery(conn, cfg, generatedSQL)
		if err == nil {
			db.PrintResults(results, cols)
		} else {
			fmt.Printf("Query error: %v\n", err)
		}
		askFeedback(reader, question, "postgres", schemaInfo, RAString, generatedSQL)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "web" {
		server.Start(8080)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("RA-SQL Agent Interface")
	fmt.Println("Commands: agent, web, quit")

	for {
		fmt.Print("\n> ")
		cmdRaw, _ := reader.ReadString('\n')
		cmd := strings.TrimSpace(strings.ToLower(cmdRaw))

		switch cmd {
		case "agent": runAgentMode(reader)
		case "web":   server.Start(8080)
		case "quit":  return
		default:     fmt.Println("Unknown command.")
		}
	}
}