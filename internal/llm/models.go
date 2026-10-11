package llm

import "encoding/json"

type chatRequest struct {
	Model               string          `json:"model"`
	Messages            []Message       `json:"messages"`
	ResponseFormat      *responseFormat `json:"response_format,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string           `json:"type"`
	JSONSchema structuredSchema `json:"json_schema"`
}

type structuredSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type chatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string  `json:"content"`
			Refusal *string `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
}

// CompletionRequest describes provider-independent inputs for one generation.
// Schema is optional: nil requests an ordinary text completion.
type CompletionRequest struct {
	Messages            []Message
	Schema              *JSONSchema
	MaxCompletionTokens int
}

type JSONSchema struct {
	Name   string
	Schema json.RawMessage
}
