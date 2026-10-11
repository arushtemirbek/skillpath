package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *Config) Validate() error {
	if strings.TrimSpace(c.OpenAI.APIKey) == "" {
		return errors.New("OPENAI_API_KEY is required")
	}
	if strings.TrimSpace(c.OpenAI.Model) == "" {
		return errors.New("OPENAI_MODEL is required")
	}
	if c.HTTP.Port == "" {
		return errors.New("HTTP_PORT is required")
	}
	port, err := strconv.Atoi(c.HTTP.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("HTTP_PORT must be an integer between 1 and 65535")
	}
	for _, setting := range []struct {
		name    string
		timeout time.Duration
	}{
		{"HTTP_READ_TIMEOUT", c.HTTP.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", c.HTTP.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", c.HTTP.IdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", c.HTTP.ShutdownTimeout},
		{"REQUEST_TIMEOUT", c.HTTP.RequestTimeout},
		{"OPENAI_TIMEOUT", c.OpenAI.Timeout},
	} {
		if setting.timeout <= 0 {
			return fmt.Errorf("%s must be positive", setting.name)
		}
	}
	if c.HTTP.WriteTimeout <= c.HTTP.RequestTimeout || c.HTTP.WriteTimeout <= c.OpenAI.Timeout {
		return errors.New("HTTP_WRITE_TIMEOUT must exceed REQUEST_TIMEOUT")
	}
	baseURL, err := url.Parse(c.OpenAI.BaseURL)
	if err != nil || baseURL.Hostname() == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.ForceQuery || baseURL.Fragment != "" {
		return errors.New("OPENAI_BASE_URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	switch strings.ToLower(c.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("LOG_LEVEL must be debug, info, warn or error")
	}
	return nil
}
