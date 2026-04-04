package api

import (
	"sync"

	"github.com/google/uuid"
	"github.com/keyanvakil/agentic-code-review/internal/orchestrator"
)

type SSEManager struct {
	mu          sync.RWMutex
	sources     map[uuid.UUID]chan orchestrator.Event
	subscribers map[uuid.UUID][]chan orchestrator.Event
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		sources:     make(map[uuid.UUID]chan orchestrator.Event),
		subscribers: make(map[uuid.UUID][]chan orchestrator.Event),
	}
}

func (m *SSEManager) Register(reviewID uuid.UUID, source chan orchestrator.Event) {
	m.mu.Lock()
	m.sources[reviewID] = source
	m.mu.Unlock()

	go m.broadcast(reviewID, source)
}

func (m *SSEManager) Unregister(reviewID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sources, reviewID)
	for _, sub := range m.subscribers[reviewID] {
		close(sub)
	}
	delete(m.subscribers, reviewID)
}

func (m *SSEManager) Subscribe(reviewID uuid.UUID) chan orchestrator.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sources[reviewID]; !exists {
		return nil
	}

	ch := make(chan orchestrator.Event, 100)
	m.subscribers[reviewID] = append(m.subscribers[reviewID], ch)
	return ch
}

func (m *SSEManager) broadcast(reviewID uuid.UUID, source chan orchestrator.Event) {
	for event := range source {
		m.mu.RLock()
		subs := m.subscribers[reviewID]
		for _, sub := range subs {
			select {
			case sub <- event:
			default:
			}
		}
		m.mu.RUnlock()
	}
}
