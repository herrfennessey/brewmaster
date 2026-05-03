package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OpenAIProvider implements Provider using the official OpenAI Go client.
type OpenAIProvider struct {
	tracer trace.Tracer
	model  openai.ChatModel
	client openai.Client
}

// NewOpenAIProvider creates an OpenAIProvider from environment variables.
// Requires OPENAI_API_KEY. AI_MODEL is optional and defaults to gpt-5.4-nano.
// If tracer is nil, no spans are emitted (fully no-op tracing).
func NewOpenAIProvider(tracer trace.Tracer) (*OpenAIProvider, error) {
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
		tracer: tracer,
	}, nil
}

// startSpan creates a child span when a tracer is configured; otherwise returns
// the original context and a nil span (callers must nil-check).
func (p *OpenAIProvider) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if p.tracer == nil {
		return ctx, nil
	}
	return p.tracer.Start(ctx, name)
}

func endSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func setUsageAttrs(span trace.Span, usage *openai.CompletionUsage) {
	if span == nil {
		return
	}
	span.SetAttributes(
		attribute.Int64("tokens_in", usage.PromptTokens),
		attribute.Int64("tokens_out", usage.CompletionTokens),
		attribute.Int64("tokens_total", usage.TotalTokens),
	)
}

// CompleteWithImage forces the model to call the tool defined in req using image + text input.
func (p *OpenAIProvider) CompleteWithImage(ctx context.Context, req *CompletionRequest, imageData []byte, mediaType string) (_ string, err error) {
	ctx, span := p.startSpan(ctx, "openai.complete_with_image")
	defer func() { endSpan(span, err) }()
	if span != nil {
		span.SetAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", p.model),
			attribute.String("ai.tool.name", req.Tool.Name),
			attribute.Bool("ai.deterministic", req.Deterministic),
			attribute.Int("ai.input.length", len(req.UserMessage)),
			attribute.Int("ai.image.bytes", len(imageData)),
			attribute.String("ai.image.media_type", mediaType),
		)
	}

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
	imgParams := openai.ChatCompletionNewParams{
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
	}
	if req.Deterministic {
		imgParams.Temperature = openai.Float(0)
	}

	start := time.Now()
	resp, err := p.client.Chat.Completions.New(ctx, imgParams)
	dur := time.Since(start)
	if span != nil {
		span.SetAttributes(attribute.Int64("ai.duration_ms", dur.Milliseconds()))
	}
	if err != nil {
		return "", fmt.Errorf("openai vision completion failed: %w", err)
	}
	setUsageAttrs(span, &resp.Usage)
	slog.InfoContext(ctx, "openai vision completion",
		"tool", req.Tool.Name, "duration_ms", dur.Milliseconds(),
		"tokens_in", resp.Usage.PromptTokens, "tokens_out", resp.Usage.CompletionTokens)
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		err = fmt.Errorf("openai returned no tool calls")
		return "", err
	}
	return resp.Choices[0].Message.ToolCalls[0].Function.Arguments, nil
}

// Complete forces the model to call the tool defined in req and returns its arguments as JSON.
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (_ string, err error) {
	ctx, span := p.startSpan(ctx, "openai.complete")
	defer func() { endSpan(span, err) }()
	if span != nil {
		span.SetAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", p.model),
			attribute.String("ai.tool.name", req.Tool.Name),
			attribute.Bool("ai.deterministic", req.Deterministic),
			attribute.Int("ai.input.length", len(req.UserMessage)),
		)
	}

	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 2048
	}

	tool := openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        req.Tool.Name,
		Description: openai.String(req.Tool.Description),
		Parameters:  req.Tool.Schema,
	})
	chatParams := openai.ChatCompletionNewParams{
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
	}
	if req.Deterministic {
		chatParams.Temperature = openai.Float(0)
	}

	start := time.Now()
	resp, err := p.client.Chat.Completions.New(ctx, chatParams)
	dur := time.Since(start)
	if span != nil {
		span.SetAttributes(attribute.Int64("ai.duration_ms", dur.Milliseconds()))
	}
	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}
	setUsageAttrs(span, &resp.Usage)
	slog.InfoContext(ctx, "openai completion",
		"tool", req.Tool.Name, "duration_ms", dur.Milliseconds(),
		"tokens_in", resp.Usage.PromptTokens, "tokens_out", resp.Usage.CompletionTokens)
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		err = fmt.Errorf("openai returned no tool calls")
		return "", err
	}
	return resp.Choices[0].Message.ToolCalls[0].Function.Arguments, nil
}

// FindRoasterContent uses the Responses API with web search to find the roaster's
// product page and return a plain-text summary of the bean details found.
func (p *OpenAIProvider) FindRoasterContent(ctx context.Context, roasterName, hint string) (_ string, err error) {
	ctx, span := p.startSpan(ctx, "openai.web_search")
	defer func() { endSpan(span, err) }()
	if span != nil {
		span.SetAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", p.model),
			attribute.String("ai.tool.name", "web_search"),
			attribute.String("ai.roaster_name", roasterName),
			attribute.Bool("ai.has_hint", hint != ""),
		)
	}

	query := fmt.Sprintf("Find the official website for %q specialty coffee roaster", roasterName)
	if hint != "" {
		query += fmt.Sprintf(" (%s)", hint)
	}
	query += ". Find their product page for this specific bean and return all details: altitude, varietal, process, flavor notes, origin region, roast level, and any other coffee bean information."

	start := time.Now()
	resp, err := p.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: p.model,
		Input: responses.ResponseNewParamsInputUnion{OfString: param.NewOpt(query)},
		Tools: []responses.ToolUnionParam{
			responses.ToolParamOfWebSearch(responses.WebSearchToolTypeWebSearch),
		},
	})
	dur := time.Since(start)
	if span != nil {
		span.SetAttributes(attribute.Int64("ai.duration_ms", dur.Milliseconds()))
	}
	if err != nil {
		return "", fmt.Errorf("web search failed: %w", err)
	}
	text := resp.OutputText()
	if span != nil {
		span.SetAttributes(
			attribute.Int64("tokens_in", resp.Usage.InputTokens),
			attribute.Int64("tokens_out", resp.Usage.OutputTokens),
			attribute.Int64("tokens_total", resp.Usage.TotalTokens),
			attribute.Int("ai.output.length", len(text)),
		)
	}
	slog.InfoContext(ctx, "openai web search",
		"duration_ms", dur.Milliseconds(),
		"tokens_in", resp.Usage.InputTokens, "tokens_out", resp.Usage.OutputTokens,
		"output_chars", len(text))
	if text == "" {
		err = fmt.Errorf("web search returned no content")
		return "", err
	}
	return text, nil
}
