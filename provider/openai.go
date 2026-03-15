package provider

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "time"

  "github.com/dknr/bantam/logging"
  "github.com/dknr/bantam/tracing"
  "go.opentelemetry.io/otel/attribute"
  "go.opentelemetry.io/otel/codes"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
type OpenAIProvider struct {
	apiKey      string
	apiBase     string
	model       string
	httpClient  *http.Client
}

// NewOpenAIProvider creates a new OpenAI-compatible provider.
func NewOpenAIProvider(apiKey, apiBase, model string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:     apiKey,
		apiBase:    apiBase,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Chat sends a chat completion request to the OpenAI-compatible API.
  	func (p *OpenAIProvider) Chat(ctx context.Context, messages []map[string]any, tools []map[string]any) (*Response, error) {
    		// Create span for this LLM call
    		ctx, chatSpan := tracing.StartActiveSpan(ctx, "llm.provider_call", map[string]string{
    			"model":          p.model,
    			"messages_count": fmt.Sprintf("%d", len(messages)),
    			"tools_count":    fmt.Sprintf("%d", len(tools)),
    		})
    		if chatSpan != nil {
    			chatSpan.SetAttributes(attribute.String("llm.model", p.model))
    		}

 reqBody := map[string]any{
 		"model":    p.model,
 		"messages": messages,
 	}

 	// Enable prefix caching if supported by provider
 	reqBody["cache_prompt"] = true

	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

resp, err := p.httpClient.Do(req)
   	if err != nil {
    		if chatSpan != nil {
    			chatSpan.SetStatus(codes.Error, err.Error())
    			chatSpan.End()
    		}
    		return nil, fmt.Errorf("HTTP request failed: %w", err)
    	}
 	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
 		if chatSpan != nil {
 			chatSpan.SetStatus(codes.Error, "non-200 status code")
 			chatSpan.End()
 		}
 		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
 	}

	// Log request and response in verbose mode
	if logging.IsVerbose(ctx) {
		logging.PrintJSON("Provider Request", reqBody)
		var apiResult map[string]any
		json.Unmarshal(body, &apiResult)
		logging.PrintJSON("Provider Response", apiResult)
	}

var apiResult map[string]any
 	if err := json.Unmarshal(body, &apiResult); err != nil {
 		if chatSpan != nil {
 			chatSpan.SetStatus(codes.Error, "failed to parse response")
 			chatSpan.End()
 		}
 		return nil, fmt.Errorf("failed to parse response: %w", err)
 	}

 	// Extract choices[0].message
 	choices, ok := apiResult["choices"].([]any)
 	if !ok || len(choices) == 0 {
 		if chatSpan != nil {
 			chatSpan.SetStatus(codes.Error, "no choices in response")
 			chatSpan.End()
 		}
 		return nil, fmt.Errorf("no choices in response")
 	}

 	choice, ok := choices[0].(map[string]any)
 	if !ok {
 		if chatSpan != nil {
 			chatSpan.SetStatus(codes.Error, "invalid choice format")
 			chatSpan.End()
 		}
 		return nil, fmt.Errorf("invalid choice format")
 	}

 	// Message object is nested under "message" key
 	message, ok := choice["message"].(map[string]any)
 	if !ok {
 		if chatSpan != nil {
 			chatSpan.SetStatus(codes.Error, "invalid message format in choice")
 			chatSpan.End()
 		}
 		return nil, fmt.Errorf("invalid message format in choice")
 	}

	content, _ := message["content"].(string)

// Extract tool calls if present
 	var toolCalls []ToolCall
 	if toolCallsRaw, ok := message["tool_calls"].([]any); ok {
 		for _, tc := range toolCallsRaw {
 			if tcMap, ok := tc.(map[string]any); ok {
 				id, _ := tcMap["id"].(string)
 				funcObj, _ := tcMap["function"].(map[string]any)
 				name, _ := funcObj["name"].(string)

 				var args map[string]any
 				// Arguments can be either a string (JSON) or a map
 				if argsRaw, ok := funcObj["arguments"].(string); ok {
 					json.Unmarshal([]byte(argsRaw), &args)
 				} else if argsMap, ok := funcObj["arguments"].(map[string]any); ok {
 					args = argsMap
 				}

 				toolCalls = append(toolCalls, ToolCall{
 					ID:        id,
 					Name:      name,
 					Arguments: args,
 				})
 			}
 		}
 	}

	// Extract finish reason
	finishReason := "stop"
	if fr, ok := message["finish_reason"].(string); ok {
		finishReason = fr
	}

// Extract token usage from response
 	tokens := make(map[string]int)
 	if usage, ok := apiResult["usage"].(map[string]any); ok {
 		if v, ok := usage["prompt_tokens"].(float64); ok {
 			tokens["prompt"] = int(v)
 		}
 		if v, ok := usage["completion_tokens"].(float64); ok {
 			tokens["completion"] = int(v)
 		}
 		if v, ok := usage["total_tokens"].(float64); ok {
 			tokens["total"] = int(v)
 		}
 	}

 // Extract timing info if available (llama.cpp and some providers)
 	var timing *Timing
 	if timings, ok := apiResult["timings"].(map[string]any); ok {
 		t := &Timing{}
 		if v, ok := timings["prompt_ms"].(float64); ok {
 			t.PromptMs = v
 		}
 		if v, ok := timings["prompt_per_second"].(float64); ok {
 			t.PromptPerSecond = v
 		}
 		if v, ok := timings["predicted_ms"].(float64); ok {
 			t.PredictedMs = v
 		}
 		if v, ok := timings["predicted_per_second"].(float64); ok {
 			t.PredictedPerSecond = v
 		}
 		timing = t
 	}

 	llmResponse := NewResponse(content, toolCalls, finishReason)
    	llmResponse.SetTokens(tokens)
    	llmResponse.SetTiming(timing)
    	if chatSpan != nil {
     		chatSpan.SetAttributes(attribute.Int("llm.token.prompt", tokens["prompt"]))
     		chatSpan.SetAttributes(attribute.Int("llm.token.completion", tokens["completion"]))
     		chatSpan.SetAttributes(attribute.Int("llm.token.total", tokens["total"]))
     		chatSpan.SetAttributes(attribute.Int("response.has_tool_calls", boolToInt(len(toolCalls) > 0)))
     		chatSpan.SetAttributes(attribute.Int("response.content_length", len(content)))
     		chatSpan.End()
     	}
     	return llmResponse, nil
    }

  // boolToInt converts bool to int for OpenTelemetry attributes.
  func boolToInt(b bool) int {
  	if b {
  		return 1
  	}
  	return 0
  }

// GetDefaultModel returns the default model.
func (p *OpenAIProvider) GetDefaultModel() string {
	return p.model
}

// mapKeys extracts keys from a map
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
