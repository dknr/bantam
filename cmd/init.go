package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dknr/bantam/defaults"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize workspace and config",
	Long:  `Initialize base directory, create config.yaml, and soul.md for agent identity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if workspace == "" {
			workspace = filepath.Join(os.Getenv("HOME"), ".bantam/workspace")
		}

		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("HOME"), ".bantam")
		}

		configPath := filepath.Join(baseDir, "config.yaml")
		soulPath := filepath.Join(workspace, "soul.md")

		// Create directories
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create base directory: %v\n", err)
			return err
		}

		if err := os.MkdirAll(workspace, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create workspace: %v\n", err)
			return err
		}

		// Write config
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
			return err
		}

		// Write soul.md from embedded defaults
		if err := os.WriteFile(soulPath, []byte(defaults.SoulMD), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write soul.md: %v\n", err)
			return err
		}

		fmt.Printf("Base directory: %s\n", baseDir)
		fmt.Printf("  - config.yaml: Default configuration\n")
		fmt.Printf("Workspace: %s\n", workspace)
		fmt.Printf("  - soul.md: Agent identity and instructions\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
