package session

import (
	"fmt"

	"github.com/dknr/bantam/paths"
	"github.com/dknr/bantam/session"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear [session-key]",
	Short: "Clear a session",
	Long:  `Clear a session by removing its database file. Clears default session (cli:direct) if no key provided.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessKey := "cli:direct"
		if len(args) > 0 {
			sessKey = args[0]
		}

		sessions := session.NewManager(paths.SessionsDir)

		if err := sessions.ClearSession(sessKey); err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}

		fmt.Printf("\033[90mSession %s cleared. Type your message.\033[0m\n", sessKey)
		return nil
	},
}
