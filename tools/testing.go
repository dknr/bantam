package tools

import (
	"context"
	"fmt"
	"time"
)

// TimeTool returns the current time in ISO 8601 format.
type TimeTool struct{}

// NewTimeTool creates a new time tool.
func NewTimeTool() *TimeTool {
	return &TimeTool{}
}

// Name returns the tool name.
func (t *TimeTool) Name() string {
	return "time"
}

// Execute returns the current time in RFC3339 format.
func (t *TimeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	return time.Now().Format(time.RFC3339), nil
}

// EchoTool returns the input message unchanged.
type EchoTool struct{}

// NewEchoTool creates a new echo tool.
func NewEchoTool() *EchoTool {
	return &EchoTool{}
}

// Name returns the tool name.
func (t *EchoTool) Name() string {
	return "echo"
}

// Execute returns the message parameter unchanged.
 	func (t *EchoTool) Execute(ctx context.Context, args map[string]any) (any, error) {
 		msg, ok := args["message"]
 		if !ok {
 			return nil, fmt.Errorf("missing required parameter: message")
 		}

 		message, ok := msg.(string)
 		if !ok {
 			return nil, fmt.Errorf("parameter 'message' must be a string")
 		}

 		return message, nil
 	}
