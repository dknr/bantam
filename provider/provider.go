// Package provider defines the LLM provider interface.
package provider

import (
	"context"
)

// ToolCall represents a tool call request from the LLM.
type ToolCall struct {
	ID       string
	Name     string
	Arguments map[string]any
}

// Response is the result from an LLM call.
type Response struct {
	content      string
	toolCalls    []ToolCall
	finishReason string
	tokens       map[string]int
}

// NewResponse creates a new Response.
func NewResponse(content string, toolCalls []ToolCall, finishReason string) *Response {
	return &Response{
		content:      content,
		toolCalls:    toolCalls,
		finishReason: finishReason,
		tokens:       make(map[string]int),
	}
}

// SetTokens sets the token usage information.
func (r *Response) SetTokens(tokens map[string]int) {
	r.tokens = tokens
}

// TokenCount returns the total token count.
func (r *Response) TokenCount() int {
	total := 0
	for _, v := range r.tokens {
		total += v
	}
	return total
}

// TokenDetails returns the token details map.
func (r *Response) TokenDetails() map[string]int {
	return r.tokens
}

// Content returns the response content.
func (r *Response) Content() string {
	return r.content
}

// ToolCalls returns any tool calls in the response.
func (r *Response) ToolCalls() []ToolCall {
	return r.toolCalls
}

// HasToolCalls checks if the response contains tool calls.
func (r *Response) HasToolCalls() bool {
	return len(r.toolCalls) > 0
}

// FinishReason returns the finish reason.
func (r *Response) FinishReason() string {
	return r.finishReason
}

// Provider is the interface for LLM providers.
type Provider interface {
	// Chat sends a chat completion request.
	Chat(ctx context.Context, messages []map[string]any, tools []map[string]any) (*Response, error)

	// GetDefaultModel returns the default model for this provider.
	GetDefaultModel() string
}
