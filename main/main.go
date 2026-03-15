// Package main provides the Bantam CLI entry point.
//
// Bantam is a lightweight agent with unified message routing.
//
// Usage:
//   bantam [--verbose]
//   bantam --prompt "your message"
//   bantam --session "channel:id"
//   bantam --clear
//   bantam --clear-all
//   bantam --list-sessions
//   bantam --init-config
//   bantam --init
//
// Flags:
//   --verbose       Enable verbose console logging
//   --prompt        Send a single message and exit
//   --session       Session key to use (default: cli:direct)
//   --clear         Clear default session (cli:direct) and exit
//   --clear-all     Clear all sessions and exit
//   --list-sessions List all sessions and exit
//   --init-config   Write default config file and exit
//   --init          Initialize workspace (config + soul.md) and exit
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dknr/bantam/agent"
	"github.com/dknr/bantam/channel"
	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/provider"
	"github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tools"
	"github.com/dknr/bantam/tracing"
	"gopkg.in/yaml.v3"
)

//go:embed defaults/soul.md
var defaultSoul string

// Config represents the bantam configuration
type Config struct {
	Workspace      string `yaml:"workspace"`
	SystemPrompt   string `yaml:"systemPrompt,omitempty"`
	Tracing        struct {
		Endpoint    string `yaml:"endpoint"`
		ServiceName string `yaml:"serviceName"`
	} `yaml:"tracing"`
	Provider struct {
		APIKey  string `yaml:"apiKey"`
		APIBase string `yaml:"apiBase"`
		Model   string `yaml:"model"`
	} `yaml:"provider"`
}

