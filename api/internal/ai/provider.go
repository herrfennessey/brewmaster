// Package ai provides a provider-agnostic interface for LLM completions.
package ai

import "context"

// Provider is the interface all AI backends must implement.
type Provider interface {
	Complete(ctx context.Context, req *CompletionRequest) (string, error)
}

// CompletionRequest holds the inputs for a structured tool-call completion.
// The AI is forced to call Tool and return its arguments as JSON.
type CompletionRequest struct {
	Tool         Tool
	SystemPrompt string
	UserMessage  string
	MaxTokens    int
}

// Tool describes the function the AI must call, with its JSON schema.
type Tool struct {
	Schema      map[string]any
	Name        string
	Description string
}
