package domain

import (
	"testing"
	"time"
)

// Default test values matching previous hard-coded constants
const (
	defaultTimeout  = 30 * time.Minute
	defaultMaxTurns = 10
)

// TestNewConversationSession tests session creation and initialization
func TestNewConversationSession(t *testing.T) {
	userID := "U1234567890abcdef"
	session := NewConversationSession(userID, defaultTimeout, defaultMaxTurns)

	if session.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, session.UserID)
	}

	if len(session.Messages) != 0 {
		t.Errorf("expected empty Messages slice, got %d messages", len(session.Messages))
	}

	if session.LastAccessTime.IsZero() {
		t.Error("expected LastAccessTime to be set, got zero value")
	}
}

// TestConversationSessionIsExpired tests session expiration check logic
func TestConversationSessionIsExpired(t *testing.T) {
	session := NewConversationSession("U1234567890abcdef", defaultTimeout, defaultMaxTurns)

	// New session should not be expired
	if session.IsExpired() {
		t.Error("expected new session to not be expired")
	}

	// Session with LastAccessTime 31 minutes ago should be expired
	session.LastAccessTime = time.Now().Add(-31 * time.Minute)
	if !session.IsExpired() {
		t.Error("expected session with LastAccessTime 31 minutes ago to be expired")
	}

	// Session with LastAccessTime 29 minutes ago should not be expired
	session.LastAccessTime = time.Now().Add(-29 * time.Minute)
	if session.IsExpired() {
		t.Error("expected session with LastAccessTime 29 minutes ago to not be expired")
	}
}

// TestConversationSessionAddTurn tests adding turns and FIFO removal
func TestConversationSessionAddTurn(t *testing.T) {
	session := NewConversationSession("U1234567890abcdef", defaultTimeout, defaultMaxTurns)

	// Add a turn
	userMsg := ChatMessage{Role: ChatMessageRoleUser, Content: "Hello"}
	assistantMsg := ChatMessage{Role: ChatMessageRoleAssistant, Content: "Hi there!"}
	session.AddTurn(userMsg, assistantMsg)

	history := session.GetHistory()
	if len(history) != 2 {
		t.Errorf("expected 2 messages after adding 1 turn, got %d", len(history))
	}

	if history[0].Role != ChatMessageRoleUser || history[0].Content != "Hello" {
		t.Errorf("expected first message to be user message 'Hello', got %v", history[0])
	}

	if history[1].Role != ChatMessageRoleAssistant || history[1].Content != "Hi there!" {
		t.Errorf("expected second message to be assistant message 'Hi there!', got %v", history[1])
	}
}

// TestConversationSessionHistoryLimitEnforcement tests FIFO removal when at max turns
func TestConversationSessionHistoryLimitEnforcement(t *testing.T) {
	session := NewConversationSession("U1234567890abcdef", defaultTimeout, defaultMaxTurns)

	// Add defaultMaxTurns turns
	for i := 0; i < defaultMaxTurns; i++ {
		userMsg := ChatMessage{Role: ChatMessageRoleUser, Content: "user message"}
		assistantMsg := ChatMessage{Role: ChatMessageRoleAssistant, Content: "assistant message"}
		session.AddTurn(userMsg, assistantMsg)
	}

	// Should have exactly defaultMaxTurns * 2 messages
	if len(session.Messages) != defaultMaxTurns*2 {
		t.Errorf("expected %d messages at max, got %d", defaultMaxTurns*2, len(session.Messages))
	}

	// Add one more turn with identifiable content
	newUserMsg := ChatMessage{Role: ChatMessageRoleUser, Content: "new user message"}
	newAssistantMsg := ChatMessage{Role: ChatMessageRoleAssistant, Content: "new assistant message"}
	session.AddTurn(newUserMsg, newAssistantMsg)

	// Should still have defaultMaxTurns * 2 messages (FIFO removal)
	if len(session.Messages) != defaultMaxTurns*2 {
		t.Errorf("expected %d messages after FIFO removal, got %d", defaultMaxTurns*2, len(session.Messages))
	}

	// The newest messages should be at the end
	history := session.GetHistory()
	lastIdx := len(history) - 1
	if history[lastIdx].Content != "new assistant message" {
		t.Errorf("expected last message to be 'new assistant message', got %s", history[lastIdx].Content)
	}
	if history[lastIdx-1].Content != "new user message" {
		t.Errorf("expected second-to-last message to be 'new user message', got %s", history[lastIdx-1].Content)
	}
}

