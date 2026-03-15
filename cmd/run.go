package cmd

import (
	"context"
	"fmt"

	"github.com/dknr/bantam/channel"
	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/paths"
	bantsession "github.com/dknr/bantam/session"
	"github.com/dknr/bantam/tracing"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start interactive mode",
	Long:  `Start an interactive chat session with the agent.`,
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

		// Create CLI channel
		cli := channel.NewCLIChannel(sessions, sessionKey)

		// Start CLI channel with handler that processes messages
		go func() {
			err := cli.Start(ctx, func(ctx context.Context, sessionKey, chatID, content string) error {
				response, stats, err := ag.ProcessMessageWithStats(ctx, cli.Name(), chatID, content)
				if err != nil {
					logger.Error(err, "failed to process message")
					fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
					return err
				}

				channel.PrintResponse(response, stats.Tokens, float64(stats.DurationMs), stats.Timing)
				return nil
			})

			if err != nil {
				logger.Error(err, "channel error")
			}
		}()

		// Wait for context cancellation (from /quit command)
		<-ctx.Done()

		// Cleanup
		if err := cli.Stop(); err != nil {
			logger.Error(err, "failed to stop channel")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
