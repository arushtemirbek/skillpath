package roadmap

import (
	"fmt"
	"strings"

	"github.com/arushtemirbek/skillpath/internal/apperror"
)

type GenerateRoadmapRequest struct {
	Goal    string `json:"goal"`
	Context string `json:"context,omitempty"`
}

type Roadmap struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Sections    []RoadmapSection `json:"sections"`
}

type RoadmapSection struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Topics      []RoadmapTopic `json:"topics"`
}

// A topic is a leaf. The model deliberately has no Children field.
type RoadmapTopic struct {
	Title string `json:"title"`
}

// Validate checks business invariants independently of the LLM provider.
func (r *Roadmap) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: roadmap is empty", apperror.ErrLLMInvalidResponse)
	}
	if strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("%w: title is required", apperror.ErrLLMInvalidResponse)
	}
	if len(r.Sections) == 0 {
		return fmt.Errorf("%w: at least one section is required", apperror.ErrLLMInvalidResponse)
	}
	for i, section := range r.Sections {
		if strings.TrimSpace(section.Title) == "" {
			return fmt.Errorf("%w: sections[%d].title is required", apperror.ErrLLMInvalidResponse, i)
		}
		if len(section.Topics) == 0 {
			return fmt.Errorf("%w: sections[%d] must contain topics", apperror.ErrLLMInvalidResponse, i)
		}
		for j, topic := range section.Topics {
			if strings.TrimSpace(topic.Title) == "" {
				return fmt.Errorf("%w: sections[%d].topics[%d].title is required", apperror.ErrLLMInvalidResponse, i, j)
			}
		}
	}
	return nil
}
