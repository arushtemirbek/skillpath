package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := checkOpenAI(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkOpenAI() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot load .env: check its syntax and permissions")
	}
	key, model := os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL")
	if strings.TrimSpace(key) == "" || strings.TrimSpace(model) == "" {
		return errors.New("set OPENAI_API_KEY and OPENAI_MODEL in .env")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	endpoint, err := url.Parse(baseURL)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return errors.New("OPENAI_BASE_URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")
	if !strings.HasSuffix(endpoint.Path, "/v1") {
		endpoint.Path += "/v1"
	}
	endpoint.Path += "/chat/completions"
	endpoint.RawPath = ""
	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "Reply with exactly: OK"},
		},
		"max_completion_tokens": 64,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot create OpenAI request")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("OpenAI request failed: check network access and timeout")
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return errors.New("cannot read OpenAI response")
	}
	fmt.Printf("HTTP %s\n%s\n", resp.Status, strings.ReplaceAll(string(result), key, "[REDACTED]"))
	if resp.StatusCode != http.StatusOK {
		return errors.New("OpenAI returned an error; see the response above")
	}
	return nil
}
