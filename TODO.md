# Bantam TODO

## Memory Feature (SQLite-based)

### Phase 1: MVP (In Progress)
- [ ] Create `tools/memory/db.go` - SQLite connection, schema initialization
- [ ] Create `tools/memory/memory.go` - Tool definitions with noun-verb naming:
  - `memory_read` - Get fact by key
  - `memory_write` - Insert/update with compare-exchange (old + new value)
  - `history_search` - Search history entries
  - `memory_list` - List all memory keys
  - `history_since` - Get entries since timestamp
- [ ] Register memory tool in `cmd/root.go`
- [ ] Add schema to `tools/tools.go` DefinitionsWithSchema()

### Later Phases
- [ ] Phase 2: CLI Commands (`/mem read`, `/mem list`, `/mem history`)
- [ ] Phase 3: Auto-Consolidation (append to history, extract facts)
- [ ] Phase 4: Vector Search (embeddings, semantic search)

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
- **Bug fix** - Removed double span.End() call that was causing segfault

## Bugs to Fix

### 1. Token Stats Showing 0
**Issue:** Status line shows "0 tokens (0 in, 0 out)" instead of actual token counts from response
**Plan:**
- Check how token counts are extracted from the response
- Verify the response contains token information
- Fix token extraction to use correct keys

### 2. Working Directory Not Set to Workspace
**Issue:** Agent doesn't know its workspace directory, gets "lost"
**Plan:**
- Set working directory to configured workspace on startup
- Pass workspace to tools (file already does this)
- Ensure all operations use absolute paths within workspace

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
