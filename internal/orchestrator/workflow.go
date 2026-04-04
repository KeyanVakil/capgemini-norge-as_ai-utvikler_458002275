package orchestrator

import "github.com/keyanvakil/agentic-code-review/internal/agent"

type Phase struct {
	Agents   []agent.Agent
	Parallel bool
}

type Workflow struct {
	Phases []Phase
}

func BuildWorkflow(agents []agent.Agent) Workflow {
	var phase1Agents []agent.Agent
	var phase2Agents []agent.Agent

	for _, a := range agents {
		if a.Name() == "improvement" {
			phase2Agents = append(phase2Agents, a)
		} else {
			phase1Agents = append(phase1Agents, a)
		}
	}

	var phases []Phase
	if len(phase1Agents) > 0 {
		phases = append(phases, Phase{Agents: phase1Agents, Parallel: true})
	}
	if len(phase2Agents) > 0 {
		phases = append(phases, Phase{Agents: phase2Agents, Parallel: false})
	}

	return Workflow{Phases: phases}
}
