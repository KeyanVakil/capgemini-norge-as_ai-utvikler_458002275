package llm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseSSEStream_TextDeltas(t *testing.T) {
	sseData := `data: {"type":"content_block_start","index":0}

data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}

data: {"type":"content_block_delta","delta":{"type":"text_delta","text":" World"}}

data: {"type":"message_stop"}

`
	reader := strings.NewReader(sseData)
	var chunks []string
	result, err := parseSSEStream(reader, func(text string) {
		chunks = append(chunks, text)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", result)
	}
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestParseSSEStream_SkipsNonData(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start"}

event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"OK"}}

data: [DONE]
`
	reader := strings.NewReader(sseData)
	result, err := parseSSEStream(reader, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "OK" {
		t.Errorf("expected 'OK', got %q", result)
	}
}

func TestParseSSEStream_EmptyStream(t *testing.T) {
	reader := strings.NewReader("")
	result, err := parseSSEStream(reader, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestAnthropicClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid api key"}`)
	}))
	defer server.Close()

	client := &AnthropicClient{
		apiKey:     "bad-key",
		model:      "claude-sonnet-4-20250514",
		baseURL:    server.URL,
		httpClient: &http.Client{},
	}

	_, err := client.Complete("system", "user msg", nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention 401, got: %v", err)
	}
}

func TestAnthropicClient_StreamingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("expected anthropic-version header")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"result"}}

data: {"type":"message_stop"}

`)
	}))
	defer server.Close()

	client := &AnthropicClient{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-20250514",
		baseURL:    server.URL,
		httpClient: &http.Client{},
	}

	var chunks []string
	result, err := client.Complete("system", "user msg", func(text string) {
		chunks = append(chunks, text)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("expected 'result', got %q", result)
	}
	if len(chunks) != 1 || chunks[0] != "result" {
		t.Errorf("expected 1 chunk 'result', got %v", chunks)
	}
}

func TestMockClient_MatchesKey(t *testing.T) {
	mock := NewMockClient()
	mock.Responses["security"] = "security response"
	mock.Responses["style"] = "style response"

	result, err := mock.Complete("You are a security auditor", "review this", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "security response" {
		t.Errorf("expected 'security response', got %q", result)
	}
}

func TestMockClient_WildcardMatch(t *testing.T) {
	mock := NewMockClient()
	mock.Responses["*"] = "default response"

	result, err := mock.Complete("anything", "whatever", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "default response" {
		t.Errorf("expected 'default response', got %q", result)
	}
}

func TestMockClient_NoMatch(t *testing.T) {
	mock := NewMockClient()
	_, err := mock.Complete("no match", "here", nil)
	if err == nil {
		t.Error("expected error when no response configured")
	}
}

func TestMockClient_CallLog(t *testing.T) {
	mock := NewMockClient()
	mock.Responses["*"] = "ok"

	mock.Complete("sys1", "user1", nil)
	mock.Complete("sys2", "user2", nil)

	if len(mock.CallLog) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.CallLog))
	}
	if mock.CallLog[0].SystemPrompt != "sys1" {
		t.Errorf("expected 'sys1', got %q", mock.CallLog[0].SystemPrompt)
	}
	if mock.CallLog[1].UserMessage != "user2" {
		t.Errorf("expected 'user2', got %q", mock.CallLog[1].UserMessage)
	}
}

func TestNewAnthropicClient_DefaultModel(t *testing.T) {
	client := NewAnthropicClient("key", "")
	if client.model != "claude-sonnet-4-20250514" {
		t.Errorf("expected default model, got %s", client.model)
	}
}

func TestNewAnthropicClient_CustomModel(t *testing.T) {
	client := NewAnthropicClient("key", "claude-opus-4-20250514")
	if client.model != "claude-opus-4-20250514" {
		t.Errorf("expected custom model, got %s", client.model)
	}
}
