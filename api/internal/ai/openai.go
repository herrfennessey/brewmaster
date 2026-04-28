package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// OpenAIProvider implements Provider using the official OpenAI Go client.
type OpenAIProvider struct {
	model  openai.ChatModel
	client openai.Client
}

// NewOpenAIProvider creates an OpenAIProvider from environment variables.
// Requires OPENAI_API_KEY. AI_MODEL is optional and defaults to gpt-4.1-mini.
func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	model := openai.ChatModelGPT5_4Nano
	if env := os.Getenv("AI_MODEL"); env != "" {
		model = env
	}
	return &OpenAIProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}, nil
}

// Complete forces the model to call the tool defined in req and returns its arguments as JSON.
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (string, error) {
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 2048
	}

	tool := openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        req.Tool.Name,
		Description: openai.String(req.Tool.Description),
		Parameters:  req.Tool.Schema,
	})

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(req.SystemPrompt),
			openai.UserMessage(req.UserMessage),
		},
		Tools: []openai.ChatCompletionToolUnionParam{tool},
		ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoRequired)),
		},
		MaxCompletionTokens: openai.Int(maxTokens),
	})
	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return "", fmt.Errorf("openai returned no tool calls")
	}
	return resp.Choices[0].Message.ToolCalls[0].Function.Arguments, nil
}
