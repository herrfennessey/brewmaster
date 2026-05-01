package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

// OpenAIProvider implements Provider using the official OpenAI Go client.
type OpenAIProvider struct {
	model  openai.ChatModel
	client openai.Client
}

// NewOpenAIProvider creates an OpenAIProvider from environment variables.
// Requires OPENAI_API_KEY. AI_MODEL is optional and defaults to gpt-5.4-nano.
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

// CompleteWithImage forces the model to call the tool defined in req using image + text input.
func (p *OpenAIProvider) CompleteWithImage(ctx context.Context, req *CompletionRequest, imageData []byte, mediaType string) (string, error) {
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 2048
	}
	dataURI := "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(imageData)
	tool := openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        req.Tool.Name,
		Description: openai.String(req.Tool.Description),
		Parameters:  req.Tool.Schema,
	})
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(req.SystemPrompt),
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart(req.UserMessage),
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: dataURI}),
			}),
		},
		Tools: []openai.ChatCompletionToolUnionParam{tool},
		ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoRequired)),
		},
		MaxCompletionTokens: openai.Int(maxTokens),
	})
	if err != nil {
		return "", fmt.Errorf("openai vision completion failed: %w", err)
	}
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return "", fmt.Errorf("openai returned no tool calls")
	}
	return resp.Choices[0].Message.ToolCalls[0].Function.Arguments, nil
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

// FindRoasterContent uses the Responses API with web search to find the roaster's
// product page and return a plain-text summary of the bean details found.
func (p *OpenAIProvider) FindRoasterContent(ctx context.Context, roasterName, hint string) (string, error) {
	query := fmt.Sprintf("Find the official website for %q specialty coffee roaster", roasterName)
	if hint != "" {
		query += fmt.Sprintf(" (%s)", hint)
	}
	query += ". Find their product page for this specific bean and return all details: altitude, varietal, process, flavor notes, origin region, roast level, and any other coffee bean information."

	resp, err := p.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: p.model,
		Input: responses.ResponseNewParamsInputUnion{OfString: param.NewOpt(query)},
		Tools: []responses.ToolUnionParam{
			responses.ToolParamOfWebSearch(responses.WebSearchToolTypeWebSearch),
		},
	})
	if err != nil {
		return "", fmt.Errorf("web search failed: %w", err)
	}
	text := resp.OutputText()
	if text == "" {
		return "", fmt.Errorf("web search returned no content")
	}
	return text, nil
}
