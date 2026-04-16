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
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"strings"
)

var (
	sessionKey string
	baseDir    string
	workspace  string
	config     Config
)

// Config represents the bantam configuration
type Config struct {
	Workspace string `yaml:"workspace"`
	Provider  struct {
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
	rootCmd.PersistentFlags().StringVar(&sessionKey, "session", "cli:direct", "Session key to use (format: channel:chatID)")
	rootCmd.PersistentFlags().StringVar(&baseDir, "basedir", "", "Base directory for config/logs/sessions")
	rootCmd.PersistentFlags().StringVar(&workspace, "workspace", "", "Workspace directory")

	// Register session subcommand
	session.Register(rootCmd)
}

// getAgent creates and returns a configured agent
// Returns the memoryTool for cleanup if needed.
func getAgent() (*agent.Agent, *memory.MemoryTool, error) {
	// Create provider
	apiKey := os.Getenv("BANTAM_API_KEY")
	apiBase := os.Getenv("BANTAM_API_BASE")
	model := os.Getenv("BANTAM_MODEL")
	if model == "" {
		model = "gpt-oss-20b"
	}

	// Check if provider is configured
	if apiBase == "" {
		apiBase = "http://localhost:11434/v1"
	}

	p := provider.NewOpenAIProvider(apiKey, apiBase, model)

	// Create session manager
	sessions := bantsession.NewManager(paths.SessionsDir)

	// Create tool registry
	tr := tools.NewRegistry()
	tr.Register(tools.NewCatTool(paths.WorkspaceDir))
	tr.Register(tools.NewSedTool(paths.WorkspaceDir))
	tr.Register(tools.NewLsTool(paths.WorkspaceDir))
	tr.Register(tools.NewTimeTool())
	tr.Register(tools.NewEchoTool())
	tr.Register(tools.NewGrepTool(paths.WorkspaceDir))
	tr.Register(tools.NewGitTool(paths.WorkspaceDir))
	tr.Register(tools.NewShTool(paths.WorkspaceDir))
	memoryTool, err := memory.NewMemoryTool(paths.BaseDir)
	if err != nil {
	} else {
		tr.Register(memoryTool)
	}

	// Create agent with channel-based communication
	// Extract channel and chatID from sessionKey (format: channel:chatID)
	sessionKey := sessionKey
	channel := "cli"
	chatID := sessionKey
	if idx := strings.Index(chatID, ":"); idx != -1 {
		channel = chatID[:idx]
		chatID = chatID[idx+1:]
	}
	return agent.New(p, tr, sessions, channel, chatID), memoryTool, nil
}

// getCLIChannel creates and returns a CLI channel
func getCLIChannel(sessions *bantsession.Manager) *channel.CLIChannel {
	return channel.NewCLIChannel(sessions, sessionKey)
}
