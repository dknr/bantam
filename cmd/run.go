package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/tracing"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start interactive mode",
	Long:  `Start an interactive chat session with the agent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine base directory and workspace
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("HOME"), ".bantam")
		}
		if workspace == "" {
			workspace = filepath.Join(baseDir, "workspace")
		}

		// Setup logger
		logsDir := filepath.Join(baseDir, "logs")
		logger := logging.NewLogger(logsDir, verbose)
		ctx := logging.NewContextWithLogger(context.Background(), logger)
		ctx = logging.SetVerbose(ctx, verbose)

		// Change to workspace directory
		if err := os.Chdir(workspace); err != nil {
			logger.Error(err, "failed to change to workspace directory", "workspace", workspace)
			return err
		}

		// Setup OpenTelemetry
		otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		otelServiceName := os.Getenv("OTEL_SERVICE_NAME")
		if otelServiceName == "" {
			otelServiceName = "bantam"
		}

		// Strip http:// or https:// prefix from endpoint for gRPC
		if otelEndpoint != "" {
			if len(otelEndpoint) > 7 && otelEndpoint[:7] == "http://" {
				otelEndpoint = otelEndpoint[7:]
			} else if len(otelEndpoint) > 8 && otelEndpoint[:8] == "https://" {
				otelEndpoint = otelEndpoint[8:]
			}
		}

		if err := tracing.SetupOTEL(otelEndpoint, otelServiceName); err != nil {
			logger.Error(err, "failed to setup OpenTelemetry")
		}
		defer tracing.ShutdownOTEL()

		// Setup signal handler for graceful shutdown
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Create agent
		ag, err := getAgent(logger)
		if err != nil {
			return err
		}

		// Create session manager
		sessions := getSessionManager()

		// Print startup status
		sess := sessions.GetOrCreate(sessionKey)
		msgCount := sess.MessageCount()
		if msgCount == 0 {
			fmt.Printf("\033[90mWorkspace: %s | Session: %s | New\033[0m\n", workspace, sessionKey)
		} else {
			fmt.Printf("\033[90mWorkspace: %s | Session: %s | %d messages\033[0m\n", workspace, sessionKey, msgCount)
		}

		// Create CLI channel
		cli := getCLIChannel(sessions)

		// Start CLI channel
		go func() {
			err := cli.Start(ctx, func(ctx context.Context, sessionKey, chatID, content string) error {
				response, stats, err := ag.ProcessMessageWithStats(ctx, cli.Name(), chatID, content)
				if err != nil {
					logger.Error(err, "failed to process message")
					fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
					return err
				}

				printResponse(response, stats.Tokens, float64(stats.DurationMs), stats.Timing)
				return nil
			})

			if err != nil {
				logger.Error(err, "channel error")
			}
		}()

		// Wait for shutdown signal
		<-sigChan
		fmt.Println("\nShutting down...")

		// Cancel context to stop the CLI loop
		cancel()

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
