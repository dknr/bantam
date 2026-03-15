package session

 import (
 	"database/sql"
 	"fmt"
 	"os"
 	"path/filepath"
 	"strings"
 	"sync"
 	"time"

 	_ "modernc.org/sqlite"
 )

 // sanitizeKey converts a session key to a safe filename.
 func sanitizeKey(key string) string {
 	// Replace invalid filename characters
 	safe := strings.ReplaceAll(key, ":", "_")
 	safe = strings.ReplaceAll(safe, "/", "_")
 	safe = strings.ReplaceAll(safe, "\\", "_")
 	return safe
 }

// Manager manages conversation sessions using SQLite.
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
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
	if err := sess.initDB(m.workspace); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize session DB: %v\n", err)
		return nil
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
	// SQLite handles persistence automatically
	// This is just for in-memory cache update
}

func (m *Manager) load(key string) (*Session, error) {
	path := m.sessionPath(key)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Check if sessions table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&tableName)
	if err == sql.ErrNoRows {
		db.Close()
		return nil, fmt.Errorf("no messages table found")
	} else if err != nil {
		db.Close()
		return nil, err
	}

sess := &Session{
 		Key:       key,
 		db:        db,
 		createdAt: time.Time{},
 		updatedAt: time.Time{},
 	}

	// Get creation time
	var createdAt string
	err = db.QueryRow("SELECT created_at FROM sessions WHERE key = ?", key).Scan(&createdAt)
	if err == nil {
		sess.createdAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	// Get last message time
	var updatedAt string
	err = db.QueryRow("SELECT timestamp FROM messages WHERE session_key = ? ORDER BY timestamp DESC LIMIT 1", key).Scan(&updatedAt)
	if err == nil {
		sess.updatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	}

	return sess, nil
}

func (m *Manager) sessionPath(key string) string {
  	safeKey := sanitizeKey(key)
  	return filepath.Join(m.workspace, "sessions", safeKey+".db")
  }

// SessionPath returns the file path for a session key (exported version).
func (m *Manager) SessionPath(key string) string {
	return m.sessionPath(key)
}

// ListSessions returns a list of all session keys (filenames without .db).
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
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".db") {
			// Remove .db extension
			sessions = append(sessions, strings.TrimSuffix(entry.Name(), ".db"))
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

// ClearSession removes a specific session from memory and disk.
 func (m *Manager) ClearSession(key string) error {
 	m.mu.Lock()
 	defer m.mu.Unlock()
 
 	// Remove from memory (and close any open DB connections)
 	if sess, exists := m.sessions[key]; exists {
 		sess.Close()
 	}
 	delete(m.sessions, key)
 
 	// Remove from disk
 	path := m.sessionPath(key)
 	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
 		return err
 	}
 
 	return nil
 }
