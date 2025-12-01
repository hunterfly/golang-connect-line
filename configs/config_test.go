package configs

import (
	"os"
	"testing"
)

// setupTestEnv sets up required environment variables for config unmarshaling
func setupTestEnv() {
	// Set required environment variables to avoid unmarshal errors
	os.Setenv("APP_DEBUG", "false")
	os.Setenv("APP_ENV", "test")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USERNAME", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("POSTGRES_DATABASE", "test")
	os.Setenv("POSTGRES_SSLMODE", "false")
	os.Setenv("LINE_CHANNEL_SECRET", "test")
	os.Setenv("LINE_CHANNEL_TOKEN", "test")
	os.Setenv("LMSTUDIO_BASE_URL", "http://localhost:1234")
	os.Setenv("LMSTUDIO_MODEL", "test-model")
	os.Setenv("LMSTUDIO_TIMEOUT", "30")
	os.Setenv("LMSTUDIO_SYSTEM_PROMPT", "test prompt")
	// Session defaults - set to 0 to simulate application layer applying defaults
	os.Setenv("SESSION_TIMEOUT", "0")
	os.Setenv("SESSION_MAX_TURNS", "0")
}

// cleanupTestEnv cleans up environment variables after tests
func cleanupTestEnv() {
	os.Unsetenv("APP_DEBUG")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("APP_PORT")
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("POSTGRES_USERNAME")
	os.Unsetenv("POSTGRES_PASSWORD")
	os.Unsetenv("POSTGRES_DATABASE")
	os.Unsetenv("POSTGRES_SSLMODE")
	os.Unsetenv("LINE_CHANNEL_SECRET")
	os.Unsetenv("LINE_CHANNEL_TOKEN")
	os.Unsetenv("LMSTUDIO_BASE_URL")
	os.Unsetenv("LMSTUDIO_MODEL")
	os.Unsetenv("LMSTUDIO_TIMEOUT")
	os.Unsetenv("LMSTUDIO_SYSTEM_PROMPT")
	os.Unsetenv("SESSION_TIMEOUT")
	os.Unsetenv("SESSION_MAX_TURNS")
}

// TestSessionStructFieldsUnmarshal tests that Session struct fields are properly unmarshaled from config
func TestSessionStructFieldsUnmarshal(t *testing.T) {
	// Setup required environment variables
	setupTestEnv()
	defer cleanupTestEnv()

	// Set session-specific environment variables with custom values
	os.Setenv("SESSION_TIMEOUT", "45")
	os.Setenv("SESSION_MAX_TURNS", "15")

	// Initialize config - using relative path from configs directory
	InitViper(".", "test")

	cfg := GetViper()

	// Verify Session struct fields are properly unmarshaled
	if cfg.Session.Timeout != 45 {
		t.Errorf("Expected Session.Timeout to be 45, got %d", cfg.Session.Timeout)
	}

	if cfg.Session.MaxTurns != 15 {
		t.Errorf("Expected Session.MaxTurns to be 15, got %d", cfg.Session.MaxTurns)
	}
}

// TestSessionZeroValuesRequireApplicationDefaults tests that zero values signal the application layer to apply defaults
// When SESSION_TIMEOUT=0 or SESSION_MAX_TURNS=0, the application layer (in protocal/http.go) should apply defaults
func TestSessionZeroValuesRequireApplicationDefaults(t *testing.T) {
	// Setup required environment variables
	setupTestEnv()
	defer cleanupTestEnv()

	// Set session environment variables to 0 (zero)
	// This simulates the case where env vars are set to 0 or empty
	// The application layer (protocal/http.go) should apply defaults when values are 0
	os.Setenv("SESSION_TIMEOUT", "0")
	os.Setenv("SESSION_MAX_TURNS", "0")

	// Initialize config
	InitViper(".", "test")

	cfg := GetViper()

	// Verify that zero values are properly unmarshaled
	// The config layer passes through zero values - application layer applies defaults
	if cfg.Session.Timeout != 0 {
		t.Errorf("Expected Session.Timeout to be 0, got %d", cfg.Session.Timeout)
	}

	if cfg.Session.MaxTurns != 0 {
		t.Errorf("Expected Session.MaxTurns to be 0, got %d", cfg.Session.MaxTurns)
	}

	// Note: When cfg.Session.Timeout == 0 or cfg.Session.MaxTurns == 0,
	// the application layer (Task Group 5) should apply defaults:
	// - Default timeout: 30 minutes
	// - Default maxTurns: 10
}

// TestSessionConfigAccess tests config access via configs.GetViper().Session
func TestSessionConfigAccess(t *testing.T) {
	// Setup required environment variables
	setupTestEnv()
	defer cleanupTestEnv()

	// Set session-specific environment variables
	os.Setenv("SESSION_TIMEOUT", "30")
	os.Setenv("SESSION_MAX_TURNS", "10")

	// Initialize config
	InitViper(".", "test")

	// Access config via GetViper().Session pattern
	cfg := GetViper()

	// Verify we can access Session as a field of the Config struct
	session := cfg.Session

	// Verify Timeout field is accessible
	timeout := session.Timeout
	if timeout != 30 {
		t.Errorf("Expected cfg.Session.Timeout to be 30, got %d", timeout)
	}

	// Verify MaxTurns field is accessible
	maxTurns := session.MaxTurns
	if maxTurns != 10 {
		t.Errorf("Expected cfg.Session.MaxTurns to be 10, got %d", maxTurns)
	}

	// Verify direct access pattern works
	if cfg.Session.Timeout != 30 {
		t.Errorf("Expected direct access cfg.Session.Timeout to be 30, got %d", cfg.Session.Timeout)
	}

	if cfg.Session.MaxTurns != 10 {
		t.Errorf("Expected direct access cfg.Session.MaxTurns to be 10, got %d", cfg.Session.MaxTurns)
	}
}
