# Bantam TODO

## Completed
- Config file support with `--init-config`, `--prompt`, `--clear` flags
- Test tools: `time` and `echo`
- Fixed tool call argument parsing
- Fixed missing assistant messages bug (tool calls preserved in session)
- Fixed tool errors (errors added to session for LLM self-correction)
- Session management: `--session`, `--clear`, `--clear-all`, `--list-sessions` flags
- Removed bantam binary from git history using git-filter-repo
- OpenTelemetry: OTLP gRPC exporter with no-op when endpoint not configured
- CLI `/clear` command fixed - Pass session manager to CLI channel, added `ClearSession()` method to Manager
- System prompt implementation - Add embedded default, config file support, and environment variable override
- **Structural cleanup** - Moved `main/main.go` → `bantam.go` (root), `main/defaults/` → `defaults/` (root)

## High Priority

### 1. Multi-Session Support in CLI
**Issue:** CLI uses single chatID ("direct"), can't manage multiple sessions
**Plan:**
- User can exit CLI and restart with different session via `--session` flag
- Add `/session <key>` command if needed in future

## Medium Priority

### 2. Span Attributes for Tracing
**Issue:** Traced operations don't include useful attributes for debugging/analysis
**Plan:**
- Add `model` attribute to LLM spans
- Add `messages_count` attribute to process_message span
- Add `tools_count` attribute to tool execution spans
- Add token count attributes (input_tokens, output_tokens)

### 3. Better Tool Schema Management
**Issue:** Tool schemas are hardcoded in `DefinitionsWithSchema()`
**Plan:**
- Add schema definition methods to Tool interface
- Auto-generate schemas from tool parameters
- Support optional vs required parameters

## Low Priority

### 4. Session Persistence Verification
**Issue:** Verify `GetOrCreate()` properly loads from disk on restart
**Plan:**
- Test session persistence across multiple runs
- Confirm session key is preserved

### 5. Tool Output Formatting
**Issue:** Tool results are JSON strings in session, not pretty-printed
**Plan:**
- Parse tool results and format for better readability
- Store both raw and formatted results
