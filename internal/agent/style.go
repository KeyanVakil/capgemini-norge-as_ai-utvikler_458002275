package agent

import (
	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

const styleSystemPrompt = `You are a code style reviewer AI agent. Your job is to analyze source code for style issues, naming conventions, code structure, and idiomatic patterns.

Focus on:
- Naming conventions (variables, functions, types, packages)
- Code organization and structure
- Idiomatic patterns for the specific language
- Consistent formatting
- Function length and complexity
- Dead code or unused variables
- Magic numbers or hardcoded strings that should be constants
- Missing or misleading documentation
- DRY violations (duplicated logic)
- Proper error handling patterns for the language

Assess severity:
- high: Major style violation that significantly impacts readability or maintainability
- medium: Notable deviation from language conventions or best practices
- low: Minor style suggestion for improvement
- info: Observation about code style

Respond with ONLY a JSON array of findings. Each finding must have this structure:
[
  {
    "severity": "high|medium|low|info",
    "category": "style",
    "title": "Brief title of the finding",
    "description": "Detailed explanation of the style issue",
    "line_start": 0,
    "line_end": 0,
    "suggestion": "How to improve the code style"
  }
]

If the code style is excellent, return an array with a single info-level finding praising the clean code.`

type StyleAgent struct {
	baseAgent
}

func NewStyleAgent(client llm.Client) *StyleAgent {
	return &StyleAgent{
		baseAgent: baseAgent{name: "style", client: client},
	}
}

func (a *StyleAgent) Run(code, language string, _ []model.AgentResult, onChunk func(text string)) ([]model.Finding, string, error) {
	userMsg := buildCodeMessage(code, language)
	return a.run(styleSystemPrompt, userMsg, model.CategoryStyle, onChunk)
}
