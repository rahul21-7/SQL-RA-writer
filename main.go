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

// ─── Feedback logging ────────────────────────────────────────────────────────

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
	fmt.Println("\n  Was this answer correct?  [2=correct / 1=partial / 0=wrong / s=skip]")
	fmt.Print("  Rating: ")
	raw, _ := reader.ReadString('\n')
	rating := strings.TrimSpace(strings.ToLower(raw))
	if rating == "s" || rating == "" {
		return
	}
	if rating != "2" && rating != "1" && rating != "0" {
		return
	}
	fmt.Print("  Comment (optional): ")
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
	if err := saveFeedback(entry); err != nil {
		fmt.Printf("  Warning: could not save feedback: %v\n", err)
	} else {
		labels := map[string]string{"2": "✅ correct", "1": "🟡 partial", "0": "❌ wrong"}
		fmt.Printf("  Logged as %s\n", labels[rating])
	}
}

func printFeedbackStats() {
	f, err := os.Open("feedback_log.jsonl")
	if err != nil {
		return
	}
	defer f.Close()
	counts := map[string]int{"2": 0, "1": 0, "0": 0}
	total := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e FeedbackEntry
		if json.Unmarshal(scanner.Bytes(), &e) == nil {
			counts[e.Rating]++
			total++
		}
	}
	if total == 0 {
		return
	}
	fmt.Printf("\n📊 Feedback: %d rated  ✅ %d  🟡 %d  ❌ %d\n",
		total, counts["2"], counts["1"], counts["0"])
	fmt.Println("   Run python/train_rl.py to train on this feedback.")
}

// ─── Agent session ───────────────────────────────────────────────────────────

// AgentSession holds the live database connection and discovered schema
// so we don't reconnect or re-introspect on every question.
type AgentSession struct {
	conn       interface{ Close() error }
	sqlConn    interface{}
	cfg        *db.Config
	dbID       string       // name shown to the LLM (real DB name or "custom")
	schemaInfo string       // formatted schema string
	history    []string     // conversation history for multi-turn context
}

// runAgentMode connects to the real database from config.json,
// discovers its schema, and enters a conversational loop.
func runAgentMode(reader *bufio.Reader) {
	fmt.Println("\n🔌 Connecting to database from config.json...")
	conn, cfg, err := db.Connect()
	if err != nil {
		fmt.Printf("  ⚠️  Could not connect: %v\n", err)
		fmt.Println("  Make sure config.json has the right credentials.")
		fmt.Println("  Falling back to Spider mode (type 'spider' command instead).")
		return
	}
	defer conn.Close()
	fmt.Printf("  ✅ Connected to %s\n", cfg.Driver)

	// Auto-discover the schema
	fmt.Println("  🔍 Discovering schema...")
	schemaInfo, err := db.DiscoverSchema(conn, cfg)
	if err != nil {
		fmt.Printf("  ⚠️  Schema discovery failed: %v\n", err)
		return
	}

	// Extract a DB name from the DSN for the LLM prompt
	dbID := extractDBName(cfg.DSN)

	fmt.Printf("  📋 Schema discovered for '%s'\n", dbID)
	fmt.Println("\n────────────────────────────────────────────────")
	fmt.Print(schemaInfo)
	fmt.Println("────────────────────────────────────────────────")
	fmt.Println("\n  Ask questions about your database.")
	fmt.Println("  Type 'schema' to see the schema again.")
	fmt.Println("  Type 'history' to see this session's questions.")
	fmt.Println("  Type 'back' to return to the main menu.")
	fmt.Println()

	var history []string

	for {
		fmt.Print("  ❓ Question: ")
		questionRaw, _ := reader.ReadString('\n')
		question := strings.TrimSpace(questionRaw)

		switch strings.ToLower(question) {
		case "back", "exit", "quit", "":
			return
		case "schema":
			fmt.Println("\n" + schemaInfo)
			continue
		case "history":
			if len(history) == 0 {
				fmt.Println("  No questions yet this session.")
			}
			for i, q := range history {
				fmt.Printf("  %d. %s\n", i+1, q)
			}
			continue
		}

		history = append(history, question)

		// Build context — include last question for multi-turn awareness
		contextualQuestion := question
		if len(history) > 1 {
			prev := history[len(history)-2]
			contextualQuestion = fmt.Sprintf("[Previous: %s] %s", prev, question)
		}

		// Call LLM
		fmt.Println("\n  🤖 Thinking...")
		RAString, err := agent.PredictRA(contextualQuestion, dbID, schemaInfo, "local-model")
		if err != nil {
			fmt.Printf("  ⚠️  AI server error: %v\n", err)
			fmt.Println("  Is python/server.py running?")
			continue
		}
		RAString = strings.TrimSpace(RAString)

		// Parse RA → SQL
		parsedRA := algebra.ParseRA(RAString)
		generatedSQL := parsedRA.ToSQL()

		fmt.Printf("\n  📐 RA:  %s\n", parsedRA.String())
		fmt.Printf("  🛠️  SQL: %s\n", generatedSQL)

		// Execute against the real database
		results, cols, execErr := db.ExecuteQuery(conn, cfg, generatedSQL)
		if execErr != nil {
			fmt.Printf("\n  ⚠️  Query failed: %v\n", execErr)

			// Auto-retry: strip aliases and try again
			fmt.Println("  ↻ Retrying with simplified query...")
			simplified := strings.ReplaceAll(generatedSQL, " NATURAL JOIN ", " JOIN ")
			results, cols, execErr = db.ExecuteQuery(conn, cfg, simplified)
			if execErr != nil {
				fmt.Printf("  ⚠️  Retry also failed: %v\n", execErr)
				fmt.Println("  Tip: The model may have used wrong table/column names.")
				fmt.Println("       Type 'schema' to check available tables.")
			} else {
				generatedSQL = simplified
				db.PrintResults(results, cols)
			}
		} else {
			db.PrintResults(results, cols)
		}

		// Feedback
		askFeedback(reader, question, dbID, schemaInfo, RAString, generatedSQL)
	}
}

