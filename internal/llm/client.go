package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client interface {
	Complete(systemPrompt, userMessage string, onChunk func(text string)) (string, error)
}

type AnthropicClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicClient{
		apiKey:     apiKey,
		model:      model,
		baseURL:    "https://api.anthropic.com",
		httpClient: &http.Client{},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *AnthropicClient) Complete(systemPrompt, userMessage string, onChunk func(text string)) (string, error) {
	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userMessage},
		},
		Stream: true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return parseSSEStream(resp.Body, onChunk)
}

func parseSSEStream(reader io.Reader, onChunk func(text string)) (string, error) {
	scanner := bufio.NewScanner(reader)
	var fullText strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			fullText.WriteString(event.Delta.Text)
			if onChunk != nil {
				onChunk(event.Delta.Text)
			}
		}
	}

	return fullText.String(), scanner.Err()
}
