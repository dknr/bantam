// Package agent provides the core agent loop with unified message routing.
//
// Key design principle: ALL messages flow through one code path,
// regardless of source (CLI, gateway, etc.). This ensures OpenTelemetry
// tracing works consistently.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/paths"
	"github.com/dknr/bantam/provider"
	"github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tools"
	"github.com/dknr/bantam/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"github.com/go-logr/logr"
)

// OutputMessageType represents the type of output message.
type OutputMessageType int

const (
	OutputStatsType OutputMessageType = iota
	OutputToolStatusType
	OutputIntermediateResponseType
	OutputErrorType
	OutputFinalResponseType
)

// OutputMessage is the interface for all output messages from the agent.
type OutputMessage interface {
	Type() OutputMessageType
}

// OutputStats contains timing and token information.
type OutputStats struct {
	Tokens     map[string]int
	DurationMs float64
	Timing     interface{}
}

func (OutputStats) Type() OutputMessageType { return OutputStatsType }

// OutputToolStatus represents the status of a tool execution.
type OutputToolStatus struct {
	ToolName string
	Args     map[string]any
}

func (OutputToolStatus) Type() OutputMessageType { return OutputToolStatusType }

// OutputIntermediateResponse represents an intermediate response from the LLM.
type OutputIntermediateResponse struct {
	Content string
}

func (OutputIntermediateResponse) Type() OutputMessageType { return OutputIntermediateResponseType }

// OutputError represents an error that occurred during processing.
type OutputError struct {
	Err error
}

func (OutputError) Type() OutputMessageType { return OutputErrorType }

// OutputFinalResponse represents the final response from the agent.
type OutputFinalResponse struct {
	Content string
}

func (OutputFinalResponse) Type() OutputMessageType { return OutputFinalResponseType }

// loadSystemPrompt loads the system prompt from soul.md or returns a fallback.
func loadSystemPrompt(logger logr.Logger, workspaceDir string) string {
	soulPath := filepath.Join(workspaceDir, "soul.md")
	if data, err := os.ReadFile(soulPath); err == nil {
		logger.Info("Loaded system prompt from soul.md", "path", soulPath)
		return string(data)
	}
	// Fallback if soul.md doesn't exist
	fallback := "Read soul.md from your workspace for your identity and instructions."
	logger.Info("soul.md not found, using fallback system prompt", "path", soulPath)
	return fallback
}

// Agent is the core agent instance that communicates via channels.
type Agent struct {
	provider     provider.Provider
	toolRegistry *tools.Registry
	sessionMgr   *session.Manager
	InputChan    chan string      // Only message content
	OutputChan   chan OutputMessage
	channel      string           // Fixed per agent (e.g., "cli")
	chatID       string           // Fixed per agent
}

// New creates a new Agent instance with channel-based communication.
func New(p provider.Provider, tools *tools.Registry, sessions *session.Manager, channel, chatID string) *Agent {
	return &Agent{
		provider:     p,
		toolRegistry: tools,
		sessionMgr:   sessions,
		InputChan:    make(chan string, 10),
		OutputChan:   make(chan OutputMessage, 10),
		channel:      channel,
		chatID:       chatID,
	}
}

// Start begins the agent loop, reading from inputChan and writing to outputChan.
// The loop runs until the provided context is cancelled.
func (a *Agent) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-a.InputChan:
			if !ok {
				return
			}
			// Process the message (this is the core agent logic)
			a.processMessageWithTiming(ctx, msg)
		}
	}
}

