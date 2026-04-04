package agent

import (
	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

const securitySystemPrompt = `You are a security auditor AI agent. Your job is to analyze source code for security vulnerabilities and potential risks.

Focus on:
- Injection vulnerabilities (SQL injection, command injection, XSS)
- Hardcoded secrets, API keys, or credentials
- Insecure cryptographic practices
- Buffer overflows or memory safety issues
- Authentication and authorization flaws
- Insecure deserialization
- Path traversal vulnerabilities
- Race conditions that could be exploited
- Unvalidated input at system boundaries

For each finding, assess the severity:
- high: Exploitable vulnerability that could lead to data breach or system compromise
- medium: Security weakness that could be exploited under certain conditions
- low: Minor security concern or best practice violation
- info: Informational observation about security posture

Respond with ONLY a JSON array of findings. Each finding must have this structure:
[
  {
    "severity": "high|medium|low|info",
    "category": "security",
    "title": "Brief title of the finding",
    "description": "Detailed explanation of the vulnerability",
    "line_start": 0,
    "line_end": 0,
    "suggestion": "How to fix the issue"
  }
]

If no security issues are found, return an array with a single info-level finding confirming the code appears secure.`

type SecurityAgent struct {
	baseAgent
}

func NewSecurityAgent(client llm.Client) *SecurityAgent {
	return &SecurityAgent{
		baseAgent: baseAgent{name: "security", client: client},
	}
}

func (a *SecurityAgent) Run(code, language string, _ []model.AgentResult, onChunk func(text string)) ([]model.Finding, string, error) {
	userMsg := buildCodeMessage(code, language)
	return a.run(securitySystemPrompt, userMsg, model.CategorySecurity, onChunk)
}
