package channel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/glamour/v2"
	"github.com/chzyer/readline"
	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/paths"
	"github.com/dknr/bantam/provider"
	"github.com/dknr/bantam/session"
	"golang.org/x/term"
)

// CLIChannel implements the Channel interface for terminal input/output.
type CLIChannel struct {
	running    bool
	sessionMgr *session.Manager
	sessionKey string
	termWidth  int
	reader     *readline.Instance
}

// NewCLIChannel creates a new CLI channel.
func NewCLIChannel(smgr *session.Manager, sessionKey string) *CLIChannel {
	return &CLIChannel{
		sessionMgr: smgr,
		sessionKey: sessionKey,
		termWidth:  getTerminalWidthStatic(),
	}
}

// Name returns the channel name.
func (c *CLIChannel) Name() string {
	return "cli"
}

// Start begins receiving messages from stdin.
func (c *CLIChannel) Start(ctx context.Context, handler func(ctx context.Context, sessionKey, chatID, content string) error) error {
	c.running = true
	logger := logging.FromContext(ctx)

	// Setup readline with history file
	historyPath := paths.BaseDir + "/history"
	if err := os.MkdirAll(paths.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}
	var err error
	c.reader, err = readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     historyPath,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer c.reader.Close()

	// Extract session key parts
	sessionKey := c.sessionKey
	chatID := sessionKey
	if idx := strings.Index(chatID, ":"); idx != -1 {
		chatID = chatID[idx+1:]
	}

	for c.running {
		line, err := c.reader.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// readline clears the line buffer on interrupt
				// Check if there was content using Line()
				if result := c.reader.Line(); result != nil && result.Line != "" {
					// There was text on the line - just continue (line already cleared by readline)
					continue
				}
				// Empty line or no result - exit
				fmt.Println("\nGoodbye!")
				return nil
			}
			if err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				return nil
			}
			logger.Error(err, "failed to read input")
			continue
		}

		line = strings.TrimSpace(line)

		// Handle commands
		if strings.HasPrefix(line, "/") {
			if strings.EqualFold(line, "/quit") || strings.EqualFold(line, "/exit") {
				fmt.Println("Goodbye!")
				return nil
			}
			if strings.EqualFold(line, "/clear") {
				if err := c.sessionMgr.ClearSession(sessionKey); err != nil {
					fmt.Printf("Error clearing session: %v\n", err)
				} else {
					fmt.Println("Session cleared.")
				}
				continue
			}
			fmt.Printf("Unknown command: %s\n", line)
			continue
		}

		if line == "" {
			continue
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return nil
		default:
			// Process message through handler
			if err := handler(ctx, sessionKey, chatID, line); err != nil {
				logger.Error(err, "failed to process message")
				fmt.Printf("Error: %v\n", err)
				continue
			}
		}
	}

	return nil
}

// Stop ends the channel.
func (c *CLIChannel) Stop() error {
	c.running = false
	if c.reader != nil {
		c.reader.Close()
	}
	return nil
}

