package cmd

import (
	"fmt"
	"os"

	"github.com/dknr/bantam/agent"
	"github.com/dknr/bantam/channel"
	"github.com/dknr/bantam/cmd/session"
	"github.com/dknr/bantam/paths"
	"github.com/dknr/bantam/provider"
	bantsession "github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tools"
	"github.com/dknr/bantam/tools/memory"
	"github.com/dknr/bantam/tracing"
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
	Workspace string `yaml:"workspace"`
	Tracing   struct {
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
	Short: "Bantam - A lightweight agent with OpenAI-compatible API support",
	Long: `Bantam is a lightweight agent with OpenAI-compatible provider support.

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
					if os.Getenv("BANTAM_API_KEY") == "" && config.Provider.APIKey != "" {
						os.Setenv("BANTAM_API_KEY", config.Provider.APIKey)
					}
					if os.Getenv("BANTAM_API_BASE") == "" && config.Provider.APIBase != "" {
						os.Setenv("BANTAM_API_BASE", config.Provider.APIBase)
					}
					if os.Getenv("BANTAM_MODEL") == "" && config.Provider.Model != "" {
						os.Setenv("BANTAM_MODEL", config.Provider.Model)
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
// Returns the memoryTool for cleanup if needed.
func getAgent(logger logr.Logger) (*agent.Agent, *memory.MemoryTool, error) {
	// Create provider
	apiKey := os.Getenv("BANTAM_API_KEY")
	apiBase := os.Getenv("BANTAM_API_BASE")
	model := os.Getenv("BANTAM_MODEL")
	if model == "" {
		model = "gpt-oss-20b"
	}

	// Check if provider is configured
	if apiBase == "" {
		logger.Info("BANTAM_API_BASE not set, assuming local Ollama at http://localhost:11434/v1")
		apiBase = "http://localhost:11434/v1"
	}

	p := provider.NewOpenAIProvider(apiKey, apiBase, model)

	// Create session manager
	sessions := bantsession.NewManager(paths.SessionsDir)

	// Create tool registry
	tr := tools.NewRegistry()
	tr.Register(tools.NewExecTool())
	tr.Register(tools.NewFileTool(paths.WorkspaceDir))
	tr.Register(tools.NewTimeTool())
	tr.Register(tools.NewEchoTool())
	memoryTool, err := memory.NewMemoryTool(paths.BaseDir)
	if err != nil {
		logger.Error(err, "failed to initialize memory tool")
	} else {
		tr.Register(memoryTool)
	}

	// Create agent
	return agent.New(p, tr, sessions), memoryTool, nil
}

// getCLIChannel creates and returns a CLI channel
func getCLIChannel(sessions *bantsession.Manager) *channel.CLIChannel {
	return channel.NewCLIChannel(sessions, sessionKey)
}

// setupTracing configures OpenTelemetry tracing.
func setupTracing(logger logr.Logger) error {
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

	return tracing.SetupOTEL(otelEndpoint, otelServiceName)
}
