package agent

import (
	"testing"

	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

func TestParseFindings_ValidJSON(t *testing.T) {
	input := `Here are my findings:
[
  {
    "severity": "high",
    "category": "security",
    "title": "SQL Injection",
    "description": "Unsafe query construction",
    "line_start": 10,
    "line_end": 10,
    "suggestion": "Use parameterized queries"
  },
  {
    "severity": "low",
    "category": "security",
    "title": "Missing input validation",
    "description": "User input not validated",
    "suggestion": "Add validation"
  }
]`
	findings := ParseFindings(input, model.CategorySecurity)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].Severity != model.SeverityHigh {
		t.Errorf("expected high severity, got %s", findings[0].Severity)
	}
	if findings[0].Title != "SQL Injection" {
		t.Errorf("expected 'SQL Injection', got %s", findings[0].Title)
	}
	if findings[0].LineStart != 10 {
		t.Errorf("expected line_start 10, got %d", findings[0].LineStart)
	}
}

func TestParseFindings_WrappedJSON(t *testing.T) {
	input := `{"findings": [{"severity": "medium", "title": "Test", "description": "Wrapped format"}]}`
	findings := ParseFindings(input, model.CategoryStyle)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Title != "Test" {
		t.Errorf("expected 'Test', got %s", findings[0].Title)
	}
	if findings[0].Category != model.CategoryStyle {
		t.Errorf("expected style category, got %s", findings[0].Category)
	}
}

func TestParseFindings_PlainText(t *testing.T) {
	input := "The code looks clean with no issues found."
	findings := ParseFindings(input, model.CategorySecurity)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != model.SeverityInfo {
		t.Errorf("expected info severity, got %s", findings[0].Severity)
	}
	if findings[0].Description != input {
		t.Errorf("expected raw text in description")
	}
}

func TestParseFindings_MalformedJSON(t *testing.T) {
	input := `[{"severity": "high", "title": "broken json`
	findings := ParseFindings(input, model.CategorySecurity)
	if len(findings) != 1 {
		t.Fatalf("expected 1 fallback finding, got %d", len(findings))
	}
	if findings[0].Severity != model.SeverityInfo {
		t.Errorf("expected info severity fallback, got %s", findings[0].Severity)
	}
}

func TestParseFindings_TruncatedArraySalvagesCompleteObjects(t *testing.T) {
	input := `[
  {
    "severity": "info",
    "category": "test",
    "title": "Test run_user_command",
    "description": "Covers command execution behavior",
    "suggestion": "def test_run_user_command():\n    assert True"
  },
  {
    "severity": "info",
    "category": "test",
    "title": "Test divide",
    "description": "Covers divide edge cases",
    "suggestion": "def test_divide():\n    assert True"
  },
  {
    "severity": "info",
    "category": "test",
    "title": "Cut off mid-object"`

	findings := ParseFindings(input, model.CategoryTest)
	if len(findings) != 2 {
		t.Fatalf("expected 2 salvaged findings, got %d", len(findings))
	}
	if findings[0].Title != "Test run_user_command" {
		t.Errorf("expected first salvaged finding, got %s", findings[0].Title)
	}
	if findings[1].Title != "Test divide" {
		t.Errorf("expected second salvaged finding, got %s", findings[1].Title)
	}
}

func TestParseFindings_TruncatedWrappedJSONFallsBack(t *testing.T) {
	input := `{"findings":[{"severity":"info","title":"One","description":"ok"},{"severity":"info","title":"Two"`

	findings := ParseFindings(input, model.CategoryTest)
	if len(findings) != 1 {
		t.Fatalf("expected fallback finding, got %d", len(findings))
	}
	if findings[0].Title != "Review Complete" {
		t.Errorf("expected fallback title, got %s", findings[0].Title)
	}
}

func TestParseFindings_DefaultCategory(t *testing.T) {
	input := `[{"severity": "high", "title": "Test", "description": "No category set"}]`
	findings := ParseFindings(input, model.CategoryImprovement)
	if findings[0].Category != model.CategoryImprovement {
		t.Errorf("expected improvement category, got %s", findings[0].Category)
	}
}

func TestParseFindings_DefaultSeverity(t *testing.T) {
	input := `[{"title": "Test", "description": "No severity set", "category": "style"}]`
	findings := ParseFindings(input, model.CategoryStyle)
	if findings[0].Severity != model.SeverityInfo {
		t.Errorf("expected info severity default, got %s", findings[0].Severity)
	}
}

