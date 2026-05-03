// Package ai provides a provider-agnostic interface for LLM completions.
package ai

import "context"

// Provider is the interface all AI backends must implement.
type Provider interface {
	Complete(ctx context.Context, req *CompletionRequest) (string, error)
	CompleteWithImage(ctx context.Context, req *CompletionRequest, imageData []byte, mediaType string) (string, error)
	// FindRoasterContent searches the web for a roaster's product page and returns
	// a plain-text summary of the bean details found. hint is optional extra context
	// (e.g. origin country or producer name) to help disambiguate the roaster.
	FindRoasterContent(ctx context.Context, roasterName, hint string) (string, error)
}

// CompletionRequest holds the inputs for a structured tool-call completion.
// The AI is forced to call Tool and return its arguments as JSON.
type CompletionRequest struct {
	Tool          Tool
	SystemPrompt  string
	UserMessage   string
	MaxTokens     int
	Deterministic bool // if true, sets temperature=0 for reproducible output
}

// Tool describes the function the AI must call, with its JSON schema.
type Tool struct {
	Schema      map[string]any
	Name        string
	Description string
}
