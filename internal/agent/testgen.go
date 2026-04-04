package agent

import (
	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

const testGenSystemPrompt = `You are a test generator AI agent. Your job is to produce comprehensive, runnable unit tests for the submitted source code.

Guidelines:
- Generate idiomatic tests for the specific language:
  - Go: use the testing package with table-driven tests
  - Java: use JUnit 5 with descriptive test names
  - C#: use xUnit with Theory/InlineData for parameterized tests
  - Python: use pytest with descriptive function names
  - JavaScript: use Jest with describe/it blocks
- Cover happy paths, edge cases, and error scenarios
- Each test should have a clear purpose described in its name
- Use meaningful test data, not trivial examples
- Include both positive and negative test cases
- Test boundary conditions where applicable

Respond with ONLY a JSON array of findings. Each finding should contain a test:
[
  {
    "severity": "info",
    "category": "test",
    "title": "Brief description of what this test covers",
    "description": "Explanation of the test strategy and what scenarios are covered",
    "suggestion": "The complete, runnable test code"
  }
]

Put the full test source code in the "suggestion" field. Group related tests into a single finding when they test the same function.`

type TestGenAgent struct {
	baseAgent
}

func NewTestGenAgent(client llm.Client) *TestGenAgent {
	return &TestGenAgent{
		baseAgent: baseAgent{name: "test_generator", client: client},
	}
}

func (a *TestGenAgent) Run(code, language string, _ []model.AgentResult, onChunk func(text string)) ([]model.Finding, string, error) {
	userMsg := buildCodeMessage(code, language)
	return a.run(testGenSystemPrompt, userMsg, model.CategoryTest, onChunk)
}
