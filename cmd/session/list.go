package session

import (
	"fmt"

	"github.com/dknr/bantam/paths"
	"github.com/dknr/bantam/session"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Long:  `List all existing sessions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessions := session.NewManager(paths.SessionsDir)
		sessionsList := sessions.ListSessions()

		if len(sessionsList) == 0 {
			fmt.Println("No sessions found.")
		} else {
			fmt.Println("Sessions:")
			for _, s := range sessionsList {
				fmt.Printf("  - %s\n", s)
			}
		}
		return nil
	},
}