// processMessageWithTiming is the internal implementation that handles message processing.
// It sends output messages to the output channel instead of returning values.
func (a *Agent) processMessageWithTiming(ctx context.Context, content string) {
	// Create span for this message (unified for all sources)
	ctx, processSpan := tracing.StartActiveSpan(ctx, "process_message", map[string]string{
		"channel":   a.channel,
		"chat_id":   a.chatID,
		"operation": "receive",
	})
	if processSpan != nil {
		defer processSpan.End()
	}

	logger := logging.FromContext(ctx)
	logger.Info("Processing message", "channel", a.channel, "chat_id", a.chatID)

	// Log request in verbose mode
	if logging.IsVerbose(ctx) {
		reqData := map[string]any{
			"channel":     a.channel,
			"chat_id":     a.chatID,
			"content":     content,
			"session_key": fmt.Sprintf("%s:%s", a.channel, a.chatID),
		}
		logging.PrintJSON("Request", reqData)
	}

	// Build session key (统一 session key 格式)
	sessionKey := fmt.Sprintf("%s:%s", a.channel, a.chatID)

	// Load or create session for this conversation
	sess := a.sessionMgr.GetOrCreate(sessionKey)
	// If session is new, add system prompt as first message
	if sess.MessageCount() == 0 {
		systemPrompt := loadSystemPrompt(logger, paths.WorkspaceDir)
		sess.AddMessage("system", systemPrompt)
	}
	// Add user message to session
	sess.AddMessage("user", content)

	var tokens map[string]int
	var durationMs int64
	var responseContent string
	firstIteration := true

	for {
		// Build messages for LLM (history + new message)
		messages := a.buildMessages(sess)

		// Call LLM with timing
		logger.Info("calling LLM", "messages_count", len(messages))
		if logging.IsVerbose(ctx) {
			reqData := map[string]any{
				"channel":     a.channel,
				"chat_id":     a.chatID,
				"content":     content,
				"session_key": fmt.Sprintf("%s:%s", a.channel, a.chatID),
				"messages":    messages,
			}
			logging.PrintJSON("Request", reqData)
		}
		startTime := time.Now()
		ctx, chatSpan := tracing.StartActiveSpan(ctx, "llm.chat", map[string]string{
			"messages_count": fmt.Sprintf("%d", len(messages)),
		})
		resp, err := a.provider.Chat(ctx, messages, a.toolRegistry.DefinitionsWithSchema())
		if err != nil {
			if chatSpan != nil {
				chatSpan.SetStatus(codes.Error, err.Error())
				chatSpan.End()
			}
			logger.Error(err, "LLM chat failed")
			// Send error to output channel
			select {
			case a.OutputChan <- OutputError{Err: fmt.Errorf("LLM error: %w", err)}:
			case <-ctx.Done():
				return
			}
			return
		}
		if chatSpan != nil {
			chatSpan.SetAttributes(attribute.Int("response.has_tool_calls", boolToInt(resp.HasToolCalls())))
			chatSpan.SetAttributes(attribute.Int("response.content_length", len(resp.Content())))
			chatSpan.End()
		}
		durationMsTmp := time.Since(startTime).Milliseconds()
		callTokens := resp.TokenDetails()
		callTiming := resp.Timing()
		if firstIteration {
			durationMs = durationMsTmp
			firstIteration = false
		}
		logger.Info("LLM response received", "has_tool_calls", resp.HasToolCalls(), "content_length", len(resp.Content()), "duration_ms", durationMsTmp)

		// Print stats line for every LLM response (send to output channel)
		select {
		case a.OutputChan <- OutputStats{
			Tokens:     callTokens,
			DurationMs: float64(durationMsTmp),
			Timing:     callTiming,
		}:
		case <-ctx.Done():
			return
		}

		// Print intermediate response (cyan with indent) if there's content and there are more tool calls coming
		// If no more tool calls, this is the final response which will be sent separately
		if resp.Content() != "" && resp.HasToolCalls() {
			select {
			case a.OutputChan <- OutputIntermediateResponse{
				Content: resp.Content(),
			}:
			case <-ctx.Done():
				return
			}
		}

		if !resp.HasToolCalls() {
			// Final response
			responseContent = resp.Content()
			tokens = resp.TokenDetails()
			break
		}

		// Handle tool calls
			for _, call := range resp.ToolCalls() {
				if _, exists := a.toolRegistry.Get(call.Name); exists {
					select {
					case a.OutputChan <- OutputToolStatus{
						ToolName: call.Name,
						Args:     call.Arguments,
					}:
					case <-ctx.Done():
						return
					}
				} else {
					select {
					case a.OutputChan <- OutputToolStatus{
						ToolName: call.Name,
						Args:     call.Arguments,
					}:
					case <-ctx.Done():
						return
					}
				}
			}

		// Save the assistant's tool call request message to session
		// The assistant message with tool calls needs to be preserved for conversation history
		if resp.HasToolCalls() {
			data := map[string]interface{}{
				"content": resp.Content(),
				"tool_calls": resp.ToolCalls(),
			}
			jsonData, err := json.Marshal(data)
			if err != nil {
				// fallback to just content
				sess.AddMessage("assistant", resp.Content())
			} else {
				sess.AddMessage("assistant", string(jsonData))
			}
			logger.Info("Stored assistant message with tool calls as JSON")
		} else {
			sess.AddMessage("assistant", resp.Content())
			logger.Info("Added assistant response to session")
		}

		for _, call := range resp.ToolCalls() {
			// Execute tool with span
			ctx, toolSpan := tracing.StartActiveSpan(ctx, "tool.execute", map[string]string{
				"tool.name": call.Name,
			})
			result, err := a.toolRegistry.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				logger.Error(err, "tool execution failed", "tool", call.Name)
				if toolSpan != nil {
					toolSpan.SetStatus(codes.Error, err.Error())
				}
				// Print error in red to terminal (send to output channel)
				select {
				case a.OutputChan <- OutputError{Err: fmt.Errorf("tool %s failed: %w", call.Name, err)}:
				case <-ctx.Done():
					if toolSpan != nil {
						toolSpan.End()
					}
					return
				}
				// Add error as tool result so LLM can see what went wrong and potentially retry
				sess.AddMessage("tool", fmt.Sprintf("{\"name\": \"%s\", \"content\": \"Error: %v\"}", call.Name, err))
				if toolSpan != nil {
					toolSpan.End()
				}
				continue
			}
			if toolSpan != nil {
				toolSpan.End()
			}

			// Add tool result to session
			resultStr := fmt.Sprintf("%v", result)
			sess.AddMessage("tool", fmt.Sprintf("{\"name\": \"%s\", \"content\": %s}", call.Name, resultStr))
		}
	}

	// Add assistant response to session (final response)
	logger.Info("Checking assistant response", "content_length", len(responseContent), "has_tool_calls", false)
	if responseContent != "" {
		sess.AddMessage("assistant", responseContent)
	}
	logger.Info("Added assistant response to session", "content", responseContent[:min(100, len(responseContent))])

	// Save session
	a.sessionMgr.Save(sess)

	logger.Info("response content", "content_length", len(responseContent), "tokens", tokens)

	// Log response in verbose mode
	if logging.IsVerbose(ctx) {
		respData := map[string]any{
			"content":        responseContent,
			"has_tool_calls": false,
			"tokens":         tokens,
			"duration_ms":    durationMs,
		}
		logging.PrintJSON("Response", respData)
	}

	// Send final response to output channel
	select {
	case a.OutputChan <- OutputFinalResponse{
		Content: responseContent,
	}:
	case <-ctx.Done():
		return
	}
}

