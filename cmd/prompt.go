package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dknr/bantam/logging"
	"github.com/dknr/bantam/tracing"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt [message]",
	Short: "Send a single message and exit",
	Long:  `Send a message to the agent and print the response, then exit.`,
	Args:  cobra.MinimumNArgs(1),
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

		// Combine all args into a single message
		message := strings.Join(args, " ")

		// Create agent
		ag, err := getAgent(logger)
		if err != nil {
			return err
		}

		// Extract chatID from session key
		chatID := sessionKey
		if idx := strings.Index(chatID, ":"); idx != -1 {
			chatID = chatID[idx+1:]
		}

		// Process message
		response, stats, err := ag.ProcessMessageWithStats(ctx, "cli", chatID, message)
		if err != nil {
			logger.Error(err, "failed to process prompt")
			fmt.Printf("\033[31mError: %v\033[0m\n\n", err)
			return err
		}

		// Print response
		printResponse(response, stats.Tokens, float64(stats.DurationMs), stats.Timing)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
