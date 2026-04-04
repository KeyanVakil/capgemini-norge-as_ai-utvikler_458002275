package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/keyanvakil/agentic-code-review/internal/api"
	"github.com/keyanvakil/agentic-code-review/internal/db"
	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/orchestrator"
)

func main() {
	databaseURL := getEnv("DATABASE_URL", "postgres://reviewer:reviewer@localhost:5432/reviews?sslmode=disable")
	apiKey := getEnv("ANTHROPIC_API_KEY", "")
	port := getEnv("PORT", "8080")

	if apiKey == "" {
		log.Println("WARNING: ANTHROPIC_API_KEY is not set. Reviews will fail until a valid API key is configured.")
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repo := db.NewRepository(database)
	orch := orchestrator.New(repo)
	llmClient := llm.NewAnthropicClient(apiKey, getEnv("LLM_MODEL", ""))

	staticDir := findStaticDir()
	handler := api.NewHandler(repo, orch, llmClient)
	router := api.NewRouter(handler, staticDir)

	log.Printf("server starting on :%s", port)
	log.Printf("open http://localhost:%s in your browser", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func findStaticDir() string {
	candidates := []string{
		"web/static",
		"../../web/static",
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	candidates = append(candidates, filepath.Join(dir, "../../web/static"))

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			log.Printf("serving static files from %s", abs)
			return abs
		}
	}

	log.Println("WARNING: static directory not found, using 'web/static'")
	return "web/static"
}
