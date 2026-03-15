// Package agent provides the core agent loop with unified message routing.
//
// Key design principle: ALL messages flow through one code path,
// regardless of source (CLI, gateway, etc.). This ensures OpenTelemetry
// tracing works consistently.
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/provider"
	"github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tools"
	"github.com/dknr/bantam/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Agent is the core agent instance.
type Agent struct {
	provider     provider.Provider
	toolRegistry *tools.Registry
	sessionMgr   *session.Manager
	systemPrompt string
}

// New creates a new Agent instance.
func New(p provider.Provider, tools *tools.Registry, sessions *session.Manager) *Agent {
	return NewWithSystemPrompt(p, tools, sessions, "")
}

func NewWithSystemPrompt(p provider.Provider, tools *tools.Registry, sessions *session.Manager, systemPrompt string) *Agent {
	return &Agent{
		provider:     p,
		toolRegistry: tools,
		sessionMgr:   sessions,
		systemPrompt: systemPrompt,
	}
}

// ProcessStats contains timing and token information.
type ProcessStats struct {
	DurationMs int
	Tokens     map[string]int
	Timing     interface{} // Provider timing data (map from API response)
}

// ProcessMessage handles a single message from any source (CLI, gateway, etc.).
//
// This is the UNIFIED message handler - all messages flow through here,
// ensuring consistent OpenTelemetry tracing regardless of channel.
func (a *Agent) ProcessMessage(ctx context.Context, channel, chatID, content string) (string, error) {
	content, _, err := a.processMessageWithTiming(ctx, channel, chatID, content)
	return content, err
}

// ProcessMessageWithStats handles a single message and returns timing/token stats.
func (a *Agent) ProcessMessageWithStats(ctx context.Context, channel, chatID, content string) (string, ProcessStats, error) {
	return a.processMessageWithTiming(ctx, channel, chatID, content)
}

