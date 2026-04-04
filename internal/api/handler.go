package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/keyanvakil/agentic-code-review/internal/agent"
	"github.com/keyanvakil/agentic-code-review/internal/db"
	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
	"github.com/keyanvakil/agentic-code-review/internal/orchestrator"
)

const maxCodeLines = 500

type Handler struct {
	repo       *db.Repository
	orch       *orchestrator.Orchestrator
	llmClient  llm.Client
	sseManager *SSEManager
}

func NewHandler(repo *db.Repository, orch *orchestrator.Orchestrator, llmClient llm.Client) *Handler {
	return &Handler{
		repo:       repo,
		orch:       orch,
		llmClient:  llmClient,
		sseManager: NewSSEManager(),
	}
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	var req model.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	lineCount := strings.Count(req.Code, "\n") + 1
	if lineCount > maxCodeLines {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("code exceeds %d lines (got %d)", maxCodeLines, lineCount))
		return
	}

	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if !model.SupportedLanguages[req.Language] {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported language: %s (supported: go, java, csharp, python, javascript)", req.Language))
		return
	}

	if len(req.Agents) == 0 {
		writeError(w, http.StatusBadRequest, "at least one agent must be selected")
		return
	}
	for _, name := range req.Agents {
		if !model.IsValidAgent(name) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid agent: %s", name))
			return
		}
	}

	review, err := h.repo.CreateReview(req.Code, req.Language, req.Agents)
	if err != nil {
		log.Printf("failed to create review: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create review")
		return
	}

	agents := h.buildAgents(req.Agents)
	eventCh := make(chan orchestrator.Event, 100)
	h.sseManager.Register(review.ID, eventCh)

	go func() {
		h.orch.RunReview(review.ID, req.Code, req.Language, agents, eventCh)
		h.sseManager.Unregister(review.ID)
	}()

	resp := model.CreateReviewResponse{
		ID:        review.ID,
		Status:    review.Status,
		CreatedAt: review.CreatedAt,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}
	if offset < 0 {
		offset = 0
	}

	reviews, total, err := h.repo.ListReviews(limit, offset)
	if err != nil {
		log.Printf("failed to list reviews: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list reviews")
		return
	}

	if reviews == nil {
		reviews = []model.ReviewListItem{}
	}

	writeJSON(w, http.StatusOK, model.ReviewListResponse{Reviews: reviews, Total: total})
}

func (h *Handler) GetReview(w http.ResponseWriter, r *http.Request) {
	id, err := extractUUID(r.URL.Path, "/api/reviews/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review ID")
		return
	}

	review, err := h.repo.GetReview(id)
	if err != nil {
		log.Printf("failed to get review: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get review")
		return
	}
	if review == nil {
		writeError(w, http.StatusNotFound, "review not found")
		return
	}

	writeJSON(w, http.StatusOK, review)
}

func (h *Handler) StreamReview(w http.ResponseWriter, r *http.Request) {
	pathAfterReviews := strings.TrimPrefix(r.URL.Path, "/api/reviews/")
	idStr := strings.TrimSuffix(pathAfterReviews, "/stream")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review ID")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	eventCh := h.sseManager.Subscribe(id)
	if eventCh == nil {
		review, err := h.repo.GetReview(id)
		if err != nil || review == nil {
			fmt.Fprintf(w, "event: error\ndata: {\"error\": \"review not found\"}\n\n")
			flusher.Flush()
			return
		}
		writeSSEEvent(w, flusher, "review_complete", map[string]interface{}{
			"review_id":   review.ID.String(),
			"status":      string(review.Status),
			"duration_ms": review.DurationMs,
		})
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, event.Type, event)
		}
	}
}

func (h *Handler) buildAgents(names []string) []agent.Agent {
	var agents []agent.Agent
	for _, name := range names {
		switch name {
		case "security":
			agents = append(agents, agent.NewSecurityAgent(h.llmClient))
		case "style":
			agents = append(agents, agent.NewStyleAgent(h.llmClient))
		case "test_generator":
			agents = append(agents, agent.NewTestGenAgent(h.llmClient))
		case "improvement":
			agents = append(agents, agent.NewImprovementAgent(h.llmClient))
		}
	}
	return agents
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

func extractUUID(path, prefix string) (uuid.UUID, error) {
	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.Split(idStr, "/")[0]
	return uuid.Parse(idStr)
}
