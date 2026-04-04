package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateReview(code, language string, agents []string) (*model.Review, error) {
	agentsJSON, err := json.Marshal(agents)
	if err != nil {
		return nil, fmt.Errorf("marshal agents config: %w", err)
	}

	review := &model.Review{
		ID:           uuid.New(),
		Code:         code,
		Language:     language,
		Status:       model.StatusPending,
		AgentsConfig: agents,
		CreatedAt:    time.Now().UTC(),
	}

	_, err = r.db.Exec(
		`INSERT INTO reviews (id, code, language, status, agents_config, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		review.ID, review.Code, review.Language, review.Status, agentsJSON, review.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert review: %w", err)
	}
	return review, nil
}

func (r *Repository) GetReview(id uuid.UUID) (*model.Review, error) {
	review := &model.Review{}
	var agentsJSON []byte
	err := r.db.QueryRow(
		`SELECT id, code, language, status, agents_config, created_at, completed_at, duration_ms
		 FROM reviews WHERE id = $1`, id,
	).Scan(
		&review.ID, &review.Code, &review.Language, &review.Status,
		&agentsJSON, &review.CreatedAt, &review.CompletedAt, &review.DurationMs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}
	if err := json.Unmarshal(agentsJSON, &review.AgentsConfig); err != nil {
		return nil, fmt.Errorf("unmarshal agents config: %w", err)
	}

	results, err := r.GetAgentResults(id)
	if err != nil {
		return nil, err
	}
	review.AgentResults = results
	return review, nil
}

func (r *Repository) ListReviews(limit, offset int) ([]model.ReviewListItem, int, error) {
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	rows, err := r.db.Query(
		`SELECT id, language, status, created_at, duration_ms
		 FROM reviews ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	var items []model.ReviewListItem
	for rows.Next() {
		var item model.ReviewListItem
		if err := rows.Scan(&item.ID, &item.Language, &item.Status, &item.CreatedAt, &item.DurationMs); err != nil {
			return nil, 0, fmt.Errorf("scan review: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *Repository) UpdateReviewStatus(id uuid.UUID, status model.ReviewStatus, durationMs *int) error {
	var completedAt *time.Time
	if status == model.StatusCompleted || status == model.StatusFailed {
		now := time.Now().UTC()
		completedAt = &now
	}
	_, err := r.db.Exec(
		`UPDATE reviews SET status = $1, completed_at = $2, duration_ms = $3 WHERE id = $4`,
		status, completedAt, durationMs, id,
	)
	if err != nil {
		return fmt.Errorf("update review status: %w", err)
	}
	return nil
}

func (r *Repository) CreateAgentResult(reviewID uuid.UUID, agentName string) (*model.AgentResult, error) {
	result := &model.AgentResult{
		ID:       uuid.New(),
		ReviewID: reviewID,
		AgentName: agentName,
		Status:   model.StatusPending,
		Findings: []model.Finding{},
	}
	_, err := r.db.Exec(
		`INSERT INTO agent_results (id, review_id, agent_name, status, findings)
		 VALUES ($1, $2, $3, $4, $5)`,
		result.ID, result.ReviewID, result.AgentName, result.Status, "[]",
	)
	if err != nil {
		return nil, fmt.Errorf("insert agent result: %w", err)
	}
	return result, nil
}

func (r *Repository) UpdateAgentResult(id uuid.UUID, status model.ReviewStatus, findings []model.Finding, rawOutput string, durationMs *int, errMsg string) error {
	findingsJSON, err := json.Marshal(findings)
	if err != nil {
		return fmt.Errorf("marshal findings: %w", err)
	}

	now := time.Now().UTC()
	var startedAt, completedAt *time.Time
	if status == model.StatusRunning {
		startedAt = &now
	}
	if status == model.StatusCompleted || status == model.StatusFailed {
		completedAt = &now
	}

	query := `UPDATE agent_results SET status = $1, findings = $2, raw_output = $3, duration_ms = $4, error = $5`
	args := []interface{}{status, findingsJSON, rawOutput, durationMs, sql.NullString{String: errMsg, Valid: errMsg != ""}}
	argIdx := 6

	if startedAt != nil {
		query += fmt.Sprintf(`, started_at = $%d`, argIdx)
		args = append(args, startedAt)
		argIdx++
	}
	if completedAt != nil {
		query += fmt.Sprintf(`, completed_at = $%d`, argIdx)
		args = append(args, completedAt)
		argIdx++
	}

	query += fmt.Sprintf(` WHERE id = $%d`, argIdx)
	args = append(args, id)

	if _, err := r.db.Exec(query, args...); err != nil {
		return fmt.Errorf("update agent result: %w", err)
	}
	return nil
}

func (r *Repository) SetAgentStarted(id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(
		`UPDATE agent_results SET status = $1, started_at = $2 WHERE id = $3`,
		model.StatusRunning, now, id,
	)
	return err
}

func (r *Repository) GetAgentResults(reviewID uuid.UUID) ([]model.AgentResult, error) {
	rows, err := r.db.Query(
		`SELECT id, review_id, agent_name, status, findings, raw_output, started_at, completed_at, duration_ms, error
		 FROM agent_results WHERE review_id = $1 ORDER BY agent_name`, reviewID,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent results: %w", err)
	}
	defer rows.Close()

	var results []model.AgentResult
	for rows.Next() {
		var ar model.AgentResult
		var findingsJSON []byte
		var rawOutput, errMsg sql.NullString
		if err := rows.Scan(
			&ar.ID, &ar.ReviewID, &ar.AgentName, &ar.Status,
			&findingsJSON, &rawOutput, &ar.StartedAt, &ar.CompletedAt, &ar.DurationMs, &errMsg,
		); err != nil {
			return nil, fmt.Errorf("scan agent result: %w", err)
		}
		if rawOutput.Valid {
			ar.RawOutput = rawOutput.String
		}
		if errMsg.Valid {
			ar.Error = errMsg.String
		}
		if err := json.Unmarshal(findingsJSON, &ar.Findings); err != nil {
			ar.Findings = []model.Finding{}
		}
		results = append(results, ar)
	}
	return results, nil
}
