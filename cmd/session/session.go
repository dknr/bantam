package session

import (
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Session management commands",
	Long:  `Manage chat sessions - list, clear, or delete them.`,
}

func init() {
	sessionCmd.AddCommand(clearCmd)
	sessionCmd.AddCommand(listCmd)
}

// Register adds the session command to the parent command.
func Register(parent *cobra.Command) {
	parent.AddCommand(sessionCmd)
}