func TestExtractJSON_ArrayInText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple array", `[{"key":"val"}]`, true},
		{"text before", `Here: [{"key":"val"}]`, true},
		{"text after", `[{"key":"val"}] done`, true},
		{"no json", `plain text`, false},
		{"empty", ``, false},
		{"nested braces", `{"findings":[{"a":"b"}]}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if tt.want && result == "" {
				t.Error("expected JSON to be extracted, got empty")
			}
			if !tt.want && result != "" {
				t.Errorf("expected no JSON, got %s", result)
			}
		})
	}
}

func TestSecurityAgent_PromptsCorrectly(t *testing.T) {
	mock := llm.NewMockClient()
	mock.Responses["*"] = `[{"severity":"high","category":"security","title":"Found issue","description":"Test"}]`

	agent := NewSecurityAgent(mock)
	if agent.Name() != "security" {
		t.Errorf("expected agent name 'security', got %s", agent.Name())
	}

	findings, raw, err := agent.Run("func main() {}", "go", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == "" {
		t.Error("expected non-empty raw output")
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Title != "Found issue" {
		t.Errorf("expected 'Found issue', got %s", findings[0].Title)
	}

	if len(mock.CallLog) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.CallLog))
	}
	call := mock.CallLog[0]
	if call.SystemPrompt == "" {
		t.Error("expected non-empty system prompt")
	}
	if call.UserMessage == "" {
		t.Error("expected non-empty user message")
	}
}

func TestStyleAgent_PromptsCorrectly(t *testing.T) {
	mock := llm.NewMockClient()
	mock.Responses["*"] = `[{"severity":"low","category":"style","title":"Naming","description":"Use camelCase"}]`

	agent := NewStyleAgent(mock)
	if agent.Name() != "style" {
		t.Errorf("expected 'style', got %s", agent.Name())
	}

	findings, _, err := agent.Run("var x = 1", "javascript", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestTestGenAgent_PromptsCorrectly(t *testing.T) {
	mock := llm.NewMockClient()
	mock.Responses["*"] = `[{"severity":"info","category":"test","title":"Test for add()","description":"Tests basic addition","suggestion":"def test_add(): assert add(1,2) == 3"}]`

	agent := NewTestGenAgent(mock)
	if agent.Name() != "test_generator" {
		t.Errorf("expected 'test_generator', got %s", agent.Name())
	}

	findings, _, err := agent.Run("def add(a,b): return a+b", "python", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Suggestion == "" {
		t.Error("expected non-empty suggestion with test code")
	}

	call := mock.CallLog[0]
	if !containsStr(call.SystemPrompt, "Return at most 3 findings total") {
		t.Error("expected test generator prompt to cap finding count")
	}
	if !containsStr(call.SystemPrompt, "valid JSON only") {
		t.Error("expected test generator prompt to enforce strict JSON output")
	}
}

func TestImprovementAgent_IncludesPriorResults(t *testing.T) {
	mock := llm.NewMockClient()
	mock.Responses["*"] = `[{"severity":"medium","category":"improvement","title":"Refactor","description":"Extract method"}]`

	agent := NewImprovementAgent(mock)
	if agent.Name() != "improvement" {
		t.Errorf("expected 'improvement', got %s", agent.Name())
	}

	priorResults := []model.AgentResult{
		{
			AgentName: "security",
			Status:    model.StatusCompleted,
			Findings: []model.Finding{
				{Severity: model.SeverityHigh, Title: "SQL Injection"},
			},
		},
	}

	findings, _, err := agent.Run("SELECT * FROM users", "python", priorResults, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	call := mock.CallLog[0]
	if !containsStr(call.UserMessage, "security") {
		t.Error("expected user message to include prior agent results")
	}
	if !containsStr(call.UserMessage, "SQL Injection") {
		t.Error("expected user message to include prior findings")
	}
}

func TestImprovementAgent_HandlesMissingPriorResults(t *testing.T) {
	mock := llm.NewMockClient()
	mock.Responses["*"] = `[{"severity":"info","category":"improvement","title":"Summary","description":"Code looks good"}]`

	agent := NewImprovementAgent(mock)
	_, _, err := agent.Run("print('hello')", "python", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := mock.CallLog[0]
	if !containsStr(call.UserMessage, "No prior agent results") {
		t.Error("expected message about no prior results")
	}
}

func TestAgent_StreamingCallback(t *testing.T) {
	mock := llm.NewMockClient()
	mock.Responses["*"] = `[{"severity":"info","title":"OK","description":"Fine"}]`

	agent := NewSecurityAgent(mock)
	var chunks []string
	_, _, err := agent.Run("x = 1", "python", nil, func(text string) {
		chunks = append(chunks, text)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected streaming callback to be called at least once")
	}
}

func TestAgent_LLMError(t *testing.T) {
	mock := llm.NewMockClient()
	// No responses configured — will return error

	agent := NewSecurityAgent(mock)
	_, _, err := agent.Run("x = 1", "python", nil, nil)
	if err == nil {
		t.Error("expected error when LLM fails")
	}
}

func TestBuildCodeMessage(t *testing.T) {
	msg := buildCodeMessage("fmt.Println()", "go")
	if !containsStr(msg, "go") {
		t.Error("expected language in message")
	}
	if !containsStr(msg, "fmt.Println()") {
		t.Error("expected code in message")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
