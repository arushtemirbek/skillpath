package roadmap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/llm"
)

const validGenerated = `{"title":"Go","description":"Overview","sections":[{"title":"Basics","description":"Fundamentals","topics":[{"title":"Interfaces"}]}]}`

type stubCompleter func(context.Context, llm.CompletionRequest) (string, error)

func (f stubCompleter) Complete(ctx context.Context, r llm.CompletionRequest) (string, error) {
	return f(ctx, r)
}

func TestGeneratorPromptAndSchema(t *testing.T) {
	generator := NewGenerator(stubCompleter(func(ctx context.Context, input llm.CompletionRequest) (string, error) {
		if input.Schema == nil || input.Schema.Name != "learning_roadmap" || len(input.Messages) != 2 {
			t.Fatal("missing roadmap prompt/schema")
		}
		var schema map[string]any
		if err := json.Unmarshal(input.Schema.Schema, &schema); err != nil {
			t.Fatal(err)
		}
		topic := schema["properties"].(map[string]any)["sections"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["topics"].(map[string]any)["items"].(map[string]any)
		if topic["additionalProperties"] != false || len(topic["properties"].(map[string]any)) != 1 {
			t.Fatal("topic allows children")
		}
		if !strings.Contains(input.Messages[1].Content, "Python") {
			t.Fatal("user context missing")
		}
		return validGenerated, nil
	}))
	result, err := generator.GenerateRoadmap(context.Background(), "Learn Go", "Python")
	if err != nil || result.Title != "Go" {
		t.Fatalf("result=%v error=%v", result, err)
	}
}

func TestGeneratorRejectsInvalidStructure(t *testing.T) {
	for _, content := range []string{
		"free text", `{}`, `null`, validGenerated + `{}`,
		strings.Replace(validGenerated, `"description":"Overview",`, "", 1),
		strings.Replace(validGenerated, `"title":"Interfaces"`, `"title":"Interfaces","children":[]`, 1),
		strings.Replace(validGenerated, `[{"title":"Interfaces"}]`, `null`, 1),
	} {
		generator := NewGenerator(stubCompleter(func(context.Context, llm.CompletionRequest) (string, error) { return content, nil }))
		if _, err := generator.GenerateRoadmap(context.Background(), "Go", ""); !errors.Is(err, apperror.ErrLLMInvalidResponse) {
			t.Fatalf("error=%v", err)
		}
	}
}
