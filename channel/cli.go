package channel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dknr/bantam/logging"
)

// CLIChannel implements the Channel interface for terminal input/output.
type CLIChannel struct {
	running bool
}

// NewCLIChannel creates a new CLI channel.
func NewCLIChannel() *CLIChannel {
	return &CLIChannel{}
}

// Name returns the channel name.
func (c *CLIChannel) Name() string {
	return "cli"
}

// Start begins receiving messages from stdin.
func (c *CLIChannel) Start(ctx context.Context, handler func(ctx context.Context, chatID, content string) error) error {
	c.running = true
	logger := logging.FromContext(ctx)

	reader := bufio.NewReader(os.Stdin)
	chatID := "direct" // CLI is always direct chat

	fmt.Println("Bantam CLI - Type your message (or /quit to exit)")
	fmt.Print("> ")

	for c.running {
		line, err := reader.ReadString('\n')
		if err != nil {
			logger.Error(err, "failed to read input")
			continue
		}

		line = strings.TrimSpace(line)

		// Handle commands
		if strings.HasPrefix(line, "/") {
			if strings.EqualFold(line, "/quit") || strings.EqualFold(line, "/exit") {
				fmt.Println("Goodbye!")
				return nil
			}
			if strings.EqualFold(line, "/clear") {
				fmt.Println("Clearing session...")
				continue
			}
			fmt.Printf("Unknown command: %s\n", line)
			continue
		}

		if line == "" {
			continue
		}

	

// Check for context cancellation
 		select {
 		case <-ctx.Done():
 			fmt.Println("\nGoodbye!")
 			return nil
 		default:
 			// Process message through handler
 			if err := handler(ctx, chatID, line); err != nil {
 				logger.Error(err, "failed to process message")
 				fmt.Printf("Error: %v\n", err)
 				continue
 			}

 			fmt.Print("> ")
		}
	}

	return nil
}

// Stop ends the channel.
func (c *CLIChannel) Stop() error {
	c.running = false
	return nil
}