// customDarkStyle is the dark style with no margins on document or code blocks
const customDarkStyle = `{
  "document": {
    "block_prefix": "\n",
    "block_suffix": "\n",
    "color": "252"
  },
  "block_quote": {
    "indent": 1,
    "indent_token": "│ "
  },
  "paragraph": {},
  "list": {
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "color": "39",
    "bold": true
  },
  "h1": {
    "prefix": " ",
    "suffix": " ",
    "color": "228",
    "background_color": "63",
    "bold": true
  },
  "h2": {
    "prefix": "## "
  },
  "h3": {
    "prefix": "### "
  },
  "h4": {
    "prefix": "#### "
  },
  "h5": {
    "prefix": "##### "
  },
  "h6": {
    "prefix": "###### ",
    "color": "35",
    "bold": false
  },
  "text": {},
  "strikethrough": {
    "crossed_out": true
  },
  "emph": {
    "italic": true
  },
  "strong": {
    "bold": true
  },
  "hr": {
    "color": "240",
    "format": "\n--------\n"
  },
  "item": {
    "block_prefix": "• "
  },
  "enumeration": {
    "block_prefix": ". "
  },
  "task": {
    "ticked": "[✓] ",
    "unticked": "[ ] "
  },
  "link": {
    "color": "30",
    "underline": true
  },
  "link_text": {
    "color": "35",
    "bold": true
  },
  "image": {
    "color": "212",
    "underline": true
  },
  "image_text": {
    "color": "243",
    "format": "Image: {{.text}} →"
  },
  "code": {
    "prefix": " ",
    "suffix": " ",
    "color": "203",
    "background_color": "236"
  },
  "code_block": {
    "color": "244",
    "chroma": {
      "text": {
        "color": "#C4C4C4"
      },
      "error": {
        "color": "#F1F1F1",
        "background_color": "#F05B5B"
      },
      "comment": {
        "color": "#676767"
      },
      "comment_preproc": {
        "color": "#FF875F"
      },
      "keyword": {
        "color": "#00AAFF"
      },
      "keyword_reserved": {
        "color": "#FF5FD2"
      },
      "keyword_namespace": {
        "color": "#FF5F87"
      },
      "keyword_type": {
        "color": "#6E6ED8"
      },
      "operator": {
        "color": "#EF8080"
      },
      "punctuation": {
        "color": "#E8E8A8"
      },
      "name": {
        "color": "#C4C4C4"
      },
      "name_builtin": {
        "color": "#FF8EC7"
      },
      "name_tag": {
        "color": "#B083EA"
      },
      "name_attribute": {
        "color": "#7A7AE6"
      },
      "name_class": {
        "color": "#F1F1F1",
        "underline": true,
        "bold": true
      },
      "name_constant": {},
      "name_decorator": {
        "color": "#FFFF87"
      },
      "name_exception": {},
      "name_function": {
        "color": "#00D787"
      },
      "name_other": {},
      "literal": {},
      "literal_number": {
        "color": "#6EEFC0"
      },
      "literal_date": {},
      "literal_string": {
        "color": "#C69669"
      },
      "literal_string_escape": {
        "color": "#AFFFD7"
      },
      "generic_deleted": {
        "color": "#FD5B5B"
      },
      "generic_emph": {
        "italic": true
      },
      "generic_inserted": {
        "color": "#00D787"
      },
      "generic_strong": {
        "bold": true
      },
      "generic_subheading": {
        "color": "#777777"
      },
      "background": {
        "background_color": "#373737"
      }
    }
  },
  "table": {},
  "definition_list": {},
  "definition_term": {},
  "definition_description": {
    "block_prefix": "\n🠶 "
  },
  "html_block": {},
  "html_span": {}
}`

// RenderMarkdown renders markdown text with glamour terminal rendering.
// Returns plain text fallback on error.
// Uses the CLIChannel's cached terminal width if available, otherwise uses static detection.
func (c *CLIChannel) RenderMarkdown(text string) string {
	width := c.getTerminalWidthStatic()

	// Create glamour renderer with custom dark style (no margins)
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(customDarkStyle)),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text // fallback to plain text
	}

	// Render the markdown
	result, err := r.Render(text)
	r.Close()
	if err != nil {
		return text // fallback to plain text
	}

	return result
}

// RenderMarkdownStatic renders markdown text with glamour terminal rendering using static detection.
// Returns plain text fallback on error.
func RenderMarkdownStatic(text string) string {
	width := getTerminalWidthStatic()

	// Create glamour renderer with custom dark style (no margins)
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(customDarkStyle)),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text // fallback to plain text
	}

	// Render the markdown
	result, err := r.Render(text)
	r.Close()
	if err != nil {
		return text // fallback to plain text
	}

	return result
}

// PrintStatus prints the startup status line.
func PrintStatus(workspace, sessionKey string, msgCount int) {
	if msgCount == 0 {
		fmt.Printf("\033[90mWorkspace: %s | Session: %s | New\033[0m\n", workspace, sessionKey)
	} else {
		fmt.Printf("\033[90mWorkspace: %s | Session: %s | %d messages\033[0m\n", workspace, sessionKey, msgCount)
	}
}

