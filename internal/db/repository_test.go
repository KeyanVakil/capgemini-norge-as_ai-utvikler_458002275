//go:build integration

package db

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := Connect(dbURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return NewRepository(db)
}

func TestCreateAndGetReview(t *testing.T) {
	repo := setupTestDB(t)

	review, err := repo.CreateReview("func main() {}", "go", []string{"security", "style"})
	if err != nil {
		t.Fatalf("failed to create review: %v", err)
	}
	if review.ID == uuid.Nil {
		t.Error("expected non-nil review ID")
	}
	if review.Status != model.StatusPending {
		t.Errorf("expected pending status, got %s", review.Status)
	}

	got, err := repo.GetReview(review.ID)
	if err != nil {
		t.Fatalf("failed to get review: %v", err)
	}
	if got == nil {
		t.Fatal("expected review, got nil")
	}
	if got.Code != "func main() {}" {
		t.Errorf("expected code 'func main() {}', got %s", got.Code)
	}
	if got.Language != "go" {
		t.Errorf("expected language 'go', got %s", got.Language)
	}
	if len(got.AgentsConfig) != 2 {
		t.Errorf("expected 2 agents, got %d", len(got.AgentsConfig))
	}
}

func TestGetReview_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	got, err := repo.GetReview(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent review")
	}
}

func TestListReviews(t *testing.T) {
	repo := setupTestDB(t)

	repo.CreateReview("code1", "go", []string{"security"})
	repo.CreateReview("code2", "python", []string{"style"})

	reviews, total, err := repo.ListReviews(10, 0)
	if err != nil {
		t.Fatalf("failed to list reviews: %v", err)
	}
	if total < 2 {
		t.Errorf("expected at least 2 total, got %d", total)
	}
	if len(reviews) < 2 {
		t.Errorf("expected at least 2 reviews, got %d", len(reviews))
	}
}

func TestUpdateReviewStatus(t *testing.T) {
	repo := setupTestDB(t)

	review, _ := repo.CreateReview("x = 1", "python", []string{"security"})
	durationMs := 5000
	err := repo.UpdateReviewStatus(review.ID, model.StatusCompleted, &durationMs)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	got, _ := repo.GetReview(review.ID)
	if got.Status != model.StatusCompleted {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if got.DurationMs == nil || *got.DurationMs != 5000 {
		t.Error("expected duration_ms to be 5000")
	}
	if got.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestAgentResultCRUD(t *testing.T) {
	repo := setupTestDB(t)

	review, _ := repo.CreateReview("test code", "go", []string{"security"})

	ar, err := repo.CreateAgentResult(review.ID, "security")
	if err != nil {
		t.Fatalf("failed to create agent result: %v", err)
	}
	if ar.Status != model.StatusPending {
		t.Errorf("expected pending, got %s", ar.Status)
	}

	err = repo.SetAgentStarted(ar.ID)
	if err != nil {
		t.Fatalf("failed to set agent started: %v", err)
	}

	findings := []model.Finding{
		{Severity: model.SeverityHigh, Category: model.CategorySecurity, Title: "Test Finding", Description: "Test"},
	}
	durationMs := 1500
	err = repo.UpdateAgentResult(ar.ID, model.StatusCompleted, findings, "raw output", &durationMs, "")
	if err != nil {
		t.Fatalf("failed to update agent result: %v", err)
	}

	results, err := repo.GetAgentResults(review.ID)
	if err != nil {
		t.Fatalf("failed to get results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != model.StatusCompleted {
		t.Errorf("expected completed, got %s", results[0].Status)
	}
	if len(results[0].Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(results[0].Findings))
	}
	if results[0].Findings[0].Title != "Test Finding" {
		t.Errorf("expected 'Test Finding', got %s", results[0].Findings[0].Title)
	}
}

func TestConcurrentReviews(t *testing.T) {
	repo := setupTestDB(t)

	r1, _ := repo.CreateReview("code1", "go", []string{"security"})
	r2, _ := repo.CreateReview("code2", "python", []string{"style"})

	repo.CreateAgentResult(r1.ID, "security")
	repo.CreateAgentResult(r2.ID, "style")

	results1, _ := repo.GetAgentResults(r1.ID)
	results2, _ := repo.GetAgentResults(r2.ID)

	if len(results1) != 1 || results1[0].AgentName != "security" {
		t.Error("review 1 should have security agent result")
	}
	if len(results2) != 1 || results2[0].AgentName != "style" {
		t.Error("review 2 should have style agent result")
	}
}
