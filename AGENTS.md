# Bantam Agent Development Guide

Welcome to the Bantam codebase! This guide documents what agents need to know to work effectively with this repository.

## Essential Commands

- **Build**: `go build -o bantam ./main` - Compiles the binary
- **Test**: `go test ./...` - Runs all tests
- **Run**: `bantam run` - Start interactive mode
- **Prompt**: `bantam prompt "your message"` - Send a single message and exit
- **Session Management**: 
  - `bantam session list` - List existing sessions
  - `bantam session clear` - Clear all sessions

## Code Organization

The project follows a modular structure:

```
bantam/
├── cmd/              # Cobra command implementations
├── agent/            # Core agent loop and logic
├── channel/          # Chat channel implementations (CLI, gateway)
├── provider/         # LLM provider interface and implementations
├── session/          # Session management (SQLite-backed)
├── tools/            # Tool definitions (view, edit, list, time, echo, grep, git, memory)
├── tracing/          # OpenTelemetry integration
└── paths/            # Path configuration constants
```

## Tool Pattern

All tools follow a consistent interface:

- **Name()** string - Returns tool name
- **StatusLine(args map[string]any)** string - Formatted status line for UI
- **Execute(ctx context.Context, args map[string]any)** (any, error) - Main execution logic
- Tools are registered in the tools registry and invoked through the agent

Common methods across tools:
- Path validation (especially for file operations)
- Context propagation
- Error wrapping with additional context

## Agent Pattern

The core agent resides in `agent/agent.go`:

- **ProcessMessage()** - Unified handler for all incoming messages
- **ProcessMessageWithStats()** - Returns timing/token statistics
- **processMessageWithTiming()** - Internal implementation with OpenTelemetry spans
- **buildMessages()** - Constructs message history including system prompt

Key characteristics:
- All messages flow through a single processing pipeline
- Supports iterative tool calling
- Maintains session state
- Integrates with OpenTelemetry tracing
- Handles system prompts from `soul.md`

## Naming Conventions

- **Tools**: Lowercase names (view, edit, list, time, echo, grep, git, memory)
- **Structs**: `{ToolName}Tool` (ExecTool, FileTool)
- **Methods**: Consistent signature patterns
- **Errors**: Wrapped with context using `fmt.Errorf("...: %w", err)`
- **Environment variables**: Uppercase with `BANTAM_` prefix

## Testing

Tests follow standard Go testing conventions:

- Run all tests with `go test ./...`
- Tests are typically located alongside implementation files
- Table-driven tests are encouraged
- Integration tests may require specific setup

CI configuration in `.github/workflows/ci.yml` defines the test command.

## Gotchas & Non-Obvious Patterns

- **Path Restrictions**: File tool only accepts paths relative to the workspace directory
- **Unified Message Flow**: All channels (CLI, gateway) route through the same agent processing logic
- **Tracing Semantics**: Specific span names (`"process_message"`, `"llm.chat"`, `"tool.execute"`) are used throughout
- **Verbose Logging**: Outputs JSON-formatted logs when enabled
- **System Prompt**: Loaded from `soul.md` in the workspace; fallback exists
- **Tool Calling Loop**: Iterative process that continues until no more tool calls are generated

## Project Context

- **CLI Framework**: Cobra-based commands in `cmd/`
- **Dependencies**: Includes cobra, spf13/cobra, OpenTelemetry, zap logging
- **Size Goal**: Target 1000-2000 lines of Go code
- **License**: MIT
- **Configuration**: Environment variables can override config file settings