// PrintTokenStats prints token usage statistics.
func PrintTokenStats(tokens map[string]int, durationMs float64, timing interface{}) {
	inputTokens := 0
	outputTokens := 0
	if v, ok := tokens["prompt"]; ok {
		inputTokens = v
	}
	if v, ok := tokens["completion"]; ok {
		outputTokens = v
	}
	totalTokens := inputTokens + outputTokens

	if timingStruct, ok := timing.(*provider.Timing); ok {
		if timingStruct != nil && timingStruct.PromptPerSecond > 0 && timingStruct.PredictedPerSecond > 0 {
			fmt.Printf("%d (%.1f/s) => %d (%.1f/s) => %d (%.1fs)", inputTokens, timingStruct.PromptPerSecond, outputTokens, timingStruct.PredictedPerSecond, totalTokens, durationMs/1000)
			return
		}
	}

	fmt.Printf("%d => %d => %d tokens (%.1fs)", inputTokens, outputTokens, totalTokens, durationMs/1000)
}

// PrintResponse prints an LLM response with token stats.
func PrintResponse(response string, tokens map[string]int, durationMs float64, timing interface{}) {
	// Print markdown-formatted response
	fmt.Println(RenderMarkdownStatic(response))
	// Print stats line in gray
	fmt.Printf("\033[90m%s | ", time.Now().Format("15:04:05"))
	printTokenStats(tokens, durationMs, timing)
	fmt.Println("\033[0m")
}

// printTokenStats prints token usage statistics.
func printTokenStats(tokens map[string]int, durationMs float64, timing interface{}) {
	inputTokens := 0
	outputTokens := 0
	if v, ok := tokens["prompt"]; ok {
		inputTokens = v
	}
	if v, ok := tokens["completion"]; ok {
		outputTokens = v
	}
	totalTokens := inputTokens + outputTokens

	if timingStruct, ok := timing.(*provider.Timing); ok {
		if timingStruct != nil && timingStruct.PromptPerSecond > 0 && timingStruct.PredictedPerSecond > 0 {
			fmt.Printf("%d (%.1f/s) => %d (%.1f/s) => %d (%.1fs)", inputTokens, timingStruct.PromptPerSecond, outputTokens, timingStruct.PredictedPerSecond, totalTokens, durationMs/1000)
			return
		}
	}

	fmt.Printf("%d => %d => %d tokens (%.1fs)", inputTokens, outputTokens, totalTokens, durationMs/1000)
}

// getTerminalWidthStatic attempts to get the terminal width, returns 80 on error.
func (c *CLIChannel) getTerminalWidthStatic() int {
	if c.termWidth > 0 {
		return c.termWidth
	}
	return getTerminalWidthStatic()
}

// getTerminalWidthStatic attempts to get the terminal width, returns 80 on error.
func getTerminalWidthStatic() int {
	// Try to read environment variables first
	if w := getEnvWidth("COLUMNS"); w > 0 {
		return w
	}

	// Try to get terminal size from stdout
	if term.IsTerminal(int(os.Stdout.Fd())) {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil && width > 0 {
			return width
		}
	}

	// Use a reasonable default
	return 80
}

// DebugTerminalWidth returns the detected terminal width (for debugging).
func DebugTerminalWidth() int {
	return getTerminalWidthStatic()
}

// GetTerminalWidth returns the terminal width for the current process.
func GetTerminalWidth() int {
	return getTerminalWidthStatic()
}

// NewCLIChannelWithWidth creates a new CLI channel with the specified terminal width.
func NewCLIChannelWithWidth(smgr *session.Manager, sessionKey string, termWidth int) *CLIChannel {
	return &CLIChannel{
		sessionMgr: smgr,
		sessionKey: sessionKey,
		termWidth:  termWidth,
	}
}

// getEnvWidth parses an environment variable as an integer width.
func getEnvWidth(key string) int {
	val := os.Getenv(key)
	var w int
	for _, c := range val {
		if c >= '0' && c <= '9' {
			w = w*10 + int(c-'0')
		}
	}
	return w
}
