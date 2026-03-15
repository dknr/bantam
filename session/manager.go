package session

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "time"
  "sync"
)

// Manager manages conversation sessions.
type Manager struct {
	workspace  string
	sessions   map[string]*Session
	mu         sync.RWMutex
}

// NewManager creates a new session manager.
func NewManager(workspace string) *Manager {
	return &Manager{
		workspace: workspace,
		sessions:  make(map[string]*Session),
	}
}

// GetOrCreate returns an existing session or creates a new one.
func (m *Manager) GetOrCreate(key string) *Session {
	m.mu.RLock()
	if sess, exists := m.sessions[key]; exists {
		m.mu.RUnlock()
		return sess
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if sess, exists := m.sessions[key]; exists {
		return sess
	}

	// Try to load from disk
	if sess, err := m.load(key); err == nil && sess != nil {
		m.sessions[key] = sess
		return sess
	}

	// Create new session
	sess := &Session{
		Key:       key,
		Messages:  []Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.sessions[key] = sess
	return sess
}

// Save persists a session to disk.
func (m *Manager) Save(sess *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[sess.Key] = sess
	m.saveLocked(sess)
}

func (m *Manager) saveLocked(sess *Session) {
  	path := m.sessionPath(sess.Key)

  	// Ensure directory exists
  	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
  		fmt.Fprintf(os.Stderr, "Failed to create session directory: %v\n", err)
  		return
  	}

  	// Write to temp file then rename for atomicity
  	tmpPath := path + ".tmp"
  	file, err := os.Create(tmpPath)
  	if err != nil {
  		fmt.Fprintf(os.Stderr, "Failed to create session file: %v\n", err)
  		return
  	}

  	encoder := json.NewEncoder(file)
  	encoder.SetIndent("", "  ")
  	if err := encoder.Encode(sess); err != nil {
  		file.Close()
  		os.Remove(tmpPath)
  		fmt.Fprintf(os.Stderr, "Failed to encode session: %v\n", err)
  		return
  	}

  	file.Close()
  	if err := os.Rename(tmpPath, path); err != nil {
  		os.Remove(tmpPath)
  		fmt.Fprintf(os.Stderr, "Failed to save session: %v\n", err)
  	}
  }

func (m *Manager) load(key string) (*Session, error) {
	path := m.sessionPath(key)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}

	return &sess, nil
}

func (m *Manager) sessionPath(key string) string {
  // Use simple key as filename (replace : with _)
  safeKey := filepath.Base(key)
  return filepath.Join(m.workspace, "sessions", safeKey+".json")
}

// SessionPath returns the file path for a session key (exported version).
func (m *Manager) SessionPath(key string) string {
	return m.sessionPath(key)
}

// ListSessions returns a list of all session keys (filenames without .json).
func (m *Manager) ListSessions() []string {
	sessionDir := filepath.Join(m.workspace, "sessions")
	
	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil
	}
	
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil
	}
	
	var sessions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			// Remove .json extension
			sessions = append(sessions, strings.TrimSuffix(entry.Name(), ".json"))
		}
	}
	
	return sessions
}

// Clear removes all session files from disk and clears in-memory cache.
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear in-memory sessions
	m.sessions = make(map[string]*Session)

	sessionDir := filepath.Join(m.workspace, "sessions")

	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil // Nothing to clear
	}

	// Remove entire sessions directory
	if err := os.RemoveAll(sessionDir); err != nil {
		return err
	}

	// Recreate directory for future sessions
	return os.MkdirAll(sessionDir, 0755)
}
