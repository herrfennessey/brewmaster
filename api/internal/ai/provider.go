// Package ai provides a provider-agnostic interface for LLM completions.
package ai

import "context"

// Provider is the interface all AI backends must implement.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (string, error)
}

// CompletionRequest holds the inputs for a single-turn completion.
type CompletionRequest struct {
	SystemPrompt string
	UserMessage  string
	MaxTokens    int
}
