package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

type Agent interface {
	Name() string
	Run(code, language string, priorResults []model.AgentResult, onChunk func(text string)) ([]model.Finding, string, error)
}

type baseAgent struct {
	name   string
	client llm.Client
}

func (a *baseAgent) Name() string {
	return a.name
}

func (a *baseAgent) run(systemPrompt, userMessage string, category model.Category, onChunk func(text string)) ([]model.Finding, string, error) {
	rawOutput, err := a.client.Complete(systemPrompt, userMessage, onChunk)
	if err != nil {
		return nil, "", fmt.Errorf("agent %s: %w", a.name, err)
	}

	findings := ParseFindings(rawOutput, category)
	return findings, rawOutput, nil
}

func buildCodeMessage(code, language string) string {
	return fmt.Sprintf("Programming Language: %s\n\nCode to review:\n```%s\n%s\n```", language, language, code)
}

func ParseFindings(rawOutput string, defaultCategory model.Category) []model.Finding {
	jsonStr := extractJSON(rawOutput)
	if jsonStr == "" {
		return []model.Finding{{
			Severity:    model.SeverityInfo,
			Category:    defaultCategory,
			Title:       "Review Complete",
			Description: rawOutput,
		}}
	}

	var findings []model.Finding
	if err := json.Unmarshal([]byte(jsonStr), &findings); err != nil {
		var wrapper struct {
			Findings []model.Finding `json:"findings"`
		}
		if err2 := json.Unmarshal([]byte(jsonStr), &wrapper); err2 == nil {
			findings = wrapper.Findings
		} else {
			findings = extractPartialFindings(rawOutput)
			if len(findings) == 0 {
				return []model.Finding{{
					Severity:    model.SeverityInfo,
					Category:    defaultCategory,
					Title:       "Review Complete",
					Description: rawOutput,
				}}
			}
		}
	}

	for i := range findings {
		if findings[i].Category == "" {
			findings[i].Category = defaultCategory
		}
		if findings[i].Severity == "" {
			findings[i].Severity = model.SeverityInfo
		}
	}

	return findings
}

func extractJSON(text string) string {
	start := strings.Index(text, "[")
	braceStart := strings.Index(text, "{")

	if start == -1 && braceStart == -1 {
		return ""
	}

	if start == -1 || (braceStart != -1 && braceStart < start) {
		return extractBracketed(text, braceStart, '{', '}')
	}
	return extractBracketed(text, start, '[', ']')
}

func extractBracketed(text string, start int, open, close byte) string {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == open {
			depth++
		} else if ch == close {
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

func extractPartialFindings(text string) []model.Finding {
	var findings []model.Finding

	start := strings.Index(text, "[")
	if start == -1 {
		start = strings.Index(text, "{")
	}
	if start == -1 {
		return nil
	}

	depth := 0
	inString := false
	escaped := false
	objStart := -1

	for i := start; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if inString && ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '{':
			if depth == 1 && objStart == -1 {
				objStart = i
			}
			depth++
		case '}':
			depth--
			if depth == 1 && objStart != -1 {
				var finding model.Finding
				candidate := text[objStart : i+1]
				if err := json.Unmarshal([]byte(candidate), &finding); err == nil {
					findings = append(findings, finding)
				}
				objStart = -1
			}
		}
	}

	return findings
}
