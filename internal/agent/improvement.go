package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

const improvementSystemPrompt = `You are an improvement advisor AI agent. You receive the original source code AND the results from other review agents (security, style, and test generator). Your job is to synthesize all findings and provide prioritized improvement recommendations.

Your responsibilities:
1. Review the original code with fresh eyes for refactoring opportunities
2. Consider all prior agent findings and identify the most impactful improvements
3. Suggest performance optimizations where applicable
4. Recommend better abstractions, design patterns, or architectural improvements
5. Prioritize recommendations by impact — what changes would improve the code the most?
6. Identify any issues the other agents may have missed

Respond with ONLY a JSON array of findings, ordered by priority (most important first):
[
  {
    "severity": "high|medium|low|info",
    "category": "improvement",
    "title": "Brief title of the recommendation",
    "description": "Detailed explanation of the improvement and why it matters",
    "line_start": 0,
    "line_end": 0,
    "suggestion": "Concrete code example or steps to implement the improvement"
  }
]

Include a final info-level finding that provides an overall code quality summary (1-2 sentences).`

type ImprovementAgent struct {
	baseAgent
}

func NewImprovementAgent(client llm.Client) *ImprovementAgent {
	return &ImprovementAgent{
		baseAgent: baseAgent{name: "improvement", client: client},
	}
}

func (a *ImprovementAgent) Run(code, language string, priorResults []model.AgentResult, onChunk func(text string)) ([]model.Finding, string, error) {
	userMsg := buildImprovementMessage(code, language, priorResults)
	return a.run(improvementSystemPrompt, userMsg, model.CategoryImprovement, onChunk)
}

func buildImprovementMessage(code, language string, priorResults []model.AgentResult) string {
	var sb strings.Builder
	sb.WriteString(buildCodeMessage(code, language))
	sb.WriteString("\n\n--- Prior Agent Results ---\n\n")

	if len(priorResults) == 0 {
		sb.WriteString("No prior agent results available. Provide a standalone review.\n")
		return sb.String()
	}

	for _, result := range priorResults {
		sb.WriteString(fmt.Sprintf("## %s Agent (status: %s)\n", result.AgentName, result.Status))
		if result.Status == model.StatusFailed {
			sb.WriteString(fmt.Sprintf("Failed: %s\n\n", result.Error))
			continue
		}
		if len(result.Findings) > 0 {
			findingsJSON, _ := json.MarshalIndent(result.Findings, "", "  ")
			sb.WriteString(string(findingsJSON))
		} else {
			sb.WriteString("No findings reported.")
		}
		sb.WriteString("\n\n")
	}

	return sb.String()
}
