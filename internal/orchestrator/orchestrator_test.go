package orchestrator

import (
	"testing"

	"github.com/keyanvakil/agentic-code-review/internal/agent"
	"github.com/keyanvakil/agentic-code-review/internal/llm"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

func TestBuildWorkflow_AllAgents(t *testing.T) {
	mock := llm.NewMockClient()
	agents := []agent.Agent{
		agent.NewSecurityAgent(mock),
		agent.NewStyleAgent(mock),
		agent.NewTestGenAgent(mock),
		agent.NewImprovementAgent(mock),
	}

	wf := BuildWorkflow(agents)
	if len(wf.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(wf.Phases))
	}

	if !wf.Phases[0].Parallel {
		t.Error("expected Phase 1 to be parallel")
	}
	if len(wf.Phases[0].Agents) != 3 {
		t.Errorf("expected 3 agents in Phase 1, got %d", len(wf.Phases[0].Agents))
	}

	if wf.Phases[1].Parallel {
		t.Error("expected Phase 2 to be sequential")
	}
	if len(wf.Phases[1].Agents) != 1 {
		t.Errorf("expected 1 agent in Phase 2, got %d", len(wf.Phases[1].Agents))
	}
	if wf.Phases[1].Agents[0].Name() != "improvement" {
		t.Errorf("expected improvement agent in Phase 2, got %s", wf.Phases[1].Agents[0].Name())
	}
}

func TestBuildWorkflow_OnlySecurity(t *testing.T) {
	mock := llm.NewMockClient()
	agents := []agent.Agent{agent.NewSecurityAgent(mock)}

	wf := BuildWorkflow(agents)
	if len(wf.Phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(wf.Phases))
	}
	if len(wf.Phases[0].Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(wf.Phases[0].Agents))
	}
}

func TestBuildWorkflow_OnlyImprovement(t *testing.T) {
	mock := llm.NewMockClient()
	agents := []agent.Agent{agent.NewImprovementAgent(mock)}

	wf := BuildWorkflow(agents)
	if len(wf.Phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(wf.Phases))
	}
	if wf.Phases[0].Parallel {
		t.Error("expected phase to be sequential for single improvement agent")
	}
}

func TestBuildWorkflow_NoAgents(t *testing.T) {
	wf := BuildWorkflow(nil)
	if len(wf.Phases) != 0 {
		t.Fatalf("expected 0 phases, got %d", len(wf.Phases))
	}
}

func TestBuildWorkflow_Phase1Only(t *testing.T) {
	mock := llm.NewMockClient()
	agents := []agent.Agent{
		agent.NewSecurityAgent(mock),
		agent.NewStyleAgent(mock),
	}

	wf := BuildWorkflow(agents)
	if len(wf.Phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(wf.Phases))
	}
	if !wf.Phases[0].Parallel {
		t.Error("expected Phase 1 to be parallel")
	}
	if len(wf.Phases[0].Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(wf.Phases[0].Agents))
	}
}

type fakeAgent struct {
	name     string
	findings []model.Finding
	err      error
}

func (f *fakeAgent) Name() string { return f.name }
func (f *fakeAgent) Run(code, lang string, prior []model.AgentResult, onChunk func(string)) ([]model.Finding, string, error) {
	if f.err != nil {
		return nil, "", f.err
	}
	return f.findings, "raw output", nil
}

func TestBuildWorkflow_OrderPreserved(t *testing.T) {
	agents := []agent.Agent{
		&fakeAgent{name: "improvement"},
		&fakeAgent{name: "security"},
		&fakeAgent{name: "style"},
	}

	wf := BuildWorkflow(agents)
	if len(wf.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(wf.Phases))
	}

	phase1Names := make(map[string]bool)
	for _, a := range wf.Phases[0].Agents {
		phase1Names[a.Name()] = true
	}
	if !phase1Names["security"] || !phase1Names["style"] {
		t.Error("expected security and style in Phase 1")
	}
	if phase1Names["improvement"] {
		t.Error("improvement should not be in Phase 1")
	}
}
