package model

import (
	"time"

	"github.com/google/uuid"
)

type ReviewStatus string

const (
	StatusPending   ReviewStatus = "pending"
	StatusRunning   ReviewStatus = "running"
	StatusCompleted ReviewStatus = "completed"
	StatusFailed    ReviewStatus = "failed"
)

type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
	SeverityInfo   Severity = "info"
)

type Category string

const (
	CategorySecurity    Category = "security"
	CategoryStyle       Category = "style"
	CategoryTest        Category = "test"
	CategoryImprovement Category = "improvement"
)

var SupportedLanguages = map[string]bool{
	"go":         true,
	"java":       true,
	"csharp":     true,
	"python":     true,
	"javascript": true,
}

var AgentNames = []string{"security", "style", "test_generator", "improvement"}

func IsValidAgent(name string) bool {
	for _, n := range AgentNames {
		if n == name {
			return true
		}
	}
	return false
}

type Finding struct {
	Severity    Severity `json:"severity"`
	Category    Category `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	LineStart   int      `json:"line_start,omitempty"`
	LineEnd     int      `json:"line_end,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
}

type AgentResult struct {
	ID          uuid.UUID    `json:"id"`
	ReviewID    uuid.UUID    `json:"review_id"`
	AgentName   string       `json:"agent_name"`
	Status      ReviewStatus `json:"status"`
	Findings    []Finding    `json:"findings"`
	RawOutput   string       `json:"raw_output,omitempty"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	DurationMs  *int         `json:"duration_ms,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type Review struct {
	ID           uuid.UUID     `json:"id"`
	Code         string        `json:"code"`
	Language     string        `json:"language"`
	Status       ReviewStatus  `json:"status"`
	AgentsConfig []string      `json:"agents_config"`
	CreatedAt    time.Time     `json:"created_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	DurationMs   *int          `json:"duration_ms,omitempty"`
	AgentResults []AgentResult `json:"agent_results,omitempty"`
}

type ReviewListItem struct {
	ID         uuid.UUID    `json:"id"`
	Language   string       `json:"language"`
	Status     ReviewStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	DurationMs *int         `json:"duration_ms,omitempty"`
}

type CreateReviewRequest struct {
	Code     string   `json:"code"`
	Language string   `json:"language"`
	Agents   []string `json:"agents"`
}

type CreateReviewResponse struct {
	ID        uuid.UUID    `json:"id"`
	Status    ReviewStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
}

type ReviewListResponse struct {
	Reviews []ReviewListItem `json:"reviews"`
	Total   int              `json:"total"`
}
