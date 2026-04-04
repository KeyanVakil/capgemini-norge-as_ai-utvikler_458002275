package orchestrator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/keyanvakil/agentic-code-review/internal/agent"
	"github.com/keyanvakil/agentic-code-review/internal/db"
	"github.com/keyanvakil/agentic-code-review/internal/model"
)

type Event struct {
	Type      string          `json:"type"`
	AgentName string          `json:"agent_name,omitempty"`
	Status    string          `json:"status,omitempty"`
	Partial   string          `json:"partial,omitempty"`
	Findings  []model.Finding `json:"findings,omitempty"`
	DurationMs int            `json:"duration_ms,omitempty"`
	Error     string          `json:"error,omitempty"`
	ReviewID  string          `json:"review_id,omitempty"`
}

type Orchestrator struct {
	repo *db.Repository
}

func New(repo *db.Repository) *Orchestrator {
	return &Orchestrator{repo: repo}
}

type agentOutcome struct {
	AgentName string
	Result    model.AgentResult
	Err       error
}

func (o *Orchestrator) RunReview(reviewID uuid.UUID, code, language string, agents []agent.Agent, eventCh chan<- Event) {
	defer close(eventCh)

	startTime := time.Now()

	if err := o.repo.UpdateReviewStatus(reviewID, model.StatusRunning, nil); err != nil {
		log.Printf("failed to update review status: %v", err)
		return
	}

	agentResultIDs := make(map[string]uuid.UUID)
	for _, a := range agents {
		ar, err := o.repo.CreateAgentResult(reviewID, a.Name())
		if err != nil {
			log.Printf("failed to create agent result for %s: %v", a.Name(), err)
			continue
		}
		agentResultIDs[a.Name()] = ar.ID
	}

	workflow := BuildWorkflow(agents)
	var allResults []model.AgentResult
	allFailed := true

	for _, phase := range workflow.Phases {
		results := o.runPhase(phase, reviewID, code, language, allResults, agentResultIDs, eventCh)
		for _, r := range results {
			allResults = append(allResults, r)
			if r.Status == model.StatusCompleted {
				allFailed = false
			}
		}
	}

	durationMs := int(time.Since(startTime).Milliseconds())
	finalStatus := model.StatusCompleted
	if allFailed && len(agents) > 0 {
		finalStatus = model.StatusFailed
	}

	if err := o.repo.UpdateReviewStatus(reviewID, finalStatus, &durationMs); err != nil {
		log.Printf("failed to update final review status: %v", err)
	}

	eventCh <- Event{
		Type:       "review_complete",
		ReviewID:   reviewID.String(),
		Status:     string(finalStatus),
		DurationMs: durationMs,
	}
}

func (o *Orchestrator) runPhase(phase Phase, reviewID uuid.UUID, code, language string, priorResults []model.AgentResult, resultIDs map[string]uuid.UUID, eventCh chan<- Event) []model.AgentResult {
	if phase.Parallel && len(phase.Agents) > 1 {
		return o.runParallel(phase.Agents, reviewID, code, language, priorResults, resultIDs, eventCh)
	}
	var results []model.AgentResult
	for _, a := range phase.Agents {
		r := o.runSingleAgent(a, reviewID, code, language, priorResults, resultIDs, eventCh)
		results = append(results, r)
	}
	return results
}

func (o *Orchestrator) runParallel(agents []agent.Agent, reviewID uuid.UUID, code, language string, priorResults []model.AgentResult, resultIDs map[string]uuid.UUID, eventCh chan<- Event) []model.AgentResult {
	outCh := make(chan agentOutcome, len(agents))
	var wg sync.WaitGroup

	for _, a := range agents {
		wg.Add(1)
		go func(ag agent.Agent) {
			defer wg.Done()
			r := o.runSingleAgent(ag, reviewID, code, language, priorResults, resultIDs, eventCh)
			outCh <- agentOutcome{AgentName: ag.Name(), Result: r}
		}(a)
	}

	wg.Wait()
	close(outCh)

	var results []model.AgentResult
	for outcome := range outCh {
		results = append(results, outcome.Result)
	}
	return results
}

func (o *Orchestrator) runSingleAgent(a agent.Agent, reviewID uuid.UUID, code, language string, priorResults []model.AgentResult, resultIDs map[string]uuid.UUID, eventCh chan<- Event) model.AgentResult {
	arID := resultIDs[a.Name()]

	eventCh <- Event{Type: "agent_status", AgentName: a.Name(), Status: "running"}

	if err := o.repo.SetAgentStarted(arID); err != nil {
		log.Printf("failed to set agent started: %v", err)
	}

	agentStart := time.Now()

	onChunk := func(text string) {
		eventCh <- Event{Type: "agent_progress", AgentName: a.Name(), Partial: text}
	}

	findings, rawOutput, err := a.Run(code, language, priorResults, onChunk)

	durationMs := int(time.Since(agentStart).Milliseconds())

	result := model.AgentResult{
		ID:        arID,
		ReviewID:  reviewID,
		AgentName: a.Name(),
		DurationMs: &durationMs,
	}

	if err != nil {
		result.Status = model.StatusFailed
		result.Error = err.Error()
		result.Findings = []model.Finding{}

		if dbErr := o.repo.UpdateAgentResult(arID, model.StatusFailed, []model.Finding{}, "", &durationMs, err.Error()); dbErr != nil {
			log.Printf("failed to update agent result: %v", dbErr)
		}

		eventCh <- Event{
			Type:      "agent_error",
			AgentName: a.Name(),
			Error:     fmt.Sprintf("Agent failed: %v", err),
		}
	} else {
		result.Status = model.StatusCompleted
		result.Findings = findings
		result.RawOutput = rawOutput

		if dbErr := o.repo.UpdateAgentResult(arID, model.StatusCompleted, findings, rawOutput, &durationMs, ""); dbErr != nil {
			log.Printf("failed to update agent result: %v", dbErr)
		}

		eventCh <- Event{
			Type:       "agent_complete",
			AgentName:  a.Name(),
			Findings:   findings,
			DurationMs: durationMs,
		}
	}

	return result
}
