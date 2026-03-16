package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

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
		ctx, cancel := context.WithCancel(context.Background())
		ctx = logging.NewContextWithLogger(ctx, logger)
		ctx = logging.SetVerbose(ctx, verbose)

		// Setup OpenTelemetry
		if err := setupTracing(logger); err != nil {
			return err
		}
		defer tracing.ShutdownOTEL()

		// Create agent
		ag, memoryTool, err := getAgent(logger)
		if err != nil {
			return err
		}
		defer func() {
			if memoryTool != nil {
				memoryTool.Close()
			}
		}()

		// Create session manager
		sessions := bantsession.NewManager(paths.SessionsDir)

		// Print startup status
		sess := sessions.GetOrCreate(sessionKey)
		msgCount := sess.MessageCount()
		channel.PrintStatus(paths.WorkspaceDir, sessionKey, msgCount)

		// Capture terminal width at startup (stdout is still a terminal)
		termWidth := channel.GetTerminalWidth()

		// Create CLI channel
		cli := channel.NewCLIChannelWithWidth(sessions, sessionKey, termWidth)

		// Set up signal handler for graceful exit
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		go func() {
			<-sigChan
			cancel()
		}()

		// Start CLI channel with handler that processes messages
		go func() {
			err := cli.Start(ctx, func(ctx context.Context, sessionKey, chatID, content string) error {
				response, stats, err := ag.ProcessMessageWithStats(ctx, cli.Name(), chatID, content)
				if err != nil {
					logger.Error(err, "failed to process message")
					fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
					return err
				}

				// Use the channel's RenderMarkdown with cached terminal width
				fmt.Println(cli.RenderMarkdown(response))
				// Print stats line in gray
				fmt.Printf("\033[90m%s | ", time.Now().Format("15:04:05"))
				channel.PrintTokenStats(stats.Tokens, float64(stats.DurationMs), stats.Timing)
				fmt.Println("\033[0m")
				return nil
			})

			if err != nil {
				logger.Error(err, "channel error")
			}
			// Channel exited (EOF, /quit, etc.) - cancel context to exit
			cancel()
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
