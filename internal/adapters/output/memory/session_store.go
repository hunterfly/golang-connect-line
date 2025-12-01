package memory

import (
	"sync"
	"time"

	"golang-template/internal/domain"
	"golang-template/internal/ports/output"
)

// Compile-time check to ensure MemorySessionStore implements SessionStore interface
var _ output.SessionStore = (*MemorySessionStore)(nil)

// MemorySessionStore struct - Output adapter for in-memory session storage
// Uses sync.Map for thread-safe concurrent access to conversation sessions.
// Stores session configuration for timeout and maxTurns to be used when creating new sessions.
type MemorySessionStore struct {
	sessions sync.Map
	timeout  time.Duration
	maxTurns int
}

// NewMemorySessionStore creates a new in-memory session store with configurable session parameters.
// timeout: Duration after which sessions expire
// maxTurns: Maximum number of conversation turns to retain in session history
func NewMemorySessionStore(timeout time.Duration, maxTurns int) *MemorySessionStore {
	return &MemorySessionStore{
		timeout:  timeout,
		maxTurns: maxTurns,
	}
}

// GetTimeout returns the configured session timeout duration.
// This value is used when creating new conversation sessions.
func (m *MemorySessionStore) GetTimeout() time.Duration {
	return m.timeout
}

// GetMaxTurns returns the configured maximum conversation turns.
// This value is used when creating new conversation sessions.
func (m *MemorySessionStore) GetMaxTurns() int {
	return m.maxTurns
}

// GetSession retrieves a conversation session by LINE user ID.
// Returns the session if found and not expired, or nil if the session
// does not exist or has expired. Expired sessions are deleted (lazy cleanup).
// LastAccessTime is updated for valid sessions.
func (m *MemorySessionStore) GetSession(userID string) (*domain.ConversationSession, error) {
	value, exists := m.sessions.Load(userID)
	if !exists {
		return nil, nil
	}

	session, ok := value.(*domain.ConversationSession)
	if !ok {
		// If data is malformed, delete and return nil
		m.sessions.Delete(userID)
		return nil, nil
	}

	// Check if session is expired
	if session.IsExpired() {
		// Lazy cleanup: delete expired session
		m.sessions.Delete(userID)
		return nil, nil
	}

	// Update LastAccessTime for valid session
	session.LastAccessTime = time.Now()

	return session, nil
}

// UpdateSession creates or updates a conversation session.
// The session's LastAccessTime is updated to the current time before storing.
func (m *MemorySessionStore) UpdateSession(session *domain.ConversationSession) error {
	// Update LastAccessTime
	session.LastAccessTime = time.Now()

	// Store session with userID as key
	m.sessions.Store(session.UserID, session)

	return nil
}

// DeleteSession removes a conversation session by LINE user ID.
// This operation is idempotent - deleting a non-existent session does not return an error.
func (m *MemorySessionStore) DeleteSession(userID string) error {
	m.sessions.Delete(userID)
	return nil
}
