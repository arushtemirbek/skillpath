package roadmap

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/arushtemirbek/skillpath/internal/apperror"
)

type stubLLM func(context.Context, string, string) (*Roadmap, error)

func (f stubLLM) GenerateRoadmap(ctx context.Context, goal, userContext string) (*Roadmap, error) {
	return f(ctx, goal, userContext)
}

func validRoadmap() *Roadmap {
	return &Roadmap{Title: "Go Backend", Description: "Overview", Sections: []RoadmapSection{{Title: "Go", Description: "Fundamentals", Topics: []RoadmapTopic{{Title: "Interfaces"}, {Title: "Error handling"}}}}}
}

func TestServiceGenerate(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "trace")
	expected := validRoadmap()
	calls := 0
	service := NewService(stubLLM(func(actual context.Context, goal, userContext string) (*Roadmap, error) {
		calls++
		if actual != ctx || goal != "Learn Go" || userContext != "Python developer" {
			t.Fatalf("arguments = %q %q", goal, userContext)
		}
		return expected, nil
	}))
	result, err := service.Generate(ctx, "  Learn Go  ", " Python developer ")
	if err != nil || !reflect.DeepEqual(result, expected) || calls != 1 {
		t.Fatalf("result=%v, error=%v, calls=%d", result, err, calls)
	}
}

func TestServiceEmptyGoalDoesNotCallLLM(t *testing.T) {
	service := NewService(stubLLM(func(context.Context, string, string) (*Roadmap, error) {
		t.Fatal("LLM called for empty goal")
		return nil, nil
	}))
	for _, goal := range []string{"", " ", "\t\n", "\u00a0"} {
		if _, err := service.Generate(context.Background(), goal, ""); !errors.Is(err, apperror.ErrGoalRequired) {
			t.Fatalf("error=%v", err)
		}
	}
}

func TestServiceProviderError(t *testing.T) {
	service := NewService(stubLLM(func(context.Context, string, string) (*Roadmap, error) { return nil, apperror.ErrLLMRequestFailed }))
	if _, err := service.Generate(context.Background(), "Go", ""); !errors.Is(err, apperror.ErrLLMRequestFailed) {
		t.Fatal(err)
	}
}

func TestServiceRejectsInvalidRoadmap(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result *Roadmap
	}{
		{"nil", nil}, {"empty", &Roadmap{}},
		{"blank title", &Roadmap{Title: " ", Sections: validRoadmap().Sections}},
		{"no sections", &Roadmap{Title: "Go"}},
		{"blank section", &Roadmap{Title: "Go", Sections: []RoadmapSection{{Title: " ", Topics: []RoadmapTopic{{Title: "Go"}}}}}},
		{"no topics", &Roadmap{Title: "Go", Sections: []RoadmapSection{{Title: "Go"}}}},
		{"blank topic", &Roadmap{Title: "Go", Sections: []RoadmapSection{{Title: "Go", Topics: []RoadmapTopic{{Title: " "}}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(stubLLM(func(context.Context, string, string) (*Roadmap, error) { return tc.result, nil }))
			if _, err := service.Generate(context.Background(), "Go", ""); !errors.Is(err, apperror.ErrLLMInvalidResponse) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestServiceCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(stubLLM(func(context.Context, string, string) (*Roadmap, error) {
		t.Fatal("LLM called after cancellation")
		return nil, nil
	}))
	_, err := service.Generate(ctx, "Go", "")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, apperror.ErrLLMRequestFailed) {
		t.Fatalf("error=%v", err)
	}
}
