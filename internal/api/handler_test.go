package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keyanvakil/agentic-code-review/internal/model"
)

func TestCreateReview_EmptyCode(t *testing.T) {
	body := `{"code":"","language":"go","agents":["security"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorContains(t, w, "code is required")
}

func TestCreateReview_WhitespaceOnlyCode(t *testing.T) {
	body := `{"code":"   \n  \t  ","language":"go","agents":["security"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateReview_UnsupportedLanguage(t *testing.T) {
	body := `{"code":"print('hi')","language":"rust","agents":["security"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorContains(t, w, "unsupported language")
}

func TestCreateReview_NoAgents(t *testing.T) {
	body := `{"code":"x = 1","language":"python","agents":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorContains(t, w, "at least one agent")
}

func TestCreateReview_InvalidAgent(t *testing.T) {
	body := `{"code":"x = 1","language":"python","agents":["nonexistent"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorContains(t, w, "invalid agent")
}

func TestCreateReview_CodeTooLong(t *testing.T) {
	lines := make([]string, 501)
	for i := range lines {
		lines[i] = "x = 1"
	}
	code := strings.Join(lines, "\n")

	reqBody := model.CreateReviewRequest{
		Code:     code,
		Language: "python",
		Agents:   []string{"security"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
}

func TestCreateReview_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateReview_LanguageCaseInsensitive(t *testing.T) {
	body := `{"code":"x = 1","language":"PYTHON","agents":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.CreateReview(w, req)

	// Should fail on agents validation (empty), not language validation
	assertErrorContains(t, w, "at least one agent")
}

func TestGetReview_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/reviews/not-a-uuid", nil)
	w := httptest.NewRecorder()

	handler := &Handler{}
	handler.GetReview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestQueryInt_Defaults(t *testing.T) {
	tests := []struct {
		query    string
		key      string
		def      int
		expected int
	}{
		{"", "limit", 20, 20},
		{"limit=10", "limit", 20, 10},
		{"limit=abc", "limit", 20, 20},
		{"offset=5", "offset", 0, 5},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
		got := queryInt(req, tt.key, tt.def)
		if got != tt.expected {
			t.Errorf("queryInt(%s, %s, %d) = %d, want %d", tt.query, tt.key, tt.def, got, tt.expected)
		}
	}
}

func TestExtractUUID_Valid(t *testing.T) {
	id, err := extractUUID("/api/reviews/550e8400-e29b-41d4-a716-446655440000", "/api/reviews/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("wrong UUID: %s", id)
	}
}

func TestExtractUUID_WithTrailingPath(t *testing.T) {
	id, err := extractUUID("/api/reviews/550e8400-e29b-41d4-a716-446655440000/stream", "/api/reviews/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("wrong UUID: %s", id)
	}
}

func TestExtractUUID_Invalid(t *testing.T) {
	_, err := extractUUID("/api/reviews/not-valid", "/api/reviews/")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["hello"] != "world" {
		t.Errorf("unexpected body: %v", body)
	}
}

func assertErrorContains(t *testing.T, w *httptest.ResponseRecorder, substr string) {
	t.Helper()
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if !strings.Contains(body["error"], substr) {
		t.Errorf("expected error containing %q, got %q", substr, body["error"])
	}
}
