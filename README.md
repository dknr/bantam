# Bantam - Lightweight Agent

Bantam is a lightweight AI agent with unified message routing.

## Key Features

- **Unified routing**: All messages flow through one code path
- **OpenTelemetry**: Full distributed tracing out of the box
- **OpenAI-compatible**: Works with any OpenAI-compatible API
- **CLI client**: Terminal-based interaction
- **Simple tools**: Shell commands, file read/write

## Installation

```bash
git clone https://github.com/dknr/bantam.git
cd bantam
go build -o bantam main/main.go
```

## Configuration

```bash
# Set your API key
export OPENAI_API_KEY="sk-..."

# Set your API base (optional, defaults to OpenAI)
export OPENAI_API_BASE="https://api.openai.com/v1"

# Set model (optional)
export OPENAI_MODEL="gpt-4o-mini"

# Set OpenTelemetry endpoint (optional)
export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
export OTEL_SERVICE_NAME="bantam"
```

## Usage

```bash
./bantam
```

### CLI Commands

- Type your message and press Enter
- `/quit` or `/exit` - Exit the CLI
- `/clear` - Clear session history

## Architecture

```
bantam/
├── main.go           # Entry point
├── agent/
│   └── agent.go      # Core loop (unified routing)
├── channel/
│   └── cli.go        # CLI channel
├── provider/
│   └── openai.go     # OpenAI-compatible API
├── session/
│   └── manager.go    # Session management
├── tools/
│   ├── shell.go      # Shell commands
│   └── filesystem.go # File operations
└── tracing/
    └── otel.go       # OpenTelemetry integration
```

## Size Goal

Target: 1000-2000 lines of Go code (vs nanobot's 16,800+ lines)

## License

MIT
