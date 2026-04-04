package model

import "testing"

func TestSupportedLanguages(t *testing.T) {
	supported := []string{"go", "java", "csharp", "python", "javascript"}
	for _, lang := range supported {
		if !SupportedLanguages[lang] {
			t.Errorf("expected %s to be supported", lang)
		}
	}

	unsupported := []string{"rust", "ruby", "typescript", ""}
	for _, lang := range unsupported {
		if SupportedLanguages[lang] {
			t.Errorf("expected %s to be unsupported", lang)
		}
	}
}

func TestIsValidAgent(t *testing.T) {
	valid := []string{"security", "style", "test_generator", "improvement"}
	for _, name := range valid {
		if !IsValidAgent(name) {
			t.Errorf("expected %s to be valid", name)
		}
	}

	invalid := []string{"Security", "STYLE", "test", "performance", ""}
	for _, name := range invalid {
		if IsValidAgent(name) {
			t.Errorf("expected %s to be invalid", name)
		}
	}
}

func TestAgentNames(t *testing.T) {
	if len(AgentNames) != 4 {
		t.Errorf("expected 4 agent names, got %d", len(AgentNames))
	}
}

func TestReviewStatusConstants(t *testing.T) {
	statuses := []ReviewStatus{StatusPending, StatusRunning, StatusCompleted, StatusFailed}
	expected := []string{"pending", "running", "completed", "failed"}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], s)
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	severities := []Severity{SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
	expected := []string{"high", "medium", "low", "info"}
	for i, s := range severities {
		if string(s) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], s)
		}
	}
}
