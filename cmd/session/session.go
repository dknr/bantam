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
	rootCmd.AddCommand(sessionCmd)
}
