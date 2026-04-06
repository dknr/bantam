package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dknr/bantam/agent"
	"github.com/dknr/bantam/channel"
	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/paths"
	bantsession "github.com/dknr/bantam/session"
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

		// Extract chatID from session key
		chatID := sessionKey
		if idx := strings.Index(chatID, ":"); idx != -1 {
			chatID = chatID[idx+1:]
		}

		// Combine all args into a single message
		message := strings.Join(args, " ")

		// Start the agent
		go ag.Start(ctx)

		// Send message to agent's input channel
		select {
		case ag.InputChan <- message:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Wait for final response from agent's output channel
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg := <-ag.OutputChan:
				// Handle different message types - we only care about final response for prompt command
				switch msg.(type) {
				case agent.OutputStats:
					// Print stats line for every LLM response
					stats := msg.(agent.OutputStats)
					channel.PrintStatsLine(stats.Tokens, stats.DurationMs, stats.Timing)
				case agent.OutputToolStatus:
					// Print tool status
					toolStatus := msg.(agent.OutputToolStatus)
					fmt.Printf("\033[33m%s(%v)\033[0m\n", toolStatus.ToolName, toolStatus.Args)
				case agent.OutputIntermediateResponse:
					// Print intermediate response
					intermediate := msg.(agent.OutputIntermediateResponse)
					fmt.Printf("\033[36m> %s\033[0m\n", intermediate.Content)
				case agent.OutputError:
					// Print error
					errMsg := msg.(agent.OutputError)
					logger.Error(errMsg.Err, "failed to process prompt")
					fmt.Printf("\033[31mError: %v\033[0m\n\n", errMsg.Err)
					return errMsg.Err
				case agent.OutputFinalResponse:
					// Print final response
					final := msg.(agent.OutputFinalResponse)
					channel.PrintResponse(final.Content, nil, 0, nil) // We don't have stats/timing here, but the print function can handle nil
					return nil
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
