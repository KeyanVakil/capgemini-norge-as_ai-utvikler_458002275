package llm

import "fmt"

type MockClient struct {
	Responses map[string]string
	CallLog   []MockCall
}

type MockCall struct {
	SystemPrompt string
	UserMessage  string
}

func NewMockClient() *MockClient {
	return &MockClient{
		Responses: make(map[string]string),
	}
}

func (m *MockClient) Complete(systemPrompt, userMessage string, onChunk func(text string)) (string, error) {
	m.CallLog = append(m.CallLog, MockCall{SystemPrompt: systemPrompt, UserMessage: userMessage})

	// Check specific keys first (deterministic), then fall back to wildcard
	for key, response := range m.Responses {
		if key != "*" && (contains(systemPrompt, key) || contains(userMessage, key)) {
			if onChunk != nil {
				onChunk(response)
			}
			return response, nil
		}
	}
	if response, ok := m.Responses["*"]; ok {
		if onChunk != nil {
			onChunk(response)
		}
		return response, nil
	}

	return "", fmt.Errorf("no mock response configured for prompt")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
