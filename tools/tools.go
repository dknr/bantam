package tools

import (
	"context"
	"fmt"
)

// Tool is an interface for agent tools.
type Tool interface {
	Name() string
	Execute(ctx context.Context, args map[string]any) (any, error)
}

// StatusLineTool is an optional interface for tools that can provide a status line.
type StatusLineTool interface {
	StatusLine(args map[string]any) string
}

// Registry holds available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// Execute executes a tool by name.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, args)
}

// Definitions returns OpenAI-style tool definitions.
func (r *Registry) Definitions() []map[string]any {
	defs := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name(),
				"description": "Tool implementation",
				"parameters": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		})
	}
	return defs
}

// DefinitionsWithSchema returns OpenAI-style tool definitions with parameter schemas.
func (r *Registry) DefinitionsWithSchema() []map[string]any {
	defs := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.tools {
		schema := map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}

		// Add specific parameter schemas for known tools
		switch tool.Name() {
		case "echo":
			schema["properties"].(map[string]any)["message"] = map[string]any{
				"type":        "string",
				"description": "The message to echo back",
			}
			schema["required"] = []string{"message"}
		case "exec":
			schema["properties"].(map[string]any)["command"] = map[string]any{
				"type":        "string",
				"description": "The command to execute",
			}
			schema["required"] = []string{"command"}
		case "file":
			schema["properties"].(map[string]any)["action"] = map[string]any{
				"type":        "string",
				"description": "The action to perform: read, write, or list",
			}
			schema["properties"].(map[string]any)["path"] = map[string]any{
				"type":        "string",
				"description": "The file or directory path",
			}
			schema["required"] = []string{"action", "path"}
		case "memory":
			schema["properties"].(map[string]any)["action"] = map[string]any{
				"type":        "string",
				"description": "The action to perform: read, write, list, search, or since",
			}
			schema["properties"].(map[string]any)["key"] = map[string]any{
				"type":        "string",
				"description": "The memory key (required for read/write actions)",
			}
			schema["properties"].(map[string]any)["old_value"] = map[string]any{
				"type":        "string",
				"description": "The expected current value (for compare-exchange write, empty string if new)",
			}
			schema["properties"].(map[string]any)["new_value"] = map[string]any{
				"type":        "string",
				"description": "The new value to store",
			}
			schema["properties"].(map[string]any)["query"] = map[string]any{
				"type":        "string",
				"description": "The search query for history entries",
			}
			schema["properties"].(map[string]any)["timestamp"] = map[string]any{
				"type":        "string",
				"description": "Timestamp in ISO8601 format for history_since action",
			}
			schema["required"] = []string{"action"}
		}

		defs = append(defs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name(),
				"description": "Tool implementation",
				"parameters":  schema,
			},
		})
	}
	return defs
}
