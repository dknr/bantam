# Bantam - Lightweight Agent

Bantam is a lightweight AI agent with OpenAI-compatible API support.

## Key Features

- **OpenAI-compatible**: Works with any OpenAI-compatible API (Ollama, LM Studio, OpenRouter, etc.)
- **CLI client**: Terminal-based interaction with markdown rendering
- **Simple tools**: Shell commands, file read/write

## Installation

```bash
git clone https://github.com/dknr/bantam.git
cd bantam
go build -o bantam .
```

## Configuration

```bash
# Set your API key
export BANTAM_API_KEY="sk-..."

# Set your API base (optional, defaults to Ollama)
export BANTAM_API_BASE="http://localhost:11434/v1"

# Set model (optional)
export BANTAM_MODEL="gpt-oss-20b"
```

## Usage

```bash
# Interactive mode
bantam run

# One-shot prompt
bantam prompt "your message here"

# Session management
bantam session list
bantam session clear
```

## Architecture

```
bantam/
├── cmd/              # Cobra command implementations
├── agent/            # Core agent loop
├── channel/          # Chat channel implementations (CLI, gateway)
├── provider/         # LLM provider interface
├── session/          # Session management (SQLite)
├── tools/            # Tool definitions (view, edit, list, time, echo, grep, git, memory)
├── tracing/          # OpenTelemetry integration
└── paths/            # Path configuration
```

## Size Goal

Target: 1000-2000 lines of Go code (vs nanobot's 16,800+ lines)

## License

MIT
