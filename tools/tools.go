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

// SchemaTool is an optional interface for tools that can provide their own parameter schema.
type SchemaTool interface {
	ToolSchema() map[string]any
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
		var schema map[string]any
		// Check if the tool provides its own schema
		if schemaTool, ok := tool.(SchemaTool); ok {
			schema = schemaTool.ToolSchema()
		} else {
			// This should not happen if all tools properly implement SchemaTool
			// Keeping minimal schema for safety during transition
			schema = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
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