func main() {
	// Parse flags
	verbose := flag.Bool("verbose", false, "Enable verbose console logging")
	prompt := flag.String("prompt", "", "Send a single message and exit")
	sessionKey := flag.String("session", "cli:direct", "Session key to use (format: channel:chatID)")
	listSessions := flag.Bool("list-sessions", false, "List all sessions and exit")
	clear := flag.Bool("clear", false, "Clear default session (cli:direct) and exit")
	clearAll := flag.Bool("clear-all", false, "Clear all sessions and exit")
	initConfig := flag.Bool("init-config", false, "Write default config file and exit")
	initWorkspace := flag.Bool("init", false, "Initialize workspace (config + soul.md) and exit")
	flag.Parse()

// Handle --init flag first (before any other logic)
 	if *initWorkspace {
 		workspace := os.Getenv("BANTAM_WORKSPACE")
 		if workspace == "" {
 			workspace = os.Getenv("HOME") + "/.bantam"
 		}

 		if *prompt != "" || *clear {
 			fmt.Fprintln(os.Stderr, "Error: --init cannot be combined with other operations")
 			os.Exit(1)
 		}

 		configPath := workspace + "/config.yaml"
 		soulPath := workspace + "/soul.md"
 		if _, err := os.Stat(configPath); err == nil {
 			fmt.Fprintf(os.Stderr, "Error: Config file already exists at %s\n", configPath)
			os.Exit(1)
		}
		if _, err := os.Stat(soulPath); err == nil {
			fmt.Fprintf(os.Stderr, "Error: soul.md already exists at %s\n", soulPath)
			os.Exit(1)
		}

		if err := os.MkdirAll(workspace, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create workspace: %v\n", err)
			os.Exit(1)
		}

		configContent := `workspace: ` + workspace + `
systemPrompt: |
  Read soul.md from your workspace for your identity and instructions.
tracing:
  endpoint: ""
  serviceName: bantam
provider:
  apiKey: ""
  apiBase: http://localhost:11434/v1
  model: gpt-oss-20b
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write config: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(soulPath, []byte(defaultSoul), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write soul.md: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Workspace initialized at %s\n", workspace)
		fmt.Printf("  - config.yaml: Default configuration\n")
		fmt.Printf("  - soul.md: Agent identity and instructions\n")
		os.Exit(0)
	}

	// Handle --init-config flag (legacy, config only)
	if *initConfig {
		workspace := os.Getenv("HOME") + "/.bantam"

		if *prompt != "" || *clear {
			fmt.Fprintln(os.Stderr, "Error: --init-config cannot be combined with other operations")
			os.Exit(1)
		}

		configPath := workspace + "/config.yaml"
		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stderr, "Error: Config file already exists at %s\n", configPath)
			os.Exit(1)
		}

		if err := os.MkdirAll(workspace, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create workspace: %v\n", err)
			os.Exit(1)
		}

		configContent := `workspace: ` + workspace + `
tracing:
  endpoint: ""
  serviceName: bantam
provider:
  apiKey: ""
  apiBase: http://localhost:11434/v1
  model: gpt-oss-20b
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Default config written to %s\n", configPath)
		os.Exit(0)
	}

	// Get workspace from environment or use default
	workspace := os.Getenv("BANTAM_WORKSPACE")
	if workspace == "" {
		workspace = os.Getenv("HOME") + "/.bantam"
	}

	// Load config file if it exists
	configPath := workspace + "/config.yaml"
	var config Config
	if _, err := os.Stat(configPath); err == nil {
		configData, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read config file: %v\n", err)
		} else {
			if err := yaml.Unmarshal(configData, &config); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse config file: %v\n", err)
			} else {
				// Override workspace from config if set
				if config.Workspace != "" {
					workspace = config.Workspace
				}
				// Apply provider settings from config (only if not set via env var)
				if os.Getenv("OPENAI_API_KEY") == "" && config.Provider.APIKey != "" {
					os.Setenv("OPENAI_API_KEY", config.Provider.APIKey)
				}
				if os.Getenv("OPENAI_API_BASE") == "" && config.Provider.APIBase != "" {
					os.Setenv("OPENAI_API_BASE", config.Provider.APIBase)
				}
				if os.Getenv("OPENAI_MODEL") == "" && config.Provider.Model != "" {
					os.Setenv("OPENAI_MODEL", config.Provider.Model)
				}
				// Apply tracing settings from config
				if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && config.Tracing.Endpoint != "" {
					os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", config.Tracing.Endpoint)
				}
				if os.Getenv("OTEL_SERVICE_NAME") == "" && config.Tracing.ServiceName != "" {
					os.Setenv("OTEL_SERVICE_NAME", config.Tracing.ServiceName)
				}
			}
		}
	}

	// Setup logging with logs directory
	logsDir := workspace + "/logs"
	logger := logging.NewLogger(logsDir, *verbose)
	ctx := logging.NewContextWithLogger(context.Background(), logger)
	ctx = logging.SetVerbose(ctx, *verbose)

	// Setup OpenTelemetry
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	otelServiceName := os.Getenv("OTEL_SERVICE_NAME")
	if otelServiceName == "" {
		otelServiceName = "bantam"
	}

	// Strip http:// or https:// prefix from endpoint for gRPC
	if otelEndpoint != "" {
		if len(otelEndpoint) > 7 && otelEndpoint[:7] == "http://" {
			otelEndpoint = otelEndpoint[7:]
		} else if len(otelEndpoint) > 8 && otelEndpoint[:8] == "https://" {
			otelEndpoint = otelEndpoint[8:]
		}
	}

	if err := tracing.SetupOTEL(otelEndpoint, otelServiceName); err != nil {
		logger.Error(err, "failed to setup OpenTelemetry")
	}

	// Create provider (OpenAI-compatible)
	apiKey := os.Getenv("OPENAI_API_KEY")
	apiBase := os.Getenv("OPENAI_API_BASE")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-oss-20b"
	}

	// Check if provider is configured
	if apiBase == "" {
		logger.Info("OPENAI_API_BASE not set, assuming local Ollama at http://localhost:11434/v1")
		apiBase = "http://localhost:11434/v1"
	}

	p := provider.NewOpenAIProvider(apiKey, apiBase, model)

	// Create session manager
	sessions := session.NewManager(workspace)

	// Create tool registry
	tr := tools.NewRegistry()
	tr.Register(tools.NewShellTool())
	tr.Register(tools.NewFileSystemTool(workspace))
	tr.Register(tools.NewTimeTool())
	tr.Register(tools.NewEchoTool())

	// Determine system prompt (env var > config > default)
	systemPrompt := os.Getenv("BANTAM_SYSTEM_PROMPT")
	if systemPrompt == "" && config.SystemPrompt != "" {
		systemPrompt = config.SystemPrompt
	}
	if systemPrompt == "" {
		systemPrompt = "Read soul.md from your workspace for your identity and instructions."
	}

	// Create agent
	ag := agent.NewWithSystemPrompt(p, tr, sessions, systemPrompt)

	// Handle --list-sessions flag
	if *listSessions {
		sessionsList := sessions.ListSessions()
		if len(sessionsList) == 0 {
			fmt.Println("No sessions found.")
		} else {
			fmt.Println("Sessions:")
			for _, s := range sessionsList {
				fmt.Printf("  - %s\n", s)
			}
		}
		os.Exit(0)
	}

	// Handle --clear flag (clear default session only)
	if *clear {
		sessionPath := sessions.SessionPath(*sessionKey)
		if err := os.Remove(sessionPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Session %s does not exist.\n", *sessionKey)
			} else {
				logger.Error(err, "failed to clear session", "session", *sessionKey)
				os.Exit(1)
			}
		}
		fmt.Printf("Session %s cleared.\n", *sessionKey)
		os.Exit(0)
	}

	// Handle --clear-all flag
	if *clearAll {
		if err := sessions.Clear(); err != nil {
			logger.Error(err, "failed to clear sessions")
			os.Exit(1)
		}
		fmt.Println("All sessions cleared.")
		os.Exit(0)
	}

	// Handle --prompt flag (one-shot mode)
	if *prompt != "" {
		// Extract chatID from session key (remove channel prefix if present)
		chatID := *sessionKey
		if idx := strings.Index(chatID, ":"); idx != -1 {
			chatID = chatID[idx+1:]
		}
		response, stats, err := ag.ProcessMessageWithStats(ctx, "cli", chatID, *prompt)
		if err != nil {
			logger.Error(err, "failed to process prompt")
			fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
			os.Exit(1)
		}

		// Print response with header
		printResponse(ctx, response, stats.Tokens, float64(stats.DurationMs))
		os.Exit(0)
	}

	// Create CLI channel
	cli := channel.NewCLIChannel(sessions)

	// Create a context that can be cancelled for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start CLI channel
	go func() {
		err := cli.Start(ctx, func(ctx context.Context, chatID, content string) error {
			// Process message through unified agent loop
			logger := logging.FromContext(ctx)
			response, stats, err := ag.ProcessMessageWithStats(ctx, cli.Name(), chatID, content)
			if err != nil {
				logger.Error(err, "failed to process message")
				fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
				return err
			}

			// Print response with header
			printResponse(ctx, response, stats.Tokens, float64(stats.DurationMs))

			// Print prompt once
			fmt.Print("> ")
			return nil
		})

		if err != nil {
			logger.Error(err, "channel error")
		}
	}()

// Wait for shutdown signal
 	<-sigChan
 	fmt.Println("\nShutting down...")

 	// Cancel context to stop the CLI loop
 	cancel()

 	// Cleanup
 	if err := cli.Stop(); err != nil {
 		logger.Error(err, "failed to stop channel")
 	}
 	if err := tracing.ShutdownOTEL(); err != nil {
 		logger.Error(err, "failed to shutdown OpenTelemetry")
 	}
 }

// printResponse prints the LLM response with a header showing time and token stats
func printResponse(ctx context.Context, response string, tokens map[string]int, durationMs float64) {
	fmt.Printf("\n\033[90m%s | ", time.Now().Format("15:04:05"))
	printTokenStats(tokens, durationMs)
	fmt.Println("\033[0m")
	fmt.Printf("\033[36m%s\033[0m\n\n", response)
}

// printTokenStats prints token usage statistics
func printTokenStats(tokens map[string]int, durationMs float64) {
	inputTokens := tokens["input_tokens"]
	outputTokens := tokens["output_tokens"]
	totalTokens := inputTokens + outputTokens
	fmt.Printf("%d tokens (%d in, %d out) | %.1fs", totalTokens, inputTokens, outputTokens, durationMs/1000)
}