// ============================================================================
// Task 2.1: Focused tests for configurable session behavior
// ============================================================================

// TestNewConversationSessionAcceptsTimeoutAndMaxTurns tests that NewConversationSession
// accepts timeout and maxTurns parameters and stores them correctly
func TestNewConversationSessionAcceptsTimeoutAndMaxTurns(t *testing.T) {
	userID := "U1234567890abcdef"
	customTimeout := 15 * time.Minute
	customMaxTurns := 5

	session := NewConversationSession(userID, customTimeout, customMaxTurns)

	// Verify basic initialization still works
	if session.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, session.UserID)
	}

	if len(session.Messages) != 0 {
		t.Errorf("expected empty Messages slice, got %d messages", len(session.Messages))
	}

	if session.LastAccessTime.IsZero() {
		t.Error("expected LastAccessTime to be set, got zero value")
	}

	// Note: timeout and maxTurns are private fields, so we verify behavior in subsequent tests
}

// TestIsExpiredUsesInstanceTimeout tests that IsExpired() uses the instance's
// timeout field instead of a constant
func TestIsExpiredUsesInstanceTimeout(t *testing.T) {
	// Create session with 5-minute timeout
	shortTimeout := 5 * time.Minute
	session := NewConversationSession("U1234567890abcdef", shortTimeout, defaultMaxTurns)

	// Session with LastAccessTime 6 minutes ago should be expired with 5-minute timeout
	session.LastAccessTime = time.Now().Add(-6 * time.Minute)
	if !session.IsExpired() {
		t.Error("expected session with 5-minute timeout and LastAccessTime 6 minutes ago to be expired")
	}

	// Session with LastAccessTime 4 minutes ago should not be expired with 5-minute timeout
	session.LastAccessTime = time.Now().Add(-4 * time.Minute)
	if session.IsExpired() {
		t.Error("expected session with 5-minute timeout and LastAccessTime 4 minutes ago to not be expired")
	}

	// Create session with 60-minute timeout
	longTimeout := 60 * time.Minute
	longSession := NewConversationSession("U1234567890abcdef", longTimeout, defaultMaxTurns)

	// Session with LastAccessTime 31 minutes ago should NOT be expired with 60-minute timeout
	longSession.LastAccessTime = time.Now().Add(-31 * time.Minute)
	if longSession.IsExpired() {
		t.Error("expected session with 60-minute timeout and LastAccessTime 31 minutes ago to NOT be expired")
	}
}

