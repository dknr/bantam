// Package channel defines the channel interface.
package channel

import (
	"context"
)

// Channel is the interface for all chat channels.
type Channel interface {
	// Name returns the channel name.
	Name() string

	// Start begins receiving messages from the channel.
	// Messages are sent to the handler function.
	Start(ctx context.Context, handler func(ctx context.Context, chatID, content string) error) error

	// Stop ends the channel.
	Stop() error
}
