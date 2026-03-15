package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dknr/bantam/channel"
	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/paths"
	bantsession "github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tracing"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt [message]",
	Short: "Send a single message and exit",
	Long:  `Send a message to the agent and print the response, then exit.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup logger
		logger := logging.NewLogger(paths.LogsDir, verbose)
		ctx := logging.NewContextWithLogger(context.Background(), logger)
		ctx = logging.SetVerbose(ctx, verbose)

		// Setup OpenTelemetry
		if err := setupTracing(logger); err != nil {
			return err
		}
		defer tracing.ShutdownOTEL()

		// Create agent
		ag, err := getAgent(logger)
		if err != nil {
			return err
		}

		// Create session manager
		sessions := bantsession.NewManager(paths.SessionsDir)

		// Print startup status
		sess := sessions.GetOrCreate(sessionKey)
		msgCount := sess.MessageCount()
		channel.PrintStatus(paths.WorkspaceDir, sessionKey, msgCount)

		// Extract chatID from session key
		chatID := sessionKey
		if idx := strings.Index(chatID, ":"); idx != -1 {
			chatID = chatID[idx+1:]
		}

		// Combine all args into a single message
		message := strings.Join(args, " ")

		// Process message
		response, stats, err := ag.ProcessMessageWithStats(ctx, "cli", chatID, message)
		if err != nil {
			logger.Error(err, "failed to process prompt")
			fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
			return err
		}

		// Print response
		channel.PrintResponse(response, stats.Tokens, float64(stats.DurationMs), stats.Timing)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