// TestAddTurnRespectsInstanceMaxTurns tests that AddTurn() uses the instance's
// maxTurns field instead of a constant
func TestAddTurnRespectsInstanceMaxTurns(t *testing.T) {
	// Create session with maxTurns = 3
	customMaxTurns := 3
	session := NewConversationSession("U1234567890abcdef", defaultTimeout, customMaxTurns)

	// Add 3 turns (reaching the limit)
	for i := 0; i < customMaxTurns; i++ {
		userMsg := ChatMessage{Role: ChatMessageRoleUser, Content: "user message"}
		assistantMsg := ChatMessage{Role: ChatMessageRoleAssistant, Content: "assistant message"}
		session.AddTurn(userMsg, assistantMsg)
	}

	// Should have exactly customMaxTurns * 2 = 6 messages
	if len(session.Messages) != customMaxTurns*2 {
		t.Errorf("expected %d messages at max, got %d", customMaxTurns*2, len(session.Messages))
	}

	// Add one more turn with identifiable content
	newUserMsg := ChatMessage{Role: ChatMessageRoleUser, Content: "new user message"}
	newAssistantMsg := ChatMessage{Role: ChatMessageRoleAssistant, Content: "new assistant message"}
	session.AddTurn(newUserMsg, newAssistantMsg)

	// Should still have customMaxTurns * 2 = 6 messages (FIFO removal)
	if len(session.Messages) != customMaxTurns*2 {
		t.Errorf("expected %d messages after FIFO removal, got %d", customMaxTurns*2, len(session.Messages))
	}

	// Verify the newest messages are at the end
	history := session.GetHistory()
	lastIdx := len(history) - 1
	if history[lastIdx].Content != "new assistant message" {
		t.Errorf("expected last message to be 'new assistant message', got %s", history[lastIdx].Content)
	}
}

// TestSessionWithCustomTimeoutAndMaxTurnsBehavesCorrectly tests that a session
// with custom timeout and maxTurns values behaves correctly for both features
func TestSessionWithCustomTimeoutAndMaxTurnsBehavesCorrectly(t *testing.T) {
	// Create session with custom values: 10-minute timeout, 2 max turns
	customTimeout := 10 * time.Minute
	customMaxTurns := 2
	session := NewConversationSession("U1234567890abcdef", customTimeout, customMaxTurns)

	// Test timeout behavior with custom 10-minute timeout
	t.Run("custom timeout", func(t *testing.T) {
		// 11 minutes ago should be expired
		session.LastAccessTime = time.Now().Add(-11 * time.Minute)
		if !session.IsExpired() {
			t.Error("expected session to be expired after 11 minutes with 10-minute timeout")
		}

		// 9 minutes ago should not be expired
		session.LastAccessTime = time.Now().Add(-9 * time.Minute)
		if session.IsExpired() {
			t.Error("expected session to not be expired after 9 minutes with 10-minute timeout")
		}
	})

	// Reset LastAccessTime for maxTurns testing
	session.LastAccessTime = time.Now()

	// Test maxTurns behavior with custom 2 max turns
	t.Run("custom maxTurns", func(t *testing.T) {
		// Add 2 turns (hitting the limit)
		for i := 0; i < customMaxTurns; i++ {
			userMsg := ChatMessage{Role: ChatMessageRoleUser, Content: "message " + string(rune('A'+i))}
			assistantMsg := ChatMessage{Role: ChatMessageRoleAssistant, Content: "response " + string(rune('A'+i))}
			session.AddTurn(userMsg, assistantMsg)
		}

		// Should have 4 messages (2 turns * 2 messages per turn)
		if len(session.Messages) != customMaxTurns*2 {
			t.Errorf("expected %d messages, got %d", customMaxTurns*2, len(session.Messages))
		}

		// Add one more turn
		session.AddTurn(
			ChatMessage{Role: ChatMessageRoleUser, Content: "new message"},
			ChatMessage{Role: ChatMessageRoleAssistant, Content: "new response"},
		)

		// Should still have 4 messages (oldest turn removed)
		if len(session.Messages) != customMaxTurns*2 {
			t.Errorf("expected %d messages after FIFO removal, got %d", customMaxTurns*2, len(session.Messages))
		}

		// The first message should now be "message B" (oldest turn "A" was removed)
		if session.Messages[0].Content != "message B" {
			t.Errorf("expected first message to be 'message B', got %s", session.Messages[0].Content)
		}

		// The last message should be "new response"
		lastIdx := len(session.Messages) - 1
		if session.Messages[lastIdx].Content != "new response" {
			t.Errorf("expected last message to be 'new response', got %s", session.Messages[lastIdx].Content)
		}
	})
}
