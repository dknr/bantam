package cmd

import (
	"context"
	"fmt"

	"github.com/dknr/bantam/agent"
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

		// Start the agent
		go ag.Start(ctx)

		// Start CLI channel with handler that processes messages
		go func() {
			err := cli.Start(ctx, func(ctx context.Context, sessionKey, chatID, content string) error {
				// Send message to agent's input channel
				select {
				case ag.InputChan <- content:
				case <-ctx.Done():
					return ctx.Err()
				}

				// Wait for response from agent's output channel
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case msg := <-ag.OutputChan:
						// Handle different message types
						switch msg.(type) {
						case agent.OutputStats:
							stats := msg.(agent.OutputStats)
							channel.PrintStatsLine(stats.Tokens, stats.DurationMs, stats.Timing)
						case agent.OutputToolStatus:
							toolStatus := msg.(agent.OutputToolStatus)
							fmt.Printf("\033[33m%s(%v)\033[0m\n", toolStatus.ToolName, toolStatus.Args)
						case agent.OutputIntermediateResponse:
							intermediate := msg.(agent.OutputIntermediateResponse)
							fmt.Printf("\033[36m> %s\033[0m\n", intermediate.Content)
						case agent.OutputError:
							errMsg := msg.(agent.OutputError)
							logger.Error(errMsg.Err, "failed to process message")
							fmt.Printf("\033[31mError: %v\033[0m\n\n", errMsg.Err)
							return errMsg.Err
						case agent.OutputFinalResponse:
							final := msg.(agent.OutputFinalResponse)
							// Use the channel's RenderMarkdown with cached terminal width
							fmt.Println(cli.RenderMarkdown(final.Content))
							return nil
						}
					}
				}
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