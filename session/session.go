// Package session provides conversation session management.
package session

import (
	"time"
)

// Message represents a single message in a session.
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session is a conversation session.
type Session struct {
	Key       string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AddMessage adds a message to the session.
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// History returns all messages.
func (s *Session) History() []Message {
	return s.Messages
}

// Clear removes all messages.
func (s *Session) Clear() {
	s.Messages = []Message{}
	s.UpdatedAt = time.Now()
}