// extractDBName pulls the database name out of a DSN string.
func extractDBName(dsn string) string {
	// Postgres: "host=... dbname=mydb ..."
	for _, part := range strings.Fields(dsn) {
		if strings.HasPrefix(part, "dbname=") {
			return strings.TrimPrefix(part, "dbname=")
		}
	}
	// MySQL: "user:pass@tcp(host)/dbname"
	if idx := strings.LastIndex(dsn, "/"); idx != -1 {
		candidate := dsn[idx+1:]
		if candidate != "" && !strings.Contains(candidate, "=") {
			return candidate
		}
	}
	return "database"
}

// ─── Spider mode (for testing with Spider dataset) ───────────────────────────

func runSpiderQuery(reader *bufio.Reader, allSchemas []schema.SpiderSchema) {
	questions, err := schema.LoadQuestions("./data/train_spider.json")
	if err != nil {
		fmt.Printf("  ⚠️  Could not load Spider questions: %v\n", err)
		return
	}
	fmt.Printf("  Spider has %d questions. Enter index (0-%d): ", len(questions), len(questions)-1)
	var idx int
	fmt.Fscan(os.Stdin, &idx)
	reader.ReadString('\n')
	if idx < 0 || idx >= len(questions) {
		fmt.Println("  Index out of range.")
		return
	}
	q := questions[idx]
	fmt.Printf("\n  Question : %s\n  DB       : %s\n  Gold SQL : %s\n", q.Question, q.DBID, q.SQL)

	dbSchema := schema.GetSchemaForDB(q.DBID, allSchemas)
	schemaInfo := schema.FormatSchema(dbSchema)

	fmt.Println("\n  🤖 Thinking...")
	RAString, err := agent.PredictRA(q.Question, q.DBID, schemaInfo, "local-model")
	if err != nil {
		fmt.Printf("  ⚠️  AI error: %v\n", err)
		return
	}
	RAString = strings.TrimSpace(RAString)
	parsedRA := algebra.ParseRA(RAString)
	generatedSQL := parsedRA.ToSQL()

	fmt.Printf("\n  📐 RA:  %s\n", parsedRA.String())
	fmt.Printf("  🛠️  SQL: %s\n", generatedSQL)

	sqlitePath := fmt.Sprintf("./data/database/%s/%s.sqlite", q.DBID, q.DBID)
	fmt.Println()
	if err := db.ExecuteAndPrint(sqlitePath, generatedSQL); err != nil {
		fmt.Printf("  ⚠️  %v\n", err)
	}
	fmt.Printf("\n  🎯 Gold SQL: %s\n", q.SQL)
	askFeedback(reader, q.Question, q.DBID, schemaInfo, RAString, generatedSQL)
}

// ─── Main ────────────────────────────────────────────────────────────────────

func main() {
	// Check for command line arguments first to support automation (e.g. from launch_forge.bat)
	if len(os.Args) > 1 {
		cmd := strings.TrimSpace(strings.ToLower(os.Args[1]))
		switch cmd {
		case "web":
			server.Start(8080)
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)

	// Load Spider metadata (used in spider mode and list command)
	allSchemas, err := schema.LoadMetadata("./data/tables.json")
	if err != nil {
		log.Fatalf("Error loading metadata: %v", err)
	}

	fmt.Println("════════════════════════════════════════════════")
	fmt.Println("  RA-SQL Agent")
	fmt.Println("  Commands:")
	fmt.Println("    agent   — connect to your database (config.json)")
	fmt.Println("    spider  — test with a Spider dataset question")
	fmt.Println("    web     — start 'The SQL Forge' web interface")
	fmt.Println("    list    — list Spider databases")
	fmt.Println("    stats   — feedback log summary")
	fmt.Println("    quit    — exit")
	fmt.Println("════════════════════════════════════════════════")

	for {
		fmt.Print("\n> ")
		cmdRaw, _ := reader.ReadString('\n')
		cmd := strings.TrimSpace(strings.ToLower(cmdRaw))

		switch cmd {
		case "agent":
			runAgentMode(reader)

		case "spider":
			runSpiderQuery(reader, allSchemas)

		case "web":
			server.Start(8080)

		case "list":
			fmt.Printf("\n  %d Spider databases:\n", len(allSchemas))
			for i, s := range allSchemas {
				fmt.Printf("  %3d. %s\n", i+1, s.DBID)
			}

		case "stats":
			printFeedbackStats()

		case "quit", "exit", "q":
			printFeedbackStats()
			fmt.Println("Bye!")
			return

		default:
			fmt.Println("  Commands: agent, spider, list, stats, quit")
		}
	}
}