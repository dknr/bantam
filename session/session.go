// Package session provides conversation session management using SQLite.
package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Message represents a single message in a session.
type Message struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session is a conversation session backed by SQLite.
type Session struct {
	Key       string
	db        *sql.DB
	createdAt time.Time
	updatedAt time.Time
}

func (s *Session) initDB(dbPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Create sessions table
	_, err = db.Exec(`
 		CREATE TABLE IF NOT EXISTS sessions (
 			key TEXT PRIMARY KEY,
 			created_at TEXT NOT NULL,
 			updated_at TEXT NOT NULL
 		)
 	`)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create messages table
	_, err = db.Exec(`
 		CREATE TABLE IF NOT EXISTS messages (
 			id INTEGER PRIMARY KEY AUTOINCREMENT,
 			session_key TEXT NOT NULL,
 			role TEXT NOT NULL,
 			content TEXT NOT NULL,
 			timestamp TEXT NOT NULL,
 			FOREIGN KEY (session_key) REFERENCES sessions(key)
 		)
 	`)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Insert session record
	_, err = db.Exec(`
 		INSERT OR REPLACE INTO sessions (key, created_at, updated_at)
 		VALUES (?, ?, ?)
 	`, s.Key, s.createdAt.Format(time.RFC3339), s.updatedAt.Format(time.RFC3339))
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to insert session: %w", err)
	}

	s.db = db
	return nil
}

// AddMessage adds a message to the session.
func (s *Session) AddMessage(role, content string) {
	if s.db == nil {
		return
	}

	timestamp := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO messages (session_key, role, content, timestamp)
		VALUES (?, ?, ?, ?)
	`, s.Key, role, content, timestamp.Format(time.RFC3339))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to add message: %v\n", err)
		return
	}

	// Update updated_at in sessions table
	s.db.Exec(`
		UPDATE sessions SET updated_at = ? WHERE key = ?
	`, timestamp.Format(time.RFC3339), s.Key)

	s.updatedAt = timestamp
}

// History returns all messages in the session.
func (s *Session) History() []Message {
	if s.db == nil {
		return nil
	}

	rows, err := s.db.Query(`
 		SELECT id, role, content, timestamp 
 		FROM messages 
 		WHERE session_key = ? 
 		ORDER BY id
 	`, s.Key)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var timestampStr string
		err := rows.Scan(&m.ID, &m.Role, &m.Content, &timestampStr)
		if err != nil {
			continue
		}
		m.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		messages = append(messages, m)
	}

	return messages
}

// MessageCount returns the total number of messages in the session.
func (s *Session) MessageCount() int {
	if s.db == nil {
		return 0
	}

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_key = ?", s.Key).Scan(&count)
	if err != nil {
		return 0
	}

	return count
}

// RecentMessages returns the last N messages.
func (s *Session) RecentMessages(n int) []Message {
	if s.db == nil {
		return nil
	}

	rows, err := s.db.Query(`
		SELECT id, role, content, timestamp 
		FROM messages 
		WHERE session_key = ? 
		ORDER BY id DESC 
		LIMIT ?
	`, s.Key, n)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var timestampStr string
		err := rows.Scan(&m.ID, &m.Role, &m.Content, &timestampStr)
		if err != nil {
			continue
		}
		m.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		messages = append(messages, m)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages
}

// Clear removes all messages from the session.
func (s *Session) Clear() {
	if s.db == nil {
		return
	}

	s.db.Exec(`DELETE FROM messages WHERE session_key = ?`, s.Key)
	s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE key = ?`, time.Now().Format(time.RFC3339), s.Key)
	s.updatedAt = time.Now()
}

// Close closes the database connection.
func (s *Session) Close() error {
	if s.db != nil {
		err := s.db.Close()
		s.db = nil
		return err
	}
	return nil
}
