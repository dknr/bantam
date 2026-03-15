package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/dknr/bantam/agent"
	"github.com/dknr/bantam/channel"
	"github.com/dknr/bantam/cmd/session"
	"github.com/dknr/bantam/paths"
	"github.com/dknr/bantam/provider"
	bantsession "github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tools"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	verbose    bool
	sessionKey string
	baseDir    string
	workspace  string
	config     Config
)

// Config represents the bantam configuration
type Config struct {
	Workspace    string `yaml:"workspace"`
	SystemPrompt string `yaml:"systemPrompt,omitempty"`
	Tracing      struct {
		Endpoint    string `yaml:"endpoint"`
		ServiceName string `yaml:"serviceName"`
	} `yaml:"tracing"`
	Provider struct {
		APIKey  string `yaml:"apiKey"`
		APIBase string `yaml:"apiBase"`
		Model   string `yaml:"model"`
	} `yaml:"provider"`
}

var rootCmd = &cobra.Command{
	Use:   "bantam",
	Short: "Bantam - A lightweight agent with unified message routing",
	Long: `Bantam is a lightweight agent with unified message routing to avoid
OpenTelemetry tracing issues experienced with separate gateway/CLI code paths.

Usage:
  bantam run           - Start interactive mode
  bantam prompt        - Send a single message and exit
  bantam session       - Session management commands
  bantam init          - Initialize workspace and config`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize paths (uses baseDir/workspace flags or defaults)
		if err := paths.Init(baseDir, workspace); err != nil {
			return err
		}

		// Update baseDir/workspace from paths package
		baseDir = paths.BaseDir
		workspace = paths.WorkspaceDir

		// Load config if it exists
		if _, err := os.Stat(paths.ConfigPath); err == nil {
			configData, err := os.ReadFile(paths.ConfigPath)
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
					// Apply provider settings from config
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

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose console logging")
	rootCmd.PersistentFlags().StringVar(&sessionKey, "session", "cli:direct", "Session key to use (format: channel:chatID)")
	rootCmd.PersistentFlags().StringVar(&baseDir, "basedir", "", "Base directory for config/logs/sessions")
	rootCmd.PersistentFlags().StringVar(&workspace, "workspace", "", "Workspace directory")

	// Register session subcommand
	session.Register(rootCmd)
}

// getAgent creates and returns a configured agent
func getAgent(logger logr.Logger) (*agent.Agent, error) {
	// Create provider
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
	sessions := bantsession.NewManager(paths.SessionsDir)

	// Create tool registry
	tr := tools.NewRegistry()
	tr.Register(tools.NewShellTool())
	tr.Register(tools.NewFileSystemTool(paths.WorkspaceDir))
	tr.Register(tools.NewTimeTool())
	tr.Register(tools.NewEchoTool())

	// Determine system prompt
	systemPrompt := os.Getenv("BANTAM_SYSTEM_PROMPT")
	if systemPrompt == "" {
		if config.SystemPrompt != "" {
			systemPrompt = config.SystemPrompt
		} else {
			// Agent will read soul.md from workspace
			systemPrompt = "Read soul.md from your workspace for your identity and instructions."
		}
	}

	// Create agent
	return agent.NewWithSystemPrompt(p, tr, sessions, systemPrompt), nil
}

// getProvider creates and returns a configured provider
func getProvider(logger logr.Logger) (provider.Provider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	apiBase := os.Getenv("OPENAI_API_BASE")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-oss-20b"
	}

	if apiBase == "" {
		logger.Info("OPENAI_API_BASE not set, assuming local Ollama at http://localhost:11434/v1")
		apiBase = "http://localhost:11434/v1"
	}

	return provider.NewOpenAIProvider(apiKey, apiBase, model), nil
}

// getSessionManager creates and returns a session manager
func getSessionManager() *bantsession.Manager {
	return bantsession.NewManager(paths.SessionsDir)
}

// getCLIChannel creates and returns a CLI channel
func getCLIChannel(sessions *bantsession.Manager) *channel.CLIChannel {
	return channel.NewCLIChannel(sessions, sessionKey)
}

// printResponse prints the LLM response with stats
func printResponse(response string, tokens map[string]int, durationMs float64, timing interface{}) {
	fmt.Printf("\033[36m%s\033[0m\n", response)
	fmt.Printf("\033[90m%s | ", time.Now().Format("15:04:05"))
	printTokenStats(tokens, durationMs, timing)
	fmt.Println("\033[0m")
}

// printTokenStats prints token usage statistics
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
