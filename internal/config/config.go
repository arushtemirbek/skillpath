package config

import (
	"errors"
	"fmt"
	"os"
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
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	// Загружаем .env, если файл существует.
	// Если его нет — это не ошибка, например в production
	// переменные могут передаваться напрямую через environment.
	_ = godotenv.Load()

	timeouts := make(map[string]time.Duration)
	for _, setting := range []struct {
		name     string
		fallback time.Duration
	}{
		{"HTTP_READ_TIMEOUT", 5 * time.Second},
		{"HTTP_WRITE_TIMEOUT", 35 * time.Second},
		{"HTTP_IDLE_TIMEOUT", 60 * time.Second},
		{"HTTP_SHUTDOWN_TIMEOUT", 10 * time.Second},
		{"OPENAI_TIMEOUT", 30 * time.Second},
	} {
		duration, err := getDuration(setting.name, setting.fallback)
		if err != nil {
			return nil, err
		}
		timeouts[setting.name] = duration
	}
	cfg := &Config{
		HTTP: HTTPConfig{
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     timeouts["HTTP_READ_TIMEOUT"],
			WriteTimeout:    timeouts["HTTP_WRITE_TIMEOUT"],
			IdleTimeout:     timeouts["HTTP_IDLE_TIMEOUT"],
			ShutdownTimeout: timeouts["HTTP_SHUTDOWN_TIMEOUT"],
		},

		OpenAI: OpenAIConfig{
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Model:   os.Getenv("OPENAI_MODEL"),
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
			Timeout: timeouts["OPENAI_TIMEOUT"],
		},

		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	for _, setting := range []struct {
		name  string
		value time.Duration
	}{
		{"HTTP_READ_TIMEOUT", c.HTTP.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", c.HTTP.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", c.HTTP.IdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", c.HTTP.ShutdownTimeout},
		{"OPENAI_TIMEOUT", c.OpenAI.Timeout},
	} {
		if setting.value <= 0 {
			return fmt.Errorf("%s must be positive", setting.name)
		}
	}
	if c.HTTP.WriteTimeout <= c.OpenAI.Timeout {
		return errors.New("HTTP_WRITE_TIMEOUT must exceed OPENAI_TIMEOUT")
	}

	if c.HTTP.Port == "" {
		return errors.New("HTTP_PORT is required")
	}

	if c.OpenAI.APIKey == "" {
		return errors.New("OPENAI_API_KEY is required")
	}

	if c.OpenAI.Model == "" {
		return errors.New("OPENAI_MODEL is required")
	}

	return nil
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return duration, nil
}
