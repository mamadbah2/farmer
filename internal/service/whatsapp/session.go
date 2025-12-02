package whatsapp

import (
	"sync"

	"github.com/mamadbah2/farmer/pkg/clients/anthropic"
)

// SessionManager handles user conversation states.
type SessionManager struct {
	sessions map[string]anthropic.ConversationState
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]anthropic.ConversationState),
	}
}

// GetSession retrieves the current state for a user.
func (sm *SessionManager) GetSession(userID string) anthropic.ConversationState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if state, exists := sm.sessions[userID]; exists {
		return state
	}
	return anthropic.ConversationState{Step: "COLLECTING"}
}

// UpdateSession updates the state for a user.
func (sm *SessionManager) UpdateSession(userID string, state anthropic.ConversationState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[userID] = state
}

// ClearSession removes a user's session.
func (sm *SessionManager) ClearSession(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, userID)
}
