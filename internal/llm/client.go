package llm

import "net/http"

// HTTPClient is the narrow transport dependency used by the OpenAI adapter.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