// buildMessages constructs the message history for the LLM.
func (a *Agent) buildMessages(sess *session.Session) []map[string]any {
	history := sess.History()

	// Build messages array
	messages := make([]map[string]any, 0, len(history)+1)

	for _, msg := range history {
		if msg.Role == "assistant" {
			// Check if the content is a JSON string with tool_calls
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Content), &data); err == nil {
				if toolCalls, ok := data["tool_calls"]; ok {
					// This is an assistant message with tool calls
					m := map[string]any{
						"role":    msg.Role,
						"content": data["content"],
					}
					if toolCalls != nil {
						// Convert internal tool calls to LLM-expected format
						var converted []map[string]any
						for _, tc := range toolCalls.([]interface{}) {
							if tcMap, ok := tc.(map[string]interface{}); ok {
								id := tcMap["ID"]
								name := tcMap["Name"]
								args := tcMap["Arguments"]
								// Ensure arguments is a JSON string
								var argsStr string
								if argsMap, ok := args.(map[string]interface{}); ok {
									b, _ := json.Marshal(argsMap)
									argsStr = string(b)
								} else if argsStrRaw, ok := args.(string); ok {
									argsStr = argsStrRaw
								} else {
									// fallback
									b, _ := json.Marshal(args)
									argsStr = string(b)
								}
								converted = append(converted, map[string]any{
									"id":   id,
									"type": "function",
									"function": map[string]any{
										"name":     name,
										"arguments": argsStr,
									},
								})
							}
						}
						m["tool_calls"] = converted
					}
					messages = append(messages, m)
					continue
				}
			}
		}
		// Default case
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

// Close closes the agent and any resources it holds.
func (a *Agent) Close() {
	// Memory tool is closed by the caller (cmd/root.go)
	// Close channels to prevent goroutine leaks
	close(a.InputChan)
	close(a.OutputChan)
}