package ai

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements Provider using the OpenAI API.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates an OpenAIProvider from environment variables.
// Requires OPENAI_API_KEY and AI_MODEL to be set.
func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = openai.GPT4oMini
	}
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		model:  model,
	}, nil
}

// Complete sends a single-turn chat completion request to OpenAI.
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: req.SystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: req.UserMessage},
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
