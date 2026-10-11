package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/config"
)

const maxResponseBodyBytes = 2 << 20

type OpenAI struct {
	client   HTTPClient
	endpoint string
	apiKey   string
	model    string
	log      *slog.Logger
}

func NewOpenAI(cfg config.OpenAIConfig, log *slog.Logger) (*OpenAI, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Hostname() == "" || (base.Scheme != "https" && base.Scheme != "http") || base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		return nil, errors.New("invalid OPENAI_BASE_URL")
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" || cfg.Timeout <= 0 {
		return nil, errors.New("OpenAI client requires API key, model and a positive timeout")
	}
	base.Path = strings.TrimRight(base.Path, "/")
	if !strings.HasSuffix(base.Path, "/v1") {
		base.Path += "/v1"
	}
	base.Path += "/chat/completions"
	base.RawPath = ""
	return &OpenAI{
		client: &http.Client{
			Timeout: cfg.Timeout,
			// Do not follow redirects carrying API credentials or convert POST to GET.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		endpoint: base.String(), apiKey: cfg.APIKey, model: cfg.Model, log: log,
	}, nil
}

func (c *OpenAI) Complete(ctx context.Context, input CompletionRequest) (result string, err error) {
	started := time.Now()
	status := 0
	defer func() {
		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
		}
		// Never log request headers, prompts, response bodies or raw transport errors.
		c.log.Log(ctx, level, "LLM request", "model", c.model, "duration", time.Since(started), "status", status, "error", err)
	}()
	if len(input.Messages) == 0 {
		return "", errors.New("completion requires messages")
	}
	request := chatRequest{Model: c.model, Messages: input.Messages, MaxCompletionTokens: input.MaxCompletionTokens}
	if input.Schema != nil {
		if input.Schema.Name == "" || !json.Valid(input.Schema.Schema) {
			return "", errors.New("completion requires a named valid JSON schema")
		}
		request.ResponseFormat = &responseFormat{Type: "json_schema", JSONSchema: structuredSchema{Name: input.Schema.Name, Strict: true, Schema: input.Schema.Schema}}
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode OpenAI request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%w: cannot create request", apperror.ErrLLMRequestFailed)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return "", errors.Join(apperror.ErrLLMRequestFailed, context.Canceled)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return "", errors.Join(apperror.ErrLLMRequestFailed, context.DeadlineExceeded)
		}
		return "", fmt.Errorf("%w: transport error", apperror.ErrLLMRequestFailed)
	}
	defer resp.Body.Close()
	status = resp.StatusCode
	if status != http.StatusOK {
		return "", fmt.Errorf("%w: upstream HTTP status %d", apperror.ErrLLMRequestFailed, status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil {
		return "", fmt.Errorf("%w: cannot read response", apperror.ErrLLMRequestFailed)
	}
	if len(data) > maxResponseBodyBytes {
		return "", fmt.Errorf("%w: response too large", apperror.ErrLLMInvalidResponse)
	}
	var completion chatResponse
	if err := json.Unmarshal(data, &completion); err != nil || len(completion.Choices) != 1 {
		return "", fmt.Errorf("%w: malformed completion", apperror.ErrLLMInvalidResponse)
	}
	choice := completion.Choices[0]
	if choice.FinishReason != "stop" || choice.Message.Refusal != nil {
		return "", fmt.Errorf("%w: refused or incomplete completion", apperror.ErrLLMInvalidResponse)
	}
	if strings.TrimSpace(choice.Message.Content) == "" {
		return "", fmt.Errorf("%w: empty completion", apperror.ErrLLMInvalidResponse)
	}
	return choice.Message.Content, nil
}

// Check performs a small real generation request to verify key and model access.
func (c *OpenAI) Check(ctx context.Context) error {
	content, err := c.Complete(ctx, CompletionRequest{
		Messages:            []Message{{Role: "user", Content: "Reply with exactly: OK"}},
		MaxCompletionTokens: 64,
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) != "OK" {
		return fmt.Errorf("%w: unexpected health probe response", apperror.ErrLLMInvalidResponse)
	}
	return nil
}
