package roadmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/arushtemirbek/skillpath/internal/apperror"
	"github.com/arushtemirbek/skillpath/internal/llm"
)

// Completer is the generic LLM capability used by the roadmap adapter.
type Completer interface {
	Complete(context.Context, llm.CompletionRequest) (string, error)
}

type Generator struct{ client Completer }

var _ LLMClient = (*Generator)(nil)

func NewGenerator(client Completer) *Generator { return &Generator{client: client} }

func (g *Generator) GenerateRoadmap(ctx context.Context, goal, userContext string) (*Roadmap, error) {
	prompt, err := userPrompt(goal, userContext)
	if err != nil {
		return nil, fmt.Errorf("encode roadmap prompt: %w", err)
	}
	content, err := g.client.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: prompt}},
		Schema:   &llm.JSONSchema{Name: "learning_roadmap", Schema: json.RawMessage(roadmapSchema)},
	})
	if err != nil {
		return nil, err
	}
	return decodeRoadmap(content)
}

// Pointers distinguish missing/null fields from required empty strings.
// Semantic validation of titles and arrays belongs to Service.
type roadmapOutput struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	Sections    *[]sectionOutput `json:"sections"`
}

type sectionOutput struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Topics      *[]topicOutput `json:"topics"`
}

type topicOutput struct {
	Title *string `json:"title"`
}

func decodeRoadmap(content string) (*Roadmap, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	var output roadmapOutput
	if err := decoder.Decode(&output); err != nil {
		return nil, fmt.Errorf("%w: invalid roadmap JSON", apperror.ErrLLMInvalidResponse)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("%w: trailing roadmap data", apperror.ErrLLMInvalidResponse)
	}
	if output.Title == nil || output.Description == nil || output.Sections == nil {
		return nil, fmt.Errorf("%w: missing roadmap fields", apperror.ErrLLMInvalidResponse)
	}
	result := &Roadmap{Title: *output.Title, Description: *output.Description, Sections: make([]RoadmapSection, 0, len(*output.Sections))}
	for _, section := range *output.Sections {
		if section.Title == nil || section.Description == nil || section.Topics == nil {
			return nil, fmt.Errorf("%w: missing section fields", apperror.ErrLLMInvalidResponse)
		}
		parsed := RoadmapSection{Title: *section.Title, Description: *section.Description, Topics: make([]RoadmapTopic, 0, len(*section.Topics))}
		for _, topic := range *section.Topics {
			if topic.Title == nil {
				return nil, fmt.Errorf("%w: missing topic title", apperror.ErrLLMInvalidResponse)
			}
			parsed.Topics = append(parsed.Topics, RoadmapTopic{Title: *topic.Title})
		}
		result.Sections = append(result.Sections, parsed)
	}
	return result, nil
}
