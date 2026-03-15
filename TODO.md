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

## Pending
- Add span attributes to tracked operations (model, messages_count, tools_count, tokens)
- Add `otlp/otlphttp` exporter option for HTTP transportelemetry tracing with console exporter

## Pending Work

### 0. Remove bantam binary from git history
**Priority:** Medium
**Issue:** Build artifact committed to repo
**Plan:**
1. Clone clean repo from remote
2. Install git-filter-repo (done via pip3)
3. Use `git filter-repo --path bantam --invert-paths` to remove binary
4. Force push to remote
5. Add `bantam` to `.gitignore` in new repo clone

### 1. OpenTelemetry Console Exporter Not Showing Spans
**Issue:** Console exporter is configured but spans aren't visible in terminal output
**Plan:**
- Verify stdouttrace exporter configuration
- Check if ForceFlush is being called after each span
- Test with `OTEL_EXPORTER_OTLP_ENDPOINT=console` or similar

### 2. Context Window Management
**Priority:** High
**Issue:** No limit on conversation history size
**Plan:**
- Add config option for max history size
- Implement sliding window (remove oldest messages)
- Consider token-based truncation

### 2. CLI `/clear` Command
**Priority:** Medium
**Issue:** `/clear` command in CLI prints a message but doesn't actually clear the session
**Plan:**
- Update CLI channel to call `sessionMgr.ClearSession()` or reset session
- Need to track current session key in CLI channel

### 3. Multi-Session Support in CLI
**Priority:** Medium
**Issue:** CLI uses single chatID ("direct"), can't manage multiple sessions
**Plan:**
- Add `/session <key>` command to switch sessions
- Show current session in prompt
- `/list-sessions` command in CLI

### 4. Better Tool Schema Management
**Priority:** Medium
**Issue:** Tool schemas are hardcoded in `DefinitionsWithSchema()`
**Plan:**
- Add schema definition methods to Tool interface
- Auto-generate schemas from tool parameters
- Support optional vs required parameters

### 5. Session Persistence Verification
**Priority:** Low
**Issue:** Verify `GetOrCreate()` properly loads from disk on restart
**Plan:**
- Test session persistence across multiple runs
- Confirm session key is preserved

### 6. OpenTelemetry Implementation
**Priority:** Low
**Issue:** OTEL is a placeholder - returns nil
**Plan:**
- Research OpenTelemetry Go SDK compatibility
- Implement actual tracing spans

### 7. Tool Output Formatting
**Priority:** Low
**Issue:** Tool results are JSON strings in session, not pretty-printed
**Plan:**
- Parse tool results and format for better readability
- Store both raw and formatted results
