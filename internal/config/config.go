package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP   HTTPConfig
	OpenAI OpenAIConfig
	Log    LogConfig
}

type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	RequestTimeout  time.Duration
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

type LogConfig struct{ Level string }

// Load reads an optional local .env without overriding process environment.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		// dotenv parser errors can contain values, including secrets.
		return nil, errors.New("failed to load .env: check file syntax and permissions")
	}
	return load(os.LookupEnv)
}

type lookupEnv func(string) (string, bool)

// load keeps configuration parsing independent of process-global environment.
func load(lookup lookupEnv) (*Config, error) {
	cfg := &Config{
		HTTP: HTTPConfig{Port: value(lookup, "HTTP_PORT", "8080")},
		OpenAI: OpenAIConfig{
			APIKey:  value(lookup, "OPENAI_API_KEY", ""),
			Model:   value(lookup, "OPENAI_MODEL", ""),
			BaseURL: value(lookup, "OPENAI_BASE_URL", "https://api.openai.com"),
		},
		Log: LogConfig{Level: strings.ToLower(value(lookup, "LOG_LEVEL", "info"))},
	}
	// REQUEST_TIMEOUT is the name from the specification. OPENAI_TIMEOUT is
	// accepted for compatibility only when REQUEST_TIMEOUT is not set.
	timeoutKey := "REQUEST_TIMEOUT"
	if _, exists := lookup(timeoutKey); !exists {
		if _, legacy := lookup("OPENAI_TIMEOUT"); legacy {
			timeoutKey = "OPENAI_TIMEOUT"
		}
	}
	for _, setting := range []struct {
		name     string
		fallback time.Duration
		target   *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", 5 * time.Second, &cfg.HTTP.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", 35 * time.Second, &cfg.HTTP.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", 60 * time.Second, &cfg.HTTP.IdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", 10 * time.Second, &cfg.HTTP.ShutdownTimeout},
		{timeoutKey, 30 * time.Second, &cfg.HTTP.RequestTimeout},
	} {
		parsed, err := parseDuration(lookup, setting.name, setting.fallback)
		if err != nil {
			return nil, err
		}
		*setting.target = parsed
	}
	cfg.OpenAI.Timeout = cfg.HTTP.RequestTimeout
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func value(lookup lookupEnv, key, fallback string) string {
	if result, exists := lookup(key); exists {
		return result
	}
	return fallback
}

func parseDuration(lookup lookupEnv, key string, fallback time.Duration) (time.Duration, error) {
	raw, exists := lookup(key)
	if !exists {
		return fallback, nil
	}
	result, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 30s", key)
	}
	if result <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return result, nil
}
