package roadmap

import (
	"context"
	"fmt"
	"strings"

	"github.com/arushtemirbek/skillpath/internal/apperror"
)

// LLMClient is owned by its consumer. Provider HTTP details stay outside service.
type LLMClient interface {
	GenerateRoadmap(ctx context.Context, goal, userContext string) (*Roadmap, error)
}

type Service struct{ client LLMClient }

func NewService(client LLMClient) *Service { return &Service{client: client} }

func (s *Service) Generate(ctx context.Context, goal, userContext string) (*Roadmap, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, apperror.ErrGoalRequired
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", apperror.ErrLLMRequestFailed, err)
	}
	result, err := s.client.GenerateRoadmap(ctx, goal, strings.TrimSpace(userContext))
	if err != nil {
		return nil, fmt.Errorf("generate roadmap: %w", err)
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}