// processMessageWithTiming is the internal implementation that handles both methods.
func (a *Agent) processMessageWithTiming(ctx context.Context, channel, chatID, content string) (string, ProcessStats, error) {
	// Create span for this message (unified for all sources)
	ctx, processSpan := tracing.StartActiveSpan(ctx, "process_message", map[string]string{
		"channel":   channel,
		"chat_id":   chatID,
		"operation": "receive",
	})
	if processSpan != nil {
		defer processSpan.End()
	}

	logger := logging.FromContext(ctx)
	logger.Info("Processing message", "channel", channel, "chat_id", chatID)

	// Log request in verbose mode
	if logging.IsVerbose(ctx) {
		reqData := map[string]any{
			"channel":     channel,
			"chat_id":     chatID,
			"content":     content,
			"session_key": fmt.Sprintf("%s:%s", channel, chatID),
		}
		logging.PrintJSON("Request", reqData)
	}

	// Build session key (统一 session key 格式)
	sessionKey := fmt.Sprintf("%s:%s", channel, chatID)

	// Load or create session for this conversation
	sess := a.sessionMgr.GetOrCreate(sessionKey)

	// Add user message to session
	sess.AddMessage("user", content)

	// Build messages for LLM (history + new message)
	messages := a.buildMessages(sess)

	// Call LLM with timing
	logger.Info("calling LLM", "messages_count", len(messages))
	startTime := time.Now()
	ctx, chatSpan := tracing.StartActiveSpan(ctx, "llm.chat", map[string]string{
		"messages_count": fmt.Sprintf("%d", len(messages)),
	})
	resp, err := a.provider.Chat(ctx, messages, a.toolRegistry.DefinitionsWithSchema())
	durationMs := time.Since(startTime).Milliseconds()
	if err != nil {
		if chatSpan != nil {
			chatSpan.SetStatus(codes.Error, err.Error())
			chatSpan.End()
		}
		logger.Error(err, "LLM chat failed")
		return "", ProcessStats{}, fmt.Errorf("LLM error: %w", err)
	}
	if chatSpan != nil {
		chatSpan.SetAttributes(attribute.Int("response.has_tool_calls", boolToInt(resp.HasToolCalls())))
		chatSpan.SetAttributes(attribute.Int("response.content_length", len(resp.Content())))
		chatSpan.End()
	}
	logger.Info("LLM response received", "has_tool_calls", resp.HasToolCalls(), "content_length", len(resp.Content()), "duration_ms", durationMs)

	// Handle tool calls - loop until no more tool calls (supports iterative tool calling)
	var tokens map[string]int
	for resp.HasToolCalls() {
		// Print tool calls with status lines if available
		for _, call := range resp.ToolCalls() {
			tool, exists := a.toolRegistry.Get(call.Name)
			if exists {
				// Check if tool implements StatusLine
				if statusTool, ok := tool.(tools.StatusLineTool); ok {
					fmt.Printf("\033[33m%s\033[0m\n", statusTool.StatusLine(call.Arguments))
				} else {
					fmt.Printf("\033[33m%s(%v)\033[0m\n", call.Name, call.Arguments)
				}
			} else {
				fmt.Printf("\033[33m%s(%v)\033[0m\n", call.Name, call.Arguments)
			}
		}

		// Save the assistant's tool call request message to session
		// The assistant message with tool calls needs to be preserved for conversation history
		sess.AddMessage("assistant", "")
		logger.Info("Added assistant tool call message to session")

		for _, call := range resp.ToolCalls() {
			// Execute tool with span
			ctx, toolSpan := tracing.StartActiveSpan(ctx, "tool.execute", map[string]string{
				"tool.name": call.Name,
			})
			result, err := a.toolRegistry.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				logger.Error(err, "tool execution failed", "tool", call.Name)
				toolSpan.SetStatus(codes.Error, err.Error())
				// Add error as tool result so LLM can see what went wrong and potentially retry
				sess.AddMessage("tool", fmt.Sprintf("{\"name\": \"%s\", \"content\": \"Error: %v\"}", call.Name, err))
				toolSpan.End()
				continue
			}
			toolSpan.End()

			// Add tool result to session
			resultStr := fmt.Sprintf("%v", result)
			sess.AddMessage("tool", fmt.Sprintf("{\"name\": \"%s\", \"content\": %s}", call.Name, resultStr))
		}

		// Call LLM again with tool results
		messages = a.buildMessages(sess)
		resp, err = a.provider.Chat(ctx, messages, a.toolRegistry.DefinitionsWithSchema())
		if err != nil {
			logger.Error(err, "LLM chat after tools failed")
			return "", ProcessStats{}, fmt.Errorf("LLM error: %w", err)
		}

		// Print intermediate response (cyan with indent) if there's content and there are more tool calls coming
		// If no more tool calls, this is the final response which printResponse will handle
		if resp.Content() != "" && resp.HasToolCalls() {
			fmt.Printf("\033[36m> %s\033[0m\n", resp.Content())
		}
	}

	// Get token details from response
	tokens = resp.TokenDetails()

	// Add assistant response to session
	logger.Info("Checking assistant response", "content_length", len(resp.Content()), "has_tool_calls", resp.HasToolCalls())
	if resp.Content() != "" {
		sess.AddMessage("assistant", resp.Content())
		logger.Info("Added assistant response to session", "content", resp.Content()[:min(100, len(resp.Content()))])
	}

	// Save session
	a.sessionMgr.Save(sess)

	responseContent := resp.Content()
	logger.Info("response content", "content_length", len(responseContent), "tokens", tokens)

	// Log response in verbose mode
	if logging.IsVerbose(ctx) {
		respData := map[string]any{
			"content":        responseContent,
			"has_tool_calls": resp.HasToolCalls(),
			"tokens":         tokens,
			"duration_ms":    durationMs,
		}
		logging.PrintJSON("Response", respData)
	}

	// Get timing data from response
	timing := resp.Timing()

	return responseContent, ProcessStats{DurationMs: int(durationMs), Tokens: tokens, Timing: timing}, nil
}

// buildMessages constructs the message history for the LLM.
func (a *Agent) buildMessages(sess *session.Session) []map[string]any {
	history := sess.History()

	// Build messages array
	messages := make([]map[string]any, 0, len(history)+1)

	// Prepend system prompt if set
	if a.systemPrompt != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": a.systemPrompt,
		})
	}

	for _, msg := range history {
		messages = append(messages, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	return messages
}

// boolToInt converts bool to int for OpenTelemetry attributes.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
